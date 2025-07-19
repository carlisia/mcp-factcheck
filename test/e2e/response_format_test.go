package e2e_test

import (
	"context"
	"strings"
	"testing"

	"github.com/carlisia/mcp-factcheck/internal/capabilities"
	mcptools "github.com/carlisia/mcp-factcheck/pkg/mcp/tools"
	"github.com/carlisia/mcp-factcheck/test/e2e"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestQuickClaimResponseFormat verifies that quick claim responses follow the required format
func TestQuickClaimResponseFormat(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	vectorDB, generator := e2e.SetupTestEnv(t)

	tests := []struct {
		name           string
		claim          string
		expectAccurate bool
		formatChecks   []formatCheck
	}{
		{
			name:           "accurate claim format",
			claim:          "MCP uses JSON-RPC 2.0",
			expectAccurate: true,
			formatChecks: []formatCheck{
				{name: "starts with checkmark and ACCURATE", check: startsWithFormat("✓ ACCURATE")},
				{name: "contains quotes or absence statement", check: containsQuotesOrAbsence},
				{name: "contains explanation", check: containsExplanation},
			},
		},
		{
			name:           "inaccurate claim format",
			claim:          "MCP enforces rate limits",
			expectAccurate: false,
			formatChecks: []formatCheck{
				{name: "starts with X and INACCURATE", check: startsWithFormat("✗ INACCURATE")},
				{name: "contains quotes or absence statement", check: containsQuotesOrAbsence},
				{name: "contains explanation", check: containsExplanation},
			},
		},
		{
			name:           "negative claim format",
			claim:          "MCP does not enforce rate limits",
			expectAccurate: true,
			formatChecks: []formatCheck{
				{name: "starts with checkmark and ACCURATE", check: startsWithFormat("✓ ACCURATE")},
				{name: "contains rate limit quotes", check: containsQuotes("SHOULD implement rate limiting")},
				{name: "explains SHOULD vs enforcement", check: containsShouldExplanation},
			},
		},
		{
			name:           "compound negative claim format",
			claim:          "MCP never forwards raw model traffic or enforces rate limits",
			expectAccurate: true,
			formatChecks: []formatCheck{
				{name: "starts with checkmark and ACCURATE", check: startsWithFormat("✓ ACCURATE")},
				{name: "addresses both concepts", check: addressesMultipleConcepts("model traffic", "rate limit")},
				{name: "contains quotes or absence for each", check: containsQuotesOrAbsence},
				{name: "explains why claim is accurate", check: containsExplanation},
			},
		},
		{
			name:           "claim about non-existent feature",
			claim:          "MCP provides quantum entanglement",
			expectAccurate: false,
			formatChecks: []formatCheck{
				{name: "starts with X and INACCURATE", check: startsWithFormat("✗ INACCURATE")},
				{name: "mentions spec doesn't contain concept", check: mentionsAbsenceFromSpec("quantum")},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			args := map[string]any{
				"claim":       tc.claim,
				"specVersion": capabilities.Latest,
			}

			result, err := mcptools.HandleQuickClaimValidation(ctx, vectorDB, generator, args)
			require.NoError(t, err)
			require.NotEmpty(t, result)

			// Extract text content
			var allText string
			for _, content := range result {
				if textContent, ok := content.(mcp.TextContent); ok {
					allText += textContent.Text
				}
			}

			t.Logf("Response:\n%s", allText)

			// Verify verdict
			if tc.expectAccurate {
				assert.Regexp(t, `✓\s*ACCURATE`, allText, "Should have ✓ ACCURATE verdict")
			} else {
				assert.Regexp(t, `✗\s*INACCURATE`, allText, "Should have ✗ INACCURATE verdict")
			}

			// Run format checks
			for _, check := range tc.formatChecks {
				t.Run(check.name, func(t *testing.T) {
					t.Parallel()
					assert.True(t, check.check(allText),
						"Format check failed: %s\nResponse was:\n%s", check.name, allText)
				})
			}
		})
	}
}

// TestClaimValidationResponseFormat verifies that full claim validation follows the format
func TestClaimValidationResponseFormat(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	vectorDB, generator := e2e.SetupTestEnv(t)

	tests := []struct {
		name         string
		content      string
		formatChecks []formatCheck
	}{
		{
			name:    "single claim with explanation",
			content: "MCP uses JSON-RPC 2.0 for communication",
			formatChecks: []formatCheck{
				{name: "has verdict header", check: containsVerdictHeader},
				{name: "shows individual claims", check: containsClaimsSection},
				{name: "includes explanations", check: containsExplanationForClaims},
			},
		},
		{
			name:    "compound claim with explanations",
			content: "MCP enforces authentication, authorization, and rate limiting",
			formatChecks: []formatCheck{
				{name: "has verdict header", check: containsVerdictHeader},
				{name: "lists all subclaims", check: containsMultipleClaims(3)},
				{name: "includes explanation for each", check: containsExplanationForClaims},
				{name: "shows corrections", check: containsCorrections},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			args := map[string]any{
				"content":     tc.content,
				"specVersion": capabilities.Latest,
			}

			result, err := mcptools.HandleClaimsValidation(ctx, vectorDB, generator, args)
			require.NoError(t, err)
			require.NotEmpty(t, result)

			// Extract text content
			var allText string
			for _, content := range result {
				if textContent, ok := content.(mcp.TextContent); ok {
					allText += textContent.Text + "\n"
				}
			}

			t.Logf("Response:\n%s", allText)

			// Run format checks
			for _, check := range tc.formatChecks {
				t.Run(check.name, func(t *testing.T) {
					t.Parallel()
					assert.True(t, check.check(allText),
						"Format check failed: %s", check.name)
				})
			}
		})
	}
}

// formatCheck represents a specific format validation
type formatCheck struct {
	name  string
	check func(string) bool
}

// Format check functions

func startsWithFormat(prefix string) func(string) bool {
	return func(text string) bool {
		trimmed := strings.TrimSpace(text)
		lines := strings.Split(trimmed, "\n")
		if len(lines) == 0 {
			return false
		}
		// Check first non-empty line after claim
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasSuffix(line, ":") {
				return strings.HasPrefix(line, prefix)
			}
		}
		return false
	}
}

func containsQuotes(expectedQuote string) func(string) bool {
	return func(text string) bool {
		// Look for quoted text containing the expected content
		return strings.Contains(text, `"`) && strings.Contains(strings.ToLower(text), strings.ToLower(expectedQuote))
	}
}

func containsQuotesOrAbsence(text string) bool {
	// Should either have quotes or mention that spec doesn't mention something
	hasQuotes := strings.Contains(text, `"`)
	mentionsAbsence := strings.Contains(strings.ToLower(text), "doesn't mention") ||
		strings.Contains(strings.ToLower(text), "does not mention") ||
		strings.Contains(strings.ToLower(text), "not mentioned") ||
		strings.Contains(strings.ToLower(text), "no mention")
	return hasQuotes || mentionsAbsence
}

func containsShouldExplanation(text string) bool {
	lower := strings.ToLower(text)
	return (strings.Contains(lower, "should") || strings.Contains(lower, "recommendation")) &&
		(strings.Contains(lower, "not enforcement") ||
			strings.Contains(lower, "not enforce") ||
			strings.Contains(lower, "recommendations for implementations") ||
			strings.Contains(lower, "not functions of mcp") ||
			strings.Contains(lower, "not mandatory functions") ||
			strings.Contains(lower, "not an enforcement action by mcp") ||
			strings.Contains(lower, "recommendation for the parties") ||
			strings.Contains(lower, "recommendation for implementations") ||
			strings.Contains(lower, "recommendation rather than an enforcement"))
}

func containsExplanation(text string) bool {
	// Should have substantial explanation beyond just the verdict
	lines := strings.Split(text, "\n")
	explanationLines := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Count non-empty lines that aren't the verdict or confidence
		if line != "" &&
			!strings.HasPrefix(line, "✓") &&
			!strings.HasPrefix(line, "✗") &&
			!strings.HasPrefix(line, "**Confidence") {
			explanationLines++
		}
	}
	return explanationLines >= 2
}

func addressesMultipleConcepts(concepts ...string) func(string) bool {
	return func(text string) bool {
		lower := strings.ToLower(text)
		for _, concept := range concepts {
			if !strings.Contains(lower, strings.ToLower(concept)) {
				return false
			}
		}
		return true
	}
}

func mentionsAbsenceFromSpec(concept string) func(string) bool {
	return func(text string) bool {
		lower := strings.ToLower(text)
		return strings.Contains(lower, strings.ToLower(concept)) &&
			(strings.Contains(lower, "doesn't mention") ||
				strings.Contains(lower, "does not mention") ||
				strings.Contains(lower, "not mentioned") ||
				strings.Contains(lower, "no mention") ||
				strings.Contains(lower, "not in the spec"))
	}
}

func containsVerdictHeader(text string) bool {
	return strings.Contains(text, "✅ Content is ACCURATE") ||
		strings.Contains(text, "❌ Content is INACCURATE")
}

func containsClaimsSection(text string) bool {
	return strings.Contains(text, "Claims:")
}

func containsMultipleClaims(minCount int) func(string) bool {
	return func(text string) bool {
		// Count occurrences of "✓ Accurate" or "✗ Inaccurate"
		count := strings.Count(text, "✓ Accurate") + strings.Count(text, "✗ Inaccurate")
		return count >= minCount
	}
}

func containsExplanationForClaims(text string) bool {
	// Look for explanations after verdict markers
	lines := strings.Split(text, "\n")
	foundVerdict := false
	foundExplanation := false

	for i, line := range lines {
		if strings.Contains(line, "✓ Accurate") || strings.Contains(line, "✗ Inaccurate") {
			foundVerdict = true
			// Check next few lines for explanation
			for j := i + 1; j < len(lines) && j < i+5; j++ {
				if len(strings.TrimSpace(lines[j])) > 20 {
					foundExplanation = true
					break
				}
			}
		}
	}

	return foundVerdict && foundExplanation
}

func containsCorrections(text string) bool {
	return strings.Contains(text, "Correction:")
}
