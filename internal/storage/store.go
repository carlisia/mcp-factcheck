package storage

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
)

// SearchResult represents a search result with similarity score
type SearchResult struct {
	Content    string  `json:"content"`
	ChunkID    string  `json:"chunk_id"`
	Similarity float64 `json:"similarity"`
	Version    string  `json:"version"`
	Rank       int     `json:"rank"`
}

// embeddedChunk represents a chunk of text with its embedding
type embeddedChunk struct {
	ID        string    `json:"id"`
	Version   string    `json:"version"`
	Content   string    `json:"content"`
	Embedding []float64 `json:"embedding"`
}

// specEmbedding represents embeddings for a specification version
type specEmbedding struct {
	Version string          `json:"version"`
	Chunks  []embeddedChunk `json:"chunks"`
	Count   int             `json:"count"`
}

// Store handles storage and retrieval of embeddings from the filesystem.
type Store struct {
	dataDir      string
	useEmbedded  bool
}

// NewStore creates a new vector store with the specified data directory.
// The directory will be created if it doesn't exist when storing embeddings.
func NewStore(dataDir string) *Store {
	return &Store{
		dataDir:     dataDir,
		useEmbedded: false,
	}
}

// NewEmbeddedStore creates a new vector store that uses embedded data.
func NewEmbeddedStore() *Store {
	return &Store{
		useEmbedded: true,
	}
}

// StoreEmbeddings saves embeddings to the filesystem as a JSON file.
// The file is named using the spec version (e.g., "draft.json").
func (s *Store) StoreEmbeddings(version string, chunks []embeddedChunk) error {
	// Ensure data directory exists
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	spec := &specEmbedding{
		Version: version,
		Chunks:  chunks,
		Count:   len(chunks),
	}

	// Save to JSON file
	filename := filepath.Join(s.dataDir, fmt.Sprintf("%s.json", version))
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Warning: Failed to close file %s: %v\n", filename, err)
		}
	}()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(spec); err != nil {
		return fmt.Errorf("failed to encode spec embedding: %w", err)
	}

	return nil
}

// Search performs similarity search against a spec version
func (s *Store) Search(version string, queryEmbedding []float64, topK int) ([]SearchResult, error) {
	var spec specEmbedding
	
	if s.useEmbedded {
		// Load from embedded data
		data, err := LoadEmbeddedEmbeddings(version)
		if err != nil {
			return nil, fmt.Errorf("failed to load embedded embeddings: %w", err)
		}
		if err := json.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("failed to decode embedded spec embedding: %w", err)
		}
	} else {
		// Load from filesystem
		filename := filepath.Join(s.dataDir, fmt.Sprintf("%s.json", version))
		file, err := os.Open(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to open file: %w", err)
		}
		defer func() {
			if err := file.Close(); err != nil {
				fmt.Printf("Warning: Failed to close file %s: %v\n", filename, err)
			}
		}()

		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&spec); err != nil {
			return nil, fmt.Errorf("failed to decode spec embedding: %w", err)
		}
	}

	// Calculate similarities
	var results []SearchResult
	for _, chunk := range spec.Chunks {
		similarity := cosineSimilarity(queryEmbedding, chunk.Embedding)
		results = append(results, SearchResult{
			Content:    chunk.Content,
			ChunkID:    chunk.ID,
			Similarity: similarity,
			Version:    chunk.Version,
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

	results = results[:topK]
	for i := range results {
		results[i].Rank = i + 1
	}

	return results, nil
}

// ListVersions returns all available spec versions in the database
func (s *Store) ListVersions() ([]string, error) {
	if s.useEmbedded {
		// Return embedded versions
		return GetAvailableVersions(), nil
	}
	
	// List from filesystem
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