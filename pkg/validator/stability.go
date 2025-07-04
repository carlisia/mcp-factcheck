package validator

import (
	"crypto/md5"
	"encoding/hex"
	"strings"
	"unicode/utf8"
)

// ContentStabilityChecker helps detect when validation results in no meaningful changes
type ContentStabilityChecker struct {
	// Track content hashes to detect circular validation
	contentHistory []string
	maxHistory     int
}

// NewContentStabilityChecker creates a new stability checker
func NewContentStabilityChecker() *ContentStabilityChecker {
	return &ContentStabilityChecker{
		contentHistory: make([]string, 0, 5),
		maxHistory:     5,
	}
}

// Reset clears the content history, useful for testing or starting fresh validation cycles
func (c *ContentStabilityChecker) Reset() {
	c.contentHistory = make([]string, 0, 5)
}

// normalizeContent removes formatting variations that don't affect meaning
func normalizeContent(content string) string {
	// Normalize whitespace
	normalized := strings.TrimSpace(content)

	// Normalize line endings first
	normalized = strings.ReplaceAll(normalized, "\r\n", "\n")

	// Normalize bullet points and whitespace per line
	lines := strings.Split(normalized, "\n")
	for i, line := range lines {
		// Normalize whitespace within each line
		trimmed := strings.TrimSpace(line)
		trimmed = strings.Join(strings.Fields(trimmed), " ")

		// Normalize different bullet styles to standard dash
		if len(trimmed) > 0 {
			// Check for various bullet point characters using rune comparison
			firstRune, _ := utf8.DecodeRuneInString(trimmed)
			switch firstRune {
			case '•', '*', '·', '◦', '▪', '▫', '‣', '⁃', '●':
				// Skip the first rune and prepend dash
				_, size := utf8.DecodeRuneInString(trimmed)
				lines[i] = "- " + strings.TrimSpace(trimmed[size:])
			default:
				lines[i] = trimmed
			}
		} else {
			lines[i] = ""
		}
	}

	return strings.Join(lines, "\n")
}

// contentHash generates a hash of normalized content
func contentHash(content string) string {
	normalized := normalizeContent(content)
	hash := md5.Sum([]byte(normalized))
	return hex.EncodeToString(hash[:])
}

// CheckStability checks if content has stabilized (no changes) or is in a loop
func (c *ContentStabilityChecker) CheckStability(originalContent, validatedContent string) StabilityResult {
	origHash := contentHash(originalContent)
	validatedHash := contentHash(validatedContent)

	result := StabilityResult{
		IsStable:            origHash == validatedHash,
		IsInLoop:            false,
		LoopLength:          0,
		NormalizedOriginal:  normalizeContent(originalContent),
		NormalizedValidated: normalizeContent(validatedContent),
	}

	// Check if we've seen this content before
	for i, prevHash := range c.contentHistory {
		if prevHash == validatedHash {
			result.IsInLoop = true
			result.LoopLength = len(c.contentHistory) - i
			break
		}
	}

	// Add to history
	c.contentHistory = append(c.contentHistory, validatedHash)
	if len(c.contentHistory) > c.maxHistory {
		c.contentHistory = c.contentHistory[1:]
	}

	return result
}

// StabilityResult contains the analysis of content stability
type StabilityResult struct {
	IsStable            bool   // Content hasn't changed meaningfully
	IsInLoop            bool   // Content is cycling through states
	LoopLength          int    // How many validations before it repeats
	NormalizedOriginal  string // Original content after normalization
	NormalizedValidated string // Validated content after normalization
}

// GetStabilityMessage returns a user-friendly message about the stability
func (r StabilityResult) GetStabilityMessage() string {
	if r.IsStable {
		return "✓ Content is already valid - no changes needed"
	}
	if r.IsInLoop {
		return "⚠️ Validation loop detected - content is cycling between states"
	}
	return ""
}
