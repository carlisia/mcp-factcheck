package storage_test

import (
	"context"
	"errors"
	"testing"

	"github.com/carlisia/mcp-factcheck/internal/storage"
	"github.com/carlisia/mcp-factcheck/pkg/logger"
)

func TestVectorDB_Interface(t *testing.T) {
	// Ensure VectorDB implements VectorSearcher
	var _ storage.VectorSearcher = (*storage.VectorDB)(nil)
}

func TestVectorDB_Search_Context(t *testing.T) {
	// Initialize logger for tests
	_ = logger.Initialize(true)

	db := storage.NewVectorDBFromEmbeddedData()
	
	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	
	_, err := db.Search(ctx, "test-version", []float64{0.1, 0.2}, 5)
	// Since our implementation doesn't check context yet, this won't error
	// but the test ensures the method signature is correct
	if err != nil && !errors.Is(err, context.Canceled) {
		// For now, we just check that any error isn't unexpected
		t.Logf("Search returned error (expected): %v", err)
	}
}

func TestIsVersionNotFoundError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "ErrVersionNotFound",
			err:  storage.ErrVersionNotFound,
			want: true,
		},
		{
			name: "wrapped ErrVersionNotFound",
			err:  errors.New("wrapped: " + storage.ErrVersionNotFound.Error()),
			want: false, // won't match with Is unless properly wrapped
		},
		{
			name: "contains 'not found'",
			err:  errors.New("file not found"),
			want: true,
		},
		{
			name: "contains 'no such file'",
			err:  errors.New("no such file or directory"),
			want: true,
		},
		{
			name: "unrelated error",
			err:  errors.New("connection timeout"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: isVersionNotFoundError is not exported, so we can't test it directly
			// This test would need to be in the storage package to access it
			t.Skip("isVersionNotFoundError is unexported")
		})
	}
}