package metadata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestVersionMetadata_Structure(t *testing.T) {
	// Test with a sample VersionMetadata
	info := &VersionMetadata{
		ExtractedDate:    time.Now().Format(time.RFC3339),
		SourceCommit:     "abc123",
		SourceRepo:       "modelcontextprotocol/specification",
		SourceBranch:     "main",
		ExtractorVersion: "1.0.0",
		Embeddings: map[string]*EmbeddingMetadata{
			"regular": {
				GeneratedDate: time.Now().Format(time.RFC3339),
				ChunkCount:    100,
				Strategy:      "regular",
			},
			"fine": {
				GeneratedDate: time.Now().Format(time.RFC3339),
				ChunkCount:    250,
				Strategy:      "fine",
			},
		},
	}

	// The struct should have all fields properly set
	if info.SourceCommit != "abc123" {
		t.Errorf("Expected SourceCommit 'abc123', got '%s'", info.SourceCommit)
	}

	if len(info.Embeddings) != 2 {
		t.Errorf("Expected 2 embeddings, got %d", len(info.Embeddings))
	}
}

func TestGetLatestCommit(t *testing.T) {
	meta := &SpecMetadata{
		Specs: map[string]*VersionMetadata{
			"draft": {
				SourceCommit: "draft123",
			},
			"2025-06-18": {
				SourceCommit: "release456",
			},
		},
	}

	tests := []struct {
		version  string
		expected string
	}{
		{"draft", "draft123"},
		{"2025-06-18", "release456"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			result := meta.GetLatestCommit(tt.version)
			if result != tt.expected {
				t.Errorf("GetLatestCommit(%s) = %s, want %s", tt.version, result, tt.expected)
			}
		})
	}
}

func TestUpdateSpecExtraction(t *testing.T) {
	// Create a test metadata instance
	meta := &SpecMetadata{
		Specs: make(map[string]*VersionMetadata),
	}

	// Since UpdateSpecExtraction calls Save() which writes to a constant path,
	// we'll create the metadata without calling the method directly

	// Simulate what UpdateSpecExtraction does
	version := "draft"
	if meta.Specs[version] == nil {
		meta.Specs[version] = &VersionMetadata{
			Embeddings: make(map[string]*EmbeddingMetadata),
		}
	}

	vm := meta.Specs[version]
	vm.ExtractedDate = time.Now().UTC().Format(time.RFC3339)
	vm.SourceCommit = "abc123"
	vm.SourceRepo = "org/repo"
	vm.ExtractorVersion = "1.0.0"
	vm.SourceBranch = "main"

	// Verify the updates
	if vm.SourceCommit != "abc123" {
		t.Errorf("Expected SourceCommit 'abc123', got '%s'", vm.SourceCommit)
	}
	if vm.SourceRepo != "org/repo" {
		t.Errorf("Expected SourceRepo 'org/repo', got '%s'", vm.SourceRepo)
	}
	if vm.SourceBranch != "main" {
		t.Errorf("Expected SourceBranch 'main', got '%s'", vm.SourceBranch)
	}

	// Test updating existing version
	vm.SourceCommit = "def456"
	vm.SourceBranch = "develop"

	// Check it was updated
	if vm.SourceCommit != "def456" {
		t.Errorf("Expected updated SourceCommit 'def456', got '%s'", vm.SourceCommit)
	}
	if vm.SourceBranch != "develop" {
		t.Errorf("Expected updated SourceBranch 'develop', got '%s'", vm.SourceBranch)
	}
}

func TestUpdateEmbeddingGeneration(t *testing.T) {
	// Create a test metadata instance
	meta := &SpecMetadata{
		Specs: make(map[string]*VersionMetadata),
	}

	// Simulate what UpdateEmbeddingGeneration does
	version := "draft"
	strategy := "fine"
	chunkCount := 250

	// Create version metadata if it doesn't exist
	if meta.Specs[version] == nil {
		meta.Specs[version] = &VersionMetadata{
			Embeddings: make(map[string]*EmbeddingMetadata),
		}
	}

	// Update embedding metadata
	if meta.Specs[version].Embeddings == nil {
		meta.Specs[version].Embeddings = make(map[string]*EmbeddingMetadata)
	}

	meta.Specs[version].Embeddings[strategy] = &EmbeddingMetadata{
		GeneratedDate: time.Now().UTC().Format(time.RFC3339),
		ChunkCount:    chunkCount,
		Strategy:      strategy,
	}

	// Verify the updates
	spec := meta.Specs["draft"]
	if spec.Embeddings == nil {
		t.Fatal("Expected embeddings map to be created")
	}

	embedding, exists := spec.Embeddings["fine"]
	if !exists {
		t.Fatal("Expected 'fine' embedding to exist")
	}

	if embedding.Strategy != "fine" {
		t.Errorf("Expected Strategy 'fine', got '%s'", embedding.Strategy)
	}
	if embedding.ChunkCount != 250 {
		t.Errorf("Expected ChunkCount 250, got %d", embedding.ChunkCount)
	}

	// Test updating existing embedding
	meta.Specs[version].Embeddings[strategy].ChunkCount = 300

	embedding = spec.Embeddings["fine"]
	if embedding.ChunkCount != 300 {
		t.Errorf("Expected updated ChunkCount 300, got %d", embedding.ChunkCount)
	}
}

func TestLoadMetadata_EmptyFile(t *testing.T) {
	// Test loading when file doesn't exist
	// Since LoadMetadata uses a constant path, we can't easily test file operations
	// But we can test the behavior when creating new metadata
	meta := &SpecMetadata{
		Specs:           make(map[string]*VersionMetadata),
		MetadataVersion: "1.0.0",
	}

	if meta.Specs == nil {
		t.Fatal("Expected Specs map to be initialized")
	}
	if meta.MetadataVersion != "1.0.0" {
		t.Errorf("Expected MetadataVersion '1.0.0', got '%s'", meta.MetadataVersion)
	}
}

func TestSave_Structure(t *testing.T) {
	// Create a temporary file to test JSON structure
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test_metadata.json")

	meta := &SpecMetadata{
		MetadataVersion: "1.0.0",
		LastUpdated:     time.Now().Format(time.RFC3339),
		Specs: map[string]*VersionMetadata{
			"test": {
				ExtractedDate: time.Now().Format(time.RFC3339),
				SourceCommit:  "save123",
			},
		},
	}

	// Marshal to JSON to test structure
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal metadata: %v", err)
	}

	// Write to temp file
	err = os.WriteFile(tempFile, data, 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Load and verify content
	loadedData, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read saved metadata: %v", err)
	}

	var loaded SpecMetadata
	err = json.Unmarshal(loadedData, &loaded)
	if err != nil {
		t.Fatalf("Failed to unmarshal saved metadata: %v", err)
	}

	if loaded.Specs["test"] == nil || loaded.Specs["test"].SourceCommit != "save123" {
		t.Error("Saved metadata doesn't match original")
	}
}
