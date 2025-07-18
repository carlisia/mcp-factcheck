package e2e

import (
	"context"
	"strings"
	"testing"

	"github.com/carlisia/mcp-factcheck/internal/capabilities"
	mcptools "github.com/carlisia/mcp-factcheck/pkg/mcp/tools"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// checkClaimArgs provides typed arguments for check claim tests
type checkClaimArgs struct {
	Content     string
	SpecVersion string
	UseChunking bool
}

// toMap converts typed args to map for handler
func (c checkClaimArgs) toMap() map[string]any {
	m := map[string]any{"content": c.Content}
	if c.SpecVersion != "" {
		m["specVersion"] = c.SpecVersion
	}
	m["useChunking"] = c.UseChunking
	return m
}

func TestValidator_HandleCheckMCPClaim_WithValidInput(t *testing.T) {
	ctx := context.Background()
	vectorDB, generator := setupTestEnv(t)

	tests := []struct {
		name   string
		args   checkClaimArgs
		assert func(t *testing.T, got []mcp.Content)
	}{
		{
			name: "validate content claim with all parameters",
			args: checkClaimArgs{
				Content:     "MCP provides tools and resources for building AI applications",
				SpecVersion: capabilities.Latest,
				UseChunking: false,
			},
			assert: assertNonEmpty,
		},
		{
			// Chunking is used for long content; this test ensures the handler processes it correctly.
			// When content is long, chunking can help the LLM process it in manageable pieces.
			name: "validate with chunking enabled",
			args: checkClaimArgs{
				Content: "This is a very long content that would benefit from chunking to properly validate against the MCP specification. " +
					"When dealing with extensive documentation or large code samples, the chunking feature helps break down the content " +
					"into smaller, more manageable pieces that can be validated individually against the MCP specification.",
				UseChunking: true,
			},
			assert: assertNonEmpty,
		},
		{
			name: "valid claim with no spec match",
			args: checkClaimArgs{
				Content: "MCP is a galactic mind-meld framework for superintelligence",
			},
			assert: assertNonEmpty, // Handler returns validation results even for unmatched content
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mcptools.HandleClaimsValidation(ctx, vectorDB, generator, tt.args.toMap())
			// Note: These tests will fail with real API calls using test key
			assertErr(t, err, false)
			if tt.assert != nil {
				tt.assert(t, got)
			}
		})
	}
}

// TestValidator_HandleCheckMCPClaim_DirectTestifyExample shows direct testify usage
func TestValidator_HandleCheckMCPClaim_DirectTestifyExample(t *testing.T) {
	ctx := context.Background()
	vectorDB, generator := setupTestEnv(t)

	t.Run("successful validation with testify", func(t *testing.T) {
		args := map[string]any{
			"content":     "MCP uses JSON-RPC 2.0 for communication",
			"specVersion": capabilities.Latest,
		}

		result, err := mcptools.HandleClaimsValidation(ctx, vectorDB, generator, args)

		// Using testify directly
		require.NoError(t, err, "validation should not fail")
		require.NotNil(t, result, "result should not be nil")
		assert.NotEmpty(t, result, "should return at least one content item")

		// Check content type - MCP content items are values, not pointers
		for i, content := range result {
			tc, ok := content.(mcp.TextContent)
			if assert.True(t, ok, "content[%d] should be TextContent", i) {
				// Log content for debugging
				t.Logf("Content[%d]: %s", i, tc.Text)
			}
		}

		// Verify content contains expected validation result
		var allText string
		for _, content := range result {
			if tc, ok := content.(mcp.TextContent); ok {
				allText += tc.Text + "\n"
			}
		}
		assert.Contains(t, allText, "MCP", "validation result should mention MCP")
	})

	t.Run("error handling with testify", func(t *testing.T) {
		args := map[string]any{
			"content": "", // empty content should fail
		}

		result, err := mcptools.HandleClaimsValidation(ctx, vectorDB, generator, args)

		// Using testify for error assertions
		require.Error(t, err, "empty content should return error")
		assert.Contains(t, err.Error(), "content", "error should mention content")
		assert.Nil(t, result, "result should be nil on error")
	})
}

func TestValidator_HandleCheckMCPClaim_CompoundClaimRegression(t *testing.T) {
	ctx := context.Background()
	vectorDB, generator := setupTestEnv(t)

	tests := []struct {
		name                   string
		args                   checkClaimArgs
		expectRateLimitMention bool
		expectInaccurate       bool
		description            string
	}{
		{
			name: "compound claim with rate limits enforcement",
			args: checkClaimArgs{
				Content:     "MCP never forwards raw model traffic; enforces ACLs, rate limits, and provenance",
				SpecVersion: "2025-06-18",
			},
			expectRateLimitMention: true,
			expectInaccurate:       true, // Should be inaccurate because MCP doesn't enforce these
			description:            "Should find and mention that rate limiting is a SHOULD recommendation, not enforcement",
		},
		{
			name: "accurate negative claim about rate limits",
			args: checkClaimArgs{
				Content:     "MCP does not enforce rate limits",
				SpecVersion: "2025-06-18",
			},
			expectRateLimitMention: true,
			expectInaccurate:       false, // This is accurate - MCP doesn't enforce, it's a SHOULD
			description:            "Should correctly identify that MCP not enforcing rate limits is accurate",
		},
		{
			name: "compound claim with comma-separated list",
			args: checkClaimArgs{
				Content:     "MCP enforces authentication, authorization, rate limiting, and access control",
				SpecVersion: "2025-06-18",
			},
			expectRateLimitMention: true,
			expectInaccurate:       true, // MCP doesn't enforce these
			description:            "Should properly parse comma-separated list and find rate limiting spec",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := mcptools.HandleClaimsValidation(ctx, vectorDB, generator, tc.args.toMap())
			require.NoError(t, err)
			require.NotEmpty(t, result)

			// Extract all text content
			var allText string
			for _, content := range result {
				if textContent, ok := content.(mcp.TextContent); ok {
					allText += textContent.Text + "\n"
				}
			}

			// Log for debugging
			t.Logf("Full result text:\n%s", allText)

			// Check if result mentions rate limiting appropriately
			if tc.expectRateLimitMention {
				assert.Contains(t, strings.ToLower(allText), "rate limit",
					"Result should mention rate limiting for: %s", tc.description)

				// Should explain it's a SHOULD recommendation
				foundShouldExplanation := strings.Contains(strings.ToLower(allText), "should implement") ||
					strings.Contains(strings.ToLower(allText), "should recommendation") ||
					strings.Contains(strings.ToLower(allText), "parties should")

				assert.True(t, foundShouldExplanation,
					"Result should explain rate limiting is a SHOULD recommendation, not enforcement")
			}

			// Check accuracy assessment - be careful about substring matching
			if tc.expectInaccurate {
				// Check for "INACCURATE" but not as a substring of another word
				assert.Regexp(t, `\bINACCURATE\b|❌.*INACCURATE`, allText,
					"Should be marked as inaccurate: %s", tc.description)
			} else {
				// Check for "ACCURATE" but ensure it's not part of "INACCURATE"
				assert.Regexp(t, `\bACCURATE\b|✅.*ACCURATE`, allText,
					"Should be marked as accurate: %s", tc.description)
				// Also ensure it doesn't contain standalone INACCURATE
				assert.NotRegexp(t, `\bINACCURATE\b|❌.*INACCURATE`, allText,
					"Should not be marked as inaccurate: %s", tc.description)
			}
		})
	}
}

func TestValidator_HandleCheckMCPClaim_WithInvalidInput(t *testing.T) {
	ctx := context.Background()
	vectorDB, generator := setupTestEnv(t)

	tests := []struct {
		name    string
		args    any
		wantErr bool
	}{
		{
			name: "missing content parameter",
			args: map[string]any{
				"useChunking": false,
			},
			wantErr: true,
		},
		{
			name:    "invalid arguments type",
			args:    []string{"not", "a", "map"},
			wantErr: true,
		},
		{
			name: "empty content",
			args: map[string]any{
				"content": "",
			},
			wantErr: true,
		},
		{
			name: "content as non-string type",
			args: map[string]any{
				"content":     12345,
				"useChunking": false,
			},
			wantErr: true,
		},
		{
			name: "invalid useChunking type",
			args: map[string]any{
				"content":     "Valid content",
				"useChunking": "yes please", // should be bool
			},
			wantErr: false, // Handler coerces to default (false) when type is invalid
		},
		{
			name: "invalid spec version",
			args: map[string]any{
				"content":     "MCP content to validate",
				"specVersion": "future-2099",
			},
			wantErr: true,
		},
		{
			name: "nil content value",
			args: map[string]any{
				"content":     nil,
				"useChunking": false,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := mcptools.HandleClaimsValidation(ctx, vectorDB, generator, tt.args)
			assertErr(t, err, tt.wantErr)
		})
	}
}
