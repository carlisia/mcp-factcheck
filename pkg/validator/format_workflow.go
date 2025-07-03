package validator

import (
	"fmt"
	"strings"
)

// Constants for formatting
const (
	headerTitle      = "## MCP Content Validation Report\n\n"
	statusAccurate   = "✓ Accurate"
	statusInaccurate = "✗ Inaccurate"
)

// WorkflowFormatter handles formatting of validation results into user-friendly reports
type WorkflowFormatter struct {
	result  ValidationResult
	content string
	sb      strings.Builder
}

// NewWorkflowFormatter creates a new formatter instance
func NewWorkflowFormatter(result ValidationResult, content string) *WorkflowFormatter {
	return &WorkflowFormatter{
		result:  result,
		content: content,
	}
}

// FormatValidationWorkflow formats validation results into a step-by-step workflow
func FormatValidationWorkflow(result ValidationResult, content string) string {
	formatter := NewWorkflowFormatter(result, content)
	return formatter.Format()
}

// Format generates the complete validation workflow report
func (f *WorkflowFormatter) Format() string {
	f.writeHeader()
	f.writeClaimExtraction()
	f.writeValidationResults()
	f.writeBestPractices()
	f.writeLanguageIssues()
	f.writeSummary()

	if !f.result.IsValid {
		f.writeCorrectedContent()
		f.writeNextSteps()
	}

	return f.sb.String()
}

func (f *WorkflowFormatter) writeHeader() {
	f.sb.WriteString(headerTitle)
}

func (f *WorkflowFormatter) writeClaimExtraction() {
	f.sb.WriteString("## Step 1: Claim Extraction\n\n")
	f.sb.WriteString("I've extracted the following claims from your content:\n")

	claims := f.extractClaims()
	if len(claims) == 0 {
		// If no claims extracted, but we have content, treat the whole content as a claim
		if f.content != "" {
			fmt.Fprintf(&f.sb, "1. %s\n", strings.TrimSpace(f.content))
		} else {
			f.sb.WriteString("No claims found in content.\n")
		}
	} else {
		for i, claim := range claims {
			fmt.Fprintf(&f.sb, "%d. %s\n", i+1, claim)
		}
	}
}

func (f *WorkflowFormatter) extractClaims() []string {
	// Use detailed claims if available
	if f.result.FactCheckResult != nil && len(f.result.FactCheckResult.Claims) > 0 {
		claims := make([]string, len(f.result.FactCheckResult.Claims))
		for i, claim := range f.result.FactCheckResult.Claims {
			claims[i] = claim.Claim
		}
		return claims
	}

	// Use parsed claims if available
	if len(f.result.ParsedClaims) > 0 {
		return f.result.ParsedClaims
	}

	// Extract from content
	return f.extractClaimsFromContent()
}

func (f *WorkflowFormatter) extractClaimsFromContent() []string {
	if f.content == "" {
		return nil
	}

	var claims []string
	bulletPrefixes := []string{"-", "*", "•"}

	for _, line := range strings.Split(f.content, "\n") {
		line = strings.TrimSpace(line)
		for _, prefix := range bulletPrefixes {
			if strings.HasPrefix(line, prefix) {
				claim := strings.TrimSpace(strings.TrimPrefix(line, prefix))
				if claim != "" {
					claims = append(claims, claim)
				}
				break
			}
		}
	}

	// If no bullet points found but content exists, treat sentences as claims
	if len(claims) == 0 && f.content != "" {
		// Split by sentence-ending punctuation
		sentences := strings.FieldsFunc(f.content, func(r rune) bool {
			return r == '.' || r == ';'
		})
		for _, sentence := range sentences {
			sentence = strings.TrimSpace(sentence)
			if sentence != "" {
				claims = append(claims, sentence)
			}
		}
	}

	return claims
}

func (f *WorkflowFormatter) writeValidationResults() {
	f.sb.WriteString("\n## Step 2: Validation Results\n\n")

	if f.result.FactCheckResult != nil && len(f.result.FactCheckResult.Claims) > 0 {
		f.writeDetailedClaimResults()
	} else {
		f.writeBasicValidationResults()
	}
}

func (f *WorkflowFormatter) writeDetailedClaimResults() {
	for _, claim := range f.result.FactCheckResult.Claims {
		fmt.Fprintf(&f.sb, "### Claim: %q\n", claim.Claim)

		if claim.IsAccurate {
			fmt.Fprintf(&f.sb, "**Status:** %s\n", statusAccurate)
		} else {
			fmt.Fprintf(&f.sb, "**Status:** %s\n", statusInaccurate)
			if claim.Correction != "" {
				fmt.Fprintf(&f.sb, "**Issue:** %s\n", claim.Correction)
				fmt.Fprintf(&f.sb, "**Correction:** %s\n", claim.Correction)
			}
		}

		if claim.Explanation != "" {
			fmt.Fprintf(&f.sb, "**Spec Reference:** %s\n", claim.Explanation)
		}
		f.sb.WriteString("\n")
	}
}

func (f *WorkflowFormatter) writeBasicValidationResults() {
	for i, issue := range f.result.Issues {
		fmt.Fprintf(&f.sb, "### Issue %d: %s\n", i+1, issue)
		fmt.Fprintf(&f.sb, "**Status:** %s\n", statusInaccurate)

		if i < len(f.result.Suggestions) {
			fmt.Fprintf(&f.sb, "**Correction:** %s\n", f.result.Suggestions[i])
		}
		f.sb.WriteString("\n")
	}
}

func (f *WorkflowFormatter) writeBestPractices() {
	if f.result.FactCheckResult == nil || len(f.result.FactCheckResult.MissingBestPractices) == 0 {
		return
	}

	f.sb.WriteString("## Step 3: Missing Best Practices\n\n")
	f.sb.WriteString("The following SHOULD requirements from the MCP specification are not addressed in your content:\n")

	for _, bp := range f.result.FactCheckResult.MissingBestPractices {
		fmt.Fprintf(&f.sb, "- %s\n", bp)
	}
	f.sb.WriteString("\n")
}

func (f *WorkflowFormatter) writeLanguageIssues() {
	if f.result.FactCheckResult == nil || len(f.result.FactCheckResult.AdvisoryLanguageIssues) == 0 {
		return
	}

	f.sb.WriteString("## Step 4: Modal Verb Issues\n\n")
	f.sb.WriteString("These issues relate to incorrect usage of requirement levels (MUST/SHOULD/MAY):\n")

	for _, issue := range f.result.FactCheckResult.AdvisoryLanguageIssues {
		fmt.Fprintf(&f.sb, "- %s\n", issue)
	}
	f.sb.WriteString("\n")
}

func (f *WorkflowFormatter) writeSummary() {
	f.sb.WriteString("## Summary\n\n")

	accuracyStatus := "Content contains inaccuracies"
	if f.result.IsValid {
		accuracyStatus = "Content is accurate"
	}

	fmt.Fprintf(&f.sb, "**Overall Accuracy:** %s\n", accuracyStatus)
	fmt.Fprintf(&f.sb, "**Confidence Score:** %.0f%%\n", f.result.Confidence*100)
	fmt.Fprintf(&f.sb, "**Spec Version:** %s\n\n", f.result.SpecVersion)
}

func (f *WorkflowFormatter) writeCorrectedContent() {
	f.sb.WriteString("## Corrected Content\n\n")
	f.sb.WriteString("Based on the validation findings, here's the corrected version of your content:\n\n")

	if f.result.CorrectedVersion != "" {
		f.sb.WriteString(f.result.CorrectedVersion)
	} else {
		claims := f.extractClaims()
		f.sb.WriteString(f.generateCorrectedContent(claims))
	}
}

func (f *WorkflowFormatter) writeNextSteps() {
	f.sb.WriteString("\n## Next Steps\n\n")
	f.sb.WriteString("Would you like me to:\n")
	f.sb.WriteString("1. Validate the corrected content to ensure all issues are resolved?\n")
	f.sb.WriteString("2. Provide more details about specific corrections?\n")
	f.sb.WriteString("3. Help you understand the specification requirements better?\n\n")
	f.sb.WriteString("Please let me know which step you'd like to proceed with.\n")
}

// generateCorrectedContent creates a corrected version of the content
func (f *WorkflowFormatter) generateCorrectedContent(claims []string) string {
	var sb strings.Builder

	sb.WriteString("**MCP Server Design** (based on the specification):\n\n")

	// Add accurate claims and known good patterns
	hasAccurateClaims := false
	if f.result.FactCheckResult != nil && len(f.result.FactCheckResult.Claims) > 0 {
		for _, claim := range f.result.FactCheckResult.Claims {
			if claim.IsAccurate {
				fmt.Fprintf(&sb, "- %s\n", claim.Claim)
				hasAccurateClaims = true
			}
		}
	}

	// If no accurate claims, provide what we know about MCP servers
	if !hasAccurateClaims {
		sb.WriteString("The MCP specification defines that servers:\n")
		sb.WriteString("- Expose resources, tools, and prompts via MCP primitives\n")
		sb.WriteString("- Participate in capability negotiation during initialization\n")
		sb.WriteString("- Respond to client requests within the protocol framework\n")
		sb.WriteString("- Can optionally support progress notifications\n")
		sb.WriteString("- Define their capabilities through the protocol handshake\n")
	}

	// Add missing best practices
	if f.result.FactCheckResult != nil && len(f.result.FactCheckResult.MissingBestPractices) > 0 {
		sb.WriteString("\n**Additional Best Practices (from MCP specification):**\n\n")
		for _, bp := range f.result.FactCheckResult.MissingBestPractices {
			fmt.Fprintf(&sb, "- %s\n", bp)
		}
	}

	sb.WriteString("\n**Note:** This corrected version aligns with the MCP specification by:\n")
	sb.WriteString("- Using accurate terminology and claims\n")
	sb.WriteString("- Distinguishing between what MCP provides vs what implementations should do\n")
	sb.WriteString("- Including relevant best practices and recommendations\n")

	return sb.String()
}
