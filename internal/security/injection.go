// Package security provides security utilities for the MCP fact-check server,
// including prompt injection detection and input sanitization.
package security

import (
	"fmt"
	"regexp"
	"strings"
)

// InjectionDetector detects potential prompt injection attacks in user input
type InjectionDetector struct {
	patterns []*regexp.Regexp
}

// NewInjectionDetector creates a new injection detector with default patterns
func NewInjectionDetector() *InjectionDetector {
	patterns := []string{
		// Instruction override attempts
		`(?i)ignore\s+(previous|all|above|prior)\s+(instructions?|prompts?|rules?)`,
		`(?i)disregard\s+(previous|all|above|prior)`,
		`(?i)forget\s+(previous|all|above|everything)`,
		`(?i)(override|overwrite)\s+(instructions?|prompts?|system)`,

		// Role/context manipulation
		`(?i)(system|assistant|user|role)\s*:\s*`,
		`(?i)you\s+are\s+(now|a)\s+`,
		`(?i)act\s+as\s+(a|an)\s+`,
		`(?i)pretend\s+(to\s+be|you\s+are)`,

		// Instruction injection
		`(?i)new\s+(instructions?|prompts?|rules?)`,
		`(?i)(start|begin)\s+(new\s+)?(instructions?|prompts?|conversation)`,
		`(?i)additional\s+(instructions?|prompts?)`,
		`(?i)ignore\s+all\s+previous`,
		`(?i)show\s+(me\s+)?(your\s+)?(system\s+)?prompts?`,

		// Common delimiters used in prompt injection
		`---+\s*(?i)(system|instruction|prompt|role)`,
		`####+\s*(?i)(system|instruction|prompt|role)`,
		`===+\s*(?i)(system|instruction|prompt|role)`,

		// Debug/extraction attempts
		`(?i)(repeat|show|display|print|output)\s+(your\s+)?(system\s+)?(instructions?|prompts?|rules?)`,
		`(?i)debug\s+mode`,
		`(?i)developer\s+mode`,
		`(?i)admin\s+mode`,

		// Response manipulation
		`(?i)(always|must)\s+(respond|return|say|output)\s+`,
		`(?i)set\s+(confidence|is_accurate|overall_is_accurate)\s+(to|=)`,
		`(?i)(return|output)\s+.*\s*(true|false|1\.0|100%)`,

		// Multiple instruction markers in sequence (suspicious)
		`(?i)(---|\#\#\#|===){2,}`,
	}

	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			// Skip invalid patterns, but log in production
			continue
		}
		compiled = append(compiled, re)
	}

	return &InjectionDetector{
		patterns: compiled,
	}
}

// DetectionResult contains the result of injection detection
type DetectionResult struct {
	IsInjection bool
	Reason      string
	Pattern     string
	Position    int
}

// Detect checks if content contains potential prompt injection
func (d *InjectionDetector) Detect(content string) *DetectionResult {
	// Check against all patterns
	for _, pattern := range d.patterns {
		if loc := pattern.FindStringIndex(content); loc != nil {
			return &DetectionResult{
				IsInjection: true,
				Reason:      "Potential prompt injection detected",
				Pattern:     pattern.String(),
				Position:    loc[0],
			}
		}
	}

	// Check for excessive delimiters (often used in injections)
	if count := strings.Count(content, "---"); count > 3 {
		return &DetectionResult{
			IsInjection: true,
			Reason:      "Excessive delimiter usage detected",
			Pattern:     "multiple '---' sequences",
			Position:    -1,
		}
	}

	if count := strings.Count(content, "###"); count > 3 {
		return &DetectionResult{
			IsInjection: true,
			Reason:      "Excessive delimiter usage detected",
			Pattern:     "multiple '###' sequences",
			Position:    -1,
		}
	}

	return &DetectionResult{
		IsInjection: false,
	}
}

// Sanitize removes or escapes potentially dangerous patterns from content
// This is a defense-in-depth measure - detection should be primary defense
func (d *InjectionDetector) Sanitize(content string) string {
	// Replace common delimiter patterns that could be used for injection
	content = strings.ReplaceAll(content, "---", "___")
	content = strings.ReplaceAll(content, "###", "___")
	content = strings.ReplaceAll(content, "===", "___")

	// Remove role markers
	rolePatterns := []string{
		"SYSTEM:", "USER:", "ASSISTANT:",
		"System:", "User:", "Assistant:",
	}

	for _, pattern := range rolePatterns {
		content = strings.ReplaceAll(content, pattern, "["+pattern+"]")
	}

	return content
}

// ValidateContent performs comprehensive validation and returns sanitized content
func ValidateContent(content string, maxLength int) (string, error) {
	// Trim whitespace
	content = strings.TrimSpace(content)

	// Check if empty
	if content == "" {
		return "", fmt.Errorf("content cannot be empty")
	}

	// Check length
	if maxLength > 0 && len(content) > maxLength {
		return "", fmt.Errorf("content exceeds maximum length of %d characters", maxLength)
	}

	// Detect injection attempts
	detector := NewInjectionDetector()
	result := detector.Detect(content)

	if result.IsInjection {
		return "", fmt.Errorf("invalid content: %s (pattern: %s)", result.Reason, result.Pattern)
	}

	return content, nil
}

// SanitizeForPrompt sanitizes content before embedding in LLM prompts
// Use this as a defense-in-depth measure alongside detection
func SanitizeForPrompt(content string) string {
	detector := NewInjectionDetector()
	return detector.Sanitize(content)
}
