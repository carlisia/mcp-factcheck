package embedding

import (
	"github.com/carlisia/mcp-factcheck/embedding"
	"github.com/carlisia/mcp-factcheck/vectorstore"
)

// VectorDB handles MCP-specific vector database operations for the runtime server
type VectorDB struct {
	store *vectorstore.Store
}

// NewVectorDB creates a new MCP vector database
func NewVectorDB(dataDir string) *VectorDB {
	return &VectorDB{
		store: vectorstore.NewStore(dataDir),
	}
}

// Search performs similarity search against a spec version (MCP tool functionality)
func (db *VectorDB) Search(version string, queryEmbedding []float64, topK int) ([]embedding.SearchResult, error) {
	return db.store.Search(version, queryEmbedding, topK)
}

// ListVersions returns all available spec versions (MCP tool functionality)
func (db *VectorDB) ListVersions() ([]string, error) {
	return db.store.ListVersions()
}