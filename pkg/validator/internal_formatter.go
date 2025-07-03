package validator

import (
	"bytes"
	"fmt"
	"text/template"
)

// Template content embedded directly
const formatTemplate = `{{- /* Template for formatting validation results into step-by-step workflow */ -}}
{{- $data := .validation_data -}}
## MCP Content Validation Report

## Step 1: Claim Extraction

I've extracted the following claims from your content:
{{- range $i, $claim := $data.parsed_claims }}
{{ add $i 1 }}. {{ $claim }}
{{- end }}

## Step 2: Validation Results
{{ range $claim := $data.claims }}
### Claim: "{{ $claim.claim }}"
**Status:** {{ if $claim.is_accurate }}✓ Accurate{{ else }}✗ Inaccurate{{ end }}
{{- if not $claim.is_accurate }}
**Issue:** {{ $claim.issue }}
**Correction:** {{ $claim.correction }}
{{- end }}
{{- if $claim.explanation }}
**Spec Reference:** {{ $claim.explanation }}
{{- end }}
{{ end }}
{{- if $data.missing_best_practices }}

## Step 3: Missing Best Practices

The following SHOULD requirements from the MCP specification are not addressed in your content:
{{- range $bp := $data.missing_best_practices }}
- {{ $bp }}
{{- end }}
{{- end }}
{{- if $data.advisory_language_issues }}

## Step 4: Modal Verb Issues

These issues relate to incorrect usage of requirement levels (MUST/SHOULD/MAY):
{{- range $issue := $data.advisory_language_issues }}
- {{ $issue }}
{{- end }}
{{- end }}

## Summary

**Overall Accuracy:** {{ if $data.validation_result.is_valid }}Content is accurate{{ else }}Content contains inaccuracies{{ end }}
**Confidence Score:** {{ printf "%.0f%%" (mul $data.validation_result.confidence 100) }}
**Spec Version:** {{ $data.validation_result.spec_version }}
{{ if not $data.validation_result.is_valid }}
## Corrected Content

Based on the validation findings, here's the corrected version of your content:

{{ $data.corrected_version }}

## Next Steps

Would you like me to:
1. Validate the corrected content to ensure all issues are resolved?
2. Provide more details about specific corrections?
3. Help you understand the specification requirements better?

Please let me know which step you'd like to proceed with.
{{- end }}`

// InternalFormatter handles internal template-based formatting of validation results.
// It uses Go's text/template package to generate consistent, structured output
// for MCP content validation reports.
type InternalFormatter struct {
	tmpl *template.Template
}

// NewInternalFormatter creates a new internal formatter with the embedded validation template.
// The formatter includes custom template functions for formatting validation data.
func NewInternalFormatter() (*InternalFormatter, error) {
	// Define template functions
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"mul": func(a, b float64) float64 { return a * b },
	}

	// Parse the template
	tmpl, err := template.New("validation").Funcs(funcMap).Parse(formatTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse validation template: %w", err)
	}

	return &InternalFormatter{tmpl: tmpl}, nil
}

// FormatValidationData formats validation data using the internal template.
// It takes a ValidationResponse and produces a structured markdown report
// showing claim extraction, validation results, missing best practices, and corrections.
func (f *InternalFormatter) FormatValidationData(data *ValidationResponse) (string, error) {
	var buf bytes.Buffer

	// Wrap data in expected structure
	templateData := map[string]interface{}{
		"validation_data": data,
	}

	err := f.tmpl.Execute(&buf, templateData)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// FormatWithTemplate formats validation results using the internal template approach.
// It converts a ValidationResult and list of ValidationMatch entries into a formatted
// validation report. If template formatting fails, callers should fall back to
// FormatValidationWorkflow for direct formatting.
func FormatWithTemplate(result ValidationResult, matches []ValidationMatch) (string, error) {
	// Build the validation response data
	response := buildValidationResponse(result, matches)

	// Create formatter
	formatter, err := NewInternalFormatter()
	if err != nil {
		return "", err
	}

	// Format using template
	formatted, err := formatter.FormatValidationData(&response)
	if err != nil {
		return "", err
	}

	return formatted, nil
}
