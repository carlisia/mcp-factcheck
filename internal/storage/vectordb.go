package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/carlisia/mcp-factcheck/pkg/logger"
	"go.uber.org/zap"
)

// Constants for VectorDB operations
const (
	// fineGrainedSuffix is appended to version names to indicate fine-grained embeddings
	fineGrainedSuffix = "-fine"
)

// Common errors
var (
	// ErrVersionNotFound indicates the requested version doesn't exist in the database
	ErrVersionNotFound = errors.New("version not found")
)

// VectorSearcher defines the interface for vector similarity search operations.
// This interface facilitates testing and allows for alternative implementations.
type VectorSearcher interface {
	// Search performs similarity search against a spec version
	Search(ctx context.Context, version string, queryEmbedding []float64, topK int) ([]SearchResult, error)
	// ListVersions returns all available spec versions
	ListVersions() ([]string, error)
}

// VectorDB handles MCP-specific vector database operations for the runtime server.
// It provides semantic search capabilities over embedded MCP specification content
// with support for fine-grained embeddings when available.
type VectorDB struct {
	store *Store
	log   *zap.Logger
}

// Ensure VectorDB implements VectorSearcher
var _ VectorSearcher = (*VectorDB)(nil)

// NewVectorDB creates a new MCP vector database instance.
// The dataDir parameter specifies the directory containing embedding files.
func NewVectorDB(dataDir string) *VectorDB {
	return &VectorDB{
		store: NewStore(dataDir),
		log:   logger.Get().Named("vectordb"),
	}
}

// NewVectorDBFromEmbeddedData creates a new MCP vector database instance using embedded data.
// This allows the binary to work without external embedding files.
func NewVectorDBFromEmbeddedData() *VectorDB {
	return &VectorDB{
		store: NewEmbeddedStore(),
		log:   logger.Get().Named("vectordb"),
	}
}

// NewEmbeddedVectorDB creates a new MCP vector database instance using embedded data.
// Deprecated: Use NewVectorDBFromEmbeddedData for clarity.
func NewEmbeddedVectorDB() *VectorDB {
	return NewVectorDBFromEmbeddedData()
}

// Search performs similarity search against a spec version.
// It attempts to use fine-grained embeddings (version-fine) if available,
// falling back to regular embeddings. Returns the top K most similar results.
func (db *VectorDB) Search(ctx context.Context, version string, queryEmbedding []float64, topK int) ([]SearchResult, error) {
	// Try fine-grained version first if available
	fineVersion := version + fineGrainedSuffix
	results, err := db.store.Search(fineVersion, queryEmbedding, topK)
	if err == nil {
		// Successfully found fine-grained embeddings
		return results, nil
	}

	// Check if this is a version not found error vs other errors
	if !isVersionNotFoundError(err) {
		return nil, fmt.Errorf("fine-grained search error: %w", err)
	}

	// Log the fallback for debugging
	db.log.Debug("falling back to regular embeddings",
		zap.String("version", version),
		zap.String("fine_version", fineVersion),
	)

	// Fall back to regular version
	results, err = db.store.Search(version, queryEmbedding, topK)
	if err != nil {
		return nil, fmt.Errorf("fallback search error: %w", err)
	}

	return results, nil
}

// isVersionNotFoundError checks if an error indicates a missing version
func isVersionNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// Check for our sentinel error or common file not found patterns
	return errors.Is(err, ErrVersionNotFound) ||
		strings.Contains(err.Error(), "not found") ||
		strings.Contains(err.Error(), "no such file")
}

// ListVersions returns all available spec versions in the vector database.
// The returned versions can be used with the Search method and other MCP tools.
func (db *VectorDB) ListVersions() ([]string, error) {
	return db.store.ListVersions()
}
