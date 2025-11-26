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

// TestNegativeClaimValidation tests that negative claims about rate limits are processed
// without errors and produce meaningful responses.
// Note: LLM verdicts (IsValid) are non-deterministic and cannot be reliably asserted.
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
		expectRateLimit bool
	}{
		{
			name:            "Quick claim - MCP does not enforce rate limits",
			claim:           "MCP does not enforce rate limits",
			useQuick:        true,
			expectRateLimit: true,
		},
		{
			name:            "Full validation - MCP never forwards raw model traffic or enforces rate limits",
			claim:           "MCP never forwards raw model traffic or enforces rate limits.",
			useQuick:        false,
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

			// Verify result structure is populated
			// Note: We don't assert IsValid as LLM responses are non-deterministic

			// Check if rate limiting is mentioned when expected
			allText := strings.Join(append(result.ParsedClaims, result.Issues...), " ")
			if tc.expectRateLimit {
				assert.Regexp(t, `(?i)rate.?limit`, allText,
					"Result should mention rate limiting")
			}
		})
	}
}
