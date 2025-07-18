package contentprep

import (
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/textsplitter"
)

// Chunk represents a logical piece of content for validation
type Chunk struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Position int    `json:"position"`
	Type     string `json:"type"`            // "paragraph", "heading", "code_block", "list_item"
	Level    int    `json:"level,omitempty"` // For headings (1-6)
}

// ChunkResult contains the chunked content and metadata
type ChunkResult struct {
	Chunks          []Chunk
	Claims          []string
	Summary         string
	OriginalContent string
	TotalChunks     int `json:"total_chunks"`
	TotalChars      int `json:"total_chars"`
	EstTokens       int `json:"estimated_tokens"`
}

// Split splits content into logical chunks for validation using langchaingo
func Split(content string) *ChunkResult {
	if strings.TrimSpace(content) == "" {
		return &ChunkResult{
			Chunks:          []Chunk{},
			OriginalContent: content,
			TotalChunks:     0,
			TotalChars:      0,
			EstTokens:       0,
		}
	}

	// Choose splitter based on content type
	var splitter textsplitter.TextSplitter

	// Use markdown splitter if content contains markdown-like patterns
	if strings.Contains(content, "#") || strings.Contains(content, "```") ||
		strings.Contains(content, "- ") || strings.Contains(content, "* ") {
		splitter = textsplitter.NewMarkdownTextSplitter(
			textsplitter.WithChunkSize(800),    // Smaller chunks for better granularity
			textsplitter.WithChunkOverlap(100), // Overlap for context preservation
		)
	} else {
		// Use recursive character splitter for plain text
		splitter = textsplitter.NewRecursiveCharacter(
			textsplitter.WithChunkSize(800),    // Smaller chunks for better granularity
			textsplitter.WithChunkOverlap(100), // Overlap for context preservation
		)
	}

	// Split the content
	docs, err := splitter.SplitText(content)
	if err != nil {
		// Fallback to simple splitting if the splitter fails
		docs = []string{content}
	}

	// Convert to our Chunk format
	chunks := make([]Chunk, len(docs))
	for i, doc := range docs {
		chunks[i] = Chunk{
			ID:       generateChunkID("chunk", i),
			Text:     strings.TrimSpace(doc),
			Position: i,
			Type:     "text_chunk", // langchaingo doesn't classify types, so use generic
		}
	}

	// Calculate metadata
	totalChars := len(content)
	estTokens := totalChars / 4 // Rough approximation

	return &ChunkResult{
		Chunks:          chunks,
		OriginalContent: content,
		TotalChunks:     len(chunks),
		TotalChars:      totalChars,
		EstTokens:       estTokens,
	}
}

// Truncate truncates content to a reasonable length for embedding/search
func Truncate(content string) string {
	const maxLength = 500
	if len(content) <= maxLength {
		return content
	}
	// Try to truncate at a sentence boundary
	truncated := content[:maxLength]
	lastPeriod := strings.LastIndex(truncated, ". ")
	if lastPeriod > 0 {
		return truncated[:lastPeriod+1]
	}
	// Otherwise just truncate at word boundary
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > 0 {
		return truncated[:lastSpace] + "..."
	}
	return truncated + "..."
}

func generateChunkID(prefix string, position int) string {
	return fmt.Sprintf("%s-%d", prefix, position)
}
