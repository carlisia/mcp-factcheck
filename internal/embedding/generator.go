package embedding

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
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
	totalChunks := len(chunks)
	processedCount := 0

	for i, chunk := range chunks {
		if len(chunk) == 0 {
			continue // Skip empty chunks
		}

		// Log progress every 10 chunks
		if i > 0 && i%10 == 0 {
			log.Printf("Progress: %d/%d chunks embedded (%.1f%%)", i, totalChunks, float64(i)/float64(totalChunks)*100)
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
		processedCount++
	}

	log.Printf("Completed: %d/%d chunks embedded (100%%)", processedCount, totalChunks)

	return &SpecEmbedding{
		Version: version,
		Chunks:  embeddedChunks,
		Count:   len(embeddedChunks),
	}, nil
}

// ProcessSpecBatch generates embeddings for all chunks in a spec version using batch processing.
// It takes a function that can generate embeddings for multiple texts at once.
func ProcessSpecBatch(ctx context.Context, version string, chunks []string, batchSize int, generateEmbeddingsBatch func(context.Context, []string) ([][]float64, error)) (*SpecEmbedding, error) {
	var embeddedChunks []EmbeddedChunk
	totalChunks := len(chunks)

	// Process chunks in batches
	for i := 0; i < totalChunks; i += batchSize {
		end := min(i+batchSize, totalChunks)

		// Get current batch
		batch := chunks[i:end]
		batchIndices := make([]int, 0, len(batch))
		batchTexts := make([]string, 0, len(batch))

		// Filter out empty chunks and track indices
		for j, chunk := range batch {
			if len(chunk) > 0 {
				batchTexts = append(batchTexts, chunk)
				batchIndices = append(batchIndices, i+j)
			}
		}

		if len(batchTexts) == 0 {
			continue
		}

		// Log progress
		log.Printf("Processing batch %d-%d of %d chunks (%.1f%%)", i, end, totalChunks, float64(end)/float64(totalChunks)*100)

		// Generate embeddings for batch
		embeddings, err := generateEmbeddingsBatch(ctx, batchTexts)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embeddings for batch starting at %d: %w", i, err)
		}

		// Create embedded chunks
		for j, embedding := range embeddings {
			idx := batchIndices[j]
			chunkID := generateChunkID(version, idx, batchTexts[j])

			embeddedChunk := EmbeddedChunk{
				ID:        chunkID,
				Version:   version,
				Content:   batchTexts[j],
				Embedding: embedding,
				Metadata: map[string]any{
					"chunk_index": idx,
					"length":      len(batchTexts[j]),
				},
			}

			embeddedChunks = append(embeddedChunks, embeddedChunk)
		}
	}

	log.Printf("Completed: %d chunks embedded (100%%)", len(embeddedChunks))

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
