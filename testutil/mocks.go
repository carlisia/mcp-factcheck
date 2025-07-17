package testutil

import (
	"context"
	"errors"
	"fmt"

	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools"
)

// Mock creation helpers for common scenarios

// NewMockVectorDBWithError creates a VectorDB that always returns an error
func NewMockVectorDBWithError(err error) *MockVectorDB {
	return &MockVectorDB{
		SearchFunc: func(version string, queryEmbedding []float64, topK int) ([]tools.SearchResult, error) {
			return nil, err
		},
		ListVersionsFunc: func() ([]string, error) {
			return nil, err
		},
	}
}

// NewMockVectorDBWithResults creates a VectorDB that returns specific results
func NewMockVectorDBWithResults(results []tools.SearchResult) *MockVectorDB {
	return &MockVectorDB{
		SearchFunc: func(version string, queryEmbedding []float64, topK int) ([]tools.SearchResult, error) {
			// Respect topK limit
			if topK > 0 && len(results) > topK {
				return results[:topK], nil
			}
			return results, nil
		},
	}
}

// NewMockEmbeddingGeneratorWithError creates an EmbeddingGenerator that always returns an error
func NewMockEmbeddingGeneratorWithError(err error) *MockEmbeddingGenerator {
	return &MockEmbeddingGenerator{
		GenerateFunc: func(ctx context.Context, content string) ([]float64, error) {
			// Check context first
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, err
		},
	}
}

// NewMockEmbeddingGeneratorWithEmbedding creates an EmbeddingGenerator that returns a specific embedding
func NewMockEmbeddingGeneratorWithEmbedding(embedding []float64) *MockEmbeddingGenerator {
	return &MockEmbeddingGenerator{
		GenerateFunc: func(ctx context.Context, content string) ([]float64, error) {
			// Check context first
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return embedding, nil
		},
	}
}

// NewMockEmbeddingGeneratorSlow creates an EmbeddingGenerator that simulates slow operation
func NewMockEmbeddingGeneratorSlow() *MockEmbeddingGenerator {
	return &MockEmbeddingGenerator{
		GenerateFunc: func(ctx context.Context, content string) ([]float64, error) {
			// This will block until context is cancelled
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}
}

// CreateTestSearchResults creates sample search results for testing
func CreateTestSearchResults(contents ...string) []tools.SearchResult {
	results := make([]tools.SearchResult, len(contents))
	for i, content := range contents {
		results[i] = tools.SearchResult{
			Content:    content,
			ChunkID:    fmt.Sprintf("test-chunk-%d", i),
			Similarity: 0.9 - float64(i)*0.1, // Decreasing similarity
			Rank:       i + 1,
			Version:    "test",
			Section:    "test-section",
			Metadata:   map[string]any{"index": i},
		}
	}
	return results
}

// Common test errors
var (
	ErrTestDatabase  = errors.New("test database error")
	ErrTestEmbedding = errors.New("test embedding error")
	ErrTestTimeout   = errors.New("test timeout error")
)
