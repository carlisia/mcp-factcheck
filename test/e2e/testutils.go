// Package e2e contains end-to-end tests and test utilities for the MCP fact-check server.
package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/carlisia/mcp-factcheck/internal/embedding/core"
	"github.com/carlisia/mcp-factcheck/pkg/embedtypes"
	mcpembedding "github.com/carlisia/mcp-factcheck/internal/embedding"
	"github.com/mark3labs/mcp-go/mcp"
)

// setupTestEnv initializes a default VectorDB and Generator for use in tests.
// It returns mock-backed, isolated instances with default content.
func setupTestEnv(t *testing.T) (*mcpembedding.VectorDB, *core.Generator) {
	return createTestVectorDB(t), createTestGenerator(t)
}

// defaultMetadata creates a default metadata map with the given section.
// This is useful for creating consistent test data.
func defaultMetadata(section string) map[string]any {
	return map[string]any{"section": section}
}

// createTestVectorDB creates a temporary vector database for testing with default chunks.
// It sets up a basic test environment with protocol and overview sections.
func createTestVectorDB(t *testing.T) *mcpembedding.VectorDB {
	// Use the default spec version to match what handlers expect
	defaultVersion := "2025-06-18"
	defaultChunks := []embedtypes.EmbeddedChunk{
		{
			ID:        "chunk1",
			Version:   defaultVersion,
			Section:   "protocol",
			Content:   "MCP uses JSON-RPC 2.0 for communication",
			Embedding: generateMockEmbedding("MCP uses JSON-RPC 2.0"),
			Metadata:  defaultMetadata("protocol"),
		},
		{
			ID:        "chunk2",
			Version:   defaultVersion,
			Section:   "overview",
			Content:   "The Model Context Protocol enables seamless integration",
			Embedding: generateMockEmbedding("Model Context Protocol"),
			Metadata:  defaultMetadata("overview"),
		},
		{
			ID:        "chunk3",
			Version:   defaultVersion,
			Section:   "query",
			Content:   "MCP query interface allows flexible searching",
			Embedding: generateMockEmbedding("MCP query interface"),
			Metadata:  defaultMetadata("query"),
		},
	}
	return createTestVectorDBWithChunks(t, defaultVersion, defaultChunks)
}

// createTestVectorDBWithChunks creates a test vector database with custom chunks.
// This allows tests to specify custom content and versions for specific test scenarios.
func createTestVectorDBWithChunks(t *testing.T, version string, chunks []embedtypes.EmbeddedChunk) *mcpembedding.VectorDB {
	t.Helper()
	tempDir := t.TempDir()

	data := embedtypes.SpecEmbedding{
		Version: version,
		Chunks:  chunks,
		Count:   len(chunks),
	}

	testData, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Failed to marshal test embeddings: %v", err)
	}

	path := filepath.Join(tempDir, version+".json")
	if err := os.WriteFile(path, testData, 0644); err != nil {
		t.Fatalf("Failed to write embedding file: %v", err)
	}

	return mcpembedding.NewVectorDB(tempDir)
}

// createTestGenerator creates a test embedding generator.
// It respects existing OPENAI_API_KEY environment variables to avoid overwriting developer credentials.
func createTestGenerator(t *testing.T) *core.Generator {
	// Set test API key only if not already set
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Setenv("OPENAI_API_KEY", "test-key")
	}
	gen, err := core.NewGenerator()
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}
	return gen
}

// generateMockEmbedding creates a deterministic mock embedding based on text length.
// The embedding is consistent for the same input text, making tests predictable.
func generateMockEmbedding(text string) []float64 {
	emb := make([]float64, 1536)
	scale := float64(len(text))
	for i := range emb {
		emb[i] = scale / float64(i+1) // explicit float conversion for clarity
	}
	return emb
}

// assertErr checks if error matches expectation.
// It fails the test if the error state doesn't match the expected state.
func assertErr(t *testing.T, err error, wantErr bool) {
	t.Helper()
	if (err != nil) != wantErr {
		t.Fatalf("Error mismatch: got err=%v, wantErr=%v", err, wantErr)
	}
}

// assertNonEmpty checks that result is not empty.
// Use this for tests that expect at least one result.
func assertNonEmpty(t *testing.T, result []mcp.Content) {
	t.Helper()
	if len(result) == 0 {
		t.Fatalf("Expected non-empty result, got none")
	}
}


// assertMinContentCount checks that result has at least expected number of items.
// Use this when the exact count may vary but a minimum is required.
func assertMinContentCount(t *testing.T, result []mcp.Content, min int) {
	t.Helper()
	if len(result) < min {
		t.Fatalf("Expected at least %d content items, got %d", min, len(result))
	}
}

// assertSuccess checks for success and non-empty content.
// This is a convenience function for the common case of expecting no error and at least one result.
func assertSuccess(t *testing.T, err error, result []mcp.Content) {
	t.Helper()
	assertErr(t, err, false)
	assertNonEmpty(t, result)
}


// assertContainsVersion checks that the result contains a specific version string.
// This is useful for verifying that list operations include expected versions.
func assertContainsVersion(t *testing.T, result []mcp.Content, version string) {
	t.Helper()
	for _, content := range result {
		// Check in text content for the version
		if textContent, ok := content.(*mcp.TextContent); ok {
			if contains(textContent.Text, version) {
				return
			}
		} else if textContent, ok := content.(mcp.TextContent); ok {
			if contains(textContent.Text, version) {
				return
			}
		}
	}
	t.Fatalf("Expected result to contain version %q", version)
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsAt(s, substr)
}

// containsAt checks if string contains substring at any position
func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
