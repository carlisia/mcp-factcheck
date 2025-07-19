package e2e

import (
	"context"
	"strings"
	"testing"

	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools"
	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools/validation"
	"github.com/carlisia/mcp-factcheck/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNegativeClaimValidation(t *testing.T) {
	t.Parallel()
	// Setup test environment with real embeddings
	vectorDB, llmClient := setupTestEnv(t)

	// Create tool functions
	embedFunc := func(ctx context.Context, content string) ([]float64, error) {
		return llmClient.CreateEmbedding(ctx, content)
	}

	searchFunc := func(version string, queryEmbedding []float64, topK int) ([]tools.SearchResult, error) {
		results, err := vectorDB.Search(context.Background(), version, queryEmbedding, topK)
		if err != nil {
			return nil, err
		}

		// Convert to tools.SearchResult
		var converted []tools.SearchResult
		for _, r := range results {
			converted = append(converted, tools.SearchResult{
				Content:    r.Content,
				Similarity: r.Similarity,
			})
		}
		return converted, nil
	}

	llmFunc := func(ctx context.Context, prompt string, temperature float64, maxTokens int) (string, error) {
		opts := llm.CompletionOptions{
			Temperature: float32(temperature),
			MaxTokens:   maxTokens,
		}
		return llmClient.Complete(ctx, prompt, opts)
	}

	testCases := []struct {
		name            string
		claim           string
		useQuick        bool
		expectValid     bool
		expectRateLimit bool
	}{
		{
			name:            "Quick claim - MCP does not enforce rate limits",
			claim:           "MCP does not enforce rate limits",
			useQuick:        true,
			expectValid:     true,
			expectRateLimit: true,
		},
		{
			name:            "Full validation - MCP never forwards raw model traffic or enforces rate limits",
			claim:           "MCP never forwards raw model traffic or enforces rate limits.",
			useQuick:        false,
			expectValid:     true,
			expectRateLimit: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			var result *validation.Result
			var err error

			if tc.useQuick {
				req := validation.QuickClaimRequest{
					Claim:       tc.claim,
					SpecVersion: "draft",
				}
				result, err = validation.QuickClaim(ctx, req, embedFunc, searchFunc, llmFunc)
			} else {
				req := validation.ClaimsRequest{
					Content:     tc.claim,
					SpecVersion: "draft",
					UseChunking: true,
				}
				result, err = validation.Claims(ctx, req, embedFunc, searchFunc, llmFunc)
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			// Log the result for debugging
			t.Logf("IsValid: %v", result.IsValid)
			t.Logf("ParsedClaims: %v", result.ParsedClaims)
			t.Logf("Issues: %v", result.Issues)

			// Check validity
			assert.Equal(t, tc.expectValid, result.IsValid,
				"Expected IsValid=%v for claim: %s", tc.expectValid, tc.claim)

			// Check if rate limiting is mentioned
			allText := strings.Join(append(result.ParsedClaims, result.Issues...), " ")
			if tc.expectRateLimit {
				assert.Contains(t, strings.ToLower(allText), "rate limit",
					"Result should mention rate limiting")

				// Should explain it's a SHOULD recommendation
				foundShouldExplanation := strings.Contains(strings.ToLower(allText), "should implement") ||
					strings.Contains(strings.ToLower(allText), "recommendation") ||
					strings.Contains(strings.ToLower(allText), "parties should")

				assert.True(t, foundShouldExplanation,
					"Result should explain rate limiting is a SHOULD recommendation, not enforcement")
			}
		})
	}
}
