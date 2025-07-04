package e2e

import (
	"context"
	"testing"

	"github.com/carlisia/mcp-factcheck/internal/specs"
	"github.com/carlisia/mcp-factcheck/pkg/spec"
	"github.com/mark3labs/mcp-go/mcp"
)

// searchArgs provides typed arguments for search spec tests
type searchArgs struct {
	Query       string
	SpecVersion string
	TopK        float64
}

// toMap converts typed args to map for handler
func (s searchArgs) toMap() map[string]any {
	m := map[string]any{"query": s.Query}
	if s.SpecVersion != "" {
		m["specVersion"] = s.SpecVersion
	}
	if s.TopK > 0 {
		m["topK"] = s.TopK
	}
	return m
}

func TestSpec_HandleSearchSpec_WithValidQuery(t *testing.T) {
	ctx := context.Background()
	vectorDB, generator := setupTestEnv(t)

	tests := []struct {
		name   string
		args   searchArgs
		assert func(t *testing.T, got []mcp.Content)
	}{
		{
			name: "successful search with all parameters",
			args: searchArgs{
				Query:       "MCP communication protocol",
				SpecVersion: specs.DefaultSpecVersion,
				TopK:        5.0,
			},
			assert: assertNonEmpty,
		},
		{
			name: "search with default parameters",
			args: searchArgs{
				Query: "JSON-RPC",
			},
			assert: assertNonEmpty,
		},
		{
			name: "valid query with unrelated content",
			args: searchArgs{
				Query: "quantum holographic entanglement",
			},
			assert: assertNonEmpty, // Vector search returns closest matches even if unrelated
		},
		{
			name: "search with topK = 0 (should fallback to default)",
			args: searchArgs{
				Query: "MCP query interface",
				TopK:  0,
			},
			assert: assertNonEmpty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := spec.HandleSearchSpecMCP(ctx, vectorDB, generator, tt.args.toMap())
			// Note: These tests will fail with real API calls using test key
			// In a real e2e test, you'd use a valid API key or mock the external service
			assertErr(t, err, false)
			if tt.assert != nil {
				tt.assert(t, got)
			}
		})
	}
}

func TestSpec_HandleSearchSpec_WithInvalidInput(t *testing.T) {
	ctx := context.Background()
	vectorDB, generator := setupTestEnv(t)

	tests := []struct {
		name    string
		args    any
		wantErr bool
	}{
		{
			name:    "invalid arguments type",
			args:    "not a map",
			wantErr: true,
		},
		{
			name: "missing query parameter",
			args: map[string]any{
				"specVersion": "draft",
			},
			wantErr: true,
		},
		{
			name: "empty query parameter",
			args: map[string]any{
				"query": "",
			},
			wantErr: true,
		},
		{
			name: "invalid spec version",
			args: map[string]any{
				"query":       "test search",
				"specVersion": "invalid-version",
			},
			wantErr: true,
		},
		{
			name: "invalid topK type",
			args: map[string]any{
				"query": "test search",
				"topK":  "not a number",
			},
			wantErr: false, // Handler should use default when topK is invalid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := spec.HandleSearchSpecMCP(ctx, vectorDB, generator, tt.args)
			assertErr(t, err, tt.wantErr)
		})
	}
}
