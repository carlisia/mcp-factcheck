package validator

import (
	"fmt"
	"strings"
)

// ValidationError represents a detailed validation error with context
type ValidationError struct {
	Type        string   `json:"type"`        // "inaccuracy", "missing", "imprecise", "unsupported"
	Severity    string   `json:"severity"`    // "critical", "warning", "suggestion"
	Message     string   `json:"message"`     // Human readable description
	Found       string   `json:"found"`       // What was found in the content
	Expected    string   `json:"expected"`    // What should be there instead
	SpecSection string   `json:"spec_section"` // Which part of spec this relates to
	LineNumber  int      `json:"line_number,omitempty"` // Line number if available
	Suggestions []string `json:"suggestions"` // Actionable suggestions
}

// IssueType constants
const (
	IssueTypeInaccuracy  = "inaccuracy"
	IssueTypeMissing     = "missing"
	IssueTypeImprecise   = "imprecise"
	IssueTypeUnsupported = "unsupported"
)

// Severity constants
const (
	SeverityCritical   = "critical"
	SeverityWarning    = "warning" 
	SeveritySuggestion = "suggestion"
)

// NewValidationError creates a new validation error
func NewValidationError(errorType, severity, message string) *ValidationError {
	return &ValidationError{
		Type:        errorType,
		Severity:    severity,
		Message:     message,
		Suggestions: make([]string, 0),
	}
}

// WithFound sets what was found in the content
func (e *ValidationError) WithFound(found string) *ValidationError {
	e.Found = found
	return e
}

// WithExpected sets what should be there instead
func (e *ValidationError) WithExpected(expected string) *ValidationError {
	e.Expected = expected
	return e
}

// WithSpecSection sets the relevant spec section
func (e *ValidationError) WithSpecSection(section string) *ValidationError {
	e.SpecSection = section
	return e
}

// WithLineNumber sets the line number
func (e *ValidationError) WithLineNumber(line int) *ValidationError {
	e.LineNumber = line
	return e
}

// AddSuggestion adds an actionable suggestion
func (e *ValidationError) AddSuggestion(suggestion string) *ValidationError {
	e.Suggestions = append(e.Suggestions, suggestion)
	return e
}

// FormatErrorMessage creates a comprehensive error message
func (e *ValidationError) FormatErrorMessage() string {
	var parts []string
	
	// Add severity prefix
	switch e.Severity {
	case SeverityCritical:
		parts = append(parts, "❌ CRITICAL:")
	case SeverityWarning:
		parts = append(parts, "⚠️  WARNING:")
	case SeveritySuggestion:
		parts = append(parts, "💡 SUGGESTION:")
	}
	
	// Add main message
	parts = append(parts, e.Message)
	
	// Add line number if available
	if e.LineNumber > 0 {
		parts = append(parts, fmt.Sprintf("(Line %d)", e.LineNumber))
	}
	
	var details []string
	
	// Add what was found vs expected
	if e.Found != "" && e.Expected != "" {
		details = append(details, fmt.Sprintf("Found: %s", e.Found))
		details = append(details, fmt.Sprintf("Expected: %s", e.Expected))
	}
	
	// Add spec section reference
	if e.SpecSection != "" {
		details = append(details, fmt.Sprintf("Spec Reference: %s", e.SpecSection))
	}
	
	// Add suggestions
	if len(e.Suggestions) > 0 {
		details = append(details, "Suggestions:")
		for _, suggestion := range e.Suggestions {
			details = append(details, fmt.Sprintf("  • %s", suggestion))
		}
	}
	
	if len(details) > 0 {
		return fmt.Sprintf("%s\n%s", strings.Join(parts, " "), strings.Join(details, "\n"))
	}
	
	return strings.Join(parts, " ")
}

// Common validation error patterns
func NewInaccurateClaimError(found, expected, specSection string) *ValidationError {
	return NewValidationError(IssueTypeInaccuracy, SeverityCritical, "Inaccurate claim about MCP behavior").
		WithFound(found).
		WithExpected(expected).
		WithSpecSection(specSection).
		AddSuggestion("Update the claim to match the official specification").
		AddSuggestion("Use more precise language (MUST/SHOULD/MAY)")
}

func NewMissingRequirementError(requirement, specSection string) *ValidationError {
	return NewValidationError(IssueTypeMissing, SeverityWarning, "Missing required MCP specification element").
		WithExpected(requirement).
		WithSpecSection(specSection).
		AddSuggestion("Add the missing requirement to ensure completeness").
		AddSuggestion("Review the complete specification section")
}

func NewImpreciseLanguageError(found, improved, specSection string) *ValidationError {
	return NewValidationError(IssueTypeImprecise, SeveritySuggestion, "Language could be more spec-compliant").
		WithFound(found).
		WithExpected(improved).
		WithSpecSection(specSection).
		AddSuggestion("Consider using more precise terminology").
		AddSuggestion("Align language with official specification")
}

func NewUnsupportedFeatureError(feature, reason string) *ValidationError {
	return NewValidationError(IssueTypeUnsupported, SeverityWarning, fmt.Sprintf("Feature '%s' is not supported in MCP", feature)).
		WithFound(feature).
		WithExpected(reason).
		AddSuggestion("Remove references to unsupported features").
		AddSuggestion("Check the latest MCP specification for supported features")
}