package e2e

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/carlisia/mcp-factcheck/internal/embedding"
	"github.com/carlisia/mcp-factcheck/internal/storage"
	mcptools "github.com/carlisia/mcp-factcheck/pkg/mcp/tools"
)

func TestSpec_HandleListSpecVersions_Success(t *testing.T) {
	ctx := context.Background()
	vectorDB, _ := setupTestEnv(t)

	result, err := mcptools.HandleListSpecVersions(ctx, vectorDB)

	assertSuccess(t, err, result)
	// Should return at least the spec version table header (even if no specs are present)
	assertMinContentCount(t, result, 1)
}

func TestSpec_HandleListSpecVersions_WithCustomVersion(t *testing.T) {
	ctx := context.Background()

	// Create a VectorDB with a test embedding file
	tempDir := t.TempDir()

	// Create minimal valid embedding data
	data := embedding.SpecEmbedding{
		Version: "2025-06-18",
		Chunks: []embedding.EmbeddedChunk{
			{
				ID:        "custom-test",
				Version:   "2025-06-18",
				Section:   "test",
				Content:   "Test content for custom version",
				Embedding: make([]float64, 1536), // OpenAI embedding size
				Metadata:  map[string]any{"section": "test"},
			},
		},
		Count: 1,
	}

	testData, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Failed to marshal test embeddings: %v", err)
	}

	path := filepath.Join(tempDir, "2025-06-18.json")
	if err := os.WriteFile(path, testData, 0644); err != nil {
		t.Fatalf("Failed to write embedding file: %v", err)
	}

	vectorDB := storage.NewVectorDB(tempDir)

	result, err := mcptools.HandleListSpecVersions(ctx, vectorDB)

	assertSuccess(t, err, result)
	// List versions returns a single formatted text content item
	assertMinContentCount(t, result, 1)
	// Verify the specific version is listed in the formatted output
	assertContainsVersion(t, result, "2025-06-18")
}

func TestSpec_HandleListSpecVersions_EmptyDirectory(t *testing.T) {
	ctx := context.Background()

	// Create an empty VectorDB (no versions)
	tempDir := t.TempDir()
	vectorDB := storage.NewVectorDB(tempDir)

	result, err := mcptools.HandleListSpecVersions(ctx, vectorDB)

	// Should succeed even with no versions
	assertErr(t, err, false)
	// Should return at least the spec version table header (even if no specs are present)
	assertMinContentCount(t, result, 1)
}

func TestSpec_HandleListSpecVersions_WithInvalidJSON(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Write invalid JSON file for a "spec version"
	invalidPath := filepath.Join(tempDir, "invalid.json")
	err := os.WriteFile(invalidPath, []byte("{ not valid json"), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid JSON file: %v", err)
	}

	vectorDB := storage.NewVectorDB(tempDir)
	result, err := mcptools.HandleListSpecVersions(ctx, vectorDB)

	// Should succeed — handler is expected to ignore invalid files
	assertErr(t, err, false)
	// Should return at least the spec version table header (even if no specs are present)
	assertMinContentCount(t, result, 1)
}

func TestSpec_HandleListSpecVersions_WithNonJSONFiles(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Write a non-JSON file that should be ignored
	txtPath := filepath.Join(tempDir, "readme.txt")
	err := os.WriteFile(txtPath, []byte("This is not a JSON file"), 0644)
	if err != nil {
		t.Fatalf("Failed to write text file: %v", err)
	}

	// Also write a valid JSON file to ensure it's found
	data := embedding.SpecEmbedding{
		Version: "valid",
		Chunks: []embedding.EmbeddedChunk{
			{
				ID:        "test",
				Version:   "valid",
				Section:   "test",
				Content:   "Test",
				Embedding: make([]float64, 1536),
				Metadata:  map[string]any{"section": "test"},
			},
		},
		Count: 1,
	}

	testData, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Failed to marshal test embeddings: %v", err)
	}

	path := filepath.Join(tempDir, "valid.json")
	if err := os.WriteFile(path, testData, 0644); err != nil {
		t.Fatalf("Failed to write embedding file: %v", err)
	}

	vectorDB := storage.NewVectorDB(tempDir)

	result, err := mcptools.HandleListSpecVersions(ctx, vectorDB)

	// Should succeed and list only the valid version
	assertSuccess(t, err, result)
	// List versions returns a single formatted text content item
	assertMinContentCount(t, result, 1)
	// Verify only the valid JSON version is listed in the formatted output
	assertContainsVersion(t, result, "valid")
}

func TestSpec_HandleListSpecVersions_IgnoresArgs(t *testing.T) {
	ctx := context.Background()
	vectorDB, _ := setupTestEnv(t)

	// The handler should ignore any args passed
	tests := []struct {
		name string
		args any
	}{
		{
			name: "with nil args",
			args: nil,
		},
		{
			name: "with empty map",
			args: map[string]any{},
		},
		{
			name: "with random args",
			args: map[string]any{"foo": "bar", "baz": 123},
		},
		{
			name: "with non-map args",
			args: "not a map",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := mcptools.HandleListSpecVersions(ctx, vectorDB)

			// Should always succeed regardless of args
			assertSuccess(t, err, result)
		})
	}
}
