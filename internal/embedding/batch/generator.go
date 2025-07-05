// Package batch provides batch processing utilities for generating embeddings at scale.
// It handles bulk embedding generation for MCP specification processing.
package batch

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/carlisia/mcp-factcheck/internal/embedding/core"
)

// BatchGenerator handles batch embedding generation for spec processing
type BatchGenerator struct {
	generator *core.Generator
}

// NewBatchGenerator creates a new batch embedding generator
func NewBatchGenerator() (*BatchGenerator, error) {
	gen, err := core.NewGenerator()
	if err != nil {
		return nil, err
	}
	return &BatchGenerator{generator: gen}, nil
}

// NewGenerator creates a new generator (alias for compatibility)
func NewGenerator() (*core.Generator, error) {
	return core.NewGenerator()
}

// GenerateSpecEmbeddings creates embeddings for all chunks in a spec
func (g *BatchGenerator) GenerateSpecEmbeddings(ctx context.Context, version string, chunks []string) (*core.SpecEmbedding, error) {
	var embeddedChunks []core.EmbeddedChunk

	for i, chunk := range chunks {
		if len(chunk) == 0 {
			continue // Skip empty chunks
		}

		// Generate embedding
		embeddingData, err := g.generator.GenerateEmbedding(ctx, chunk)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding for chunk %d: %w", i, err)
		}

		// Create chunk ID
		chunkID := generateChunkID(version, i, chunk)

		embeddedChunk := core.EmbeddedChunk{
			ID:        chunkID,
			Version:   version,
			Content:   chunk,
			Embedding: embeddingData,
			Metadata: map[string]any{
				"chunk_index": i,
				"length":      len(chunk),
			},
		}

		embeddedChunks = append(embeddedChunks, embeddedChunk)
	}

	return &core.SpecEmbedding{
		Version: version,
		Chunks:  embeddedChunks,
		Count:   len(embeddedChunks),
	}, nil
}

// generateChunkID creates a unique ID for a chunk
func generateChunkID(version string, index int, content string) string {
	// Create a hash of the content for uniqueness
	hasher := sha256.New()
	hasher.Write([]byte(content))
	hash := fmt.Sprintf("%x", hasher.Sum(nil))[:8]

	return fmt.Sprintf("%s_%d_%s", version, index, hash)
}
