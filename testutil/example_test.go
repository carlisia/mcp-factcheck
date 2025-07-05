package testutil_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/carlisia/mcp-factcheck/pkg/embedtypes"
	"github.com/carlisia/mcp-factcheck/testutil"
	"github.com/mark3labs/mcp-go/mcp"
)

// SampleHandlerFunc demonstrates a handler that would be tested
func SampleHandlerFunc(ctx context.Context, db testutil.VectorDB, gen testutil.EmbeddingGenerator, args any) ([]mcp.Content, error) {
	// Validate arguments
	params, ok := args.(map[string]any)
	if !ok {
		return nil, errors.New("arguments must be a map")
	}

	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, errors.New("query must be a non-empty string")
	}

	// Generate embedding
	embedding, err := gen.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, err
	}

	// Search
	results, err := db.Search("test", embedding, 5)
	if err != nil {
		return nil, err
	}

	// Build response
	var content []mcp.Content
	content = append(content, mcp.NewTextContent("Search Results:\n"))
	for _, result := range results {
		content = append(content, mcp.NewTextContent(result.Chunk.Content))
	}
	return content, nil
}

// TestExampleHandler_WithHarness demonstrates using the test harness
func TestExampleHandler_WithHarness(t *testing.T) {
	testCases := []testutil.HandlerTestCase{
		// === Argument validation tests ===
		{
			Name: "nil arguments",
			Args: nil,
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				return &testutil.MockVectorDB{}, &testutil.MockEmbeddingGenerator{}
			},
			WantErr: true,
			ValidateError: func(t *testing.T, err error) {
				testutil.AssertErrorContains(t, err, "arguments must be a map")
			},
		},
		{
			Name: "missing query",
			Args: map[string]any{},
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				return &testutil.MockVectorDB{}, &testutil.MockEmbeddingGenerator{}
			},
			WantErr: true,
			ValidateError: func(t *testing.T, err error) {
				testutil.AssertErrorContains(t, err, "query must be a non-empty string")
			},
		},

		// === Error handling tests ===
		{
			Name: "embedding generation error",
			Args: map[string]any{"query": "test"},
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				return &testutil.MockVectorDB{},
					testutil.NewMockEmbeddingGeneratorWithError(testutil.ErrTestEmbedding)
			},
			WantErr: true,
			ValidateError: func(t *testing.T, err error) {
				if !errors.Is(err, testutil.ErrTestEmbedding) {
					t.Errorf("Expected ErrTestEmbedding, got %v", err)
				}
			},
		},
		{
			Name: "database error",
			Args: map[string]any{"query": "test"},
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				return testutil.NewMockVectorDBWithError(testutil.ErrTestDatabase),
					testutil.NewMockEmbeddingGeneratorWithEmbedding([]float64{0.1, 0.2})
			},
			WantErr: true,
			ValidateError: func(t *testing.T, err error) {
				if !errors.Is(err, testutil.ErrTestDatabase) {
					t.Errorf("Expected ErrTestDatabase, got %v", err)
				}
			},
		},

		// === Success cases ===
		{
			Name: "successful search with results",
			Args: map[string]any{"query": "MCP protocol"},
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				results := testutil.CreateTestSearchResults(
					"MCP uses JSON-RPC 2.0",
					"Protocol defines request/response patterns",
				)
				return testutil.NewMockVectorDBWithResults(results),
					testutil.NewMockEmbeddingGeneratorWithEmbedding([]float64{0.1, 0.2})
			},
			WantErr: false,
			ValidateResult: func(t *testing.T, result []mcp.Content) {
				testutil.AssertNonEmpty(t, result)
				testutil.AssertAllTextContent(t, result)
				testutil.AssertTextContains(t, result, "Search Results:")
				testutil.AssertTextContains(t, result, "MCP uses JSON-RPC 2.0")
				testutil.AssertTextContainsAny(t, result, "Protocol", "protocol")
				// Verify we have exactly 2 content items with actual results
				testutil.AssertTextContainsCount(t, result, "JSON-RPC", 1)
			},
		},
		{
			Name: "empty search results",
			Args: map[string]any{"query": "nonexistent"},
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				return testutil.NewMockVectorDBWithResults([]embedtypes.SearchResult{}),
					testutil.NewMockEmbeddingGeneratorWithEmbedding([]float64{0.1, 0.2})
			},
			WantErr: false,
			ValidateResult: func(t *testing.T, result []mcp.Content) {
				testutil.AssertContentCount(t, result, 1) // Just the header
				testutil.AssertTextContains(t, result, "Search Results:")
			},
		},
	}

	// Run all test cases
	testutil.RunHandlerTestCases(t, SampleHandlerFunc, testCases)
}

// TestExampleHandler_WithCustomSetup demonstrates context cancellation testing
func TestExampleHandler_WithCustomSetup(t *testing.T) {
	testCases := []testutil.HandlerTestCase{
		{
			Name: "context timeout during embedding",
			Args: map[string]any{"query": "test"},
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				return &testutil.MockVectorDB{}, testutil.NewMockEmbeddingGeneratorSlow()
			},
			WantErr: true,
			ValidateError: func(t *testing.T, err error) {
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Errorf("Expected context.DeadlineExceeded, got %v", err)
				}
			},
		},
		{
			Name: "context cancellation",
			Args: map[string]any{"query": "test"},
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				return &testutil.MockVectorDB{}, testutil.NewMockEmbeddingGeneratorSlow()
			},
			WantErr: true,
			ValidateError: func(t *testing.T, err error) {
				if !errors.Is(err, context.Canceled) {
					t.Errorf("Expected context.Canceled, got %v", err)
				}
			},
		},
	}

	// Custom setup function to handle context per test
	setupFunc := func(t *testing.T, tc *testutil.HandlerTestCase) context.Context {
		switch tc.Name {
		case "context timeout during embedding":
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
			t.Cleanup(cancel) // Ensure cancel is called when test completes
			return ctx
		case "context cancellation":
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately
			return ctx
		default:
			return context.Background()
		}
	}

	testutil.RunHandlerTestCasesWithSetup(t, SampleHandlerFunc, testCases, setupFunc)
}
