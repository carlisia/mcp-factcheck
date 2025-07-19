package validation_test

import (
	"context"
	"strings"
	"testing"

	"github.com/carlisia/mcp-factcheck/internal/capabilities"
	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools"
	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuickClaim_AccurateVsInaccurate(t *testing.T) {
	tests := []struct {
		name            string
		claim           string
		llmResponse     string
		wantIsValid     bool
		wantParsedClaim string
	}{
		{
			name:  "inaccurate claim - MCP is REST",
			claim: "MCP is the same as REST",
			llmResponse: `✗ INACCURATE  
The claim that MCP is the same as REST is incorrect because MCP is a distinct protocol with specific features such as the use of JSON-RPC for message encoding and an emphasis on OAuth 2.1 roles, which are not inherent to REST.`,
			wantIsValid:     false,
			wantParsedClaim: "✗ INACCURATE: MCP is the same as REST",
		},
		{
			name:  "accurate claim - MCP uses JSON-RPC",
			claim: "MCP uses JSON-RPC 2.0",
			llmResponse: `✓ ACCURATE
MCP uses JSON-RPC 2.0 for message encoding as stated in the specification.`,
			wantIsValid:     true,
			wantParsedClaim: "✓ ACCURATE: MCP uses JSON-RPC 2.0",
		},
		{
			name:  "inaccurate with ACCURATE in explanation",
			claim: "MCP requires REST endpoints",
			llmResponse: `✗ INACCURATE
While it's ACCURATE that MCP can use HTTP transport, it doesn't require REST endpoints. MCP uses JSON-RPC, not REST.`,
			wantIsValid:     false,
			wantParsedClaim: "✗ INACCURATE: MCP requires REST endpoints",
		},
		{
			name:            "edge case - just INACCURATE",
			claim:           "MCP is built on GraphQL",
			llmResponse:     `✗ INACCURATE`,
			wantIsValid:     false,
			wantParsedClaim: "✗ INACCURATE: MCP is built on GraphQL",
		},
		{
			name:            "edge case - just ACCURATE",
			claim:           "MCP is a protocol",
			llmResponse:     `✓ ACCURATE`,
			wantIsValid:     true,
			wantParsedClaim: "✓ ACCURATE: MCP is a protocol",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock dependencies
			mockEmbed := func(ctx context.Context, content string) ([]float64, error) {
				return []float64{0.1, 0.2, 0.3}, nil
			}

			mockSearch := func(version string, queryEmbedding []float64, topK int) ([]tools.SearchResult, error) {
				return []tools.SearchResult{
					{Content: "MCP uses JSON-RPC 2.0 for encoding messages", Similarity: 0.9},
					{Content: "MCP is not REST, it's a different protocol", Similarity: 0.85},
				}, nil
			}

			mockLLM := func(ctx context.Context, prompt string, temperature float64, maxTokens int) (string, error) {
				// Return the test's expected LLM response
				return tt.llmResponse, nil
			}

			// Create request
			req := validation.QuickClaimRequest{
				Claim:       tt.claim,
				SpecVersion: capabilities.Latest,
			}

			// Execute
			result, err := validation.QuickClaim(context.Background(), req, mockEmbed, mockSearch, mockLLM)

			// Assert
			require.NoError(t, err, "QuickClaim should not return error")
			require.NotNil(t, result, "Result should not be nil")

			assert.Equal(t, tt.wantIsValid, result.IsValid, "IsValid mismatch for claim: %s", tt.claim)

			if len(result.ParsedClaims) > 0 {
				assert.Equal(t, tt.wantParsedClaim, result.ParsedClaims[0], "ParsedClaim mismatch")
			}

			// Additional assertions for invalid claims
			if !tt.wantIsValid {
				assert.NotEmpty(t, result.Issues, "Invalid claim should have issues")
				// The issue contains the explanation part, not the full response
				assert.NotEmpty(t, result.Issues[0], "Issue should not be empty")
			}
		})
	}
}

func TestQuickClaim_ParsedClaimsFormat(t *testing.T) {
	tests := []struct {
		name           string
		claim          string
		isAccurate     bool
		expectedPrefix string
	}{
		{
			name:           "accurate claim format",
			claim:          "MCP supports tools",
			isAccurate:     true,
			expectedPrefix: "✓ ACCURATE:",
		},
		{
			name:           "inaccurate claim format",
			claim:          "MCP is REST-based",
			isAccurate:     false,
			expectedPrefix: "✗ INACCURATE:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLLM := func(ctx context.Context, prompt string, temperature float64, maxTokens int) (string, error) {
				if tt.isAccurate {
					return "✓ ACCURATE\nThe claim is correct.", nil
				}
				return "✗ INACCURATE\nThe claim is incorrect.", nil
			}

			req := validation.QuickClaimRequest{
				Claim:       tt.claim,
				SpecVersion: capabilities.Latest,
			}

			result, err := validation.QuickClaim(
				context.Background(),
				req,
				mockEmbedFunc([]float64{0.1, 0.2}),
				mockSearchFunc(2),
				mockLLM,
			)

			require.NoError(t, err)
			require.NotEmpty(t, result.ParsedClaims)

			assert.True(t, strings.HasPrefix(result.ParsedClaims[0], tt.expectedPrefix),
				"ParsedClaim should start with %s, got: %s", tt.expectedPrefix, result.ParsedClaims[0])
			assert.Contains(t, result.ParsedClaims[0], tt.claim,
				"ParsedClaim should contain the original claim")
		})
	}
}

func TestQuickClaim_BugFix_InaccurateContainingAccurate(t *testing.T) {
	// This is the specific bug case from telemetry - response contains "ACCURATE"
	// but is actually INACCURATE and was being parsed wrong
	llmResponse := `✗ INACCURATE  
The claim that MCP is the same as REST is incorrect because MCP is a distinct protocol with specific features such as the use of JSON-RPC for message encoding and an emphasis on OAuth 2.1 roles, which are not inherent to REST. While both MCP and REST may utilize HTTP, they serve different purposes and implement different architectural principles.`

	mockLLM := func(ctx context.Context, prompt string, temperature float64, maxTokens int) (string, error) {
		return llmResponse, nil
	}

	req := validation.QuickClaimRequest{
		Claim:       "MCP is the same as REST",
		SpecVersion: capabilities.Latest,
	}

	result, err := validation.QuickClaim(
		context.Background(),
		req,
		mockEmbedFunc([]float64{0.1, 0.2}),
		mockSearchFunc(5),
		mockLLM,
	)

	require.NoError(t, err)
	require.NotNil(t, result)

	// The key assertion - this should be false, not true (which was the bug)
	assert.False(t, result.IsValid, "Response starting with '✗ INACCURATE' should be parsed as invalid")
	assert.Equal(t, 0.9, result.Confidence)
	assert.Equal(t, "✗ INACCURATE: MCP is the same as REST", result.ParsedClaims[0])
	assert.NotEmpty(t, result.Issues, "Should have issues for inaccurate claim")
}

func TestQuickClaim_ConfidenceAndCorrections(t *testing.T) {
	tests := []struct {
		name              string
		llmResponse       string
		wantConfidence    float64
		wantHasCorrection bool
		checkCorrection   string
	}{
		{
			name: "high confidence accurate",
			llmResponse: `✓ ACCURATE
The specification clearly states this.`,
			wantConfidence:    0.9,
			wantHasCorrection: false,
		},
		{
			name: "high confidence inaccurate with correction",
			llmResponse: `✗ INACCURATE
The claim is wrong. MCP should be described as using JSON-RPC, not REST.`,
			wantConfidence:    0.9,
			wantHasCorrection: true,
			checkCorrection:   "should",
		},
		{
			name:              "uncertain response",
			llmResponse:       `The claim needs more context to verify properly.`,
			wantConfidence:    0.5,
			wantHasCorrection: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLLM := func(ctx context.Context, prompt string, temperature float64, maxTokens int) (string, error) {
				return tt.llmResponse, nil
			}

			req := validation.QuickClaimRequest{
				Claim:       "Test claim",
				SpecVersion: capabilities.Latest,
			}

			result, err := validation.QuickClaim(
				context.Background(),
				req,
				mockEmbedFunc([]float64{0.1, 0.2}),
				mockSearchFunc(2),
				mockLLM,
			)

			require.NoError(t, err)

			// Check confidence
			assert.Equal(t, tt.wantConfidence, result.Confidence, "Confidence mismatch")

			// Check corrections
			if tt.wantHasCorrection {
				assert.NotEmpty(t, result.Suggestions, "Expected corrections in suggestions")
				if tt.checkCorrection != "" {
					assert.Contains(t, result.Suggestions[0], tt.checkCorrection,
						"Correction should contain expected text")
				}
			} else {
				assert.Empty(t, result.Suggestions, "Expected no corrections")
			}
		})
	}
}

func TestQuickClaim_CompoundClaimDetection(t *testing.T) {
	tests := []struct {
		name                string
		claim               string
		shouldBeCompound    bool
		expectedIssuePrefix string
	}{
		{
			name:                "semicolon compound claim",
			claim:               "MCP Never forwards raw model traffic; enforces ACLs, rate limits, and provenance",
			shouldBeCompound:    true,
			expectedIssuePrefix: "Compound claim detected:",
		},
		{
			name:             "simple single claim",
			claim:            "MCP uses JSON-RPC 2.0",
			shouldBeCompound: false,
		},
		{
			name:                "multiple conjunctions",
			claim:               "MCP provides tools and resources and enables authentication and supports prompts",
			shouldBeCompound:    true,
			expectedIssuePrefix: "Compound claim detected:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock dependencies
			mockEmbed := func(ctx context.Context, content string) ([]float64, error) {
				return []float64{0.1, 0.2, 0.3}, nil
			}

			mockSearch := func(version string, queryEmbedding []float64, topK int) ([]tools.SearchResult, error) {
				return []tools.SearchResult{
					{Content: "MCP uses JSON-RPC 2.0", Similarity: 0.9},
				}, nil
			}

			mockLLM := func(ctx context.Context, prompt string, temperature float64, maxTokens int) (string, error) {
				return "✓ ACCURATE\nThe claim is correct.", nil
			}

			// Create request
			req := validation.QuickClaimRequest{
				Claim:       tt.claim,
				SpecVersion: capabilities.Latest,
			}

			// Execute
			result, err := validation.QuickClaim(context.Background(), req, mockEmbed, mockSearch, mockLLM)

			// Assert
			require.NoError(t, err, "QuickClaim should not return error")
			require.NotNil(t, result, "Result should not be nil")

			if tt.shouldBeCompound {
				// Check that compound claim was detected
				assert.False(t, result.IsValid, "Compound claims should return IsValid=false")
				assert.Equal(t, 1.0, result.Confidence, "Should be confident about compound detection")

				// Check issues contain compound claim message
				require.NotEmpty(t, result.Issues, "Should have issues for compound claim")
				assert.Contains(t, result.Issues[0], tt.expectedIssuePrefix)

				// Check suggestions
				require.NotEmpty(t, result.Suggestions, "Should have suggestions for compound claim")
				assert.Contains(t, result.Suggestions[0], "check_mcp_claim")
			} else {
				// For single claims, it should process normally
				// The mock returns ACCURATE, so IsValid should be true
				assert.True(t, result.IsValid, "Single claims should be processed normally")
			}
		})
	}
}

func TestQuickClaim_CompoundClaimIndicators(t *testing.T) {
	// Test that specific indicators are detected
	req := validation.QuickClaimRequest{
		Claim:       "MCP enforces authentication, authorization, rate limiting, and access control",
		SpecVersion: capabilities.Latest,
	}

	result, err := validation.QuickClaim(
		context.Background(),
		req,
		mockEmbedFunc([]float64{0.1, 0.2}),
		mockSearchFunc(1),
		func(ctx context.Context, prompt string, temperature float64, maxTokens int) (string, error) {
			return "✓ ACCURATE", nil
		},
	)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Should detect as compound due to comma-separated list
	assert.False(t, result.IsValid)
	assert.Contains(t, result.Issues[2], "comma-separated feature list")
}
