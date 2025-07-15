package embedding

import (
	"context"
	"crypto/sha256"
	"fmt"
)

// EmbeddedChunk represents a chunk of text with its embedding vector.
type EmbeddedChunk struct {
	ID        string         `json:"id"`
	Version   string         `json:"version"`
	FilePath  string         `json:"file_path,omitempty"`
	Section   string         `json:"section,omitempty"`
	Content   string         `json:"content"`
	Embedding []float64      `json:"embedding"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// SpecEmbedding represents all embeddings for a specific MCP spec version.
type SpecEmbedding struct {
	Version string          `json:"version"`
	Chunks  []EmbeddedChunk `json:"chunks"`
	Count   int             `json:"count"`
}


// ProcessSpec generates embeddings for all chunks in a spec version.
// It takes a function that can generate embeddings and applies it to all chunks.
func ProcessSpec(ctx context.Context, version string, chunks []string, generateEmbedding func(context.Context, string) ([]float64, error)) (*SpecEmbedding, error) {
	var embeddedChunks []EmbeddedChunk

	for i, chunk := range chunks {
		if len(chunk) == 0 {
			continue // Skip empty chunks
		}

		// Generate embedding
		embeddingData, err := generateEmbedding(ctx, chunk)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding for chunk %d: %w", i, err)
		}

		// Create chunk ID
		chunkID := generateChunkID(version, i, chunk)

		embeddedChunk := EmbeddedChunk{
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

	return &SpecEmbedding{
		Version: version,
		Chunks:  embeddedChunks,
		Count:   len(embeddedChunks),
	}, nil
}

// generateChunkID creates a unique ID for a chunk based on version, index, and content
func generateChunkID(version string, index int, content string) string {
	// Create the full string first, then hash it
	data := fmt.Sprintf("%s:%d:%s", version, index, content)
	h := sha256.New()
	h.Write([]byte(data))
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}