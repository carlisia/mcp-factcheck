package validation

import (
	"strings"
)

// ClaimType indicates whether a claim should be handled by quick or full validation
type ClaimType int

const (
	SingleClaim ClaimType = iota
	CompoundClaim
)

// ClaimClassification contains the result of classifying a claim
type ClaimClassification struct {
	Type       ClaimType
	ClaimCount int
	Indicators []string // What made us classify it as compound
	Suggestion string   // Suggestion for the user
}

// ClassifyClaim determines whether a claim should be handled by quick claim
// or full validation based on complexity indicators.
//
// Boundary rules for compound claims:
// 1. Contains semicolons (;) separating independent statements
// 2. Contains multiple "and" connecting independent claims
// 3. Contains comma-separated lists of capabilities/features
// 4. Contains 3+ distinct verifiable statements
func ClassifyClaim(claim string) ClaimClassification {
	result := ClaimClassification{
		Type:       SingleClaim,
		ClaimCount: 1,
		Indicators: []string{},
	}

	normalized := strings.ToLower(claim)

	// Check for semicolons - strong indicator of multiple independent claims
	if strings.Contains(claim, ";") {
		parts := strings.Split(claim, ";")
		result.ClaimCount = len(parts)
		result.Indicators = append(result.Indicators, "semicolon-separated claims")
	}

	// Check for multiple "and" conjunctions that connect independent claims
	andCount := strings.Count(normalized, " and ")
	if andCount >= 2 {
		result.Indicators = append(result.Indicators, "multiple 'and' conjunctions")
		if result.ClaimCount < andCount+1 {
			result.ClaimCount = andCount + 1
		}
	}

	// Check for "or" conjunctions that connect independent claims
	// Exception: negative claims about what MCP doesn't do (e.g., "MCP never X or Y")
	// are often better handled as single claims
	orCount := strings.Count(normalized, " or ")
	if orCount >= 1 && containsMultipleVerbs(normalized) {
		// Check if this is a negative claim about what MCP doesn't do
		isNegativeClaim := strings.Contains(normalized, "never") ||
			strings.Contains(normalized, "does not") ||
			strings.Contains(normalized, "doesn't") ||
			strings.Contains(normalized, "not enforce")

		if !isNegativeClaim {
			result.Indicators = append(result.Indicators, "'or' connecting independent claims")
			if result.ClaimCount < 2 {
				result.ClaimCount = 2
			}
		}
	}

	// Check for comma-separated lists of features/capabilities
	if containsFeatureList(claim) {
		result.Indicators = append(result.Indicators, "comma-separated feature list")
		// Count commas in feature lists
		commaCount := strings.Count(claim, ",")
		if commaCount >= 2 && result.ClaimCount < commaCount+1 {
			result.ClaimCount = commaCount + 1
		}
	}

	// Check for multiple distinct action verbs indicating separate claims
	verbCount := countDistinctVerbs(normalized)
	if verbCount >= 3 {
		result.Indicators = append(result.Indicators, "multiple distinct verbs")
		if result.ClaimCount < verbCount {
			result.ClaimCount = verbCount
		}
	}

	// Classify based on indicators and count
	if len(result.Indicators) > 0 || result.ClaimCount >= 3 {
		result.Type = CompoundClaim
		result.Suggestion = "This appears to contain multiple claims. Using full validation for comprehensive analysis."
	} else {
		result.Suggestion = "This appears to be a single claim suitable for quick validation."
	}

	return result
}

// containsFeatureList checks if the claim contains a list of features/capabilities
func containsFeatureList(claim string) bool {
	// Look for patterns like "X, Y, and Z" or "X, Y, Z"
	lower := strings.ToLower(claim)

	// Common patterns that indicate feature lists
	listPatterns := []string{
		"provides", "supports", "enables", "implements", "enforces",
		"includes", "offers", "features", "allows",
	}

	for _, pattern := range listPatterns {
		if strings.Contains(lower, pattern) {
			// Check if followed by comma-separated items
			afterPattern := lower[strings.Index(lower, pattern)+len(pattern):]
			if strings.Count(afterPattern, ",") >= 1 {
				return true
			}
		}
	}

	return false
}

// countDistinctVerbs counts the number of distinct action verbs in the claim
func countDistinctVerbs(claim string) int {
	// Common verbs in MCP claims
	verbs := []string{
		"provides", "supports", "enables", "implements", "enforces",
		"validates", "requires", "allows", "exposes", "handles",
		"processes", "manages", "controls", "limits", "forwards",
		"accepts", "rejects", "verifies", "authenticates", "authorizes",
	}

	count := 0
	for _, verb := range verbs {
		if strings.Contains(claim, " "+verb+" ") || strings.Contains(claim, " "+verb+"s ") {
			count++
		}
	}

	return count
}

// containsMultipleVerbs checks if the claim contains more than one verb
func containsMultipleVerbs(claim string) bool {
	return countDistinctVerbs(claim) >= 2
}

// IsCompoundClaim is a simple helper to check if a claim should use full validation
func IsCompoundClaim(claim string) bool {
	return ClassifyClaim(claim).Type == CompoundClaim
}

// ShouldUseQuickClaim determines if a claim is suitable for quick validation
func ShouldUseQuickClaim(claim string) bool {
	classification := ClassifyClaim(claim)

	// Additional checks for quick claim suitability
	if len(claim) > 500 {
		return false // Too long for quick claim
	}

	if strings.Count(claim, "\n") > 2 {
		return false // Multi-line claims should use full validation
	}

	return classification.Type == SingleClaim
}
