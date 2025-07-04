package specs

import (
	"strings"
	"testing"
)

// Test helper functions for string assertions
// These are available to all test files in the specs package

// AssertContains verifies that text contains the expected string
func AssertContains(t *testing.T, text string, want string) {
	t.Helper()
	if !strings.Contains(text, want) {
		t.Errorf("Expected to find '%s' in text", want)
	}
}

// AssertNotContains verifies that text does not contain the forbidden string
func AssertNotContains(t *testing.T, text string, forbidden string) {
	t.Helper()
	if strings.Contains(text, forbidden) {
		t.Errorf("Did not expect to find '%s' in text (should be processed/removed)", forbidden)
	}
}

// AssertChunkNotEmpty verifies that a chunk contains meaningful content
func AssertChunkNotEmpty(t *testing.T, chunk string, index int) {
	t.Helper()
	trimmed := strings.TrimSpace(chunk)
	if trimmed == "" {
		t.Errorf("Chunk %d is empty or only whitespace", index)
	}
}

// AssertChunkSizeReasonable verifies chunk size is within expected bounds
func AssertChunkSizeReasonable(t *testing.T, chunk string, index int, minSize, maxSize int) {
	t.Helper()
	size := len(chunk)
	if size > maxSize {
		t.Errorf("Chunk %d is too large (%d chars > %d max), should be split", index, size, maxSize)
	}
	if size < minSize && strings.TrimSpace(chunk) != "" {
		t.Errorf("Chunk %d is suspiciously small (%d chars < %d min): %q", index, size, minSize, chunk)
	}
}

// AssertAllContains verifies that text contains all expected strings
func AssertAllContains(t *testing.T, text string, expected []string) {
	t.Helper()
	for _, want := range expected {
		AssertContains(t, text, want)
	}
}

// AssertEqual verifies that two strings are equal with context
func AssertEqual(t *testing.T, got, want string, context string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %q, want %q", context, got, want)
	}
}

// AssertChunkResult verifies common properties of ChunkResult
func AssertChunkResult(t *testing.T, chunk ChunkResult, index int) {
	t.Helper()
	if chunk.Position < 0 {
		t.Errorf("Chunk %d has invalid Position: %d", index, chunk.Position)
	}
	if chunk.Content == "" {
		t.Errorf("Chunk %d has empty content", index)
	}
	if chunk.Type == "" {
		t.Errorf("Chunk %d has empty Type", index)
	}
}

// TruncateString is a helper function to truncate long strings for logging
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 10 {
		return s[:maxLen]
	}
	// Show start and end for better context
	startLen := (maxLen - 3) * 2 / 3
	endLen := maxLen - startLen - 3
	return s[:startLen] + "..." + s[len(s)-endLen:]
}

// LogChunkDistribution logs detailed chunk size distribution for analysis
func LogChunkDistribution(t *testing.T, chunks []string) {
	t.Helper()
	if len(chunks) == 0 {
		return
	}

	// Calculate size statistics
	sizes := make([]int, len(chunks))
	minSize, maxSize := len(chunks[0]), len(chunks[0])
	for i, chunk := range chunks {
		sizes[i] = len(chunk)
		if sizes[i] < minSize {
			minSize = sizes[i]
		}
		if sizes[i] > maxSize {
			maxSize = sizes[i]
		}
	}

	t.Logf("  Chunk size distribution: min=%d, max=%d, spread=%d",
		minSize, maxSize, maxSize-minSize)
}
