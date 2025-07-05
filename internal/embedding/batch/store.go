package batch

import (
	"github.com/carlisia/mcp-factcheck/internal/embedding"
	"github.com/carlisia/mcp-factcheck/internal/embedding/core"
)

// EmbeddingStore handles storage of embeddings for the specloader utility
type EmbeddingStore struct {
	store *embedding.Store
}

// NewEmbeddingStore creates a new embedding store for batch operations
func NewEmbeddingStore(dataDir string) *EmbeddingStore {
	return &EmbeddingStore{
		store: embedding.NewStore(dataDir),
	}
}

// Store saves a spec embedding to the database
func (es *EmbeddingStore) Store(specEmbedding *core.SpecEmbedding) error {
	return es.store.Store(specEmbedding)
}
