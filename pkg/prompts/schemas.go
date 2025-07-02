package prompts

import (
	"encoding/json"
	"fmt"
)

// ResponseSchema defines the expected structure for LLM responses
type ResponseSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Version     string                 `json:"version"`
	Fields      []ResponseField        `json:"fields"`
	Required    []string               `json:"required"`
	Examples    []json.RawMessage      `json:"examples,omitempty"`
}

// ResponseField defines a field in the response schema
type ResponseField struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Required    bool                   `json:"required"`
	Properties  map[string]ResponseField `json:"properties,omitempty"`
	Items       *ResponseField         `json:"items,omitempty"`
	Example     any                    `json:"example,omitempty"`
}

// FactCheckResponseSchema defines the structured response for fact-checking
var FactCheckResponseSchema = ResponseSchema{
	Name:        "fact_check_response",
	Description: "Structured response for MCP content fact-checking",
	Version:     "1.0.0",
	Fields: []ResponseField{
		{
			Name:        "claims",
			Type:        "array",
			Description: "List of extracted claims with validation results",
			Required:    true,
			Items: &ResponseField{
				Type: "object",
				Properties: map[string]ResponseField{
					"claim": {
						Name:        "claim",
						Type:        "string",
						Description: "Full expanded claim text",
						Required:    true,
					},
					"is_accurate": {
						Name:        "is_accurate",
						Type:        "boolean",
						Description: "Whether the claim is accurate according to the spec",
						Required:    true,
					},
					"correction": {
						Name:        "correction",
						Type:        "string",
						Description: "Suggested rewording that matches the spec's language strength",
						Required:    false,
					},
					"explanation": {
						Name:        "explanation",
						Type:        "string",
						Description: "Why this is accurate or inaccurate based on spec - must quote exact spec text",
						Required:    true,
					},
					"issue": {
						Name:        "issue",
						Type:        "string",
						Description: "Type of issue (e.g., 'language_strength', 'factual_error', 'missing_context')",
						Required:    false,
					},
				},
			},
		},
		{
			Name:        "missing_best_practices",
			Type:        "array",
			Description: "List of SHOULD requirements from the spec that the content doesn't mention",
			Required:    true,
			Items: &ResponseField{
				Type:        "string",
				Description: "A SHOULD requirement not addressed in the content",
			},
		},
		{
			Name:        "advisory_language_issues",
			Type:        "array",
			Description: "List of MAY/CAN options presented incorrectly or modal verb confusion",
			Required:    true,
			Items: &ResponseField{
				Type:        "string",
				Description: "Description of the modal verb or language strength issue",
			},
		},
		{
			Name:        "overall_is_accurate",
			Type:        "boolean",
			Description: "Overall accuracy assessment of the content",
			Required:    true,
		},
		{
			Name:        "summary",
			Type:        "string",
			Description: "Brief summary of findings including accuracy, completeness, modal verb usage, and language strength issues",
			Required:    true,
		},
		{
			Name:        "confidence_score",
			Type:        "number",
			Description: "Confidence score for the validation (0.0 to 1.0)",
			Required:    false,
		},
	},
	Required: []string{"claims", "missing_best_practices", "advisory_language_issues", "overall_is_accurate", "summary"},
}

// SchemaValidator validates JSON responses against a schema
type SchemaValidator struct {
	schema ResponseSchema
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator(schema ResponseSchema) *SchemaValidator {
	return &SchemaValidator{
		schema: schema,
	}
}

// Validate checks if a JSON response matches the expected schema
func (v *SchemaValidator) Validate(response json.RawMessage) error {
	var data map[string]any
	if err := json.Unmarshal(response, &data); err != nil {
		return fmt.Errorf("invalid JSON response: %w", err)
	}

	// Check required fields
	for _, field := range v.schema.Required {
		if _, exists := data[field]; !exists {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	// Validate field types
	for _, field := range v.schema.Fields {
		if value, exists := data[field.Name]; exists {
			if err := v.validateFieldType(field, value); err != nil {
				return fmt.Errorf("field %s: %w", field.Name, err)
			}
		} else if field.Required {
			return fmt.Errorf("missing required field: %s", field.Name)
		}
	}

	return nil
}

// validateFieldType checks if a value matches the expected field type
func (v *SchemaValidator) validateFieldType(field ResponseField, value any) error {
	switch field.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}
	case "number":
		switch value.(type) {
		case float64, int:
			// Valid number types
		default:
			return fmt.Errorf("expected number, got %T", value)
		}
	case "array":
		arr, ok := value.([]any)
		if !ok {
			return fmt.Errorf("expected array, got %T", value)
		}
		// Validate array items if schema is defined
		if field.Items != nil {
			for i, item := range arr {
				if err := v.validateFieldType(*field.Items, item); err != nil {
					return fmt.Errorf("item %d: %w", i, err)
				}
			}
		}
	case "object":
		obj, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("expected object, got %T", value)
		}
		// Validate object properties if schema is defined
		if field.Properties != nil {
			for propName, propField := range field.Properties {
				if propValue, exists := obj[propName]; exists {
					if err := v.validateFieldType(propField, propValue); err != nil {
						return fmt.Errorf("property %s: %w", propName, err)
					}
				} else if propField.Required {
					return fmt.Errorf("missing required property: %s", propName)
				}
			}
		}
	default:
		return fmt.Errorf("unknown field type: %s", field.Type)
	}
	return nil
}

// GenerateSchemaPrompt creates a prompt instruction for structured output
func GenerateSchemaPrompt(schema ResponseSchema) string {
	schemaJSON, _ := json.MarshalIndent(schema, "", "  ")
	return fmt.Sprintf(`
You MUST return your response as a JSON object that strictly follows this schema:

%s

IMPORTANT: 
- Your response MUST be valid JSON
- You MUST include all required fields
- Field types MUST match exactly as specified
- Do NOT include any text before or after the JSON object
- Do NOT wrap the JSON in markdown code blocks
`, string(schemaJSON))
}