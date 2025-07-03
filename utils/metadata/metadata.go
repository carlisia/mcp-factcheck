// Package metadata provides functionality for managing spec version metadata
// tracking extraction dates, commit hashes, and embedding generation details.
package metadata

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const metadataFile = "data/SPEC_METADATA.json"

// SpecMetadata represents metadata for all spec versions
type SpecMetadata struct {
	Specs           map[string]*VersionMetadata `json:"specs"`
	MetadataVersion string                      `json:"metadata_version"`
	LastUpdated     string                      `json:"last_updated"`
	mu              sync.Mutex                  // For thread-safe updates
}

// VersionMetadata represents metadata for a single spec version
type VersionMetadata struct {
	ReleaseDate      string                        `json:"release_date,omitempty"` // For official versions
	ExtractedDate    string                        `json:"extracted_date"`
	SourceCommit     string                        `json:"source_commit"`
	SourceRepo       string                        `json:"source_repo"`
	SourceBranch     string                        `json:"source_branch,omitempty"` // For draft
	SourceTag        string                        `json:"source_tag,omitempty"`    // For releases
	ExtractorVersion string                        `json:"extractor_version"`
	Embeddings       map[string]*EmbeddingMetadata `json:"embeddings"`
}

// EmbeddingMetadata represents metadata for a single embedding strategy
type EmbeddingMetadata struct {
	GeneratedDate string `json:"generated_date"`
	ChunkCount    int    `json:"chunk_count"`
	Strategy      string `json:"strategy"`
}

// LoadMetadata loads the metadata file or creates a new one if it doesn't exist
func LoadMetadata() (*SpecMetadata, error) {
	metadata := &SpecMetadata{
		Specs:           make(map[string]*VersionMetadata),
		MetadataVersion: "1.0.0",
	}

	// Check if file exists
	if _, err := os.Stat(metadataFile); os.IsNotExist(err) {
		// File doesn't exist, return empty metadata
		return metadata, nil
	}

	// Read existing file
	data, err := os.ReadFile(metadataFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	// Parse JSON
	if err := json.Unmarshal(data, metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	// Initialize maps if nil
	if metadata.Specs == nil {
		metadata.Specs = make(map[string]*VersionMetadata)
	}

	return metadata, nil
}

// Save writes the metadata back to the file
func (m *SpecMetadata) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update last_updated timestamp
	m.LastUpdated = time.Now().UTC().Format(time.RFC3339)

	// Create directory if it doesn't exist
	dir := filepath.Dir(metadataFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Write to file
	if err := os.WriteFile(metadataFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}

// UpdateSpecExtraction updates metadata after spec extraction
func (m *SpecMetadata) UpdateSpecExtraction(version, commit, repo, branchOrTag string, chunkCount int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create version metadata if it doesn't exist
	if m.Specs[version] == nil {
		m.Specs[version] = &VersionMetadata{
			Embeddings: make(map[string]*EmbeddingMetadata),
		}
	}

	vm := m.Specs[version]
	vm.ExtractedDate = time.Now().UTC().Format(time.RFC3339)
	vm.SourceCommit = commit
	vm.SourceRepo = repo
	vm.ExtractorVersion = "1.0.0" // Could be passed as parameter or from build info

	// Set branch or tag based on version
	if version == "draft" {
		vm.SourceBranch = branchOrTag
	} else {
		vm.SourceTag = branchOrTag
		if vm.ReleaseDate == "" {
			vm.ReleaseDate = version // Use version as release date if not set
		}
	}

	return m.Save()
}

// UpdateEmbeddingGeneration updates metadata after embedding generation
func (m *SpecMetadata) UpdateEmbeddingGeneration(version, strategy string, chunkCount int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create version metadata if it doesn't exist
	if m.Specs[version] == nil {
		m.Specs[version] = &VersionMetadata{
			Embeddings: make(map[string]*EmbeddingMetadata),
		}
	}

	// Update embedding metadata
	if m.Specs[version].Embeddings == nil {
		m.Specs[version].Embeddings = make(map[string]*EmbeddingMetadata)
	}

	m.Specs[version].Embeddings[strategy] = &EmbeddingMetadata{
		GeneratedDate: time.Now().UTC().Format(time.RFC3339),
		ChunkCount:    chunkCount,
		Strategy:      strategy,
	}

	return m.Save()
}

// GetLatestCommit returns the source commit for a given version
func (m *SpecMetadata) GetLatestCommit(version string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	if vm, exists := m.Specs[version]; exists {
		return vm.SourceCommit
	}
	return ""
}
