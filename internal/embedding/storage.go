package embedding

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Storage handles reading and writing embedding files.
type Storage struct {
	dataDir string
}

// NewEmbeddingStorage creates a new embedding storage handler.
func NewEmbeddingStorage(dataDir string) *Storage {
	return &Storage{dataDir: dataDir}
}

// WriteEmbeddings stores the embeddings to disk as JSON.
func (s *Storage) WriteEmbeddings(specEmbedding *SpecEmbedding) error {
	filename := filepath.Join(s.dataDir, fmt.Sprintf("%s.json", specEmbedding.Version))

	// Ensure directory exists
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Write the spec embedding to file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			// Log error or handle as appropriate
			_ = err
		}
	}()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(specEmbedding); err != nil {
		return fmt.Errorf("failed to encode spec embedding: %w", err)
	}

	return nil
}

// ReadEmbeddings loads embeddings from disk.
func (s *Storage) ReadEmbeddings(version string) (*SpecEmbedding, error) {
	filename := filepath.Join(s.dataDir, fmt.Sprintf("%s.json", version))

	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			// Log error or handle as appropriate
			_ = err
		}
	}()

	var specEmbedding SpecEmbedding
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&specEmbedding); err != nil {
		return nil, fmt.Errorf("failed to decode spec embedding: %w", err)
	}

	return &specEmbedding, nil
}

// LoadChunksFromJSON loads chunks from a JSON file containing spec data.
func LoadChunksFromJSON(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			// Log error or handle as appropriate
			_ = err
		}
	}()

	var data struct {
		Chunks []string `json:"chunks"`
		Count  int      `json:"count"`
	}

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	if len(data.Chunks) == 0 {
		return nil, fmt.Errorf("no chunks found in file")
	}

	return data.Chunks, nil
}

