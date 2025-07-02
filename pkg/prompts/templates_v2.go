package prompts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
)

// TemplateVersion defines versioning for templates
type TemplateVersion struct {
	Major int `json:"major"`
	Minor int `json:"minor"`
	Patch int `json:"patch"`
}

// String returns the version as a string
func (v TemplateVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// PromptResponsePair defines a paired prompt and response template
type PromptResponsePair struct {
	Name             string           `json:"name"`
	Description      string           `json:"description"`
	Version          TemplateVersion  `json:"version"`
	SpecVersions     []string         `json:"spec_versions"`
	RequestTemplate  string           `json:"request_template"`
	ResponseSchema   ResponseSchema   `json:"response_schema"`
	ValidationRules  []ValidationRule `json:"validation_rules,omitempty"`
	PostProcessing   PostProcessor    `json:"post_processing,omitempty"`
}

// ValidationRule defines custom validation logic
type ValidationRule struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Rule        string `json:"rule"`
}

// PostProcessor defines post-processing steps for responses
type PostProcessor struct {
	StripWhitespace bool     `json:"strip_whitespace"`
	RemoveNulls     bool     `json:"remove_nulls"`
	CustomSteps     []string `json:"custom_steps,omitempty"`
}

// FactCheckTemplateV2 is the formalized fact-check prompt-response pair
var FactCheckTemplateV2 = PromptResponsePair{
	Name:        "fact-check-v2",
	Description: "Validates MCP content against specification with structured response",
	Version:     TemplateVersion{Major: 2, Minor: 0, Patch: 0},
	SpecVersions: []string{"draft", "2025-06-18", "2025-03-26", "2024-11-05"},
	RequestTemplate: `MCP Claim Extraction & Fact-Checking

You are an expert MCP fact-checker. Extract ALL claims and validate against the official MCP specification.

{{.ClaimExtractionRules}}

{{.AccuracyCheckingRules}}

{{.SpecificationGuidanceNote}}

USER CONTENT TO CHECK:
{{.Content}}

OFFICIAL MCP SPECIFICATION SECTIONS:
{{range $i, $section := .SpecSections}}
=== Spec Section {{add $i 1}} ===
{{$section}}
{{end}}

` + GenerateSchemaPrompt(FactCheckResponseSchema),
	ResponseSchema: FactCheckResponseSchema,
	ValidationRules: []ValidationRule{
		{
			Name:        "claims_not_empty",
			Description: "Claims array must not be empty",
			Rule:        "len(claims) > 0",
		},
		{
			Name:        "summary_min_length",
			Description: "Summary must be at least 20 characters",
			Rule:        "len(summary) >= 20",
		},
	},
	PostProcessing: PostProcessor{
		StripWhitespace: true,
		RemoveNulls:     true,
	},
}

// TemplateRegistry manages versioned templates
type TemplateRegistry struct {
	templates map[string][]PromptResponsePair
}

// NewTemplateRegistry creates a new template registry
func NewTemplateRegistry() *TemplateRegistry {
	return &TemplateRegistry{
		templates: make(map[string][]PromptResponsePair),
	}
}

// Register adds a template to the registry
func (r *TemplateRegistry) Register(pair PromptResponsePair) error {
	if pair.Name == "" {
		return fmt.Errorf("template name cannot be empty")
	}

	// Store templates by name with version history
	r.templates[pair.Name] = append(r.templates[pair.Name], pair)
	return nil
}

// GetLatest returns the latest version of a template
func (r *TemplateRegistry) GetLatest(name string) (*PromptResponsePair, error) {
	versions, exists := r.templates[name]
	if !exists || len(versions) == 0 {
		return nil, fmt.Errorf("template not found: %s", name)
	}

	// Find the highest version
	latest := versions[0]
	for _, v := range versions[1:] {
		if v.Version.Major > latest.Version.Major ||
			(v.Version.Major == latest.Version.Major && v.Version.Minor > latest.Version.Minor) ||
			(v.Version.Major == latest.Version.Major && v.Version.Minor == latest.Version.Minor && v.Version.Patch > latest.Version.Patch) {
			latest = v
		}
	}

	return &latest, nil
}

// GetByVersion returns a specific version of a template
func (r *TemplateRegistry) GetByVersion(name string, version TemplateVersion) (*PromptResponsePair, error) {
	versions, exists := r.templates[name]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", name)
	}

	for _, v := range versions {
		if v.Version == version {
			return &v, nil
		}
	}

	return nil, fmt.Errorf("template version not found: %s v%s", name, version.String())
}

// RenderWithValidation renders a template and validates the response
func RenderWithValidation(pair PromptResponsePair, args map[string]any, response json.RawMessage) (json.RawMessage, error) {
	// Render the request template
	tmpl, err := template.New(pair.Name).Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}).Parse(pair.RequestTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, args); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	// Validate the response against schema
	validator := NewSchemaValidator(pair.ResponseSchema)
	if err := validator.Validate(response); err != nil {
		return nil, fmt.Errorf("response validation failed: %w", err)
	}

	// Apply post-processing
	processed := response
	if pair.PostProcessing.StripWhitespace || pair.PostProcessing.RemoveNulls {
		var data map[string]any
		if err := json.Unmarshal(response, &data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal for post-processing: %w", err)
		}

		if pair.PostProcessing.RemoveNulls {
			data = removeNulls(data)
		}

		processed, err = json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal after post-processing: %w", err)
		}
	}

	return processed, nil
}

// removeNulls recursively removes null values from a map
func removeNulls(data map[string]any) map[string]any {
	result := make(map[string]any)
	for k, v := range data {
		if v != nil {
			switch val := v.(type) {
			case map[string]any:
				result[k] = removeNulls(val)
			case []any:
				var filtered []any
				for _, item := range val {
					if item != nil {
						filtered = append(filtered, item)
					}
				}
				if len(filtered) > 0 {
					result[k] = filtered
				}
			default:
				result[k] = v
			}
		}
	}
	return result
}