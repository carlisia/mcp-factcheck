package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/carlisia/mcp-factcheck/embedding"
	mcpembedding "github.com/carlisia/mcp-factcheck/internal/embedding"
	"github.com/carlisia/mcp-factcheck/internal/specs"
	"github.com/carlisia/mcp-factcheck/pkg/logger"
	"github.com/carlisia/mcp-factcheck/pkg/telemetry"
	"github.com/mark3labs/mcp-go/mcp"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

const ValidateContentToolName = "validate_content"

// Helper function for debugging
func getKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Helper function to get content preview for logging
func getContentPreview(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "..."
}

// Helper functions for OpenInference
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func getMaxSimilarity(results []embedding.SearchResult) float64 {
	if len(results) == 0 {
		return 0.0
	}
	max := results[0].Similarity
	for _, result := range results {
		if result.Similarity > max {
			max = result.Similarity
		}
	}
	return max
}

func getMinSimilarity(results []embedding.SearchResult) float64 {
	if len(results) == 0 {
		return 0.0
	}
	min := results[0].Similarity
	for _, result := range results {
		if result.Similarity < min {
			min = result.Similarity
		}
	}
	return min
}

type ValidateContentArgs struct {
	Content     string `json:"content"`
	SpecVersion string `json:"spec_version,omitempty"`
	UseChunking bool   `json:"use_chunking,omitempty"` // Enable chunk-level validation
}

func GetValidateContentTool() mcp.Tool {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{
				"type":        "string",
				"description": "Content to validate against MCP specification. Checks accuracy of claims AND identifies missing spec requirements. Supports large documents (24k+ characters).",
			},
			"contextType": map[string]any{
				"type":        "string",
				"description": "Type of content being validated to determine which spec sections are relevant",
				"enum":        []string{"full-implementation", "client", "server", "transport", "protocol-overview", "tutorial", "documentation", "blog post"},
				"default":     "full-implementation",
			},
			"specVersion": map[string]any{
				"type":        "string",
				"description": "MCP specification version to validate against",
				"enum":        specs.ValidSpecVersions,
				"default":     specs.DefaultSpecVersion,
			},
			"useChunking": map[string]any{
				"type":        "boolean",
				"description": "Enable chunk-level validation for long content (default: false)",
				"default":     false,
			},
		},
		"required": []string{"content"},
	}
	schemaBytes, _ := json.Marshal(schema)

	description := `Strictly validate MCP content against the embedded official MCP specification. 

USE THIS WHEN YOU SEE:
- Any text explaining MCP concepts or architecture
- Documentation describing MCP features or behavior
- Claims about what MCP "does", "requires", or "supports"
- MCP protocol descriptions, explanations, tutorials, blog posts, etc.

Returns specific spec violations with section references and correct language from the official specification.

Be explicit about limitations: If validation tools show high confidence but you haven't verified specific claims, state that clearly rather than giving blanket approval.`

	return mcp.NewToolWithRawSchema(ValidateContentToolName, description, schemaBytes)
}

func HandleValidateContent(ctx context.Context, vectorDB *mcpembedding.VectorDB, generator *embedding.Generator, args any) ([]mcp.Content, error) {
	// Get structured logger with request ID
	log := logger.WithRequestID(ctx)
	
	params, ok := args.(map[string]any)
	if !ok {
		log.Error("Invalid arguments type", 
			zap.String("expected", "map[string]any"),
			zap.String("actual", fmt.Sprintf("%T", args)))
		return nil, fmt.Errorf("arguments must be a map")
	}

	log.Debug("Processing validate_content request", 
		zap.Strings("param_keys", getKeys(params)))

	content, ok := params["content"].(string)
	if !ok {
		log.Error("Invalid content parameter", 
			zap.String("expected", "string"),
			zap.String("actual", fmt.Sprintf("%T", params["content"])),
			zap.Any("value", params["content"]))
		return nil, fmt.Errorf("content must be a string")
	}

	specVersion, ok := params["specVersion"].(string)
	if !ok {
		specVersion = specs.DefaultSpecVersion
		log.Debug("Using default spec version", zap.String("version", specVersion))
	}

	useChunking, ok := params["useChunking"].(bool)
	if !ok {
		useChunking = false
	}

	if !specs.IsValidSpecVersion(specVersion) {
		log.Error("Invalid spec version", 
			zap.String("version", specVersion),
			zap.Strings("valid_versions", specs.ValidSpecVersions))
		return nil, fmt.Errorf("invalid spec version: %s", specVersion)
	}

	// Start parent span with actual content and parameters
	ctx, requestSpan := telemetry.StartValidationSpan(ctx, content, specVersion, useChunking)
	defer requestSpan.End()

	// Add structured logging for request details
	log.Info("Starting content validation", 
		zap.Int("content_length", len(content)),
		zap.String("spec_version", specVersion),
		zap.Bool("use_chunking", useChunking),
		zap.String("content_preview", getContentPreview(content, 100)))

	// Check if we should use chunking based on content length or explicit request
	shouldChunk := useChunking || len(content) > 500 // Auto-chunk for moderately long content

	var result []mcp.Content
	var err error

	if shouldChunk {
		requestSpan.SetAttributes(attribute.String("validation.strategy", "chunked"))
		result, err = HandleChunkedValidation(ctx, vectorDB, generator, content, specVersion)
	} else {
		requestSpan.SetAttributes(attribute.String("validation.strategy", "single"))
		result, err = handleSingleValidation(ctx, vectorDB, generator, content, specVersion)
	}

	// Add result attributes to parent span
	if err != nil {
		requestSpan.SetAttributes(attribute.String("validation.error", err.Error()))
		requestSpan.RecordError(err)
	} else {
		resultJSON, _ := json.Marshal(result)
		requestSpan.SetAttributes(
			attribute.String("output.value", string(resultJSON)),
			attribute.String("output.mime_type", "application/json"),
			attribute.Bool("validation.success", true),
		)
	}

	return result, err
}

// analyzeContentValidation determines if content is valid and provides insights
func analyzeContentValidation(content string, results []embedding.SearchResult, specVersion string) ValidationResult {
	if len(results) == 0 {
		return ValidationResult{
			IsValid:     false,
			Confidence:  0.1,
			Issues:      []string{"No relevant MCP specification content found"},
			SpecVersion: specVersion,
		}
	}

	// Calculate average similarity
	var totalSimilarity float64
	for _, result := range results {
		totalSimilarity += result.Similarity
	}
	avgSimilarity := totalSimilarity / float64(len(results))

	// Determine validation based on similarity thresholds
	isValid := avgSimilarity > 0.7
	confidence := avgSimilarity

	var issues []string
	var suggestions []string

	if !isValid {
		issues = append(issues, "Content may not align with MCP specification")
		if avgSimilarity < 0.5 {
			issues = append(issues, "Low similarity to MCP patterns detected")
		}
		suggestions = append(suggestions, "Review content against MCP specification")
		suggestions = append(suggestions, "Consider using standard MCP terminology and patterns")
	}

	return ValidationResult{
		IsValid:     isValid,
		Confidence:  confidence,
		Issues:      issues,
		Suggestions: suggestions,
		SpecVersion: specVersion,
	}
}

// summarizeContentMatches creates concise summaries from search results
func summarizeContentMatches(results []embedding.SearchResult, maxMatches int) []ValidationMatch {
	if maxMatches > len(results) {
		maxMatches = len(results)
	}

	var matches []ValidationMatch
	for i := 0; i < maxMatches; i++ {
		result := results[i]

		// Extract topic from content (first meaningful line)
		lines := strings.Split(result.Chunk.Content, "\n")
		topic := "MCP Specification"
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if len(line) > 0 && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "-") {
				if len(line) > 50 {
					topic = line[:50] + "..."
				} else {
					topic = line
				}
				break
			}
		}

		// Create brief summary
		summary := result.Chunk.Content
		if len(summary) > 200 {
			summary = summary[:200] + "..."
		}

		matches = append(matches, ValidationMatch{
			Topic:     topic,
			Relevance: result.Similarity,
			Summary:   summary,
		})
	}
	return matches
}

func handleSingleValidation(ctx context.Context, vectorDB *mcpembedding.VectorDB, generator *embedding.Generator, content, specVersion string) ([]mcp.Content, error) {
	// Start embedding generation span using telemetry builder
	embeddingCtx, embeddingSpan := telemetry.StartEmbeddingSpan(ctx, content)

	// Generate embedding for content
	contentEmbedding, err := generator.GenerateEmbedding(content)
	embeddingSpan.End()
	if err != nil {
		embeddingSpan.SetAttributes(attribute.String("embedding.error", err.Error()))
		embeddingSpan.RecordError(err)
		return nil, fmt.Errorf("failed to generate content embedding: %w", err)
	}

	// Start vector search span using telemetry builder
	searchCtx, searchSpan := telemetry.StartRetrievalSpan(embeddingCtx, specVersion, 5)

	// Search for relevant spec sections
	results, err := vectorDB.Search(specVersion, contentEmbedding, 5)
	if err != nil {
		searchSpan.SetAttributes(attribute.String("search.error", err.Error()))
		searchSpan.RecordError(err)
		searchSpan.End()
		return nil, fmt.Errorf("failed to search specifications: %w", err)
	}

	// Convert search results for telemetry
	var retrievalDocs []telemetry.RetrievalDocument
	var totalSimilarity float64
	for i, result := range results {
		retrievalDocs = append(retrievalDocs, telemetry.RetrievalDocument{
			ID:      fmt.Sprintf("mcp_doc_%d", i),
			Score:   result.Similarity,
			Content: result.Chunk.Content,
			Metadata: map[string]interface{}{
				"source":     "mcp_specification",
				"version":    specVersion,
				"chunk_type": "specification_section",
			},
		})
		totalSimilarity += result.Similarity
	}

	avgSimilarity := totalSimilarity / float64(len(results))

	// Add retrieval results to span using telemetry builder
	searchSpan.SetAttributes(
		attribute.String("retrieval.query", content[:min(200, len(content))]),
		attribute.Int("retrieval.top_k", 5),
		attribute.Float64("retrieval.similarity.avg", avgSimilarity),
		attribute.Float64("retrieval.similarity.max", getMaxSimilarity(results)),
		attribute.Float64("retrieval.similarity.min", getMinSimilarity(results)),
	)

	// Use telemetry builder to add retrieval documents properly
	// Note: Additional attributes could be set here if needed

	searchSpan.End()

	// Start validation analysis span using telemetry builder
	_, analysisSpan := telemetry.StartAnalysisSpan(searchCtx, len(results), avgSimilarity)

	// Analyze validation results
	validationResult := analyzeContentValidation(content, results, specVersion)
	matches := summarizeContentMatches(results, 3)

	analysisSpan.SetAttributes(
		attribute.Bool("validation.is_valid", validationResult.IsValid),
		attribute.Float64("validation.confidence", validationResult.Confidence),
		attribute.String("validation.spec_version", validationResult.SpecVersion),
	)
	analysisSpan.End()

	// Create optimized response
	response := FormatValidationResult(validationResult, matches)

	return []mcp.Content{mcp.NewTextContent(response)}, nil
}
