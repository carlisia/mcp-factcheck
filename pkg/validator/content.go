package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/carlisia/mcp-factcheck/embedding"
	mcpembedding "github.com/carlisia/mcp-factcheck/internal/embedding"
	"github.com/carlisia/mcp-factcheck/internal/specs"
	"github.com/carlisia/mcp-factcheck/internal/utils"
	"github.com/carlisia/mcp-factcheck/pkg/logger"
	"github.com/carlisia/mcp-factcheck/pkg/telemetry"
	"github.com/mark3labs/mcp-go/mcp"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

const (
	checkMCPClaimToolName = "check_mcp_claim"

	// Search configuration
	defaultSearchTopK = 20 // Number of spec sections to retrieve (increased for better coverage)
	chunkSearchTopK   = 10 // Number of spec sections per chunk (increased)

	// Display configuration
	defaultMatchesShown = 5 // Number of matches to show in results
	chunkMatchesShown   = 2 // Number of matches per chunk

	// Validation thresholds
	similarityValidThreshold = 0.7 // Similarity score above which content is considered valid
	similarityLowThreshold   = 0.5 // Below this is considered low similarity

	// Content processing
	autoChunkThreshold = 2000 // Content length above which to auto-enable chunking
	contentPreviewLen  = 100  // Length of content preview in logs

	// Confidence scores
	highConfidence    = 0.9 // Confidence when fact-check passes
	lowConfidence     = 0.3 // Confidence when fact-check fails
	minimumConfidence = 0.1 // Minimum confidence when no results
)

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

// truncateString safely truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

type ValidateContentArgs struct {
	Content     string `json:"content"`
	SpecVersion string `json:"spec_version,omitempty"`
	UseChunking bool   `json:"use_chunking,omitempty"` // Enable chunk-level validation
}

// Package-level stability checker (shared across validations in a session)
var stabilityChecker = NewContentStabilityChecker()

// Package-level validation counter for debugging
var validationCounter int

func GetCheckMCPClaimTool() mcp.Tool {
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
	schemaBytes := utils.MustMarshalSchema(schema, checkMCPClaimToolName)

	description := `Validate any MCP-related content against the official specification to check accuracy and completeness.

USE THIS TOOL WHEN:
- User provides ANY text about MCP (even if they don't explicitly ask for validation)
- Text contains multiple statements about MCP capabilities, design, or behavior
- Content describes what MCP does, doesn't do, or how it works
- Documentation, tutorials, or explanations about MCP
- Bullet points or lists describing MCP features

EXAMPLES OF CONTENT TO VALIDATE:
- "MCP server exposes Resources and Tools..."
- "Never forwards raw model traffic; enforces ACLs..."
- Any technical description of MCP functionality

RETURNS:
A complete validation workflow with:
- All claims extracted and individually validated
- Corrections for any inaccuracies
- Missing best practices identified
- Overall accuracy assessment

For single-sentence questions like "Does MCP support X?", use check_mcp_quick_fact instead.`

	return mcp.NewToolWithRawSchema(checkMCPClaimToolName, description, schemaBytes)
}

func HandleCheckMCPClaim(ctx context.Context, vectorDB *mcpembedding.VectorDB, generator *embedding.Generator, args any) ([]mcp.Content, error) {
	// Get structured logger with request ID
	log := logger.WithRequestID(ctx)

	params, ok := args.(map[string]any)
	if !ok {
		log.Error("Invalid arguments type",
			zap.String("expected", "map[string]any"),
			zap.String("actual", fmt.Sprintf("%T", args)))
		return nil, errArgumentsNotMap
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
		zap.String("content_preview", getContentPreview(content, contentPreviewLen)))

	// Check if we should use chunking based on content length or explicit request
	shouldChunk := useChunking || len(content) > autoChunkThreshold

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
func analyzeContentValidation(ctx context.Context, vectorDB *mcpembedding.VectorDB, generator *embedding.Generator, content string, results []embedding.SearchResult, specVersion string) ValidationResult {
	// Increment validation counter for debugging
	validationCounter++

	// Initialize debug info
	debugInfo := &ValidationDebugInfo{
		Timestamp:           time.Now().Format(time.RFC3339),
		ValidationIteration: validationCounter,
		SearchQueries:       []string{content}, // The content itself was the search query
		TopSpecMatches:      []SpecMatchDebug{},
		ClaimAnalysis:       []ClaimDebugInfo{},
	}

	if len(results) == 0 {
		return ValidationResult{
			IsValid:     false,
			Confidence:  minimumConfidence,
			Issues:      []string{"No relevant MCP specification content found"},
			SpecVersion: specVersion,
			DebugInfo:   debugInfo,
		}
	}

	// Extract spec sections for fact-checking and populate debug info
	var specSections []string
	for i, result := range results {
		specSections = append(specSections, result.Chunk.Content)

		// Add top 3 matches to debug info
		if i < 3 {
			debugInfo.TopSpecMatches = append(debugInfo.TopSpecMatches, SpecMatchDebug{
				Content:    result.Chunk.Content,
				Similarity: result.Similarity,
				ChunkID:    result.Chunk.ID,
			})
		}
	}

	// Pre-analyze compound claims
	compoundEvidence := make(map[string]string)
	log := logger.WithRequestID(ctx)

	// Simple heuristic: look for "and" in the content
	if strings.Contains(strings.ToLower(content), " and ") {
		log.Debug("Detected potential compound claims in content")

		// Extract potential compound claims
		// First try to split by sentences, but if no periods, use the whole content
		var claimsToCheck []string
		if strings.Contains(content, ". ") {
			claimsToCheck = strings.Split(content, ". ")
		} else {
			claimsToCheck = []string{content}
		}

		for _, claim := range claimsToCheck {
			claim = strings.TrimSpace(claim)
			if strings.Contains(strings.ToLower(claim), " and ") {
				compound := DecomposeCompoundClaim(claim)
				if compound.IsCompound {
					log.Debug("Processing compound claim",
						zap.String("claim", compound.OriginalClaim),
						zap.Int("subclaim_count", len(compound.SubClaims)))

					// Search for evidence for each subclaim
					err := SearchEvidenceForSubClaims(ctx, &compound, vectorDB, generator, specVersion, 10)
					if err == nil {
						evidence := FormatCompoundClaimEvidence(compound)
						compoundEvidence[compound.OriginalClaim] = evidence

						log.Debug("Generated compound evidence",
							zap.String("claim", compound.OriginalClaim),
							zap.String("evidence_summary", truncateString(evidence, 200)))
					} else {
						log.Warn("Failed to search evidence for compound claim",
							zap.String("claim", compound.OriginalClaim),
							zap.Error(err))
					}
				}
			}
		}

		log.Debug("Compound claim analysis complete",
			zap.Int("compound_count", len(compoundEvidence)))
	}

	// Use LLM to fact-check the content against spec sections
	factCheckResult, err := generator.FactCheckAgainstSpec(ctx, content, specSections, compoundEvidence)
	if err != nil {
		// Fallback to similarity-based validation if fact-checking fails
		log.Error("Fact-checking failed, falling back to similarity validation", zap.Error(err))

		// Calculate average similarity
		var totalSimilarity float64
		for _, result := range results {
			totalSimilarity += result.Similarity
		}
		avgSimilarity := totalSimilarity / float64(len(results))

		return ValidationResult{
			IsValid:     avgSimilarity > similarityValidThreshold,
			Confidence:  avgSimilarity,
			Issues:      []string{"Fact-checking unavailable, using similarity-based validation"},
			SpecVersion: specVersion,
			DebugInfo:   debugInfo,
		}
	}

	// Populate debug info with LLM reasoning if available
	if factCheckResult.RawResponse != "" {
		debugInfo.LLMReasoning = factCheckResult.RawResponse
	}

	// Populate claim analysis debug info
	for _, claim := range factCheckResult.Claims {
		claimDebug := ClaimDebugInfo{
			OriginalClaim:    claim.Claim,
			ValidationStatus: "valid",
			Confidence:       highConfidence,
		}

		if !claim.IsAccurate {
			claimDebug.ValidationStatus = "invalid"
			claimDebug.Confidence = lowConfidence
			if claim.Explanation != "" {
				claimDebug.Issues = []string{claim.Explanation}
			}
		}

		// Add spec evidence (would need to enhance FactCheckResult to include this)
		claimDebug.SpecEvidence = []string{claim.Explanation}

		debugInfo.ClaimAnalysis = append(debugInfo.ClaimAnalysis, claimDebug)
	}

	// Build validation result from fact-check
	var correctedVersion string
	if !factCheckResult.IsAccurate && len(factCheckResult.Corrections) > 0 {
		// Combine corrections into a suggested version
		correctedVersion = "Suggested corrections:\n"
		for i, correction := range factCheckResult.Corrections {
			if i < len(factCheckResult.Inaccuracies) {
				correctedVersion += fmt.Sprintf("- %s → %s\n", factCheckResult.Inaccuracies[i], correction)
			}
		}
	}

	// Calculate confidence based on fact-check results
	confidence := highConfidence
	if !factCheckResult.IsAccurate {
		confidence = lowConfidence
	}

	// Log detailed debug info for circular validation issues
	log.Debug("Validation analysis complete",
		zap.Int("iteration", validationCounter),
		zap.Bool("is_accurate", factCheckResult.IsAccurate),
		zap.Float64("confidence", confidence),
		zap.Int("claims_count", len(factCheckResult.Claims)),
		zap.Int("issues_count", len(factCheckResult.Inaccuracies)),
		zap.String("content_preview", getContentPreview(content, 100)),
	)

	return ValidationResult{
		IsValid:          factCheckResult.IsAccurate,
		Confidence:       confidence,
		ParsedClaims:     factCheckResult.ParsedClaims,
		Issues:           factCheckResult.Inaccuracies,
		Suggestions:      factCheckResult.Corrections,
		CorrectedVersion: correctedVersion,
		SpecVersion:      specVersion,
		FactCheckResult:  factCheckResult,
		DebugInfo:        debugInfo,
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

// performTargetedSearches searches for specific concepts mentioned in the content
func performTargetedSearches(ctx context.Context, vectorDB *mcpembedding.VectorDB, generator *embedding.Generator, content, specVersion string) []embedding.SearchResult {
	var allResults []embedding.SearchResult
	log := logger.WithRequestID(ctx)

	// Extract key concepts from content
	concepts := extractKeyConcepts(content)

	log.Debug("Performing targeted searches for concepts",
		zap.Strings("concepts", concepts),
		zap.Int("concept_count", len(concepts)))

	for _, concept := range concepts {
		// Get expanded queries for this concept
		queries := expandClaimForSearch(concept)

		for _, query := range queries {
			// Generate embedding for the query
			queryEmbedding, err := generator.GenerateEmbedding(ctx, query)
			if err != nil {
				log.Warn("Failed to generate embedding for query",
					zap.String("query", query),
					zap.Error(err))
				continue
			}

			// Search with this specific query
			results, err := vectorDB.Search(specVersion, queryEmbedding, 5)
			if err != nil {
				log.Warn("Failed to search for query",
					zap.String("query", query),
					zap.Error(err))
				continue
			}

			allResults = append(allResults, results...)
		}
	}

	return allResults
}

// extractKeyConcepts extracts important concepts from content for targeted searches
func extractKeyConcepts(content string) []string {
	concepts := []string{}
	normalized := strings.ToLower(content)

	// Look for key protocol concepts
	if strings.Contains(normalized, "initialization") {
		concepts = append(concepts, "initialization phase client server")
	}
	if strings.Contains(normalized, "capability") || strings.Contains(normalized, "capabilities") {
		concepts = append(concepts, "exchange capabilities")
	}
	if strings.Contains(normalized, "tools/call") {
		concepts = append(concepts, "tools/call request")
	}
	if strings.Contains(normalized, "protocol version") {
		concepts = append(concepts, "protocol version compatibility")
	}
	if strings.Contains(normalized, "security") {
		concepts = append(concepts, "security best practices")
	}

	return concepts
}

// mergeSearchResults merges and deduplicates search results
func mergeSearchResults(primary, additional []embedding.SearchResult, maxResults int) []embedding.SearchResult {
	// Use a map to track unique chunks by ID
	seen := make(map[string]bool)
	merged := []embedding.SearchResult{}

	// Add primary results first
	for _, result := range primary {
		if !seen[result.Chunk.ID] {
			seen[result.Chunk.ID] = true
			merged = append(merged, result)
		}
	}

	// Add additional results
	for _, result := range additional {
		if !seen[result.Chunk.ID] && len(merged) < maxResults {
			seen[result.Chunk.ID] = true
			merged = append(merged, result)
		}
	}

	return merged
}

// formatDebugInfo formats debug information for display
func formatDebugInfo(debug *ValidationDebugInfo) string {
	var sections []string

	sections = append(sections, "## 🔍 DEBUG INFORMATION")
	sections = append(sections, fmt.Sprintf("**Timestamp:** %s", debug.Timestamp))
	sections = append(sections, fmt.Sprintf("**Validation Iteration:** %d", debug.ValidationIteration))

	if len(debug.TopSpecMatches) > 0 {
		sections = append(sections, "\n### Top Spec Matches:")
		for i, match := range debug.TopSpecMatches {
			sections = append(sections, fmt.Sprintf("\n**Match %d (similarity: %.3f)**", i+1, match.Similarity))
			sections = append(sections, fmt.Sprintf("```\n%s\n```", match.Content))
		}
	}

	if len(debug.ClaimAnalysis) > 0 {
		sections = append(sections, "\n### Claim-by-Claim Analysis:")
		for _, claim := range debug.ClaimAnalysis {
			sections = append(sections, fmt.Sprintf("\n**Claim:** \"%s\"", claim.OriginalClaim))
			sections = append(sections, fmt.Sprintf("- Status: %s (confidence: %.2f)", claim.ValidationStatus, claim.Confidence))
			if len(claim.Issues) > 0 {
				sections = append(sections, fmt.Sprintf("- Issues: %s", strings.Join(claim.Issues, "; ")))
			}
			if len(claim.SpecEvidence) > 0 {
				sections = append(sections, fmt.Sprintf("- Evidence: %s", strings.Join(claim.SpecEvidence, "; ")))
			}
		}
	}

	if debug.LLMReasoning != "" {
		sections = append(sections, "\n### LLM Raw Response:")
		sections = append(sections, "```json")
		sections = append(sections, debug.LLMReasoning)
		sections = append(sections, "```")
	}

	return strings.Join(sections, "\n")
}

func handleSingleValidation(ctx context.Context, vectorDB *mcpembedding.VectorDB, generator *embedding.Generator, content, specVersion string) ([]mcp.Content, error) {
	// Start embedding generation span using telemetry builder
	embeddingCtx, embeddingSpan := telemetry.StartEmbeddingSpan(ctx, content)

	// Generate embedding for content
	contentEmbedding, err := generator.GenerateEmbedding(embeddingCtx, content)
	embeddingSpan.End()
	if err != nil {
		embeddingSpan.SetAttributes(attribute.String("embedding.error", err.Error()))
		embeddingSpan.RecordError(err)
		return nil, fmt.Errorf("failed to generate content embedding: %w", err)
	}

	// Start vector search span using telemetry builder
	searchCtx, searchSpan := telemetry.StartRetrievalSpan(embeddingCtx, specVersion, defaultSearchTopK)

	// Search for relevant spec sections using the original content
	results, err := vectorDB.Search(specVersion, contentEmbedding, defaultSearchTopK)
	if err != nil {
		searchSpan.SetAttributes(attribute.String("search.error", err.Error()))
		searchSpan.RecordError(err)
		searchSpan.End()
		return nil, fmt.Errorf("failed to search specifications: %w", err)
	}

	// Additionally, perform targeted searches for key concepts mentioned in the content
	// This helps find specific protocol details that might not match the overall content embedding
	additionalResults := performTargetedSearches(ctx, vectorDB, generator, content, specVersion)

	// Merge and deduplicate results
	results = mergeSearchResults(results, additionalResults, defaultSearchTopK*2)

	// Calculate average similarity for telemetry
	var totalSimilarity float64
	for _, result := range results {
		totalSimilarity += result.Similarity
	}

	avgSimilarity := totalSimilarity / float64(len(results))

	// Add retrieval results to span using telemetry builder
	searchSpan.SetAttributes(
		attribute.String("retrieval.query", truncateString(content, 200)),
		attribute.Int("retrieval.top_k", defaultSearchTopK),
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
	validationResult := analyzeContentValidation(searchCtx, vectorDB, generator, content, results, specVersion)

	analysisSpan.SetAttributes(
		attribute.Bool("validation.is_valid", validationResult.IsValid),
		attribute.Float64("validation.confidence", validationResult.Confidence),
		attribute.String("validation.spec_version", validationResult.SpecVersion),
	)
	analysisSpan.End()

	// Check content stability before formatting
	var stabilityMessage string
	if validationResult.CorrectedVersion != "" {
		stability := stabilityChecker.CheckStability(content, validationResult.CorrectedVersion)
		stabilityMessage = stability.GetStabilityMessage()

		// Log stability analysis for debugging
		if stability.IsStable || stability.IsInLoop {
			logger.WithRequestID(ctx).Warn("Content stability issue detected",
				zap.Bool("is_stable", stability.IsStable),
				zap.Bool("is_in_loop", stability.IsInLoop),
				zap.Int("loop_length", stability.LoopLength),
				zap.String("original_normalized", stability.NormalizedOriginal),
				zap.String("validated_normalized", stability.NormalizedValidated),
			)
		}
	}

	// Create response using template formatting
	matches := summarizeContentMatches(results, defaultMatchesShown)
	formatted, err := FormatWithTemplate(validationResult, matches)
	if err != nil {
		// Fallback to direct formatting if template fails
		workflow := FormatValidationWorkflow(validationResult, content)
		return []mcp.Content{mcp.NewTextContent(workflow)}, nil
	}

	// Prepend stability message if present
	if stabilityMessage != "" {
		formatted = stabilityMessage + "\n\n" + formatted
	}

	// Append debug info if available (controlled by environment variable)
	if validationResult.DebugInfo != nil && os.Getenv("MCP_DEBUG") == "true" {
		debugSection := formatDebugInfo(validationResult.DebugInfo)
		formatted = formatted + "\n\n" + debugSection
	}

	return []mcp.Content{mcp.NewTextContent(formatted)}, nil
}
