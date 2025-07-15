// Package specs provides functionality for extracting and processing MCP specifications
// from various sources including GitHub repositories and local directories.
package specs

import (
	"regexp"
	"strings"
)

// ChunkingStrategy defines how to split MCP specification content into smaller chunks
// for processing and embedding generation. Different strategies optimize for different
// use cases such as semantic search accuracy or context preservation.
type ChunkingStrategy struct {
	Name            string
	ChunkSize       int  // Target size in characters
	ChunkOverlap    int  // Overlap between chunks
	SplitBySentence bool // Whether to split on sentence boundaries
	SplitByBullet   bool // Whether to split on bullet points
	KeepHeaders     bool // Whether to keep headers with their content
}

// Strategy name constants
const (
	strategyParagraph = "paragraph"
	strategySentence  = "sentence"
	strategyBullet    = "bullet"
	strategyFine      = "fine"
)

// DefaultStrategies provides common chunking strategies for different use cases.
// The "fine" strategy is optimized for matching short queries, while "paragraph"
// preserves more context for comprehensive understanding.
var DefaultStrategies = map[string]ChunkingStrategy{
	strategyParagraph: {
		Name:         strategyParagraph,
		ChunkSize:    1000,
		ChunkOverlap: 100,
	},
	strategySentence: {
		Name:            strategySentence,
		ChunkSize:       300,
		ChunkOverlap:    50,
		SplitBySentence: true,
	},
	strategyBullet: {
		Name:          strategyBullet,
		ChunkSize:     200,
		ChunkOverlap:  0,
		SplitByBullet: true,
		KeepHeaders:   true,
	},
	strategyFine: {
		Name:            strategyFine,
		ChunkSize:       150,
		ChunkOverlap:    30,
		SplitBySentence: true,
		SplitByBullet:   true,
		KeepHeaders:     true,
	},
}

// ChunkResult represents a chunk of content with associated metadata.
// It includes the content itself, its type (header, bullet, paragraph, sentence),
// parent section information, and position within the document.
type ChunkResult struct {
	Content  string
	Type     string // "header", "bullet", "paragraph", "sentence"
	Parent   string // Parent section header if applicable
	Position int    // Position in document
}

// ParseMarkdownSectionsV2 provides fine-grained chunking of markdown content
// using the specified strategy. It recognizes headers, bullet points, and
// paragraphs, creating appropriately sized chunks based on the strategy settings.
// The function preserves document structure and context while optimizing for
// embedding generation and semantic search.
func ParseMarkdownSectionsV2(content string, strategy ChunkingStrategy) []ChunkResult {
	var chunks []ChunkResult
	position := 0

	// Track current header context
	currentHeader := ""
	headerPattern := regexp.MustCompile(`^#+\s+(.+)$`)
	bulletPattern := regexp.MustCompile(`^[-*•]\s+(.+)$`)

	lines := strings.Split(content, "\n")
	var currentChunk strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if this is a header
		if matches := headerPattern.FindStringSubmatch(trimmed); matches != nil {
			// Save any accumulated content
			if currentChunk.Len() > 0 {
				chunks = append(chunks, ChunkResult{
					Content:  strings.TrimSpace(currentChunk.String()),
					Type:     "paragraph",
					Parent:   currentHeader,
					Position: position,
				})
				position++
				currentChunk.Reset()
			}

			// Update header context
			currentHeader = matches[1]

			// Add header as its own chunk
			chunks = append(chunks, ChunkResult{
				Content:  trimmed,
				Type:     "header",
				Parent:   "",
				Position: position,
			})
			position++
			continue
		}

		// Check if this is a bullet point
		if strategy.SplitByBullet && bulletPattern.MatchString(trimmed) {
			// Save any accumulated content
			if currentChunk.Len() > 0 {
				chunks = append(chunks, ChunkResult{
					Content:  strings.TrimSpace(currentChunk.String()),
					Type:     "paragraph",
					Parent:   currentHeader,
					Position: position,
				})
				position++
				currentChunk.Reset()
			}

			// Create bullet chunk (include header context if enabled)
			bulletContent := trimmed
			if strategy.KeepHeaders && currentHeader != "" {
				bulletContent = currentHeader + "\n" + trimmed
			}

			chunks = append(chunks, ChunkResult{
				Content:  bulletContent,
				Type:     "bullet",
				Parent:   currentHeader,
				Position: position,
			})
			position++
			continue
		}

		// Handle regular content
		if trimmed == "" {
			// Empty line - might signal end of paragraph
			if currentChunk.Len() > 0 {
				chunks = append(chunks, ChunkResult{
					Content:  strings.TrimSpace(currentChunk.String()),
					Type:     "paragraph",
					Parent:   currentHeader,
					Position: position,
				})
				position++
				currentChunk.Reset()
			}
		} else {
			// Add to current chunk
			if currentChunk.Len() > 0 {
				currentChunk.WriteString(" ")
			}
			currentChunk.WriteString(trimmed)

			// Check if we should split by sentence
			if strategy.SplitBySentence && strings.HasSuffix(trimmed, ".") {
				chunks = append(chunks, ChunkResult{
					Content:  strings.TrimSpace(currentChunk.String()),
					Type:     "sentence",
					Parent:   currentHeader,
					Position: position,
				})
				position++
				currentChunk.Reset()
			}
		}
	}

	// Add any remaining content
	if currentChunk.Len() > 0 {
		chunks = append(chunks, ChunkResult{
			Content:  strings.TrimSpace(currentChunk.String()),
			Type:     "paragraph",
			Parent:   currentHeader,
			Position: position,
		})
	}

	// Apply sliding window if overlap is specified
	if strategy.ChunkOverlap > 0 {
		chunks = applySlidingWindow(chunks, strategy.ChunkSize, strategy.ChunkOverlap)
	}

	return chunks
}

// applySlidingWindow creates overlapping chunks
func applySlidingWindow(chunks []ChunkResult, targetSize, overlap int) []ChunkResult {
	var windowedChunks []ChunkResult

	for i := 0; i < len(chunks); i++ {
		// Combine chunks until we reach target size
		combined := chunks[i].Content
		parent := chunks[i].Parent
		chunkType := chunks[i].Type

		// Look ahead to combine smaller chunks
		j := i + 1
		for j < len(chunks) && len(combined) < targetSize {
			combined += " " + chunks[j].Content
			j++
		}

		windowedChunks = append(windowedChunks, ChunkResult{
			Content:  combined,
			Type:     chunkType,
			Parent:   parent,
			Position: i,
		})

		// Move forward considering overlap
		if overlap > 0 && j-i > 1 {
			// Calculate how many chunks to skip based on overlap
			skip := (targetSize - overlap) / (len(combined) / (j - i))
			if skip < 1 {
				skip = 1
			}
			i += skip - 1
		}
	}

	return windowedChunks
}

// ConvertToStrings converts ChunkResults to simple strings for backward compatibility
// with systems expecting plain string chunks. This discards metadata but preserves
// the content for embedding generation.
func ConvertToStrings(chunks []ChunkResult) []string {
	result := make([]string, len(chunks))
	for i, chunk := range chunks {
		result[i] = chunk.Content
	}
	return result
}
