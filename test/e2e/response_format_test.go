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

// TestQuickClaimResponseFormat verifies that quick claim responses follow the required format.
// Note: LLM verdicts (ACCURATE/INACCURATE) are non-deterministic, so we only check format structure
// rather than specific verdicts.
func TestQuickClaimResponseFormat(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	vectorDB, generator := e2e.SetupTestEnv(t)

	tests := []struct {
		name         string
		claim        string
		formatChecks []formatCheck
	}{
		{
			name:  "accurate claim format",
			claim: "MCP uses JSON-RPC 2.0",
			formatChecks: []formatCheck{
				{name: "starts with verdict marker", check: startsWithVerdictMarker},
				{name: "contains quotes or absence statement", check: containsQuotesOrAbsence},
				{name: "contains explanation", check: containsExplanation},
			},
		},
		{
			name:  "claim about rate limits",
			claim: "MCP enforces rate limits",
			formatChecks: []formatCheck{
				{name: "starts with verdict marker", check: startsWithVerdictMarker},
				{name: "contains quotes or absence statement", check: containsQuotesOrAbsence},
				{name: "contains explanation", check: containsExplanation},
			},
		},
		{
			name:  "negative claim format",
			claim: "MCP does not enforce rate limits",
			formatChecks: []formatCheck{
				{name: "starts with verdict marker", check: startsWithVerdictMarker},
				{name: "mentions rate limits", check: mentionsRateLimits},
				{name: "contains explanation", check: containsExplanation},
			},
		},
		{
			name:  "compound negative claim format",
			claim: "MCP never forwards raw model traffic or enforces rate limits",
			formatChecks: []formatCheck{
				{name: "starts with verdict marker", check: startsWithVerdictMarker},
				{name: "addresses rate limit concept", check: mentionsRateLimits},
				{name: "contains quotes or absence for each", check: containsQuotesOrAbsence},
				{name: "contains explanation", check: containsExplanation},
			},
		},
		{
			name:  "claim about non-existent feature",
			claim: "MCP provides quantum entanglement",
			formatChecks: []formatCheck{
				{name: "starts with verdict marker", check: startsWithVerdictMarker},
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

			// Verify response has some verdict (either ACCURATE or INACCURATE)
			// Note: We don't assert which verdict as LLM responses are non-deterministic
			assert.True(t, hasVerdictMarker(allText), "Should have a verdict marker (✓ or ✗)")

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

// hasVerdictMarker checks if the text contains either ACCURATE or INACCURATE verdict
func hasVerdictMarker(text string) bool {
	return strings.Contains(text, "✓") || strings.Contains(text, "✗") ||
		strings.Contains(text, "ACCURATE") || strings.Contains(text, "INACCURATE")
}

// startsWithVerdictMarker checks if text starts with either ✓ or ✗ verdict marker
func startsWithVerdictMarker(text string) bool {
	trimmed := strings.TrimSpace(text)
	lines := strings.Split(trimmed, "\n")
	if len(lines) == 0 {
		return false
	}
	// Check first non-empty line after claim
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasSuffix(line, ":") {
			return strings.HasPrefix(line, "✓") || strings.HasPrefix(line, "✗")
		}
	}
	return false
}

// mentionsRateLimits checks if the text discusses rate limits
func mentionsRateLimits(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(lower, "rate limit") || strings.Contains(lower, "rate-limit") ||
		strings.Contains(lower, "ratelimit")
}

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
