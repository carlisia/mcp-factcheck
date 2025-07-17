package tools

import (
	"context"
	"fmt"
)

// TODO: do we need all these fields and do they need json tags?
// SearchResult represents a unified search result from the vector database.
// This is used across all tools that perform search operations.
type SearchResult struct {
	Content    string         `json:"content"`
	ChunkID    string         `json:"chunk_id"`
	Similarity float64        `json:"similarity"`
	Rank       int            `json:"rank"`
	Version    string         `json:"version,omitempty"`
	FilePath   string         `json:"file_path,omitempty"`
	Section    string         `json:"section,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// SearchFunc performs vector database search operations.
// This is the standard signature for all search operations across tools.
type SearchFunc func(version string, queryEmbedding []float64, topK int) ([]SearchResult, error)

// EmbeddingFunc converts text content into vector embeddings for semantic similarity matching.
// This is used across multiple tools for search and validation operations.
type EmbeddingFunc func(ctx context.Context, content string) ([]float64, error)

// EmbedAndSearch combines embedding generation and search into a single operation.
// This is the standard pattern used across all tools that need to search specifications.
//
// Context handling:
//   - The context is passed through to both embedding and search operations
//   - Embedding generation: External API call to OpenAI (can be slow, 1-3s)
//   - Search operation: Local but CPU-intensive vector similarity computation
//   - Cancellation between operations prevents unnecessary work
//   - Total operation time varies by content length: 2-5s typical
//
// Returns immediately with an error if the context is cancelled at any point.
func EmbedAndSearch(ctx context.Context, content string, version string, topK int, embedFunc EmbeddingFunc, searchFunc SearchFunc) ([]SearchResult, error) {
	// Generate embedding for the content
	embedding, err := generateEmbedding(ctx, content, embedFunc)
	if err != nil {
		return nil, err
	}

	// Search specifications with the embedding
	results, err := performSearch(ctx, version, embedding, topK, searchFunc)
	if err != nil {
		return nil, err
	}

	return results, nil
}

// generateEmbedding is a helper function that generates embeddings with standard error handling.
// It checks context cancellation before performing the embedding operation.
func generateEmbedding(ctx context.Context, content string, embedFunc EmbeddingFunc) ([]float64, error) {
	// Check context before expensive operation
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("operation cancelled: %w", ctx.Err())
	default:
	}

	embedding, err := embedFunc(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("embedding generation failed: %w", err)
	}

	return embedding, nil
}

// performSearch is a helper function that performs search with standard error handling.
// It checks context cancellation before performing the search operation.
func performSearch(ctx context.Context, version string, embedding []float64, topK int, searchFunc SearchFunc) ([]SearchResult, error) {
	// Check context before expensive operation
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("operation cancelled: %w", ctx.Err())
	default:
	}

	results, err := searchFunc(version, embedding, topK)
	if err != nil {
		return nil, fmt.Errorf("specification search failed: %w", err)
	}

	return results, nil
}

