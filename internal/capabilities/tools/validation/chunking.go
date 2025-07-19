package validation

import (
	"context"
	"fmt"

	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools"
	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools/contentprep"
)

const (
	// chunkSearchTopK defines how many spec matches to retrieve per chunk
	chunkSearchTopK = 5
)

// ChunkValidationResult represents validation results for a single chunk
type ChunkValidationResult struct {
	Chunk        contentprep.Chunk    `json:"chunk"`
	Validation   *Result              `json:"validation,omitempty"`
	SearchResult []tools.SearchResult `json:"search_results,omitempty"`
	Error        string               `json:"error,omitempty"`
}

// AggregatedValidationResult contains validation results for all chunks
type AggregatedValidationResult struct {
	ChunkResults []ChunkValidationResult `json:"chunk_results"`
	Overall      *Result                 `json:"overall_validation"`
	Summary      string                  `json:"summary"`
	SpecVersion  string                  `json:"spec_version"`
}

// validateWithChunkingComprehensive performs comprehensive chunk-based validation
func validateWithChunkingComprehensive(ctx context.Context, req ClaimsRequest, embedFunc tools.EmbeddingFunc, searchFunc tools.SearchFunc, llmFunc LLMCompleteFunc) (*Result, error) {
	// Use contentprep.Split for sophisticated chunking
	chunkResult := contentprep.Split(req.Content)

	if len(chunkResult.Chunks) == 0 {
		return nil, fmt.Errorf("no valid chunks found in content")
	}

	// Use batch validation for better performance
	batchResults, err := validateChunksBatch(ctx, chunkResult.Chunks, req.SpecVersion, embedFunc, searchFunc, llmFunc)
	if err != nil {
		return nil, fmt.Errorf("batch validation failed: %w", err)
	}

	// Aggregate and return results
	return aggregateChunkResults(batchResults, req.Content, req.SpecVersion, len(chunkResult.Chunks))
}

// validateChunksBatch validates multiple chunks and attempts to use batch embedding if available
func validateChunksBatch(ctx context.Context, chunks []contentprep.Chunk, specVersion string,
	embedFunc tools.EmbeddingFunc, searchFunc tools.SearchFunc, llmFunc LLMCompleteFunc) ([]ChunkValidationResult, error) {

	// Extract text from all chunks for batch embedding
	chunkTexts := make([]string, len(chunks))
	for i, chunk := range chunks {
		chunkTexts[i] = chunk.Text
	}

	// Generate embeddings for all chunks
	// Check if batch embedding is available in context
	embeddings := make([][]float64, len(chunks))
	embeddingErrors := make([]error, len(chunks))

	// Try to get batch embed function from context
	batchEmbedFunc, hasBatch := ctx.Value(tools.ContextKeyBatchEmbedFunc).(tools.BatchEmbeddingFunc)
	if hasBatch && len(chunks) > 1 {
		// Use batch embedding for better performance
		batchEmbeddings, err := batchEmbedFunc(ctx, chunkTexts)
		if err != nil {
			// If batch fails, fall back to sequential
			hasBatch = false
		} else {
			embeddings = batchEmbeddings
		}
	}

	// If batch not available or failed, use sequential embedding
	if !hasBatch || len(chunks) == 1 {
		// Fall back to sequential embedding, but don't fail early
		for i, text := range chunkTexts {
			embedding, err := embedFunc(ctx, text)
			if err != nil {
				embeddingErrors[i] = fmt.Errorf("failed to embed chunk %d: %w", i, err)
				// Continue processing other chunks
				continue
			}
			embeddings[i] = embedding
		}
	}

	// Validate each chunk with its embedding
	results := make([]ChunkValidationResult, len(chunks))
	for i, chunk := range chunks {
		// Check if this chunk had an embedding error
		if embeddingErrors[i] != nil {
			results[i] = ChunkValidationResult{
				Chunk: chunk,
				Error: embeddingErrors[i].Error(),
			}
			continue
		}

		// Skip chunks without embeddings
		if embeddings[i] == nil || len(embeddings[i]) == 0 {
			results[i] = ChunkValidationResult{
				Chunk: chunk,
				Error: "no embedding available for chunk",
			}
			continue
		}

		// Search for relevant spec sections
		var searchResults []tools.SearchResult
		var err error

		// Check if this chunk contains negative claims that need special search
		if isNegativeClaim(chunk.Text) {
			// Use enhanced search for negative claims
			searchResults, err = searchForNegativeClaim(ctx, chunk.Text, specVersion, embedFunc, searchFunc)
		} else {
			// Use the pre-computed embedding for regular search
			searchResults, err = searchFunc(specVersion, embeddings[i], chunkSearchTopK)
		}

		if err != nil {
			results[i] = ChunkValidationResult{
				Chunk: chunk,
				Error: fmt.Sprintf("failed to search specifications: %v", err),
			}
			continue
		}

		// Perform fact-checking on this chunk
		factCheckResult, err := performClaimCheck(ctx, llmFunc, chunk.Text, searchResults)
		if err != nil {
			results[i] = ChunkValidationResult{
				Chunk: chunk,
				Error: fmt.Sprintf("validation failed: %v", err),
			}
			continue
		}

		// Build validation result for this chunk
		result := &Result{
			IsValid:          factCheckResult.IsAccurate,
			Confidence:       factCheckResult.Confidence,
			ParsedClaims:     factCheckResult.ParsedClaims,
			Issues:           factCheckResult.Inaccuracies,
			Suggestions:      factCheckResult.Suggestions,
			CorrectedVersion: factCheckResult.CorrectedVersion,
			SpecVersion:      specVersion,
			FactCheckResult:  factCheckResult,
		}

		results[i] = ChunkValidationResult{
			Chunk:        chunk,
			Validation:   result,
			SearchResult: searchResults,
		}
	}

	return results, nil
}

// aggregateChunkResults combines results from multiple chunk validations into a single result
func aggregateChunkResults(batchResults []ChunkValidationResult, originalContent, specVersion string, totalChunks int) (*Result, error) {
	var allClaims []Claim
	var allParsedClaims []string
	var allIssues []string
	var allSuggestions []string
	var allMissingBestPractices []string
	var totalConfidence float64
	var validChunks int
	var processedChunks int

	for _, chunkValidationResult := range batchResults {
		if chunkValidationResult.Error != "" {
			continue
		}

		processedChunks++

		if chunkValidationResult.Validation != nil {
			result := chunkValidationResult.Validation

			// Aggregate claims
			if result.FactCheckResult != nil {
				allClaims = append(allClaims, result.FactCheckResult.Claims...)
				allParsedClaims = append(allParsedClaims, result.FactCheckResult.ParsedClaims...)
				allIssues = append(allIssues, result.FactCheckResult.Inaccuracies...)
				allSuggestions = append(allSuggestions, result.FactCheckResult.Suggestions...)
				allMissingBestPractices = append(allMissingBestPractices, result.FactCheckResult.MissingBestPractices...)
			}

			totalConfidence += result.Confidence
			if result.IsValid {
				validChunks++
			}
		}
	}

	// Calculate overall results
	avgConfidence := 0.0
	if processedChunks > 0 {
		avgConfidence = totalConfidence / float64(processedChunks)
	}

	// Overall is valid only if ALL chunks are valid
	isValid := validChunks == totalChunks && processedChunks > 0

	// Add chunk processing information
	if processedChunks < totalChunks {
		errorSummary := fmt.Sprintf("Warning: %d of %d chunks failed validation",
			totalChunks-processedChunks, totalChunks)
		allIssues = append([]string{errorSummary}, allIssues...)
	}

	// Create corrected version if needed
	correctedVersion := ""
	if !isValid && len(allClaims) > 0 {
		correctedVersion = buildCorrectedVersion(originalContent, allClaims)
	}

	// Build final result
	return &Result{
		IsValid:          isValid,
		Confidence:       avgConfidence,
		ParsedClaims:     deduplicate(allParsedClaims),
		Issues:           deduplicate(allIssues),
		Suggestions:      deduplicate(allSuggestions),
		CorrectedVersion: correctedVersion,
		SpecVersion:      specVersion,
		FactCheckResult: &FactCheckResult{
			IsAccurate:           isValid,
			Confidence:           avgConfidence,
			ParsedClaims:         allParsedClaims,
			Inaccuracies:         deduplicate(allIssues),
			Suggestions:          deduplicate(allSuggestions),
			MissingBestPractices: deduplicate(allMissingBestPractices),
			CorrectedVersion:     correctedVersion,
			Claims:               allClaims,
		},
	}, nil
}
