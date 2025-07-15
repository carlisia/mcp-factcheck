package validation

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	
	"github.com/carlisia/mcp-factcheck/internal/tools"
)

const (
	FactCheckModel       = "gpt-4o-mini"
	FactCheckTemperature = 0.0
	FactCheckMaxTokens   = 2500
)

// Templates for fact checking
var (
	claimCheckTemplate *template.Template
	quickFactTemplate  *template.Template

	//go:embed templates/claim-check.tmpl
	claimCheckTemplateContent string

	//go:embed templates/quick-claim-check.tmpl
	quickClaimCheckTemplateContent string
)

func init() {
	// Parse templates at initialization
	var err error
	claimCheckTemplate, err = template.New("claim-check").Parse(claimCheckTemplateContent)
	if err != nil {
		panic(fmt.Sprintf("failed to parse fact-check template: %v", err))
	}

	quickFactTemplate, err = template.New("quick-claim-check").Parse(quickClaimCheckTemplateContent)
	if err != nil {
		panic(fmt.Sprintf("failed to parse quick-fact template: %v", err))
	}
}

// performClaimCheck performs fact-checking against spec matches
func performClaimCheck(ctx context.Context, llmFunc LLMCompleteFunc, content string, specMatches []tools.SearchResult) (*FactCheckResult, error) {
	// Build the context from spec matches
	var specContext strings.Builder
	for i, match := range specMatches {
		if i >= 5 { // Limit to top 5 matches for context
			break
		}
		specContext.WriteString(fmt.Sprintf("--- Section %d (Similarity: %.2f) ---\n%s\n\n",
			i+1, match.Similarity, match.Content))
	}

	// Prepare template data
	data := struct {
		Content     string
		SpecContext string
	}{
		Content:     content,
		SpecContext: specContext.String(),
	}

	// Execute template
	var prompt bytes.Buffer
	if err := claimCheckTemplate.Execute(&prompt, data); err != nil {
		return nil, fmt.Errorf("failed to execute fact-check template: %w", err)
	}

	// Call LLM
	response, err := llmFunc(ctx, FactCheckModel, prompt.String(), FactCheckTemperature, FactCheckMaxTokens)
	if err != nil {
		return nil, fmt.Errorf("LLM fact-check failed: %w", err)
	}

	// Parse response
	result, err := parseFactCheckResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse fact-check response: %w", err)
	}

	result.RawResponse = response
	return result, nil
}

// performQuickClaimCheck performs quick fact-checking for a single claim
func performQuickClaimCheck(ctx context.Context, llmFunc LLMCompleteFunc, claim string, specMatches []tools.SearchResult) (*FactCheckResult, error) {
	// Build context from spec matches
	var specContext strings.Builder
	for i, match := range specMatches {
		if i >= 8 { // Use more matches for quick fact
			break
		}
		// Truncate long sections
		content := match.Content
		if len(content) > maxSectionLength {
			content = content[:maxSectionLength] + "..."
		}
		specContext.WriteString(fmt.Sprintf("[Match %d - %.2f similarity]\n%s\n\n",
			i+1, match.Similarity, content))
	}

	// Prepare template data
	data := struct {
		Claim       string
		SpecContext string
	}{
		Claim:       claim,
		SpecContext: specContext.String(),
	}

	// Execute template
	var prompt bytes.Buffer
	if err := quickFactTemplate.Execute(&prompt, data); err != nil {
		return nil, fmt.Errorf("failed to execute quick-fact template: %w", err)
	}

	// Call LLM
	response, err := llmFunc(ctx, FactCheckModel, prompt.String(), FactCheckTemperature, 1000)
	if err != nil {
		return nil, fmt.Errorf("LLM quick fact-check failed: %w", err)
	}

	// Parse response
	return parseQuickFactResponse(response)
}

// parseFactCheckResponse parses the LLM response into a FactCheckResult
func parseFactCheckResponse(response string) (*FactCheckResult, error) {
	// Try to parse as JSON first
	var result FactCheckResult
	if err := json.Unmarshal([]byte(response), &result); err == nil {
		return &result, nil
	}

	// If JSON parsing fails, try to extract structured information
	// This is a fallback for when the LLM doesn't return valid JSON
	result = FactCheckResult{
		RawResponse: response,
	}

	// Extract verdict
	if strings.Contains(strings.ToLower(response), "accurate") &&
		!strings.Contains(strings.ToLower(response), "inaccurate") {
		result.IsAccurate = true
		result.Confidence = 0.8
	} else {
		result.IsAccurate = false
		result.Confidence = 0.8
	}

	// TODO: Implement more sophisticated parsing of non-JSON responses
	result.Explanation = response

	return &result, nil
}

// parseQuickFactResponse parses the quick fact check response
func parseQuickFactResponse(response string) (*FactCheckResult, error) {
	result := &FactCheckResult{
		RawResponse: response,
	}

	// Look for verdict markers
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "✓") || strings.Contains(response, "ACCURATE") {
		result.IsAccurate = true
		result.Confidence = 0.9
	} else if strings.HasPrefix(response, "✗") || strings.Contains(response, "INACCURATE") {
		result.IsAccurate = false
		result.Confidence = 0.9
	} else {
		// Default to uncertain
		result.IsAccurate = false
		result.Confidence = 0.5
	}

	// Extract explanation (everything after the verdict)
	lines := strings.Split(response, "\n")
	if len(lines) > 1 {
		result.Explanation = strings.Join(lines[1:], "\n")
	} else {
		result.Explanation = response
	}

	// Extract inaccuracies if mentioned
	if !result.IsAccurate && strings.Contains(response, "should") {
		// Simple extraction of correction
		if idx := strings.Index(response, "should"); idx > 0 {
			correction := strings.TrimSpace(response[idx:])
			result.Corrections = []string{correction}
		}
	}

	return result, nil
}
