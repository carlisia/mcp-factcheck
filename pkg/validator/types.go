package validator

import "encoding/json"

// ValidationResult represents a structured validation response
type ValidationResult struct {
	IsValid      bool     `json:"is_valid"`
	Confidence   float64  `json:"confidence"`
	Issues       []string `json:"issues,omitempty"`
	Suggestions  []string `json:"suggestions,omitempty"`
	CorrectedVersion string `json:"corrected_version,omitempty"`
	SpecVersion  string   `json:"spec_version"`
}

// ValidationMatch represents a summarized spec match
type ValidationMatch struct {
	Topic      string  `json:"topic"`
	Relevance  float64 `json:"relevance"`
	Summary    string  `json:"summary"`
}

// SummarizeMatches creates concise summaries from search results
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

// FormatValidationResult creates a concise response for the LLM
func FormatValidationResult(result ValidationResult, matches []ValidationMatch) string {
	response := map[string]interface{}{
		"validation": result,
		"references": matches,
	}
	
	jsonBytes, _ := json.MarshalIndent(response, "", "  ")
	return string(jsonBytes)
}