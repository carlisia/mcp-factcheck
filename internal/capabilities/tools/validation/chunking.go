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
	// chunkMatchesShown defines how many matches to show per chunk in results
	chunkMatchesShown = 3
)

// ChunkValidationResult represents validation results for a single chunk
type ChunkValidationResult struct {
	Chunk        contentprep.Chunk `json:"chunk"`
	Validation   *Result           `json:"validation,omitempty"`
	SearchResult []tools.SearchResult `json:"search_results,omitempty"`
	Error        string            `json:"error,omitempty"`
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

	// Validate each chunk
	var chunkResults []ChunkValidationResult
	var allClaims []Claim
	var allParsedClaims []string
	var allIssues []string
	var allSuggestions []string
	var allMissingBestPractices []string
	var totalConfidence float64
	var validChunks int
	var processedChunks int

	for _, chunk := range chunkResult.Chunks {
		// Validate this chunk
		chunkValidationResult := validateChunk(ctx, chunk, req.SpecVersion, embedFunc, searchFunc, llmFunc)
		chunkResults = append(chunkResults, chunkValidationResult)

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
	isValid := validChunks == len(chunkResult.Chunks) && processedChunks > 0

	// Add chunk processing information
	if processedChunks < len(chunkResult.Chunks) {
		errorSummary := fmt.Sprintf("Warning: %d of %d chunks failed validation", 
			len(chunkResult.Chunks)-processedChunks, len(chunkResult.Chunks))
		allIssues = append([]string{errorSummary}, allIssues...)
	}

	// Create corrected version if needed
	correctedVersion := ""
	if !isValid && len(allClaims) > 0 {
		correctedVersion = buildCorrectedVersion(req.Content, allClaims)
	}

	// Build final result
	return &Result{
		IsValid:          isValid,
		Confidence:       avgConfidence,
		ParsedClaims:     deduplicate(allParsedClaims),
		Issues:           deduplicate(allIssues),
		Suggestions:      deduplicate(allSuggestions),
		CorrectedVersion: correctedVersion,
		SpecVersion:      req.SpecVersion,
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

// validateChunk validates a single chunk of content
func validateChunk(ctx context.Context, chunk contentprep.Chunk, specVersion string, 
	embedFunc tools.EmbeddingFunc, searchFunc tools.SearchFunc, llmFunc LLMCompleteFunc) ChunkValidationResult {
	
	// Search for relevant spec sections for this chunk
	searchResults, err := tools.NewValidationBuilder(chunk.Text, specVersion).
		WithFunctions(embedFunc, searchFunc).
		WithSearchTopK(chunkSearchTopK).
		Search(ctx)
	
	if err != nil {
		return ChunkValidationResult{
			Chunk: chunk,
			Error: fmt.Sprintf("failed to search specifications: %v", err),
		}
	}

	// Perform fact-checking on this chunk
	factCheckResult, err := performClaimCheck(ctx, llmFunc, chunk.Text, searchResults)
	if err != nil {
		return ChunkValidationResult{
			Chunk: chunk,
			Error: fmt.Sprintf("validation failed: %v", err),
		}
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

	return ChunkValidationResult{
		Chunk:        chunk,
		Validation:   result,
		SearchResult: searchResults,
	}
}

// formatChunkValidationDetails formats detailed chunk validation results
// This can be used for debugging or when more detailed output is needed
func formatChunkValidationDetails(aggregated AggregatedValidationResult) map[string]interface{} {
	chunkSummaries := make([]map[string]interface{}, 0, len(aggregated.ChunkResults))
	
	for _, chunkResult := range aggregated.ChunkResults {
		summary := map[string]interface{}{
			"chunk_id":   chunkResult.Chunk.ID,
			"position":   chunkResult.Chunk.Position,
			"type":       chunkResult.Chunk.Type,
			"text_preview": truncateText(chunkResult.Chunk.Text, 100),
		}
		
		if chunkResult.Error != "" {
			summary["error"] = chunkResult.Error
		} else if chunkResult.Validation != nil {
			summary["is_valid"] = chunkResult.Validation.IsValid
			summary["confidence"] = chunkResult.Validation.Confidence
			summary["issues_count"] = len(chunkResult.Validation.Issues)
			summary["parsed_claims_count"] = len(chunkResult.Validation.ParsedClaims)
		}
		
		chunkSummaries = append(chunkSummaries, summary)
	}
	
	return map[string]interface{}{
		"validation_type": "comprehensive_chunked",
		"total_chunks":    len(aggregated.ChunkResults),
		"overall":         aggregated.Overall,
		"summary":         aggregated.Summary,
		"spec_version":    aggregated.SpecVersion,
		"chunk_summaries": chunkSummaries,
	}
}

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}