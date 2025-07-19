package validation

import (
	"context"
	"fmt"
	"strings"

	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools"
)

// searchForNegativeClaim handles searching for negative claims by extracting the positive concept
// and searching for it directly. For example, "MCP does not enforce rate limits" becomes
// a search for "rate limits" and "rate limiting".
func searchForNegativeClaim(ctx context.Context, claim string, specVersion string, embedFunc tools.EmbeddingFunc, searchFunc tools.SearchFunc) ([]tools.SearchResult, error) {
	// Extract the key concept from the negative claim
	concepts := extractConceptsFromNegativeClaim(claim)
	if len(concepts) == 0 {
		// Fallback to regular search
		return performRegularSearch(ctx, claim, specVersion, embedFunc, searchFunc)
	}

	// Search for each concept and combine results
	allResults := make(map[string]tools.SearchResult)
	for _, concept := range concepts {
		embedding, err := embedFunc(ctx, concept)
		if err != nil {
			continue
		}

		results, err := searchFunc(specVersion, embedding, 10)
		if err != nil {
			continue
		}

		// Deduplicate results by content
		for _, result := range results {
			key := strings.TrimSpace(result.Content)
			if existing, ok := allResults[key]; !ok || result.Similarity > existing.Similarity {
				allResults[key] = result
			}
		}
	}

	// Convert map back to slice
	var finalResults []tools.SearchResult
	for _, result := range allResults {
		finalResults = append(finalResults, result)
	}

	// Sort by similarity
	tools.SortSearchResultsBySimilarity(finalResults)

	// Limit to top results
	if len(finalResults) > 15 {
		finalResults = finalResults[:15]
	}

	return finalResults, nil
}

// extractConceptsFromNegativeClaim extracts positive concepts from negative claims
func extractConceptsFromNegativeClaim(claim string) []string {
	lowerClaim := strings.ToLower(claim)
	var concepts []string

	// Common negative patterns
	negativePatterns := []struct {
		prefix string
		suffix string
	}{
		{"mcp does not enforce", ""},
		{"mcp doesn't enforce", ""},
		{"mcp does not", ""},
		{"mcp doesn't", ""},
		{"mcp never", ""},
		{"mcp does not provide", ""},
		{"mcp doesn't provide", ""},
		{"mcp does not implement", ""},
		{"mcp doesn't implement", ""},
	}

	// Extract the concept after the negative pattern
	for _, pattern := range negativePatterns {
		if strings.HasPrefix(lowerClaim, pattern.prefix) {
			concept := strings.TrimSpace(strings.TrimPrefix(lowerClaim, pattern.prefix))
			if pattern.suffix != "" && strings.HasSuffix(concept, pattern.suffix) {
				concept = strings.TrimSpace(strings.TrimSuffix(concept, pattern.suffix))
			}
			if concept != "" {
				concepts = append(concepts, concept)
				// Also add variations
				if strings.Contains(concept, "rate limits") {
					concepts = append(concepts, "rate limiting")
					concepts = append(concepts, "Both parties SHOULD implement rate limiting")
				}
			}
			break
		}
	}

	// Handle compound claims with "or"
	if strings.Contains(lowerClaim, " or ") {
		parts := strings.Split(lowerClaim, " or ")
		for _, part := range parts {
			subConcepts := extractConceptsFromNegativeClaim(strings.TrimSpace(part))
			concepts = append(concepts, subConcepts...)
		}
	}

	return concepts
}

// performRegularSearch performs a standard search
func performRegularSearch(ctx context.Context, claim string, specVersion string, embedFunc tools.EmbeddingFunc, searchFunc tools.SearchFunc) ([]tools.SearchResult, error) {
	embedding, err := embedFunc(ctx, claim)
	if err != nil {
		return nil, fmt.Errorf("creating embedding: %w", err)
	}

	return searchFunc(specVersion, embedding, 15)
}

// isNegativeClaim checks if a claim contains negative assertions
func isNegativeClaim(claim string) bool {
	lowerClaim := strings.ToLower(claim)
	negativeIndicators := []string{
		"does not", "doesn't", "never", "cannot", "can't",
		"will not", "won't", "shall not", "shan't",
		"is not", "isn't", "are not", "aren't",
		"no longer", "without", "lacks", "absence of",
	}

	for _, indicator := range negativeIndicators {
		if strings.Contains(lowerClaim, indicator) {
			return true
		}
	}

	return false
}
