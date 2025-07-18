package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// SearchStrategy defines how to perform searches for validation
type SearchStrategy interface {
	Search(ctx context.Context, content string, version string, embedFunc EmbeddingFunc, searchFunc SearchFunc) ([]SearchResult, error)
}

// DefaultSearchStrategy performs standard search
type DefaultSearchStrategy struct {
	topK int
}

func NewDefaultSearchStrategy(topK int) *DefaultSearchStrategy {
	return &DefaultSearchStrategy{topK: topK}
}

func (s *DefaultSearchStrategy) Search(ctx context.Context, content string, version string, embedFunc EmbeddingFunc, searchFunc SearchFunc) ([]SearchResult, error) {
	return EmbedAndSearch(ctx, content, version, s.topK, embedFunc, searchFunc)
}

// AggressiveSearchStrategy performs search with fallback strategies
type AggressiveSearchStrategy struct {
	PrimaryTopK  int
	FallbackTopK int
}

func (s *AggressiveSearchStrategy) Search(ctx context.Context, content string, version string, embedFunc EmbeddingFunc, searchFunc SearchFunc) ([]SearchResult, error) {
	// Primary search
	searchResults, err := EmbedAndSearch(ctx, content, version, s.PrimaryTopK, embedFunc, searchFunc)
	if err != nil {
		return nil, err
	}

	// Always try alternative queries for better coverage, especially for enforcement-related queries
	alternativeResults, err := s.searchWithAlternatives(ctx, content, version, embedFunc, searchFunc)
	if err == nil && len(alternativeResults) > 0 {
		// Merge results, keeping unique ones
		searchResults = s.mergeSearchResults(searchResults, alternativeResults)
	}

	return searchResults, nil
}

func (s *AggressiveSearchStrategy) searchWithAlternatives(ctx context.Context, content, version string, embedFunc EmbeddingFunc, searchFunc SearchFunc) ([]SearchResult, error) {
	var allResults []SearchResult

	// Special handling for "enforces" queries
	if strings.Contains(strings.ToLower(content), "enforces") {
		// When looking for enforcement, also search for recommendations
		alternatives := s.generateEnforcementAlternatives(content)
		for _, query := range alternatives {
			results, err := EmbedAndSearch(ctx, query, version, s.FallbackTopK, embedFunc, searchFunc)
			if err != nil {
				continue
			}
			allResults = append(allResults, results...)
		}
	}

	// Extract key terms from the content
	keyTerms := s.extractKeyTerms(content)
	if len(keyTerms) == 0 {
		return allResults, nil
	}

	// Try searching with expanded queries
	expandedQueries := []string{
		fmt.Sprintf("MCP %s specification", strings.Join(keyTerms, " ")),
		fmt.Sprintf("Model Context Protocol %s", strings.Join(keyTerms, " ")),
	}

	for _, query := range expandedQueries {
		results, err := EmbedAndSearch(ctx, query, version, s.FallbackTopK, embedFunc, searchFunc)
		if err != nil {
			continue
		}

		allResults = append(allResults, results...)
	}

	return allResults, nil
}

func (s *AggressiveSearchStrategy) generateEnforcementAlternatives(content string) []string {
	// When user asks about "enforces X", also search for:
	// - "implementations should X"
	// - "X recommendation"
	// - "X best practice"
	// - Just "X" itself

	lower := strings.ToLower(content)
	alternatives := []string{}

	// Extract what MCP supposedly enforces
	if idx := strings.Index(lower, "enforces"); idx >= 0 {
		// Get the part after "enforces"
		afterEnforces := strings.TrimSpace(content[idx+len("enforces"):])

		// Clean up common endings
		afterEnforces = strings.TrimSuffix(afterEnforces, ".")
		afterEnforces = strings.TrimSuffix(afterEnforces, "?")

		// Generate alternatives
		alternatives = append(alternatives,
			"implementations should "+afterEnforces,
			"implementations SHOULD "+afterEnforces,
			afterEnforces+" recommendation",
			afterEnforces+" best practice",
			"MCP "+afterEnforces,
			afterEnforces,
		)
	}

	// Also handle "does not enforce" or "doesn't enforce"
	if strings.Contains(lower, "not enforce") || strings.Contains(lower, "n't enforce") {
		// Extract what's after the enforce part
		patterns := []string{"not enforce", "n't enforce", "not enforces", "n't enforces"}
		for _, pattern := range patterns {
			if idx := strings.Index(lower, pattern); idx >= 0 {
				afterPattern := strings.TrimSpace(content[idx+len(pattern):])
				alternatives = append(alternatives,
					"implementations should "+afterPattern,
					afterPattern+" recommendation",
					"MCP "+afterPattern,
				)
				break
			}
		}
	}

	return alternatives
}

func (s *AggressiveSearchStrategy) extractKeyTerms(text string) []string {
	// Simple keyword extraction - in production this would be more sophisticated
	words := strings.Fields(strings.ToLower(text))

	// Filter out common words
	stopWords := map[string]bool{
		"the": true, "is": true, "are": true, "a": true, "an": true,
		"and": true, "or": true, "but": true, "in": true, "on": true,
		"at": true, "to": true, "for": true, "of": true, "with": true,
		"does": true, "can": true, "has": true, "have": true, "be": true,
		"mcp": true, // We'll add MCP back contextually
	}

	var keyTerms []string
	for _, word := range words {
		word = strings.Trim(word, ".,?!")
		if !stopWords[word] && len(word) > 2 {
			keyTerms = append(keyTerms, word)
		}
	}

	return keyTerms
}

func (s *AggressiveSearchStrategy) mergeSearchResults(results1, results2 []SearchResult) []SearchResult {
	seen := make(map[string]bool)
	var merged []SearchResult

	// Add all from first set
	for _, r := range results1 {
		if !seen[r.Content] {
			seen[r.Content] = true
			merged = append(merged, r)
		}
	}

	// Add unique ones from second set
	for _, r := range results2 {
		if !seen[r.Content] {
			seen[r.Content] = true
			merged = append(merged, r)
		}
	}

	return merged
}

// sortSearchResults sorts search results by similarity in descending order
func sortSearchResults(results []SearchResult) {
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})
}

// CompoundClaimSearchStrategy performs search specifically for compound claims
// by breaking them down and searching for each subclaim separately
type CompoundClaimSearchStrategy struct {
	topK int
}

func NewCompoundClaimSearchStrategy(topK int) *CompoundClaimSearchStrategy {
	return &CompoundClaimSearchStrategy{topK: topK}
}

func (s *CompoundClaimSearchStrategy) Search(ctx context.Context, content string, version string, embedFunc EmbeddingFunc, searchFunc SearchFunc) ([]SearchResult, error) {
	var allResults []SearchResult
	resultMap := make(map[string]SearchResult) // Deduplicate by content

	// First, search with the full content to get general context
	fullResults, err := EmbedAndSearch(ctx, content, version, s.topK, embedFunc, searchFunc)
	if err != nil {
		return nil, err
	}

	// Add full search results
	for _, result := range fullResults {
		resultMap[result.Content] = result
	}

	// Break down compound claims
	subclaims := s.extractSubclaims(content)

	// Search for each subclaim separately
	for _, subclaim := range subclaims {
		// Skip very short subclaims
		if len(strings.TrimSpace(subclaim)) < 5 {
			continue
		}

		subResults, err := EmbedAndSearch(ctx, subclaim, version, 5, embedFunc, searchFunc)
		if err != nil {
			continue // Don't fail the whole search if one subclaim fails
		}

		// Add subclaim results, deduplicating
		for _, result := range subResults {
			if _, exists := resultMap[result.Content]; !exists {
				resultMap[result.Content] = result
			}
		}
	}

	// Convert map back to slice
	for _, result := range resultMap {
		allResults = append(allResults, result)
	}

	// Sort by similarity
	sortSearchResults(allResults)

	// Return top results
	if len(allResults) > s.topK {
		return allResults[:s.topK], nil
	}

	return allResults, nil
}

func (s *CompoundClaimSearchStrategy) extractSubclaims(content string) []string {
	var subclaims []string

	// Split by semicolons
	if strings.Contains(content, ";") {
		parts := strings.Split(content, ";")
		for _, part := range parts {
			subclaims = append(subclaims, strings.TrimSpace(part))
		}
	}

	// Split by "and" with commas (e.g., "X, Y, and Z")
	if strings.Contains(content, ",") && strings.Contains(content, " and ") {
		// Extract the list part
		// Simple approach: find patterns like "enforces X, Y, and Z"
		words := strings.Fields(content)
		for i, word := range words {
			if strings.Contains(word, ",") || (i > 0 && words[i-1] == "and") {
				// This is part of a list, extract individual items
				listPart := s.extractListItems(content)
				subclaims = append(subclaims, listPart...)
				break
			}
		}
	}

	// Also search for key concepts individually
	concepts := s.extractKeyConcepts(content)
	subclaims = append(subclaims, concepts...)

	return subclaims
}

func (s *CompoundClaimSearchStrategy) extractListItems(content string) []string {
	var items []string

	// Look for patterns like "enforces ACLs, rate limits, and provenance"
	// Extract the verb and apply it to each item
	parts := strings.Split(content, " ")
	verb := ""

	for i, part := range parts {
		// Common verbs that precede lists
		if strings.Contains("enforces,provides,supports,implements,enables,requires", part) {
			verb = part
		}

		// If we found a verb and this part contains a comma, we're in a list
		if verb != "" && strings.Contains(part, ",") {
			// Extract all list items
			listStart := i
			listEnd := len(parts)

			// Find the end of the list
			for j := i; j < len(parts); j++ {
				if strings.Contains(parts[j], ";") || strings.Contains(parts[j], ".") {
					listEnd = j
					break
				}
			}

			// Extract individual items
			listText := strings.Join(parts[listStart:listEnd], " ")
			// Remove trailing punctuation and clean up
			listText = strings.TrimSuffix(listText, ";")
			listText = strings.TrimSuffix(listText, ".")

			// Split by commas and "and"
			listText = strings.ReplaceAll(listText, " and ", ", ")
			individualItems := strings.Split(listText, ",")

			for _, item := range individualItems {
				item = strings.TrimSpace(item)
				// Remove any remaining punctuation
				item = strings.TrimSuffix(item, ",")
				item = strings.TrimSuffix(item, ";")
				item = strings.TrimSuffix(item, ".")

				if item != "" && item != verb {
					// Don't include the verb itself as an item
					items = append(items, verb+" "+item)
				}
			}
			break
		}
	}

	return items
}

func (s *CompoundClaimSearchStrategy) extractKeyConcepts(content string) []string {
	// Extract key terms dynamically from the content itself
	// without hardcoding any specific MCP concepts
	var concepts []string

	// Look for noun phrases and technical terms in the content
	words := strings.Fields(strings.ToLower(content))

	// Extract potential multi-word concepts (2-3 word phrases)
	for i := 0; i < len(words)-1; i++ {
		// Skip common words
		if isCommonWord(words[i]) {
			continue
		}

		// Two-word phrases
		twoWord := words[i] + " " + words[i+1]
		if !isCommonPhrase(twoWord) {
			concepts = append(concepts, "MCP "+twoWord)
		}

		// Three-word phrases if available
		if i < len(words)-2 {
			threeWord := twoWord + " " + words[i+2]
			if !isCommonPhrase(threeWord) && !isCommonWord(words[i+2]) {
				concepts = append(concepts, "MCP "+threeWord)
			}
		}
	}

	return concepts
}

func isCommonWord(word string) bool {
	common := map[string]bool{
		"the": true, "is": true, "are": true, "a": true, "an": true,
		"and": true, "or": true, "but": true, "in": true, "on": true,
		"at": true, "to": true, "for": true, "of": true, "with": true,
		"never": true, "always": true, "can": true, "may": true,
	}
	return common[word]
}

func isCommonPhrase(phrase string) bool {
	// Filter out very common phrases that wouldn't be useful for search
	common := map[string]bool{
		"is the": true, "in the": true, "on the": true,
		"to the": true, "for the": true, "and the": true,
	}
	return common[phrase]
}

// ChunkedSearchStrategy performs search on chunked content
type ChunkedSearchStrategy struct {
	ChunkSize  int
	ChunkLimit int
}

func (s *ChunkedSearchStrategy) Search(ctx context.Context, content string, version string, embedFunc EmbeddingFunc, searchFunc SearchFunc) ([]SearchResult, error) {
	// For chunked strategy, we don't search - we just process the content
	// The actual search happens per chunk in the validation logic
	return nil, nil
}
