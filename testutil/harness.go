package testutil

import (
	"context"
	"testing"

	"github.com/carlisia/mcp-factcheck/embedding"
	"github.com/mark3labs/mcp-go/mcp"
)

// HandlerTestCase defines a reusable unit test case for handler functions.
type HandlerTestCase struct {
	Name           string
	Args           any
	SetupMocks     func() (VectorDB, EmbeddingGenerator)
	WantErr        bool
	ValidateError  func(t *testing.T, err error)
	ValidateResult func(t *testing.T, result []mcp.Content)
}

// VectorDB defines the minimal interface required for handler tests.
type VectorDB interface {
	Search(version string, queryEmbedding []float64, topK int) ([]embedding.SearchResult, error)
	ListVersions() ([]string, error)
}

// EmbeddingGenerator defines the minimal interface required for embedding injection.
type EmbeddingGenerator interface {
	GenerateEmbedding(ctx context.Context, content string) ([]float64, error)
}

// HandlerFunc defines the signature for MCP handlers that can be tested with this harness.
type HandlerFunc func(ctx context.Context, db VectorDB, gen EmbeddingGenerator, args any) ([]mcp.Content, error)

// RunHandlerTestCases runs a list of HandlerTestCase against a given handler function.
func RunHandlerTestCases(t *testing.T, handler HandlerFunc, testCases []HandlerTestCase) {
	t.Helper()
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Helper()

			ctx := context.Background()

			// Setup mocks
			db, gen := tc.SetupMocks()

			// Special handling for context cancellation test
			if tc.Name == "context cancellation during embedding" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			// Execute handler
			result, err := handler(ctx, db, gen, tc.Args)

			// Check error expectation
			if (err != nil) != tc.WantErr {
				t.Errorf("error = %v, wantErr = %v", err, tc.WantErr)
				return
			}

			// Validate error if present
			if err != nil && tc.ValidateError != nil {
				tc.ValidateError(t, err)
			}

			// Validate result if no error
			if err == nil && tc.ValidateResult != nil {
				tc.ValidateResult(t, result)
			}
		})
	}
}

// MockVectorDB is a test implementation of VectorDB for unit testing
type MockVectorDB struct {
	SearchFunc       func(version string, queryEmbedding []float64, topK int) ([]embedding.SearchResult, error)
	ListVersionsFunc func() ([]string, error)
}

func (m *MockVectorDB) Search(version string, queryEmbedding []float64, topK int) ([]embedding.SearchResult, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(version, queryEmbedding, topK)
	}
	return []embedding.SearchResult{}, nil
}

func (m *MockVectorDB) ListVersions() ([]string, error) {
	if m.ListVersionsFunc != nil {
		return m.ListVersionsFunc()
	}
	return []string{"draft", "2025-06-18", "2025-03-26", "2024-11-05"}, nil
}

// MockEmbeddingGenerator is a test implementation of EmbeddingGenerator for unit testing
type MockEmbeddingGenerator struct {
	GenerateFunc func(ctx context.Context, content string) ([]float64, error)
}

func (m *MockEmbeddingGenerator) GenerateEmbedding(ctx context.Context, content string) ([]float64, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(ctx, content)
	}
	return []float64{0.1, 0.2, 0.3}, nil
}

// Compile-time interface conformance checks
var _ VectorDB = (*MockVectorDB)(nil)
var _ EmbeddingGenerator = (*MockEmbeddingGenerator)(nil)

// RunHandlerTestCasesWithSetup is a variant that allows custom setup per test case.
// This is useful when tests need different context configurations or other setup.
func RunHandlerTestCasesWithSetup(t *testing.T, handler HandlerFunc, testCases []HandlerTestCase, setupFunc func(t *testing.T, tc *HandlerTestCase) context.Context) {
	t.Helper()
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Helper()

			// Allow custom setup if provided
			var ctx context.Context
			if setupFunc != nil {
				ctx = setupFunc(t, &tc)
			} else {
				ctx = context.Background()
			}

			// Setup mocks
			db, gen := tc.SetupMocks()

			// Execute handler
			result, err := handler(ctx, db, gen, tc.Args)

			// Check error expectation
			if (err != nil) != tc.WantErr {
				t.Errorf("error = %v, wantErr = %v", err, tc.WantErr)
				return
			}

			// Validate error if present
			if err != nil && tc.ValidateError != nil {
				tc.ValidateError(t, err)
			}

			// Validate result if no error
			if err == nil && tc.ValidateResult != nil {
				tc.ValidateResult(t, result)
			}
		})
	}
}
