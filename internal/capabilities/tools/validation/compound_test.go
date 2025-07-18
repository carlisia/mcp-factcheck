package validation_test

import (
	"context"
	"testing"

	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools"
	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools/contentprep"
	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompoundEvidence(t *testing.T) {
	// Test that compound claims are properly analyzed
	compoundContent := "MCP provides authentication and authorization mechanisms"

	// Mock search results that contain evidence for both subclaims
	mockSearchResults := []tools.SearchResult{
		{
			Content:    "MCP implementations should provide authentication mechanisms to verify client identity",
			Similarity: 0.9,
		},
		{
			Content:    "The protocol supports authorization through capability negotiation and access control",
			Similarity: 0.85,
		},
	}

	// Mock LLM function that returns template response with compound evidence
	mockLLM := func(ctx context.Context, model string, prompt string, temperature float64, maxTokens int) (string, error) {
		// Verify that compound evidence is in the prompt
		assert.Contains(t, prompt, "Compound Claim:", "Expected compound evidence in prompt but not found")
		assert.Contains(t, prompt, "authentication", "Expected authentication subclaim in compound evidence")
		assert.Contains(t, prompt, "authorization", "Expected authorization subclaim in compound evidence")

		// Return a successful validation response
		return `{
			"claims": [{
				"claim": "MCP provides authentication and authorization mechanisms",
				"is_accurate": true,
				"explanation": "Both authentication and authorization are supported"
			}],
			"missing_best_practices": [],
			"advisory_language_issues": [],
			"overall_is_accurate": true,
			"summary": "Compound claim validated successfully"
		}`, nil
	}

	// Create a simple request
	req := validation.ClaimsRequest{
		Content:     compoundContent,
		SpecVersion: "draft",
	}

	// Mock embedding function
	mockEmbed := func(ctx context.Context, content string) ([]float64, error) {
		return []float64{0.1, 0.2, 0.3}, nil
	}

	// Mock search function that returns our prepared results
	mockSearch := func(version string, queryEmbedding []float64, topK int) ([]tools.SearchResult, error) {
		return mockSearchResults, nil
	}

	// Perform validation
	result, err := validation.Claims(context.Background(), req, mockEmbed, mockSearch, mockLLM)
	require.NoError(t, err, "Claims validation should not fail")

	assert.True(t, result.IsValid, "Expected compound claim to be validated as accurate")
}

func TestCompoundClaimDecomposition(t *testing.T) {
	tests := []struct {
		name     string
		claim    string
		expected int // expected number of subclaims
		wantSubs []string
	}{
		{
			name:     "simple and compound",
			claim:    "MCP supports tools and resources",
			expected: 2,
			wantSubs: []string{"MCP supports tools", "MCP supports resources"},
		},
		{
			name:     "not compound",
			claim:    "MCP supports JSON-RPC",
			expected: 1,
			wantSubs: []string{"MCP supports JSON-RPC"},
		},
		{
			name:     "multiple and",
			claim:    "MCP provides authentication and authorization and logging",
			expected: 3,
			wantSubs: []string{"MCP provides authentication", "MCP provides authorization", "MCP provides logging"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compound := contentprep.Decompose(tt.claim)

			assert.Len(t, compound.SubClaims, tt.expected, "Number of subclaims mismatch")

			for i, want := range tt.wantSubs {
				if i >= len(compound.SubClaims) {
					break
				}
				assert.Equal(t, want, compound.SubClaims[i].Text, "Subclaim %d text mismatch", i)
			}
		})
	}
}
