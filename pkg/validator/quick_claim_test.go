package validator

import (
	"context"
	"errors"
	"testing"

	"github.com/carlisia/mcp-factcheck/internal/embedding/core"
	"github.com/carlisia/mcp-factcheck/internal/specs"
	"github.com/carlisia/mcp-factcheck/testutil"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestHandleCheckMCPQuickFact(t *testing.T) {
	// Create adapters to convert testutil interfaces to our handler's interfaces
	handler := func(ctx context.Context, db testutil.VectorDB, gen testutil.EmbeddingGenerator, args any) ([]mcp.Content, error) {
		// Use centralized adapters to satisfy handler interfaces
		vectorDB := testutil.NewVectorDBAdapter(db)
		generator := testutil.NewFactCheckingEmbeddingGeneratorAdapter(gen)

		// Call the real handler with adapted interfaces
		return HandleCheckMCPQuickFact(ctx, vectorDB, generator, args)
	}

	testCases := []testutil.HandlerTestCase{
		{
			Name: "valid quick fact check - accurate claim",
			Args: map[string]any{
				"claim":       "MCP uses JSON-RPC",
				"specVersion": specs.DefaultSpecVersion,
			},
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				// Return search results that confirm the claim
				results := testutil.CreateTestSearchResults(
					"MCP uses JSON-RPC 2.0 for all communication between clients and servers",
					"The protocol is based on JSON-RPC 2.0 specification",
				)
				return testutil.NewMockVectorDBWithResults(results),
					testutil.NewMockEmbeddingGeneratorWithEmbedding([]float64{0.1, 0.2, 0.3})
			},
			WantErr: false,
			ValidateResult: func(t *testing.T, result []mcp.Content) {
				testutil.AssertContentCount(t, result, 1)
				testutil.AssertTextContains(t, result, "✓")
				testutil.AssertTextContains(t, result, "Accurate")
				testutil.AssertTextContains(t, result, "MCP uses JSON-RPC")
			},
		},
		{
			Name: "inaccurate claim",
			Args: map[string]any{
				"claim": "MCP requires XML format",
			},
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				// Return search results that contradict the claim
				results := testutil.CreateTestSearchResults(
					"MCP uses JSON-RPC 2.0 format exclusively, not XML",
					"All messages must be valid JSON-RPC 2.0",
				)
				return testutil.NewMockVectorDBWithResults(results),
					testutil.NewMockEmbeddingGeneratorWithEmbedding([]float64{0.1, 0.2, 0.3})
			},
			WantErr: false,
			ValidateResult: func(t *testing.T, result []mcp.Content) {
				testutil.AssertContentCount(t, result, 1)
				testutil.AssertTextContains(t, result, "✗")
				testutil.AssertTextContains(t, result, "Inaccurate")
				testutil.AssertTextContains(t, result, "MCP requires XML format")
			},
		},
		{
			Name: "empty claim",
			Args: map[string]any{
				"claim": "",
			},
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				return &testutil.MockVectorDB{}, &testutil.MockEmbeddingGenerator{}
			},
			WantErr: true,
			ValidateError: func(t *testing.T, err error) {
				testutil.AssertErrorContains(t, err, "claim must be a non-empty string")
			},
		},
		{
			Name: "invalid arguments type",
			Args: 123,
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				return &testutil.MockVectorDB{}, &testutil.MockEmbeddingGenerator{}
			},
			WantErr: true,
			ValidateError: func(t *testing.T, err error) {
				testutil.AssertErrorContains(t, err, errArgumentsNotMap.Error())
			},
		},
		{
			Name: "invalid spec version",
			Args: map[string]any{
				"claim":       "MCP uses JSON",
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
		{
			Name: "embedding generation error",
			Args: map[string]any{
				"claim": "MCP test claim",
			},
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				return &testutil.MockVectorDB{},
					testutil.NewMockEmbeddingGeneratorWithError(errors.New("API rate limit exceeded"))
			},
			WantErr: false, // The handler continues even if some embeddings fail
			ValidateResult: func(t *testing.T, result []mcp.Content) {
				testutil.AssertContentCount(t, result, 1)
				// When embedding fails, the handler continues but will show as inaccurate
				testutil.AssertTextContains(t, result, "✗")
				testutil.AssertTextContains(t, result, "Inaccurate")
			},
		},
		{
			Name: "vector search error",
			Args: map[string]any{
				"claim": "MCP test claim",
			},
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				return testutil.NewMockVectorDBWithError(errors.New("database connection failed")),
					testutil.NewMockEmbeddingGeneratorWithEmbedding([]float64{0.1, 0.2, 0.3})
			},
			WantErr: false, // The handler continues even if search fails
			ValidateResult: func(t *testing.T, result []mcp.Content) {
				testutil.AssertContentCount(t, result, 1)
				// When search fails, handler continues but shows as inaccurate
				testutil.AssertTextContains(t, result, "✗")
				testutil.AssertTextContains(t, result, "Inaccurate")
			},
		},
		{
			Name: "claim with special characters",
			Args: map[string]any{
				"claim": "MCP supports unicode: 你好世界 émojis 🌍",
			},
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				results := []core.SearchResult{
					{
						Chunk: core.EmbeddedChunk{
							Content: "MCP fully supports Unicode characters in all text fields",
						},
						Similarity: 0.85,
					},
				}
				return testutil.NewMockVectorDBWithResults(results),
					testutil.NewMockEmbeddingGeneratorWithEmbedding([]float64{0.1, 0.2})
			},
			WantErr: false,
			ValidateResult: func(t *testing.T, result []mcp.Content) {
				testutil.AssertTextContains(t, result, "你好世界")
				testutil.AssertTextContains(t, result, "🌍")
			},
		},
		{
			Name: "no relevant results found",
			Args: map[string]any{
				"claim": "MCP supports quantum computing",
			},
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				// Return empty results
				return testutil.NewMockVectorDBWithResults([]core.SearchResult{}),
					testutil.NewMockEmbeddingGeneratorWithEmbedding([]float64{0.1, 0.2, 0.3})
			},
			WantErr: false,
			ValidateResult: func(t *testing.T, result []mcp.Content) {
				testutil.AssertContentCount(t, result, 1)
				// With no results, shows as inaccurate
				testutil.AssertTextContains(t, result, "✗")
				testutil.AssertTextContains(t, result, "Inaccurate")
			},
		},
		{
			Name: "default spec version",
			Args: map[string]any{
				"claim": "MCP uses JSON-RPC",
			},
			SetupMocks: func() (testutil.VectorDB, testutil.EmbeddingGenerator) {
				db := &testutil.MockVectorDB{
					SearchFunc: func(version string, queryEmbedding []float64, topK int) ([]core.SearchResult, error) {
						if version != specs.DefaultSpecVersion {
							t.Errorf("Expected version=%s, got %s", specs.DefaultSpecVersion, version)
						}
						return testutil.CreateTestSearchResults("MCP uses JSON-RPC 2.0"), nil
					},
				}
				return db, testutil.NewMockEmbeddingGeneratorWithEmbedding([]float64{0.1})
			},
			WantErr: false,
			ValidateResult: func(t *testing.T, result []mcp.Content) {
				testutil.AssertTextContains(t, result, "✓")
			},
		},
	}

	testutil.RunHandlerTestCases(t, handler, testCases)
}
