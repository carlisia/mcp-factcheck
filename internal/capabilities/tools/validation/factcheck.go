package validation

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/carlisia/mcp-factcheck/internal/capabilities/rules"
	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools"
	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools/contentprep"
)

const (
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
	// Use more spec sections for compound claims to ensure all subclaims can be validated
	maxSections := 10
	if strings.Contains(content, ";") || strings.Contains(content, ",") {
		// Compound claims need more context
		maxSections = min(15, len(specMatches))
	}

	for i, match := range specMatches {
		if i >= maxSections {
			break
		}
		specContext.WriteString(fmt.Sprintf("--- Section %d (Similarity: %.2f) ---\n%s\n\n",
			i+1, match.Similarity, match.Content))
	}

	// Pre-analyze compound claims if content contains "and"
	compoundEvidence := make(map[string]string)
	if strings.Contains(strings.ToLower(content), " and ") {
		// Extract potential compound claims
		var claimsToCheck []string
		if strings.Contains(content, ". ") {
			claimsToCheck = strings.Split(content, ". ")
		} else {
			claimsToCheck = []string{content}
		}

		for _, claim := range claimsToCheck {
			claim = strings.TrimSpace(claim)
			if strings.Contains(strings.ToLower(claim), " and ") {
				compound := contentprep.Decompose(claim)
				if compound.IsCompound {
					// Build evidence for this compound claim from existing spec matches
					evidence := buildCompoundEvidence(compound, specMatches)
					if evidence != "" {
						compoundEvidence[compound.OriginalClaim] = evidence
					}
				}
			}
		}
	}

	// Prepare template data with all rule constants
	data := struct {
		// Content
		Content          string
		SpecContext      string
		CompoundEvidence map[string]string

		// Headers
		Header                      string
		SystemPrompt                string
		ExtractionRulesHeader       string
		FactCheckingHeader          string
		CompoundEvidenceHeader      string
		UserContentHeader           string
		SpecificationSectionsHeader string
		ResponseFormatHeader        string

		// Rules
		ClaimExtractionRules         string
		AccuracyCheckingRules        string
		SpecificationGuidanceNote    string
		CompoundClaimInstructions    string
		CompoundEvidenceInstructions string
		CompoundClaimEvaluation      string

		// Protocol vs Implementation
		ProtocolVsImplementation      string
		ImplementationRecommendations string
		KeyDistinction                string
		CaseSensitivityNote           string

		// Equivalence Rules
		ParaphrasingRules     string
		ConceptualEquivalence string
		SemanticUnderstanding string

		// Context Understanding
		ImportantContextUnderstanding   string
		CriticalExposureRule            string
		InitializationFlowUnderstanding string
		PatternRecognition              string

		// Explanation Requirements
		InaccuracyExplanation     string
		CriticalSearchRequirement string

		// Response Format
		CommonResponseRequirements string
		ResponseFormat             string
	}{
		// Content
		Content:          content,
		SpecContext:      specContext.String(),
		CompoundEvidence: compoundEvidence,

		// Headers
		Header:                      rules.ClaimCheckHeader,
		SystemPrompt:                rules.FactCheckSystem,
		ExtractionRulesHeader:       rules.ExtractionRulesHeader,
		FactCheckingHeader:          rules.FactCheckingRulesHeader,
		CompoundEvidenceHeader:      rules.CompoundEvidenceHeader,
		UserContentHeader:           rules.UserContentHeader,
		SpecificationSectionsHeader: rules.SpecificationSectionsHeader,
		ResponseFormatHeader:        rules.ResponseFormatHeader,

		// Rules
		ClaimExtractionRules:         rules.ClaimExtraction,
		AccuracyCheckingRules:        rules.AccuracyChecking,
		SpecificationGuidanceNote:    rules.SpecificationGuidance,
		CompoundClaimInstructions:    rules.CompoundClaimInstructions,
		CompoundEvidenceInstructions: rules.CompoundEvidenceInstructions,
		CompoundClaimEvaluation:      rules.CompoundClaimEvaluation,

		// Protocol vs Implementation
		ProtocolVsImplementation:      rules.ProtocolVsImplementation,
		ImplementationRecommendations: rules.ImplementationRecommendations,
		KeyDistinction:                rules.KeyDistinction,
		CaseSensitivityNote:           rules.CaseSensitivityNote,

		// Equivalence Rules
		ParaphrasingRules:     rules.ParaphrasingRules,
		ConceptualEquivalence: rules.ConceptualEquivalence,
		SemanticUnderstanding: rules.SemanticUnderstanding,

		// Context Understanding
		ImportantContextUnderstanding:   rules.ImportantContextUnderstanding,
		CriticalExposureRule:            rules.CriticalExposureRule,
		InitializationFlowUnderstanding: rules.InitializationFlowUnderstanding,
		PatternRecognition:              rules.PatternRecognition,

		// Explanation Requirements
		InaccuracyExplanation:     rules.InaccuracyExplanation,
		CriticalSearchRequirement: rules.CriticalSearchRequirement,

		// Response Format
		CommonResponseRequirements: rules.CommonResponseRequirements,
		ResponseFormat:             rules.ClaimCheckResponseFormat,
	}

	// Execute template
	var prompt bytes.Buffer
	if err := claimCheckTemplate.Execute(&prompt, data); err != nil {
		return nil, fmt.Errorf("failed to execute fact-check template: %w", err)
	}

	// Call LLM
	response, err := llmFunc(ctx, prompt.String(), FactCheckTemperature, FactCheckMaxTokens)
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
		Claim                      string
		SpecContext                string
		SystemPrompt               string
		ClaimHeader                string
		SpecHeader                 string
		ValidationHeader           string
		ValidationRules            string
		CommonResponseRequirements string
		ResponseFormat             string
	}{
		Claim:                      claim,
		SpecContext:                specContext.String(),
		SystemPrompt:               rules.QuickClaimSystemPrompt,
		ClaimHeader:                rules.QuickClaimHeader,
		SpecHeader:                 rules.QuickClaimSpecHeader,
		ValidationHeader:           rules.QuickClaimValidationHeader,
		ValidationRules:            rules.QuickClaimValidation,
		CommonResponseRequirements: rules.CommonResponseRequirements,
		ResponseFormat:             rules.QuickClaimResponseFormat,
	}

	// Execute template
	var prompt bytes.Buffer
	if err := quickFactTemplate.Execute(&prompt, data); err != nil {
		return nil, fmt.Errorf("failed to execute quick-fact template: %w", err)
	}

	// Call LLM
	response, err := llmFunc(ctx, prompt.String(), FactCheckTemperature, 1000)
	if err != nil {
		return nil, fmt.Errorf("LLM quick fact-check failed: %w", err)
	}

	// Parse response
	return parseQuickFactResponse(response)
}

// parseFactCheckResponse parses the LLM response into a FactCheckResult
func parseFactCheckResponse(response string) (*FactCheckResult, error) {
	// Define the expected JSON structure from the template
	type templateResponse struct {
		Claims                 []Claim  `json:"claims"`
		MissingBestPractices   []string `json:"missing_best_practices"`
		AdvisoryLanguageIssues []string `json:"advisory_language_issues"`
		OverallIsAccurate      bool     `json:"overall_is_accurate"`
		Summary                string   `json:"summary"`
	}

	// Try to parse as JSON
	var templateResp templateResponse
	if err := json.Unmarshal([]byte(response), &templateResp); err == nil {
		// Convert to FactCheckResult
		result := &FactCheckResult{
			IsAccurate:             templateResp.OverallIsAccurate,
			Claims:                 templateResp.Claims,
			MissingBestPractices:   templateResp.MissingBestPractices,
			AdvisoryLanguageIssues: templateResp.AdvisoryLanguageIssues,
			Explanation:            templateResp.Summary,
			RawResponse:            response,
		}

		// Calculate confidence based on claims
		if len(templateResp.Claims) > 0 {
			var totalConfidence float64
			for _, claim := range templateResp.Claims {
				totalConfidence += claim.Confidence
			}
			result.Confidence = totalConfidence / float64(len(templateResp.Claims))
		} else {
			result.Confidence = 0.9
		}

		// Extract parsed claims, inaccuracies, and corrections
		for _, claim := range templateResp.Claims {
			result.ParsedClaims = append(result.ParsedClaims, claim.Claim)
			if !claim.IsAccurate {
				result.Inaccuracies = append(result.Inaccuracies, claim.Explanation)
				if claim.Correction != "" {
					result.Corrections = append(result.Corrections, claim.Correction)
				}
			}
		}

		return result, nil
	}

	// If JSON parsing fails, try to extract structured information
	// This is a fallback for when the LLM doesn't return valid JSON
	result := &FactCheckResult{
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

	result.Explanation = response
	return result, nil
}

// parseQuickFactResponse parses the quick fact check response
func parseQuickFactResponse(response string) (*FactCheckResult, error) {
	result := &FactCheckResult{
		RawResponse: response,
	}

	// Look for verdict markers
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "✗") || strings.HasPrefix(response, "✗ INACCURATE") {
		result.IsAccurate = false
		result.Confidence = 0.9
	} else if strings.HasPrefix(response, "✓") || strings.HasPrefix(response, "✓ ACCURATE") {
		result.IsAccurate = true
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

// buildCompoundEvidence builds evidence string for a compound claim from spec matches
func buildCompoundEvidence(compound contentprep.Compound, specMatches []tools.SearchResult) string {
	if !compound.IsCompound {
		return ""
	}

	var output []string
	output = append(output, fmt.Sprintf("Compound Claim: %s", compound.OriginalClaim))

	allSubClaimsHaveEvidence := true

	for i, subClaim := range compound.SubClaims {
		output = append(output, fmt.Sprintf("\nSubclaim %d: %s", i+1, subClaim.Text))

		// Check if we have evidence for this subclaim in the spec matches
		hasEvidence := false
		var evidenceQuotes []string

		// Look for evidence in spec matches
		for _, match := range specMatches {
			// Check if the spec content relates to this subclaim
			if containsEvidence(match.Content, subClaim.Text) {
				hasEvidence = true
				// Extract relevant quote
				quote := contentprep.ExtractQuote(match.Content, subClaim.Text)
				if quote != "" {
					evidenceQuotes = append(evidenceQuotes, quote)
					if len(evidenceQuotes) >= 2 { // Limit to 2 quotes
						break
					}
				}
			}
		}

		if hasEvidence {
			output = append(output, "   ✓ Evidence found:")
			for _, quote := range evidenceQuotes {
				output = append(output, fmt.Sprintf("   - \"%s\"", quote))
			}
		} else {
			output = append(output, "   ✗ No clear evidence found in spec")
			allSubClaimsHaveEvidence = false
		}
	}

	// Overall conclusion
	output = append(output, "\nConclusion: ")
	if allSubClaimsHaveEvidence {
		output = append(output, "Compound claim is supported by evidence for all parts.")
	} else {
		output = append(output, "Compound claim is only partially supported. Some subclaims lack evidence.")
	}

	return strings.Join(output, "\n")
}

// containsEvidence checks if spec content contains evidence for a claim
func containsEvidence(specContent, claim string) bool {
	specLower := strings.ToLower(specContent)
	claimLower := strings.ToLower(claim)

	// Extract key terms from the claim
	keyTerms := extractKeyTerms(claimLower)

	// Check if at least 2 key terms appear in the spec content
	matchCount := 0
	for _, term := range keyTerms {
		if strings.Contains(specLower, term) {
			matchCount++
		}
	}

	return matchCount >= 2
}

// extractKeyTerms extracts important terms from a claim
func extractKeyTerms(claim string) []string {
	// Remove common words and extract key terms
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"is": true, "are": true, "was": true, "were": true,
		"has": true, "have": true, "had": true, "can": true,
		"should": true, "must": true, "may": true, "will": true,
		"to": true, "of": true, "in": true, "for": true, "with": true,
	}

	words := strings.Fields(claim)
	var terms []string

	for _, word := range words {
		word = strings.ToLower(strings.Trim(word, ".,!?;:"))
		if len(word) > 2 && !stopWords[word] {
			terms = append(terms, word)
		}
	}

	return terms
}
