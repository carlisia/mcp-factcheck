package tools

import (
	"context"
	"fmt"
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

	// If initial search has low similarity, try alternative queries
	if len(searchResults) == 0 || searchResults[0].Similarity < 0.7 {
		alternativeResults, err := s.searchWithAlternatives(ctx, content, version, embedFunc, searchFunc)
		if err == nil && len(alternativeResults) > 0 {
			// Merge results, keeping unique ones
			searchResults = s.mergeSearchResults(searchResults, alternativeResults)
		}
	}

	return searchResults, nil
}

func (s *AggressiveSearchStrategy) searchWithAlternatives(ctx context.Context, content, version string, embedFunc EmbeddingFunc, searchFunc SearchFunc) ([]SearchResult, error) {
	// Extract key terms from the content
	keyTerms := s.extractKeyTerms(content)
	if len(keyTerms) == 0 {
		return nil, nil
	}

	var allResults []SearchResult

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
		if !seen[r.ChunkID] {
			seen[r.ChunkID] = true
			merged = append(merged, r)
		}
	}

	// Add unique ones from second set
	for _, r := range results2 {
		if !seen[r.ChunkID] {
			seen[r.ChunkID] = true
			merged = append(merged, r)
		}
	}

	return merged
}

// ChunkedSearchStrategy performs search on chunked content
type ChunkedSearchStrategy struct {
	ChunkSize  int
	SearchTopK int
	ChunkFunc  func(string) []string
}

func (s *ChunkedSearchStrategy) Search(ctx context.Context, content string, version string, embedFunc EmbeddingFunc, searchFunc SearchFunc) ([]SearchResult, error) {
	// Use provided chunk function or default
	var chunks []string
	if s.ChunkFunc != nil {
		chunks = s.ChunkFunc(content)
	} else {
		chunks = s.defaultChunk(content)
	}

	var allResults []SearchResult
	seenChunks := make(map[string]bool)

	// Search each chunk
	for _, chunk := range chunks {
		results, err := EmbedAndSearch(ctx, chunk, version, s.SearchTopK, embedFunc, searchFunc)
		if err != nil {
			continue // Skip failed chunks
		}

		// Add unique results
		for _, result := range results {
			if !seenChunks[result.ChunkID] {
				seenChunks[result.ChunkID] = true
				allResults = append(allResults, result)
			}
		}
	}

	return allResults, nil
}

func (s *ChunkedSearchStrategy) defaultChunk(content string) []string {
	// Simple chunking by paragraphs or sentences
	var chunks []string
	currentChunk := ""
	chunkSize := s.ChunkSize
	if chunkSize == 0 {
		chunkSize = 2000 // default chunk size
	}

	for para := range strings.SplitSeq(content, "\n\n") {
		if len(currentChunk)+len(para) > chunkSize && currentChunk != "" {
			chunks = append(chunks, strings.TrimSpace(currentChunk))
			currentChunk = para
		} else {
			if currentChunk != "" {
				currentChunk += "\n\n"
			}
			currentChunk += para
		}
	}

	if currentChunk != "" {
		chunks = append(chunks, strings.TrimSpace(currentChunk))
	}

	return chunks
}
