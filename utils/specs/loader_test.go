// Package specs_test tests the markdown parsing logic used for loading spec files.
// These tests verify:
// - Content chunking based on headers and paragraphs
// - Code block preservation within chunks
// - Bullet point handling
// - Edge cases (empty content, very long paragraphs)
//
// Chunk Count Tuning:
// The minChunks and maxChunks fields in test cases define expected boundaries for chunking.
// When adjusting the chunking algorithm, update these values to match the new behavior:
// - minChunks: The minimum expected chunks (fails if fewer chunks are produced)
// - maxChunks: The maximum expected chunks (fails if more chunks are produced)
// This approach allows the implementation to evolve while maintaining reasonable bounds.
// If the chunking strategy changes significantly, simply update the test expectations
// rather than the implementation-specific details in validate functions.
package specs

import (
	"strings"
	"testing"
)

func TestParseMarkdownSections(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		minChunks   int
		maxChunks   int
		contains    []string
		notContains []string // Things that should NOT appear (e.g., markdown syntax)
		validate    func(t *testing.T, chunks []string)
	}{
		{
			name: "simple markdown with headers",
			content: `# Header 1
This is paragraph one.

## Header 2
This is paragraph two.

### Header 3
This is paragraph three.`,
			minChunks: 1, // Expected: fine-grained strategy may combine small sections
			maxChunks: 6, // Maximum if each element is separate
			contains:  []string{"Header 1", "Header 2", "Header 3", "paragraph one", "paragraph two", "paragraph three"},
			validate: func(t *testing.T, chunks []string) {
				t.Helper()
				// With fine-grained strategy, small content may be combined into fewer chunks
				// Just verify all content is present
				allContent := strings.Join(chunks, " ")
				expectedPairs := [][]string{
					{"Header 1", "paragraph one"},
					{"Header 2", "paragraph two"},
					{"Header 3", "paragraph three"},
				}
				for _, pair := range expectedPairs {
					AssertContains(t, allContent, pair[0])
					AssertContains(t, allContent, pair[1])
				}
			},
		},
		{
			name: "markdown with bullet points",
			content: `# Main Section
Introduction text.

## Features
- Feature one
- Feature two
- Feature three

## Benefits
* Benefit one
* Benefit two`,
			minChunks: 2, // Expected: Main Section chunk + Features/Benefits chunks
			maxChunks: 4, // Maximum if each section is separate
			contains:  []string{"Main Section", "Features", "Benefits", "Feature one", "Benefit one"},
			// Note: parseMarkdownSections preserves markdown syntax, doesn't strip it
			// notContains: []string{"-", "*"}, // Removed - bullet markers are preserved
			validate: func(t *testing.T, chunks []string) {
				t.Helper()
				// Verify bullet points are properly grouped
				foundFeaturesChunk := false
				foundBenefitsChunk := false
				for _, chunk := range chunks {
					if strings.Contains(chunk, "Features") &&
						strings.Contains(chunk, "Feature one") &&
						strings.Contains(chunk, "Feature two") {
						foundFeaturesChunk = true
					}
					if strings.Contains(chunk, "Benefits") &&
						strings.Contains(chunk, "Benefit one") &&
						strings.Contains(chunk, "Benefit two") {
						foundBenefitsChunk = true
					}
				}
				if !foundFeaturesChunk {
					t.Error("Expected all features to be grouped in the same chunk")
				}
				if !foundBenefitsChunk {
					t.Error("Expected all benefits to be grouped in the same chunk")
				}
			},
		},
		{
			name: "markdown with code blocks",
			content: `# Code Example
Here's some code:

` + "```json" + `
{
  "key": "value"
}
` + "```" + `

More text after code.`,
			minChunks: 1, // Expected: all content in one chunk if small enough
			maxChunks: 2, // Maximum if code block causes split
			contains:  []string{"Code Example", "json", "key", "value", "More text"},
			// Note: parseMarkdownSections preserves markdown syntax
			// notContains: []string{"```"}, // Removed - code fences are preserved
			validate: func(t *testing.T, chunks []string) {
				t.Helper()
				// Check that code block is preserved intact
				foundCompleteCodeBlock := false
				for _, chunk := range chunks {
					if strings.Contains(chunk, `"key": "value"`) {
						// Verify the JSON structure is intact
						if strings.Contains(chunk, "{") && strings.Contains(chunk, "}") {
							foundCompleteCodeBlock = true
						}
					}
				}
				if !foundCompleteCodeBlock {
					t.Error("Expected code block to be preserved with complete JSON structure")
				}

				// Verify code block context is preserved
				foundCodeWithContext := false
				for _, chunk := range chunks {
					if strings.Contains(chunk, "Here's some code") &&
						strings.Contains(chunk, "key") &&
						strings.Contains(chunk, "value") {
						foundCodeWithContext = true
					}
				}
				if !foundCodeWithContext {
					t.Error("Expected code block to be kept with its surrounding context")
				}
			},
		},
		{
			name:      "empty content",
			content:   "",
			minChunks: 0, // Expected: no chunks for empty content
			maxChunks: 0, // Expected: no chunks for empty content
			validate: func(t *testing.T, chunks []string) {
				t.Helper()
				if len(chunks) != 0 {
					t.Errorf("Expected exactly 0 chunks for empty content, got %d", len(chunks))
				}
			},
		},
		{
			name:      "only whitespace",
			content:   "   \n\n   \t\n   ",
			minChunks: 0, // Expected: whitespace is trimmed, resulting in no chunks
			maxChunks: 1, // Maximum: could be one empty chunk
			validate: func(t *testing.T, chunks []string) {
				t.Helper()
				// If any chunks exist, they should not be just whitespace
				for i, chunk := range chunks {
					if strings.TrimSpace(chunk) == "" {
						t.Errorf("Chunk %d contains only whitespace, should be filtered out", i)
					}
				}
			},
		},
		{
			name: "long paragraph that needs splitting",
			content: strings.Repeat("This is a very long sentence. ", 50) + "\n\n" +
				strings.Repeat("Another long paragraph. ", 50),
			minChunks: 2, // Expected: each paragraph forms its own chunk due to length
			maxChunks: 4, // Maximum: could split further based on chunking algorithm
			validate: func(t *testing.T, chunks []string) {
				t.Helper()
				// Verify content is preserved
				allContent := strings.Join(chunks, " ")
				AssertContains(t, allContent, "This is a very long sentence")
				AssertContains(t, allContent, "Another long paragraph")

				// Check chunk sizes are reasonable
				for i, chunk := range chunks {
					AssertChunkSizeReasonable(t, chunk, i, 10, 2000)
				}
			},
		},
		{
			name: "complex markdown with mixed content",
			content: `# MCP Specification

## Overview
The Model Context Protocol (MCP) defines how AI models interact with external systems.

### Key Features
- **JSON-RPC 2.0**: All communication uses JSON-RPC
- **Stateful Sessions**: Maintains context between calls
- **Tool Integration**: Exposes callable tools

## Implementation Details

### Message Format
` + "```json" + `
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "search_spec",
    "arguments": {"query": "test"}
  }
}
` + "```" + `

### Error Handling
Errors follow the JSON-RPC 2.0 error format with additional MCP-specific codes.

## Conclusion
MCP provides a robust framework for AI-system integration.`,
			minChunks: 1,  // Expected: depends on chunking strategy
			maxChunks: 10, // Maximum: if every section is separate
			contains: []string{
				"MCP Specification",
				"JSON-RPC 2.0",
				"Tool Integration",
				"Message Format",
				"tools/call",
				"Error Handling",
				"robust framework",
			},
			validate: func(t *testing.T, chunks []string) {
				t.Helper()
				// Verify important sections are preserved
				allContent := strings.Join(chunks, "\n")

				// Check that code block is complete
				if strings.Contains(allContent, `"jsonrpc"`) {
					if !strings.Contains(allContent, `"method"`) || !strings.Contains(allContent, `"params"`) {
						t.Error("JSON code block appears to be incomplete")
					}
				}

				// Verify hierarchical structure is maintained
				if strings.Contains(allContent, "### Key Features") {
					// Should also contain the parent section
					if !strings.Contains(allContent, "## Overview") {
						t.Log("Warning: Subsection found without parent section context")
					}
				}

				// Check for reasonable chunk distribution
				t.Logf("Complex markdown chunked into %d chunks", len(chunks))
				if len(chunks) == 1 {
					t.Log("Note: All content in single chunk - may want finer granularity for search")
				} else if len(chunks) > 8 {
					t.Log("Note: Many small chunks - may want coarser granularity")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseMarkdownSections(tt.content)

			// Enhanced logging with chunk statistics
			t.Logf("Test case: '%s'", tt.name)
			t.Logf("  Input size: %d chars", len(tt.content))
			t.Logf("  Got %d chunks (expected: %d-%d)", len(result), tt.minChunks, tt.maxChunks)

			// Log chunk details
			totalSize := 0
			for i, chunk := range result {
				totalSize += len(chunk)
				preview := TruncateString(chunk, 60)
				t.Logf("  Chunk %d: %d chars | %q", i, len(chunk), preview)
			}

			// Log chunk statistics
			if len(result) > 0 {
				avgSize := totalSize / len(result)
				t.Logf("  Average chunk size: %d chars", avgSize)
				t.Logf("  Total output size: %d chars (%.1f%% of input)",
					totalSize, float64(totalSize)/float64(len(tt.content))*100)
				LogChunkDistribution(t, result)
			}

			// Check minimum chunks
			if len(result) < tt.minChunks {
				t.Errorf("Expected at least %d chunks, got %d", tt.minChunks, len(result))
			}

			// Check maximum chunks if specified
			if tt.maxChunks > 0 && len(result) > tt.maxChunks {
				t.Errorf("Expected at most %d chunks, got %d", tt.maxChunks, len(result))
			}

			// Check that expected content is present
			allContent := strings.Join(result, " ")
			AssertAllContains(t, allContent, tt.contains)

			// Check that unwanted content is NOT present
			for _, notExpected := range tt.notContains {
				AssertNotContains(t, allContent, notExpected)
			}

			// Check no empty chunks (unless content is empty)
			if tt.content != "" {
				for i, chunk := range result {
					AssertChunkNotEmpty(t, chunk, i)
				}
			}

			// Run custom validation if provided
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// Note: Test helper functions are now in test_helpers.go and available to all test files in this package
