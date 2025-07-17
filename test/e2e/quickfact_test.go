package e2e

import (
	"context"
	"testing"

	"github.com/carlisia/mcp-factcheck/internal/capabilities"
	mcptools "github.com/carlisia/mcp-factcheck/pkg/mcp/tools"
	"github.com/mark3labs/mcp-go/mcp"
)

// quickFactArgs provides typed arguments for quick fact tests
type quickFactArgs struct {
	Claim       string
	SpecVersion string
}

// toMap converts typed args to map for handler
func (q quickFactArgs) toMap() map[string]any {
	m := map[string]any{"claim": q.Claim}
	if q.SpecVersion != "" {
		m["specVersion"] = q.SpecVersion
	}
	return m
}

func TestValidator_HandleCheckMCPQuickFact_WithValidInput(t *testing.T) {
	ctx := context.Background()
	vectorDB, generator := setupTestEnv(t)

	tests := []struct {
		name  string
		args  quickFactArgs
		check func(t *testing.T, got []mcp.Content)
	}{
		{
			name: "check accurate claim with spec version",
			args: quickFactArgs{
				Claim:       "MCP uses JSON-RPC",
				SpecVersion: capabilities.Latest,
			},
			check: assertNonEmpty,
		},
		{
			name: "check with default spec version",
			args: quickFactArgs{
				Claim: "MCP supports multiple transport protocols",
			},
			check: assertNonEmpty,
		},
		{
			// Even if the claim is irrelevant or semantically distant,
			// the vector search may still return the closest result.
			// This test ensures the system doesn't return an error in that case.
			name: "valid input with no matching chunk",
			args: quickFactArgs{
				Claim: "MCP enables sentient AI overlords", // not in test data
			},
			check: assertNonEmpty, // Handler returns a result even for non-matching claims
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mcptools.HandleQuickClaimValidation(ctx, vectorDB, generator, tt.args.toMap())
			assertSuccess(t, err, got)
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestValidator_HandleCheckMCPQuickFact_WithInvalidInput(t *testing.T) {
	ctx := context.Background()
	vectorDB, generator := setupTestEnv(t)

	tests := []struct {
		name string
		args any
	}{
		{
			name: "empty claim",
			args: map[string]any{
				"claim": "",
			},
		},
		{
			name: "missing claim field",
			args: map[string]any{},
		},
		{
			name: "invalid spec version",
			args: map[string]any{
				"claim":       "MCP uses JSON-RPC",
				"specVersion": "not-a-real-version",
			},
		},
		{
			name: "invalid arguments type",
			args: 123,
		},
		{
			name: "claim as non-string type",
			args: map[string]any{
				"claim": 12345,
			},
		},
		{
			name: "nil claim value",
			args: map[string]any{
				"claim": nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := mcptools.HandleQuickClaimValidation(ctx, vectorDB, generator, tt.args)
			// All invalid input tests should expect an error
			assertErr(t, err, true)
		})
	}
}
