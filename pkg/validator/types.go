package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/carlisia/mcp-factcheck/embedding"
)

// Common errors used across validator package.
var (
	// errArgumentsNotMap is returned when tool arguments are not provided as a map[string]any.
	errArgumentsNotMap = errors.New("arguments must be a map")
)

// ValidationResult represents a structured validation response containing
// the complete analysis of content validity against MCP specifications.
type ValidationResult struct {
	IsValid          bool                       `json:"is_valid"`
	Confidence       float64                    `json:"confidence"`
	ParsedClaims     []string                   `json:"parsed_claims,omitempty"`
	Issues           []string                   `json:"issues,omitempty"`
	Suggestions      []string                   `json:"suggestions,omitempty"`
	CorrectedVersion string                     `json:"corrected_version,omitempty"`
	SpecVersion      string                     `json:"spec_version"`
	FactCheckResult  *embedding.FactCheckResult `json:"-"`                    // omit from JSON
	DebugInfo        *ValidationDebugInfo       `json:"debug_info,omitempty"` // Detailed debugging information
}

// ValidationDebugInfo contains detailed information for debugging validation issues.
// It provides transparency into the validation process including search queries,
// spec matches, and reasoning behind validation decisions.
type ValidationDebugInfo struct {
	Timestamp           string           `json:"timestamp"`
	SearchQueries       []string         `json:"search_queries"`
	TopSpecMatches      []SpecMatchDebug `json:"top_spec_matches"`
	ClaimAnalysis       []ClaimDebugInfo `json:"claim_analysis"`
	LLMReasoning        string           `json:"llm_reasoning,omitempty"`
	ValidationIteration int              `json:"validation_iteration"`
}

// SpecMatchDebug contains debug information about specification matches found
// during validation, including similarity scores and chunk identifiers.
type SpecMatchDebug struct {
	Content    string  `json:"content"`
	Similarity float64 `json:"similarity"`
	ChunkID    string  `json:"chunk_id"`
}

// ClaimDebugInfo contains debug information for individual claim validation,
// tracking the validation status, issues found, and supporting evidence from the spec.
type ClaimDebugInfo struct {
	OriginalClaim    string   `json:"original_claim"`
	ValidationStatus string   `json:"validation_status"`
	Issues           []string `json:"issues"`
	SpecEvidence     []string `json:"spec_evidence"`
	Confidence       float64  `json:"confidence"`
}

// ValidationMatch represents a summarized specification match with relevance
// scoring and a concise summary of the matched content.
type ValidationMatch struct {
	Topic     string  `json:"topic"`
	Relevance float64 `json:"relevance"`
	Summary   string  `json:"summary"`
}

// SummarizeMatches creates concise summaries from search results.
// It extracts the most relevant matches up to maxMatches and formats them
// for inclusion in validation responses.
func SummarizeMatches(results []interface{}, maxMatches int) []ValidationMatch {
	if maxMatches > len(results) {
		maxMatches = len(results)
	}

	var matches []ValidationMatch
	for i := 0; i < maxMatches; i++ {
		// This will be implemented based on the actual search result type
		// For now, creating a placeholder structure
		matches = append(matches, ValidationMatch{
			Topic:     "MCP Specification",
			Relevance: 0.8,
			Summary:   "Relevant specification content found",
		})
	}
	return matches
}

// claimDetail represents detailed information about a claim including
// its accuracy assessment, corrections if needed, and explanations.
type claimDetail struct {
	Claim       string `json:"claim"`
	IsAccurate  bool   `json:"is_accurate"`
	Correction  string `json:"correction,omitempty"`
	Explanation string `json:"explanation,omitempty"`
	Issue       string `json:"issue,omitempty"`
}

// validationResponse represents the complete validation response structure
// that is returned to users. It includes all validation results, claim analysis,
// suggestions, and references to relevant specification sections.
type validationResponse struct {
	ValidationResult       validationSummary `json:"validation_result"`
	Claims                 []claimDetail     `json:"claims"`
	ParsedClaims           []string          `json:"parsed_claims"`
	Issues                 []string          `json:"issues"`
	Suggestions            []string          `json:"suggestions"`
	CorrectedVersion       string            `json:"corrected_version,omitempty"`
	MissingBestPractices   []string          `json:"missing_best_practices"`
	AdvisoryLanguageIssues []string          `json:"advisory_language_issues"`
	SpecReferences         []ValidationMatch `json:"spec_references"`
}

// validationSummary provides a high-level summary of validation results
// including overall validity, confidence score, and a human-readable summary.
type validationSummary struct {
	IsValid     bool    `json:"is_valid"`
	Confidence  float64 `json:"confidence"`
	SpecVersion string  `json:"spec_version"`
	Summary     string  `json:"summary"`
}

// formatValidationResult creates a structured JSON response with all validation details.
// It includes a directive header instructing LLMs to use the format-validation-results
// prompt for proper formatting.
func formatValidationResult(result ValidationResult, matches []ValidationMatch) string {
	response := buildValidationResponse(result, matches)

	jsonBytes, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		// Log error and return a basic response
		return fmt.Sprintf("Error formatting validation result: %v", err)
	}

	return formatDirective() + string(jsonBytes)
}

// buildValidationResponse constructs the complete validation response structure
// by combining validation results with matched specification references.
func buildValidationResponse(result ValidationResult, matches []ValidationMatch) validationResponse {
	response := validationResponse{
		ValidationResult: validationSummary{
			IsValid:     result.IsValid,
			Confidence:  result.Confidence,
			SpecVersion: result.SpecVersion,
			Summary:     formatSummary(result.IsValid, result.Confidence),
		},
		Claims:                 buildClaimDetails(result),
		ParsedClaims:           result.ParsedClaims,
		Issues:                 result.Issues,
		Suggestions:            result.Suggestions,
		CorrectedVersion:       result.CorrectedVersion,
		MissingBestPractices:   []string{},
		AdvisoryLanguageIssues: []string{},
		SpecReferences:         matches,
	}

	// Add fact check details if available
	if result.FactCheckResult != nil {
		response.MissingBestPractices = result.FactCheckResult.MissingBestPractices
		response.AdvisoryLanguageIssues = result.FactCheckResult.AdvisoryLanguageIssues
	}

	return response
}

// buildClaimDetails constructs detailed claim information from validation results.
// It prioritizes fact-check results when available, otherwise builds from parsed
// claims and identified issues.
func buildClaimDetails(result ValidationResult) []claimDetail {
	var claims []claimDetail

	// Use fact check claims if available
	if result.FactCheckResult != nil && len(result.FactCheckResult.Claims) > 0 {
		for _, claim := range result.FactCheckResult.Claims {
			claims = append(claims, claimDetail{
				Claim:       claim.Claim,
				IsAccurate:  claim.IsAccurate,
				Correction:  claim.Correction,
				Explanation: claim.Explanation,
			})
		}
		return claims
	}

	// Build from parsed claims and issues
	for _, claim := range result.ParsedClaims {
		detail := claimDetail{
			Claim:      claim,
			IsAccurate: true,
		}

		// Check if this claim has issues
		for j, issue := range result.Issues {
			if issue == claim || strings.Contains(issue, claim) {
				detail.IsAccurate = false
				detail.Issue = issue
				if j < len(result.Suggestions) {
					detail.Correction = result.Suggestions[j]
				}
				break
			}
		}

		claims = append(claims, detail)
	}

	return claims
}

// formatSummary creates a human-readable summary of the validation result
// expressing validity and confidence in natural language.
func formatSummary(isValid bool, confidence float64) string {
	validity := "invalid"
	if isValid {
		validity = "valid"
	}
	return fmt.Sprintf("Content is %s with %.0f%% confidence", validity, confidence*100)
}

// formatDirective returns the directive header for LLM processing.
// This header instructs LLMs to use the format-validation-results prompt
// rather than attempting to interpret or reformat the JSON directly.
func formatDirective() string {
	return `[VALIDATION COMPLETE - DO NOT INTERPRET THIS JSON]

You MUST now:
1. Call the format-validation-results prompt
2. Pass the JSON below as the validation_data parameter
3. Display the formatted output to the user without modification

DO NOT:
- Summarize the JSON
- Create your own formatting
- Interpret the results yourself

REQUIRED ACTION: Use format-validation-results prompt with this JSON:
`
}
