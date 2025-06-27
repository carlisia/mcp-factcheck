package embedding

import (
	"github.com/carlisia/mcp-factcheck/embedding"
	"github.com/carlisia/mcp-factcheck/vectorstore"
)

// EmbeddingStore handles storage of embeddings for the specloader utility
type EmbeddingStore struct {
	store *vectorstore.Store
}

// NewEmbeddingStore creates a new embedding store for batch operations
func NewEmbeddingStore(dataDir string) *EmbeddingStore {
	return &EmbeddingStore{
		store: vectorstore.NewStore(dataDir),
	}
}

// Store saves a spec embedding to the database
func (es *EmbeddingStore) Store(specEmbedding *embedding.SpecEmbedding) error {
	return es.store.Store(specEmbedding)
}