package vectorstore

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"

	"github.com/carlisia/mcp-factcheck/embedding"
)

// Store handles storage and retrieval of embeddings from the filesystem
type Store struct {
	dataDir string
}

// NewStore creates a new vector store
func NewStore(dataDir string) *Store {
	return &Store{dataDir: dataDir}
}

// Store saves a spec embedding to the database
func (s *Store) Store(specEmbedding *embedding.SpecEmbedding) error {
	// Ensure data directory exists
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Save to JSON file
	filename := filepath.Join(s.dataDir, fmt.Sprintf("%s.json", specEmbedding.Version))
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(specEmbedding); err != nil {
		return fmt.Errorf("failed to encode spec embedding: %w", err)
	}

	return nil
}

// Load retrieves a spec embedding from the database
func (s *Store) Load(version string) (*embedding.SpecEmbedding, error) {
	filename := filepath.Join(s.dataDir, fmt.Sprintf("%s.json", version))
	
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var specEmbedding embedding.SpecEmbedding
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&specEmbedding); err != nil {
		return nil, fmt.Errorf("failed to decode spec embedding: %w", err)
	}

	return &specEmbedding, nil
}

// Search performs similarity search against a spec version
func (s *Store) Search(version string, queryEmbedding []float64, topK int) ([]embedding.SearchResult, error) {
	// Load spec embeddings
	specEmbedding, err := s.Load(version)
	if err != nil {
		return nil, fmt.Errorf("failed to load spec embeddings: %w", err)
	}

	// Calculate similarities
	var results []embedding.SearchResult
	for _, chunk := range specEmbedding.Chunks {
		similarity := cosineSimilarity(queryEmbedding, chunk.Embedding)
		results = append(results, embedding.SearchResult{
			Chunk:      chunk,
			Similarity: similarity,
		})
	}

	// Sort by similarity (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	// Add rank and limit to topK
	if topK > len(results) {
		topK = len(results)
	}
	
	for i := 0; i < topK; i++ {
		results[i].Rank = i + 1
	}

	return results[:topK], nil
}

// ListVersions returns all available spec versions in the database
func (s *Store) ListVersions() ([]string, error) {
	files, err := filepath.Glob(filepath.Join(s.dataDir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	var versions []string
	for _, file := range files {
		base := filepath.Base(file)
		version := base[:len(base)-5] // Remove .json extension
		versions = append(versions, version)
	}

	return versions, nil
}

// cosineSimilarity calculates cosine similarity between two vectors
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0.0 || normB == 0.0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}