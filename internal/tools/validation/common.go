// Package validation provides MCP validation tool definitions for the fact-check server.
package validation

import (
	"context"
	"fmt"
	"strings"
)

// Search and request related configurations
const (
	MaxContentLength = 8000

	// maxSectionLength is the maximum length of a spec section in search results
	maxSectionLength = 200
)

// Claim represents the validation result for a single MCP claim or statement.
// Used when breaking down complex content into individual verifiable claims.
type Claim struct {
	Claim       string  `json:"claim"`                 // The actual claim text
	IsAccurate  bool    `json:"is_accurate"`           // Whether this specific claim is accurate
	Explanation string  `json:"explanation,omitempty"` // Why the claim is accurate/inaccurate
	Correction  string  `json:"correction,omitempty"`  // Suggested correction if inaccurate
	Confidence  float64 `json:"confidence,omitempty"`  // Confidence for this specific claim
}

// FactCheckResult contains the detailed output from LLM-based fact checking.
// This is the internal representation with all analysis details.
type FactCheckResult struct {
	// Overall assessment
	IsAccurate bool    `json:"is_accurate"` // Overall accuracy (false if any claim is inaccurate)
	Confidence float64 `json:"confidence"`  // Overall confidence (may be average or minimum)

	// Detailed claim analysis
	Claims []Claim `json:"claims"` // Individual claim validations (when content is decomposed)

	// Aggregate findings
	ParsedClaims           []string `json:"parsed_claims"`            // List of claims extracted from content
	Inaccuracies           []string `json:"inaccuracies"`             // All inaccurate statements found
	Corrections            []string `json:"corrections"`              // Suggested corrections
	Suggestions            []string `json:"suggestions"`              // General improvement suggestions
	MissingBestPractices   []string `json:"missing_best_practices"`   // MCP best practices not followed
	AdvisoryLanguageIssues []string `json:"advisory_language_issues"` // Issues with SHOULD/MUST language

	// Formatted outputs
	Explanation      string `json:"explanation"`       // Overall explanation
	CorrectedVersion string `json:"corrected_version"` // Full corrected content
	RawResponse      string `json:"-"`                 // Raw LLM response (not serialized)
}

// Result represents the final validation outcome returned to MCP clients.
// It provides a simplified, flattened view of the validation results.
// The fields are populated from FactCheckResult for consistent external API.
type Result struct {
	// Core validation outcome
	IsValid     bool   `json:"is_valid"`     // Overall validation result
	SpecVersion string `json:"spec_version"` // MCP version used for validation

	// Flattened data from FactCheckResult (for backward compatibility)
	Confidence       float64  `json:"confidence"`                  // From FactCheckResult.Confidence
	ParsedClaims     []string `json:"parsed_claims,omitempty"`     // From FactCheckResult.ParsedClaims
	Issues           []string `json:"issues,omitempty"`            // From FactCheckResult.Inaccuracies
	Suggestions      []string `json:"suggestions,omitempty"`       // From FactCheckResult.Suggestions
	CorrectedVersion string   `json:"corrected_version,omitempty"` // From FactCheckResult.CorrectedVersion

	// Internal reference to full details (not serialized)
	FactCheckResult *FactCheckResult `json:"-"`
}

// LLMCompleteFunc performs LLM completion operations
type LLMCompleteFunc func(ctx context.Context, model string, prompt string, temperature float64, maxTokens int) (string, error)

// SearchResult represents spec search result with scoring and content
type SearchResult struct {
	Content  string  `json:"content"`
	Section  string  `json:"section"`
	Score    float64 `json:"score"`
	SpecPath string  `json:"spec_path"`
	Version  string  `json:"version"`
}

// SearchFunc performs embedding search
type SearchFunc func(version string, queryEmbedding []float64, topK int) ([]SearchResult, error)

// EmbeddingFunc generates embeddings for content
type EmbeddingFunc func(ctx context.Context, content string) ([]float64, error)

// ValidateContentLength validates content length and trims whitespace
func ValidateContentLength(content string, fieldName string, maxLength int) (string, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "", fmt.Errorf("%s is required", fieldName)
	}
	if len(trimmed) > maxLength {
		return "", fmt.Errorf("%s is too long (max %d characters)", fieldName, maxLength)
	}
	return trimmed, nil
}
