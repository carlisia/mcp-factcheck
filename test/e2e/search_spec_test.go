package e2e

import (
	"context"
	"strings"
	"testing"

	mcptools "github.com/carlisia/mcp-factcheck/pkg/mcp/tools"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
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

func TestSpec_SearchSpec_WithValidQuery(t *testing.T) {
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
				SpecVersion: "2025-06-18",
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
			got, err := mcptools.HandleSearchSpec(ctx, vectorDB, generator, tt.args.toMap())
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
		name            string
		args            any
		expectErr       bool
		errMsgContains string // optional: check error message contains this
	}{
		{
			name:      "invalid arguments type",
			args:      "not a map",
			expectErr: true,
		},
		{
			name: "missing query parameter",
			args: map[string]any{
				"specVersion": "draft",
			},
			expectErr: true,
		},
		{
			name: "empty query parameter",
			args: map[string]any{
				"query": "",
			},
			expectErr: true,
		},
		{
			name: "query with only whitespace",
			args: map[string]any{
				"query": "   \t\n   ",
			},
			expectErr: true,
		},
		{
			name: "query exceeds max length",
			args: map[string]any{
				"query": strings.Repeat("a", 501), // MaxQueryLength is 500
			},
			expectErr: true,
		},
		{
			name: "invalid spec version",
			args: map[string]any{
				"query":       "test search",
				"specVersion": "invalid-version",
			},
			expectErr: true,
		},
		{
			name: "invalid topK type",
			args: map[string]any{
				"query": "test search",
				"topK":  "not a number",
			},
			expectErr: false, // Handler uses default when topK type is invalid
		},
		{
			name: "topK below minimum",
			args: map[string]any{
				"query": "test search",
				"topK":  float64(0),
			},
			expectErr:       true,
			errMsgContains: "topK must be at least 1",
		},
		{
			name: "topK above maximum",
			args: map[string]any{
				"query": "test search",
				"topK":  float64(100),
			},
			expectErr:       true,
			errMsgContains: "topK cannot exceed 20",
		},
		{
			name: "negative topK",
			args: map[string]any{
				"query": "test search",
				"topK":  float64(-5),
			},
			expectErr:       true,
			errMsgContains: "topK must be at least 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := mcptools.HandleSearchSpec(ctx, vectorDB, generator, tt.args)
			assertErr(t, err, tt.expectErr)
			if tt.expectErr && tt.errMsgContains != "" && err != nil {
				assert.Contains(t, err.Error(), tt.errMsgContains)
			}
		})
	}
}
