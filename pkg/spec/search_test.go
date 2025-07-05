package spec

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/carlisia/mcp-factcheck/internal/embedding/core"
	"github.com/carlisia/mcp-factcheck/internal/specs"
	"github.com/carlisia/mcp-factcheck/testutil"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestGetSearchSpecTool(t *testing.T) {
	tool := GetSearchSpecTool()

	// Test tool properties
	if tool.Name != "search_spec" {
		t.Errorf("Expected tool name 'search_spec', got '%s'", tool.Name)
	}

	if tool.Description == "" {
		t.Error("Tool description should not be empty")
	}
}

func TestSpecVersionValidation(t *testing.T) {
	tests := []struct {
		version string
		valid   bool
	}{
		{"draft", true},
		{"2025-06-18", true},
		{"2025-03-26", true},
		{"2024-11-05", true},
		{"invalid", false},
		{"", false},
		{"2023-01-01", false},
		{"DRAFT", false}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("version=%s", tt.version), func(t *testing.T) {
			result := specs.IsValidSpecVersion(tt.version)
			if result != tt.valid {
				t.Errorf("IsValidSpecVersion(%s) = %v, want %v", tt.version, result, tt.valid)
			}
		})
	}
}

func TestHandleSearchSpec(t *testing.T) {
	// Adapter to convert our local interfaces to testutil interfaces
	handler := func(ctx context.Context, db testutil.VectorDB, gen testutil.EmbeddingGenerator, args any) ([]mcp.Content, error) {
		// Use centralized adapters to satisfy handler interfaces
		localDB := testutil.NewVectorDBAdapter(db)
		localGen := testutil.NewEmbeddingGeneratorAdapter(gen)
		return HandleSearchSpec(ctx, localDB, localGen, args)
	}

	testCases := []testutil.HandlerTestCase{
		// === Argument validation ===
		{
			Name: "invalid arguments type",
			Args: "not a map",
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				return &testutil.MockVectorDB{}, &testutil.MockEmbeddingGenerator{}
			},
			WantErr: true,
			ValidateError: func(t *testing.T, err error) {
				testutil.AssertErrorContains(t, err, "arguments must be a map")
			},
		},
		{
			Name: "missing query parameter",
			Args: map[string]any{
				"specVersion": specs.DefaultSpecVersion,
			},
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				return &testutil.MockVectorDB{}, &testutil.MockEmbeddingGenerator{}
			},
			WantErr: true,
			ValidateError: func(t *testing.T, err error) {
				testutil.AssertErrorContains(t, err, "query must be a string")
			},
		},
		{
			Name: "empty query",
			Args: map[string]any{
				"query": "",
			},
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				return testutil.NewMockVectorDBWithResults([]core.SearchResult{}),
					testutil.NewMockEmbeddingGeneratorWithEmbedding([]float64{0.1, 0.2, 0.3})
			},
			WantErr: false,
			ValidateResult: func(t *testing.T, result []mcp.Content) {
				// Empty query is allowed, returns header only
				testutil.AssertContentCount(t, result, 1)
				testutil.AssertTextContains(t, result, "Search results for ''")
			},
		},
		{
			Name: "non-string query",
			Args: map[string]any{
				"query": 123,
			},
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				return &testutil.MockVectorDB{}, &testutil.MockEmbeddingGenerator{}
			},
			WantErr: true,
			ValidateError: func(t *testing.T, err error) {
				testutil.AssertErrorContains(t, err, "query must be a string")
			},
		},

		// === Spec version validation ===
		{
			Name: "invalid spec version",
			Args: map[string]any{
				"query":       "test query",
				"specVersion": "invalid-version",
			},
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				return &testutil.MockVectorDB{}, &testutil.MockEmbeddingGenerator{}
			},
			WantErr: true,
			ValidateError: func(t *testing.T, err error) {
				testutil.AssertErrorContains(t, err, "invalid spec version: invalid-version")
			},
		},

		// === Embedding errors ===
		{
			Name: "embedding generation error",
			Args: map[string]any{
				"query": "test query",
			},
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				return &testutil.MockVectorDB{},
					testutil.NewMockEmbeddingGeneratorWithError(errors.New("API rate limit exceeded"))
			},
			WantErr: true,
			ValidateError: func(t *testing.T, err error) {
				testutil.AssertErrorContains(t, err, "failed to generate embedding")
				testutil.AssertErrorContains(t, err, "API rate limit exceeded")
			},
		},
		{
			Name: "context cancellation during embedding",
			Args: map[string]any{
				"query": "test query",
			},
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				return &testutil.MockVectorDB{},
					testutil.NewMockEmbeddingGeneratorWithError(context.Canceled)
			},
			WantErr: true,
			ValidateError: func(t *testing.T, err error) {
				if !errors.Is(err, context.Canceled) {
					t.Errorf("Expected context.Canceled, got %v", err)
				}
			},
		},

		// === Vector DB errors ===
		{
			Name: "vector search error",
			Args: map[string]any{
				"query": "test query",
			},
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				return testutil.NewMockVectorDBWithError(errors.New("database connection failed")),
					testutil.NewMockEmbeddingGeneratorWithEmbedding([]float64{0.1, 0.2, 0.3})
			},
			WantErr: true,
			ValidateError: func(t *testing.T, err error) {
				testutil.AssertErrorContains(t, err, "failed to search specifications")
				testutil.AssertErrorContains(t, err, "database connection failed")
			},
		},

		// === Success cases ===
		{
			Name: "successful search with results",
			Args: map[string]any{
				"query": "MCP protocol",
			},
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				results := testutil.CreateTestSearchResults(
					"MCP uses JSON-RPC 2.0 protocol for communication",
					"The protocol defines request/response message patterns",
					"Protocol versioning follows semantic versioning",
				)
				return testutil.NewMockVectorDBWithResults(results),
					testutil.NewMockEmbeddingGeneratorWithEmbedding([]float64{0.1, 0.2, 0.3})
			},
			WantErr: false,
			ValidateResult: func(t *testing.T, result []mcp.Content) {
				// Header + 3 results
				testutil.AssertContentCount(t, result, 4)
				testutil.AssertAllTextContent(t, result)
				testutil.AssertTextContains(t, result, "Search results for 'MCP protocol'")
				testutil.AssertTextContains(t, result, "JSON-RPC 2.0")
				testutil.AssertTextContains(t, result, "request/response")
				testutil.AssertTextContains(t, result, "Rank 1")
				testutil.AssertTextContains(t, result, "similarity: 0.9000")
			},
		},
		{
			Name: "successful search with no results",
			Args: map[string]any{
				"query": "nonexistent topic xyz123",
			},
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				return testutil.NewMockVectorDBWithResults([]core.SearchResult{}),
					testutil.NewMockEmbeddingGeneratorWithEmbedding([]float64{0.1, 0.2, 0.3})
			},
			WantErr: false,
			ValidateResult: func(t *testing.T, result []mcp.Content) {
				// Just header, no results
				testutil.AssertContentCount(t, result, 1)
				testutil.AssertTextContains(t, result, "Search results for 'nonexistent topic xyz123'")
			},
		},
		{
			Name: "search with custom topK",
			Args: map[string]any{
				"query": "test",
				"topK":  float64(3), // JSON numbers are float64
			},
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				db := &testutil.MockVectorDB{
					SearchFunc: func(version string, queryEmbedding []float64, topK int) ([]core.SearchResult, error) {
						if topK != 3 {
							t.Errorf("Expected topK=3, got %d", topK)
						}
						return testutil.CreateTestSearchResults("Result 1", "Result 2", "Result 3"), nil
					},
				}
				return db, testutil.NewMockEmbeddingGeneratorWithEmbedding([]float64{0.1, 0.2})
			},
			WantErr: false,
			ValidateResult: func(t *testing.T, result []mcp.Content) {
				testutil.AssertContentCount(t, result, 4) // header + 3 results
			},
		},
		{
			Name: "search with default spec version",
			Args: map[string]any{
				"query": "test",
			},
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				db := &testutil.MockVectorDB{
					SearchFunc: func(version string, queryEmbedding []float64, topK int) ([]core.SearchResult, error) {
						if version != specs.DefaultSpecVersion {
							t.Errorf("Expected version=%s, got %s", specs.DefaultSpecVersion, version)
						}
						return []core.SearchResult{}, nil
					},
				}
				return db, testutil.NewMockEmbeddingGeneratorWithEmbedding([]float64{0.1})
			},
			WantErr: false,
			ValidateResult: func(t *testing.T, result []mcp.Content) {
				testutil.AssertTextContains(t, result, specs.DefaultSpecVersion)
			},
		},
		{
			Name: "unicode content handling",
			Args: map[string]any{
				"query": "unicode test 你好世界",
			},
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				results := []core.SearchResult{
					{
						Chunk: core.EmbeddedChunk{
							Content: "Unicode support: 你好世界 (Hello World) 🌍 émojis work too!",
						},
						Similarity: 0.95,
						Rank:       1,
					},
				}
				return testutil.NewMockVectorDBWithResults(results),
					testutil.NewMockEmbeddingGeneratorWithEmbedding([]float64{0.1, 0.2})
			},
			WantErr: false,
			ValidateResult: func(t *testing.T, result []mcp.Content) {
				testutil.AssertTextContains(t, result, "你好世界")
				testutil.AssertTextContains(t, result, "🌍")
				testutil.AssertTextContains(t, result, "émojis")
			},
		},
	}

	testutil.RunHandlerTestCases(t, handler, testCases)
}
