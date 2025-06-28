package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/carlisia/mcp-factcheck/embedding"
	mcpembedding "github.com/carlisia/mcp-factcheck/internal/embedding"
	"github.com/carlisia/mcp-factcheck/pkg/telemetry"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/tmc/langchaingo/textsplitter"
	"go.opentelemetry.io/otel/attribute"
)

// ContentChunk represents a logical piece of content for validation
type ContentChunk struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Position int    `json:"position"`
	Type     string `json:"type"` // "paragraph", "heading", "code_block", "list_item"
	Level    int    `json:"level,omitempty"` // For headings (1-6)
}

// ChunkingResult contains the chunked content and metadata
type ChunkingResult struct {
	Chunks      []ContentChunk `json:"chunks"`
	TotalChunks int           `json:"total_chunks"`
	TotalChars  int           `json:"total_chars"`
	EstTokens   int           `json:"estimated_tokens"`
}

// ChunkContent splits content into logical chunks for validation using langchaingo
func ChunkContent(content string) *ChunkingResult {
	if strings.TrimSpace(content) == "" {
		return &ChunkingResult{
			Chunks:      []ContentChunk{},
			TotalChunks: 0,
			TotalChars:  0,
			EstTokens:   0,
		}
	}

	// Choose splitter based on content type
	var splitter textsplitter.TextSplitter
	
	// Use markdown splitter if content contains markdown-like patterns
	if strings.Contains(content, "#") || strings.Contains(content, "```") || 
	   strings.Contains(content, "- ") || strings.Contains(content, "* ") {
		splitter = textsplitter.NewMarkdownTextSplitter(
			textsplitter.WithChunkSize(800),    // Smaller chunks for better granularity
			textsplitter.WithChunkOverlap(100), // Overlap for context preservation
		)
	} else {
		// Use recursive character splitter for plain text
		splitter = textsplitter.NewRecursiveCharacter(
			textsplitter.WithChunkSize(800),    // Smaller chunks for better granularity
			textsplitter.WithChunkOverlap(100), // Overlap for context preservation
		)
	}
	
	// Split the content
	docs, err := splitter.SplitText(content)
	if err != nil {
		// Fallback to simple splitting if the splitter fails
		docs = []string{content}
	}

	// Convert to our ContentChunk format
	chunks := make([]ContentChunk, len(docs))
	for i, doc := range docs {
		chunks[i] = ContentChunk{
			ID:       generateChunkID("chunk", i),
			Text:     strings.TrimSpace(doc),
			Position: i,
			Type:     "text_chunk", // langchaingo doesn't classify types, so use generic
		}
	}

	// Calculate metadata
	totalChars := len(content)
	estTokens := totalChars / 4 // Rough approximation

	return &ChunkingResult{
		Chunks:      chunks,
		TotalChunks: len(chunks),
		TotalChars:  totalChars,
		EstTokens:   estTokens,
	}
}


func generateChunkID(prefix string, position int) string {
	return fmt.Sprintf("%s-%d", prefix, position)
}

// ChunkValidationResult represents validation results for a single chunk
type ChunkValidationResult struct {
	Chunk      ContentChunk       `json:"chunk"`
	Validation ValidationResult   `json:"validation,omitempty"`
	Matches    []ValidationMatch  `json:"matches,omitempty"`
	Error      string            `json:"error,omitempty"`
}

// AggregatedValidationResult contains validation results for all chunks
type AggregatedValidationResult struct {
	ChunkResults []ChunkValidationResult `json:"chunk_results"`
	Overall      ValidationResult        `json:"overall_validation"`
	Summary      string                 `json:"summary"`
	SpecVersion  string                 `json:"spec_version"`
}

// HandleChunkedValidation processes long content by chunking it and validating each piece
func HandleChunkedValidation(ctx context.Context, vectorDB *mcpembedding.VectorDB, generator *embedding.Generator, content, specVersion string) ([]mcp.Content, error) {
	// Start content chunking span using telemetry builder
	ctx, chunkingSpan := telemetry.NewSpanBuilder().
		WithKind("CHAIN").
		WithInput(content, "text/plain").
		WithCustom(
			attribute.String("session.id", "chunked-validation"),
			attribute.Int("content.length", len(content)),
			attribute.Int("content.estimated_tokens", len(content)/4),
		).
		Start(ctx, "content.chunking")
	defer chunkingSpan.End()
	
	// Chunk the content
	chunkingResult := ChunkContent(content)
	
	// Add chunking results to span using OpenInference conventions
	chunkingSpan.SetAttributes(
		attribute.Int("chunks.total", chunkingResult.TotalChunks),
		attribute.Int("chunks.total_chars", chunkingResult.TotalChars),
		attribute.Int("chunks.estimated_tokens", chunkingResult.EstTokens),
		attribute.String("output.mime_type", "application/json"),
	)
	
	if len(chunkingResult.Chunks) == 0 {
		return nil, fmt.Errorf("no valid chunks found in content")
	}
	
	// Validate each chunk
	var chunkResults []ChunkValidationResult
	var totalSimilarity float64
	var totalChunks int
	
	for _, chunk := range chunkingResult.Chunks {
		// Start span for individual chunk validation using telemetry builder
		chunkCtx, chunkSpan := telemetry.NewSpanBuilder().
			WithKind("CHAIN").
			WithInput(chunk.Text, "text/plain").
			WithCustom(
				attribute.String("chunk.id", chunk.ID),
				attribute.String("chunk.type", chunk.Type),
				attribute.Int("chunk.length", len(chunk.Text)),
			).
			Start(ctx, "chunk.validation")
		
		// Generate embedding for this chunk using telemetry builder
		embeddingCtx, embeddingSpan := telemetry.StartEmbeddingSpan(chunkCtx, chunk.Text)
		
		chunkEmbedding, err := generator.GenerateEmbedding(chunk.Text)
		embeddingSpan.End()
		
		if err != nil {
			embeddingSpan.SetAttributes(attribute.String("embedding.error", err.Error()))
			embeddingSpan.RecordError(err)
			chunkSpan.SetAttributes(attribute.String("chunk.error", err.Error()))
			chunkSpan.RecordError(err)
			chunkSpan.End()
			
			chunkResults = append(chunkResults, ChunkValidationResult{
				Chunk: chunk,
				Error: fmt.Sprintf("failed to generate embedding: %v", err),
			})
			continue
		}
		
		// Search for relevant spec sections using telemetry builder
		searchCtx, searchSpan := telemetry.StartRetrievalSpan(embeddingCtx, specVersion, 3)
		searchSpan.SetAttributes(attribute.String("chunk_id", chunk.ID))
		
		results, err := vectorDB.Search(specVersion, chunkEmbedding, 3)
		
		if err != nil {
			searchSpan.SetAttributes(attribute.String("search.error", err.Error()))
			searchSpan.RecordError(err)
			searchSpan.End()
			chunkSpan.SetAttributes(attribute.String("chunk.error", err.Error()))
			chunkSpan.RecordError(err)
			chunkSpan.End()
			
			chunkResults = append(chunkResults, ChunkValidationResult{
				Chunk: chunk,
				Error: fmt.Sprintf("failed to search specifications: %v", err),
			})
			continue
		}
		
		// Calculate search results metrics
		var avgSimilarity float64
		if len(results) > 0 {
			var totalSim float64
			for _, result := range results {
				totalSim += result.Similarity
			}
			avgSimilarity = totalSim / float64(len(results))
		}
		
		searchSpan.SetAttributes(
			attribute.Int("document_count", len(results)),
			attribute.Float64("avg_similarity", avgSimilarity),
			attribute.Bool("has_results", len(results) > 0),
		)
		searchSpan.End()
		
		// Analyze validation for this chunk
		validation := analyzeChunkValidation(chunk.Text, results, specVersion)
		matches := summarizeChunkMatches(results, 2)
		
		// Add chunk validation results to span
		chunkSpan.SetAttributes(
			attribute.Float64("chunk.confidence", validation.Confidence),
			attribute.Bool("chunk.is_valid", validation.IsValid),
			attribute.Int("chunk.matches_count", len(matches)),
			attribute.String("output.mime_type", "application/json"),
		)
		chunkSpan.End()
		
		chunkResults = append(chunkResults, ChunkValidationResult{
			Chunk:      chunk,
			Validation: validation,
			Matches:    matches,
		})
		
		// Track overall metrics
		totalSimilarity += validation.Confidence
		totalChunks++
		
		// Use searchCtx to keep context chain
		_ = searchCtx
	}
	
	// Create overall validation summary
	avgConfidence := totalSimilarity / float64(totalChunks)
	overallValidation := ValidationResult{
		IsValid:     avgConfidence > 0.7,
		Confidence:  avgConfidence,
		SpecVersion: specVersion,
	}
	
	// Set overall issues and suggestions
	if !overallValidation.IsValid {
		overallValidation.Issues = []string{
			fmt.Sprintf("%d chunks analyzed with average confidence %.2f", totalChunks, avgConfidence),
		}
		if avgConfidence < 0.5 {
			overallValidation.Issues = append(overallValidation.Issues, "Multiple sections show low alignment with MCP specification")
		}
		overallValidation.Suggestions = []string{
			"Review flagged sections against MCP specification",
			"Consider using standard MCP terminology throughout",
		}
	}
	
	// Create aggregated result
	aggregated := AggregatedValidationResult{
		ChunkResults: chunkResults,
		Overall:      overallValidation,
		Summary:      fmt.Sprintf("Analyzed %d content chunks", len(chunkResults)),
		SpecVersion:  specVersion,
	}
	
	// Format response
	response := FormatChunkedValidationResult(aggregated)
	return []mcp.Content{mcp.NewTextContent(response)}, nil
}

// analyzeChunkValidation determines if a chunk is valid and provides insights
func analyzeChunkValidation(content string, results []embedding.SearchResult, specVersion string) ValidationResult {
	if len(results) == 0 {
		return ValidationResult{
			IsValid:     false,
			Confidence:  0.1,
			Issues:      []string{"No relevant MCP specification content found for this section"},
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
		issues = append(issues, "Content section may not align with MCP specification")
		if avgSimilarity < 0.5 {
			issues = append(issues, "Low similarity to MCP patterns detected")
		}
		suggestions = append(suggestions, "Review this section against MCP specification")
		suggestions = append(suggestions, "Consider using standard MCP terminology")
	}
	
	return ValidationResult{
		IsValid:     isValid,
		Confidence:  confidence,
		Issues:      issues,
		Suggestions: suggestions,
		SpecVersion: specVersion,
	}
}

// summarizeChunkMatches creates concise summaries from search results for a chunk
func summarizeChunkMatches(results []embedding.SearchResult, maxMatches int) []ValidationMatch {
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
		if len(summary) > 150 {
			summary = summary[:150] + "..."
		}
		
		matches = append(matches, ValidationMatch{
			Topic:     topic,
			Relevance: result.Similarity,
			Summary:   summary,
		})
	}
	return matches
}

// FormatChunkedValidationResult creates a structured response for chunked validation
func FormatChunkedValidationResult(result AggregatedValidationResult) string {
	response := map[string]interface{}{
		"validation_type": "chunked_content",
		"total_chunks":    len(result.ChunkResults),
		"overall":         result.Overall,
		"summary":         result.Summary,
		"spec_version":    result.SpecVersion,
		"chunk_details":   result.ChunkResults,
	}
	
	jsonBytes, _ := json.MarshalIndent(response, "", "  ")
	return string(jsonBytes)
}