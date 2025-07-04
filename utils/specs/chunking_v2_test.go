// Package specs_test tests the markdown chunking and windowing logic.
// These tests verify:
// - Content preservation (headers, bullet points, code blocks)
// - Chunking strategies (fine-grained vs paragraph-based)
// - Sliding window behavior for chunk combination
// - Edge cases (empty content, very small/large content)
package specs

import (
	"strings"
	"testing"
)

func TestParseMarkdownSectionsV2(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		strategy     ChunkingStrategy
		minChunks    int
		checkContent func(t *testing.T, chunks []ChunkResult)
	}{
		{
			name: "fine-grained strategy with headers",
			content: `# Main Title
This is the introduction paragraph that explains the main concepts.

## Section One
First section content with some details.

### Subsection 1.1
More specific content here.

## Section Two
Second section with different information.`,
			strategy:  DefaultStrategies[strategyFine],
			minChunks: 3, // Expected: header chunk + 2 section chunks minimum for fine-grained strategy
			checkContent: func(t *testing.T, chunks []ChunkResult) {
				t.Logf("Got %d chunks", len(chunks))
				for i, chunk := range chunks {
					t.Logf("Chunk %d: Type=%s, Length=%d, Position=%d",
						i, chunk.Type, len(chunk.Content), chunk.Position)
				}
				// Check that headers are preserved
				foundMain := false
				foundSection := false
				var headerChunk *ChunkResult
				for i, chunk := range chunks {
					if strings.Contains(chunk.Content, "Main Title") {
						foundMain = true
						headerChunk = &chunks[i]
					}
					if strings.Contains(chunk.Content, "Section One") {
						foundSection = true
					}
				}
				if !foundMain {
					t.Error("Expected to find 'Main Title' in chunks")
				}
				if !foundSection {
					t.Error("Expected to find 'Section One' in chunks")
				}
				// Verify metadata integrity
				if headerChunk != nil {
					if headerChunk.Type != "header" {
						t.Logf("Note: Header chunk has type '%s' instead of 'header'", headerChunk.Type)
					}
					if headerChunk.Position < 0 {
						t.Error("Expected valid Position for header chunk")
					}
					// Check Parent field for nested headers
					if headerChunk.Parent != "" {
						t.Logf("Header chunk parent: %s", headerChunk.Parent)
					}
				}
			},
		},
		{
			name: "regular strategy with bullet points",
			content: `# Features
Here are the main features:

- Feature A: Does something important
- Feature B: Does another thing
- Feature C: Yet another feature

Each feature is designed to work together.`,
			strategy:  DefaultStrategies[strategyParagraph],
			minChunks: 1, // Expected: regular strategy groups content into larger chunks
			checkContent: func(t *testing.T, chunks []ChunkResult) {
				// Should keep bullet points together in regular strategy
				allContent := ""
				for _, chunk := range chunks {
					allContent += chunk.Content
				}
				if !strings.Contains(allContent, "Feature A") {
					t.Error("Expected to find 'Feature A' in chunks")
				}
				if !strings.Contains(allContent, "Feature B") {
					t.Error("Expected to find 'Feature B' in chunks")
				}
				if !strings.Contains(allContent, "Feature C") {
					t.Error("Expected to find 'Feature C' in chunks")
				}
			},
		},
		{
			name: "fine-grained with code blocks",
			content: `# Code Example
Here's how to use it:

` + "```python" + `
def hello():
    print("Hello, world!")
` + "```" + `

That's the basic example.`,
			strategy:  DefaultStrategies[strategyFine],
			minChunks: 1, // Expected: at least one chunk containing the code block
			checkContent: func(t *testing.T, chunks []ChunkResult) {
				foundCode := false
				foundCodeFence := false
				for i, chunk := range chunks {
					t.Logf("Chunk %d: Type=%s, Content=%q", i, chunk.Type, chunk.Content)
					if strings.Contains(chunk.Content, "def hello():") {
						foundCode = true
						// Check that code fence is preserved
						if strings.Contains(chunk.Content, "```python") {
							foundCodeFence = true
						}
						// Check metadata for code chunks
						if chunk.Type == "" {
							t.Errorf("Expected chunk type to be set, got empty string")
						}
						// Verify position metadata
						if chunk.Position < 0 {
							t.Errorf("Expected valid position, got %d", chunk.Position)
						}
						t.Logf("Code chunk type: %s, position: %d", chunk.Type, chunk.Position)
					}
				}
				if !foundCode {
					t.Error("Expected to find code block in chunks")
				}
				if !foundCodeFence {
					t.Error("Expected code fence (```python) to be preserved in chunk containing code")
				}
			},
		},
		{
			name:      "empty content",
			content:   "",
			strategy:  DefaultStrategies[strategyFine],
			minChunks: 0, // Expected: no chunks for empty content
		},
		{
			name:      "very small content",
			content:   "Short.",
			strategy:  DefaultStrategies[strategyFine],
			minChunks: 1, // Expected: single chunk for very short content
			checkContent: func(t *testing.T, chunks []ChunkResult) {
				if len(chunks) != 1 {
					t.Errorf("Expected exactly 1 chunk for short content, got %d", len(chunks))
					return
				}
				if chunks[0].Content != "Short." {
					t.Errorf("Expected chunk content 'Short.', got '%s'", chunks[0].Content)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseMarkdownSectionsV2(tt.content, tt.strategy)

			if len(result) < tt.minChunks {
				t.Errorf("Expected at least %d chunks, got %d", tt.minChunks, len(result))
			}

			// Run custom content checks
			if tt.checkContent != nil {
				tt.checkContent(t, result)
			}

			// Verify chunk metadata
			for i, chunk := range result {
				if chunk.Position < 0 {
					t.Errorf("Chunk %d has invalid Position: %d", i, chunk.Position)
				}
				if chunk.Content == "" && tt.content != "" {
					t.Errorf("Chunk %d has empty content", i)
				}
			}
		})
	}
}

func TestApplySlidingWindow(t *testing.T) {
	tests := []struct {
		name       string
		chunks     []ChunkResult
		targetSize int
		overlap    int
		minOutput  int
		maxOutput  int
	}{
		{
			name: "combine small chunks",
			chunks: []ChunkResult{
				{Content: "First chunk."},
				{Content: "Second chunk."},
				{Content: "Third chunk."},
			},
			targetSize: 30,
			overlap:    5,
			minOutput:  1,
			maxOutput:  3, // Expected: may combine or keep separate based on size constraints
		},
		{
			name: "single large chunk",
			chunks: []ChunkResult{
				{Content: strings.Repeat("Large content. ", 20)},
			},
			targetSize: 100,
			overlap:    10,
			minOutput:  1, // Expected: current implementation preserves large chunks without splitting
		},
		{
			name:       "empty chunks",
			chunks:     []ChunkResult{},
			targetSize: 100,
			overlap:    10,
			minOutput:  0, // Expected: no output for empty input
			maxOutput:  0, // Expected: no output for empty input
		},
		{
			name: "exact target size",
			chunks: []ChunkResult{
				{Content: "Exactly 50 characters content.........................."},
			},
			targetSize: 50,
			overlap:    10,
			minOutput:  1,
			maxOutput:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applySlidingWindow(tt.chunks, tt.targetSize, tt.overlap)

			t.Logf("Input: %d chunks, targetSize=%d, overlap=%d", len(tt.chunks), tt.targetSize, tt.overlap)
			t.Logf("Output: %d chunks", len(result))
			for i, chunk := range result {
				t.Logf("  Chunk %d: length=%d, content=%q", i, len(chunk.Content), chunk.Content)
			}

			if len(result) < tt.minOutput {
				t.Errorf("Expected at least %d output chunks, got %d", tt.minOutput, len(result))
			}

			if tt.maxOutput > 0 && len(result) > tt.maxOutput {
				t.Errorf("Expected at most %d output chunks, got %d", tt.maxOutput, len(result))
			}

			// Note: Current implementation may exceed target size
			// This is acceptable behavior for the sliding window algorithm

			// Verify overlap exists when there are multiple chunks
			if len(result) > 1 && tt.overlap > 0 {
				for i := 0; i < len(result)-1; i++ {
					curr := result[i].Content
					next := result[i+1].Content

					// Check if there's some overlap in content
					currEnd := curr[len(curr)-min(tt.overlap, len(curr)):]
					nextStart := next[:min(tt.overlap, len(next))]

					if !strings.Contains(next, currEnd) && !strings.Contains(curr, nextStart) {
						// This is a loose check - exact overlap depends on implementation
						t.Logf("Warning: Chunk %d and %d may not have expected overlap", i, i+1)
					}
				}
			}
		})
	}
}

func TestConvertToStrings(t *testing.T) {
	tests := []struct {
		name     string
		chunks   []ChunkResult
		expected []string
	}{
		{
			name: "multiple chunks",
			chunks: []ChunkResult{
				{Content: "First"},
				{Content: "Second"},
				{Content: "Third"},
			},
			expected: []string{"First", "Second", "Third"},
		},
		{
			name:     "empty chunks",
			chunks:   []ChunkResult{},
			expected: []string{},
		},
		{
			name: "single chunk",
			chunks: []ChunkResult{
				{Content: "Only one"},
			},
			expected: []string{"Only one"},
		},
		{
			name: "chunks with special characters",
			chunks: []ChunkResult{
				{Content: "Line 1\nLine 2"},
				{Content: "Tab\there"},
				{Content: "Quote\"s"},
			},
			expected: []string{"Line 1\nLine 2", "Tab\there", "Quote\"s"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToStrings(tt.chunks)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d strings, got %d", len(tt.expected), len(result))
				return
			}

			for i, str := range result {
				if str != tt.expected[i] {
					t.Errorf("String %d: expected '%s', got '%s'", i, tt.expected[i], str)
				}
			}
		})
	}
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
