// Package embedding provides vector database functionality for semantic search over MCP specifications.
// It includes support for both regular and fine-grained embeddings with intelligent fallback mechanisms.
package embedding

import (
	"github.com/carlisia/mcp-factcheck/internal/embedding/core"
)

// VectorDB handles MCP-specific vector database operations for the runtime server.
// It provides semantic search capabilities over embedded MCP specification content
// with support for fine-grained embeddings when available.
type VectorDB struct {
	store *Store
}

// NewVectorDB creates a new MCP vector database instance.
// The dataDir parameter specifies the directory containing embedding files.
func NewVectorDB(dataDir string) *VectorDB {
	return &VectorDB{
		store: NewStore(dataDir),
	}
}

// Search performs similarity search against a spec version.
// It attempts to use fine-grained embeddings (version-fine) if available,
// falling back to regular embeddings. Returns the top K most similar results.
func (db *VectorDB) Search(version string, queryEmbedding []float64, topK int) ([]core.SearchResult, error) {
	// Try fine-grained version first if available
	fineVersion := version + "-fine"
	results, err := db.store.Search(fineVersion, queryEmbedding, topK)
	if err == nil {
		// Successfully found fine-grained embeddings
		return results, nil
	}

	// Fall back to regular version
	return db.store.Search(version, queryEmbedding, topK)
}

// ListVersions returns all available spec versions in the vector database.
// The returned versions can be used with the Search method and other MCP tools.
func (db *VectorDB) ListVersions() ([]string, error) {
	return db.store.ListVersions()
}
