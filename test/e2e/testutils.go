package e2e

import (
	"os"
	"strings"
	"testing"

	"github.com/carlisia/mcp-factcheck/internal/storage"
	"github.com/carlisia/mcp-factcheck/pkg/llm"
	"github.com/carlisia/mcp-factcheck/pkg/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestEnv initializes a real VectorDB and Generator for use in tests.
// It uses the actual embeddings from the embeddings directory.
func setupTestEnv(t *testing.T) (*storage.VectorDB, *llm.Client) {
	// Use the real embeddings directory - adjust path based on where test runs
	embeddingsDir := "../../internal/storage/embeddings"
	if _, err := os.Stat(embeddingsDir); os.IsNotExist(err) {
		// Try from project root if running from there
		embeddingsDir = "./internal/storage/embeddings"
		if _, err := os.Stat(embeddingsDir); os.IsNotExist(err) {
			t.Fatalf("Embeddings directory does not exist. Run 'task generate:embeddings' from project root first.")
		}
	}

	vectorDB := storage.NewVectorDB(embeddingsDir)
	return vectorDB, createTestRuntimeService(t)
}

// SetupTestEnv is the exported version for use by other test packages
func SetupTestEnv(t *testing.T) (*storage.VectorDB, *llm.Client) {
	return setupTestEnv(t)
}

// createTestRuntimeService creates a test embedding generator.
// It respects existing OPENAI_API_KEY environment variables to avoid overwriting developer credentials.
func createTestRuntimeService(t *testing.T) *llm.Client {
	// Set test API key only if not already set
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Setenv("OPENAI_API_KEY", "test-key")
	}
	// Create with no-op telemetry for tests
	telemetry := logger.NewNoOpTelemetryProvider()
	gen, err := llm.New(telemetry)
	if err != nil {
		t.Fatalf("Failed to create runtime service: %v", err)
	}
	return gen
}

// assertErr checks if error matches expectation.
// It fails the test if the error state doesn't match the expected state.
func assertErr(t *testing.T, err error, wantErr bool) {
	t.Helper()
	if wantErr {
		require.Error(t, err, "Expected error but got none")
	} else {
		require.NoError(t, err, "Expected no error but got: %v", err)
	}
}

// assertNonEmpty checks that result is not empty.
// Use this for tests that expect at least one result.
func assertNonEmpty(t *testing.T, result []mcp.Content) {
	t.Helper()
	require.NotEmpty(t, result, "Expected non-empty result")
}

// assertMinContentCount checks that result has at least expected number of items.
// Use this when the exact count may vary but a minimum is required.
func assertMinContentCount(t *testing.T, result []mcp.Content, min int) {
	t.Helper()
	assert.GreaterOrEqual(t, len(result), min,
		"Expected at least %d content items, got %d", min, len(result))
}

// assertSuccess checks for success and non-empty content.
// This is a convenience function for the common case of expecting no error and at least one result.
func assertSuccess(t *testing.T, err error, result []mcp.Content) {
	t.Helper()
	require.NoError(t, err)
	require.NotEmpty(t, result, "Expected non-empty result")
}

// assertContainsVersion checks that the result contains a specific version string.
// This is useful for verifying that list operations include expected versions.
func assertContainsVersion(t *testing.T, result []mcp.Content, version string) {
	t.Helper()
	for _, content := range result {
		// Check in text content for the version
		if textContent, ok := content.(*mcp.TextContent); ok {
			if strings.Contains(textContent.Text, version) {
				return
			}
		} else if textContent, ok := content.(mcp.TextContent); ok {
			if strings.Contains(textContent.Text, version) {
				return
			}
		}
	}
	require.Fail(t, "Version not found in result", "Expected result to contain version %q", version)
}

// Additional testify-based assertion helpers for MCP content
