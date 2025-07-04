package spec

import (
	"context"

	"github.com/carlisia/mcp-factcheck/embedding"
)

// Compile-time interface conformance checks
var _ vectorDB = (*mockSearchVectorDB)(nil)
var _ embeddingGenerator = (*mockEmbeddingGenerator)(nil)

// mockSearchVectorDB is a test implementation of vectorDB interface for search tests
type mockSearchVectorDB struct {
	searchFunc       func(version string, queryEmbedding []float64, topK int) ([]embedding.SearchResult, error)
	listVersionsFunc func() ([]string, error)
}

func (m *mockSearchVectorDB) Search(version string, queryEmbedding []float64, topK int) ([]embedding.SearchResult, error) {
	// Optional trace logging for debugging - uncomment when needed
	// fmt.Printf("[mockSearch] version=%s topK=%d embeddingLen=%d\n", version, topK, len(queryEmbedding))

	if m.searchFunc != nil {
		return m.searchFunc(version, queryEmbedding, topK)
	}

	// Default behavior: return a mock result
	return []embedding.SearchResult{
		{
			Chunk: embedding.EmbeddedChunk{
				Content: "MCP uses JSON-RPC 2.0 for communication protocol",
			},
			Similarity: 0.95,
			Rank:       1,
		},
	}, nil
}

func (m *mockSearchVectorDB) ListVersions() ([]string, error) {
	if m.listVersionsFunc != nil {
		return m.listVersionsFunc()
	}
	// Default behavior: return standard versions
	return []string{"draft", "2025-06-18", "2025-03-26", "2024-11-05"}, nil
}

// mockEmbeddingGenerator is a test implementation of embeddingGenerator interface
type mockEmbeddingGenerator struct {
	generateFunc func(ctx context.Context, content string) ([]float64, error)
}

func (m *mockEmbeddingGenerator) GenerateEmbedding(ctx context.Context, content string) ([]float64, error) {
	// Optional trace logging for debugging - uncomment when needed
	// fmt.Printf("[mockGenerate] contentLen=%d\n", len(content))

	// Check context cancellation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if m.generateFunc != nil {
		return m.generateFunc(ctx, content)
	}

	// Default behavior: return a mock embedding
	return []float64{0.1, 0.2, 0.3}, nil
}

