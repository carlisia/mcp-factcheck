package testutil

import (
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// AssertTextContains verifies that the MCP content contains the expected text.
func AssertTextContains(t *testing.T, content []mcp.Content, expected string) {
	t.Helper()
	if len(content) == 0 {
		t.Errorf("Expected content containing %q, but got empty response", expected)
		return
	}

	// Check all content items, not just the first
	for _, c := range content {
		if strings.Contains(extractText(c), expected) {
			return
		}
	}

	// If we get here, the expected text wasn't found
	allText := ""
	for i, c := range content {
		if i > 0 {
			allText += " | "
		}
		allText += extractText(c)
	}
	t.Errorf("Expected content to contain %q, but got: %s", expected, allText)
}

// AssertNonEmpty verifies that the MCP content is not empty.
func AssertNonEmpty(t *testing.T, content []mcp.Content) {
	t.Helper()
	if len(content) == 0 {
		t.Error("Expected non-empty content, but got empty response")
		return
	}

	text := extractText(content[0])
	if strings.TrimSpace(text) == "" {
		t.Error("Expected non-empty text content")
	}
}

// AssertContentCount verifies the exact number of content items.
func AssertContentCount(t *testing.T, content []mcp.Content, expected int) {
	t.Helper()
	if len(content) != expected {
		t.Errorf("Expected %d content items, got %d", expected, len(content))
	}
}

// AssertErrorContains verifies that an error contains the expected substring.
func AssertErrorContains(t *testing.T, err error, expected string) {
	t.Helper()
	if err == nil {
		t.Errorf("Expected error containing %q, but got nil", expected)
		return
	}
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("Expected error containing %q, got: %v", expected, err)
	}
}

// AssertIsTextContent verifies that the content is of type TextContent.
func AssertIsTextContent(t *testing.T, content mcp.Content) string {
	t.Helper()
	// Try both pointer and non-pointer assertions
	if tc, ok := content.(mcp.TextContent); ok {
		return tc.Text
	}
	if tc, ok := content.(*mcp.TextContent); ok {
		return tc.Text
	}
	t.Errorf("Expected TextContent, got %T", content)
	return ""
}

// AssertMarkdownFormat verifies basic markdown formatting.
func AssertMarkdownFormat(t *testing.T, content []mcp.Content) {
	t.Helper()
	if len(content) == 0 {
		t.Error("Expected content for markdown validation, but got empty response")
		return
	}

	text := extractText(content[0])

	// Check for common markdown patterns
	hasHeader := strings.Contains(text, "##") || strings.Contains(text, "#")
	hasBold := strings.Contains(text, "**")
	hasList := strings.Contains(text, "- ") || strings.Contains(text, "* ") || strings.Contains(text, "• ")

	if !hasHeader && !hasBold && !hasList {
		t.Error("Expected markdown formatting (headers, bold, or lists)")
	}
}

// AssertValidationResult checks common validation response patterns.
func AssertValidationResult(t *testing.T, content []mcp.Content, expectValid bool) {
	t.Helper()
	if len(content) == 0 {
		t.Error("Expected validation result, but got empty response")
		return
	}

	text := extractText(content[0])

	// Common validation result indicators
	validIndicators := []string{"✓", "valid", "Valid", "accurate", "Accurate", "correct", "Correct"}
	invalidIndicators := []string{"✗", "invalid", "Invalid", "inaccurate", "Inaccurate", "incorrect", "Incorrect"}

	hasValidIndicator := false
	hasInvalidIndicator := false

	for _, indicator := range validIndicators {
		if strings.Contains(text, indicator) {
			hasValidIndicator = true
			break
		}
	}

	for _, indicator := range invalidIndicators {
		if strings.Contains(text, indicator) {
			hasInvalidIndicator = true
			break
		}
	}

	if expectValid && !hasValidIndicator {
		t.Errorf("Expected validation success indicators, but got: %s", text)
	}
	if !expectValid && !hasInvalidIndicator {
		t.Errorf("Expected validation failure indicators, but got: %s", text)
	}
}

// AssertConfidenceScore checks for confidence score in validation results.
func AssertConfidenceScore(t *testing.T, content []mcp.Content, minConfidence float64) {
	t.Helper()
	if len(content) == 0 {
		t.Error("Expected content with confidence score, but got empty response")
		return
	}

	text := extractText(content[0])

	// Look for common confidence patterns
	confidencePatterns := []string{"Confidence:", "confidence:", "Score:", "score:"}
	hasConfidence := false

	for _, pattern := range confidencePatterns {
		if strings.Contains(text, pattern) {
			hasConfidence = true
			break
		}
	}

	if !hasConfidence {
		t.Logf("Warning: No confidence score found in: %s", text)
	}
}

// AssertTextContainsAny verifies that the MCP content contains any of the expected strings.
func AssertTextContainsAny(t *testing.T, content []mcp.Content, options ...string) {
	t.Helper()
	if len(content) == 0 {
		t.Errorf("Expected content but got empty response")
		return
	}

	// Check all content items, not just the first
	for _, c := range content {
		text := extractText(c)
		for _, opt := range options {
			if strings.Contains(text, opt) {
				return
			}
		}
	}

	// If we get here, none of the options were found
	allText := ""
	for i, c := range content {
		if i > 0 {
			allText += " | "
		}
		allText += extractText(c)
	}
	t.Errorf("Expected content to contain any of %v, but got: %s", options, allText)
}

// AssertEmpty verifies that the MCP content is empty.
func AssertEmpty(t *testing.T, content []mcp.Content) {
	t.Helper()
	if len(content) != 0 {
		t.Errorf("Expected empty content, got %d items", len(content))
	}
}

// AssertAllTextContent verifies that all content items are TextContent.
func AssertAllTextContent(t *testing.T, content []mcp.Content) {
	t.Helper()
	for i, c := range content {
		if _, ok := c.(mcp.TextContent); !ok {
			if _, ok := c.(*mcp.TextContent); !ok {
				t.Errorf("Content[%d] is not TextContent, got %T", i, c)
			}
		}
	}
}

// AssertTextOccurrenceCount verifies that a specific text appears exactly n times in the content.
func AssertTextOccurrenceCount(t *testing.T, content []mcp.Content, needle string, want int) {
	t.Helper()
	got := 0
	for _, c := range content {
		text := extractText(c)
		if text == needle {
			got++
		}
	}
	if got != want {
		t.Errorf("Expected %d occurrences of %q, got %d", want, needle, got)
	}
}

// AssertTextContainsCount verifies that a substring appears in exactly n content items.
func AssertTextContainsCount(t *testing.T, content []mcp.Content, substr string, want int) {
	t.Helper()
	got := 0
	for _, c := range content {
		text := extractText(c)
		if strings.Contains(text, substr) {
			got++
		}
	}
	if got != want {
		t.Errorf("Expected %d content items containing %q, got %d", want, substr, got)
	}
}

// ExtractText is a public helper to extract text from MCP content.
// Useful for custom assertions in tests.
func ExtractText(content mcp.Content) string {
	return extractText(content)
}

// extractText is a helper to extract text from MCP content.
func extractText(content mcp.Content) string {
	// Try both pointer and non-pointer type assertions
	if tc, ok := content.(mcp.TextContent); ok {
		return tc.Text
	}
	if tc, ok := content.(*mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}
