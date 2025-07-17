package validation_test

import (
	"testing"

	"github.com/carlisia/mcp-factcheck/internal/capabilities"
	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// Expected tool name constant for testing
	expectedClaimsToolName = "check-mcp-claims"
)

func TestClaimsDefinition(t *testing.T) {
	def := validation.ClaimsDefinition()

	t.Run("basic properties", func(t *testing.T) {
		tests := []struct {
			name     string
			got      any
			want     any
			checkFunc func(got, want any) bool
		}{
			{
				name: "tool name",
				got:  def.Name,
				want: expectedClaimsToolName,
				checkFunc: func(got, want any) bool {
					return got.(string) == want.(string)
				},
			},
			{
				name: "has description",
				got:  def.Description,
				want: "",
				checkFunc: func(got, want any) bool {
					return got.(string) != want.(string) // not empty
				},
			},
			{
				name: "schema type",
				got:  def.Schema["type"],
				want: "object",
				checkFunc: func(got, want any) bool {
					return got.(string) == want.(string)
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.True(t, tt.checkFunc(tt.got, tt.want), "got %v, want %v", tt.got, tt.want)
			})
		}
	})

	t.Run("description phrases", func(t *testing.T) {
		tests := []struct {
			phrase string
		}{
			{"Validate any MCP-related content"},
			{"official specification"},
			{"All claims extracted"},
			{"individually validated"},
			{"Corrections for any inaccuracies"},
			{"Documentation, tutorials"},
			{"description of MCP functionality"},
		}

		for _, tt := range tests {
			t.Run(tt.phrase, func(t *testing.T) {
				assert.Contains(t, def.Description, tt.phrase, "description missing expected phrase: %q", tt.phrase)
			})
		}
	})

	t.Run("schema properties", func(t *testing.T) {
		properties, ok := def.Schema["properties"].(map[string]any)
		require.True(t, ok, "properties should be a map[string]any")

		tests := []struct {
			name         string
			propName     string
			expectedType string
			hasDefault   bool
			defaultValue any
			hasEnum      bool
		}{
			{
				name:         "content property",
				propName:     "content",
				expectedType: "string",
				hasDefault:   false,
			},
			{
				name:         "specVersion property",
				propName:     "specVersion",
				expectedType: "string",
				hasDefault:   true,
				defaultValue: capabilities.Latest,
				hasEnum:      true,
			},
			{
				name:         "useChunking property",
				propName:     "useChunking",
				expectedType: "boolean",
				hasDefault:   true,
				defaultValue: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				prop, ok := properties[tt.propName].(map[string]any)
				require.True(t, ok, "%s should be a map", tt.propName)

				assert.Equal(t, tt.expectedType, prop["type"], "%s type mismatch", tt.propName)

				assert.NotEmpty(t, prop["description"], "%s should have a description", tt.propName)

				if tt.hasDefault {
					assert.Equal(t, tt.defaultValue, prop["default"], "%s default value mismatch", tt.propName)
				}

				if tt.hasEnum {
					if enumValue, hasEnum := prop["enum"]; hasEnum {
						enum, ok := enumValue.([]string)
						if ok {
							assert.NotEmpty(t, enum, "%s enum should not be empty", tt.propName)
						}
					}
				}
			})
		}
	})

	t.Run("required fields", func(t *testing.T) {
		required, ok := def.Schema["required"].([]string)
		require.True(t, ok, "required should be []string")

		tests := []struct {
			name     string
			field    string
			required bool
		}{
			{"content is required", "content", true},
			{"specVersion not required", "specVersion", false},
			{"useChunking not required", "useChunking", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				found := false
				for _, r := range required {
					if r == tt.field {
						found = true
						break
					}
				}
				assert.Equal(t, tt.required, found, "field %q required status mismatch", tt.field)
			})
		}
	})
}

func TestClaimsToolName(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		want     string
	}{
		{
			name: "correct tool name format",
			got:  validation.MCPClaimsToolName,
			want: expectedClaimsToolName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.got, "MCPClaimsToolName mismatch")
		})
	}
}

func TestClaimsDefinitionUseCases(t *testing.T) {
	def := validation.ClaimsDefinition()

	t.Run("use cases", func(t *testing.T) {
		tests := []struct {
			useCase string
		}{
			{"Documentation"},
			{"tutorials"},
			{"MCP capabilities"},
			{"MCP functionality"},
			{"what MCP does"},
			{"how it works"},
		}

		for _, tt := range tests {
			t.Run(tt.useCase, func(t *testing.T) {
				assert.Contains(t, def.Description, tt.useCase, "description should mention use case: %q", tt.useCase)
			})
		}
	})

	t.Run("return values", func(t *testing.T) {
		tests := []struct {
			returnValue string
		}{
			{"individually validated"},
			{"Corrections for any inaccuracies"},
			{"Missing best practices"},
			{"Overall accuracy assessment"},
		}

		for _, tt := range tests {
			t.Run(tt.returnValue, func(t *testing.T) {
				assert.Contains(t, def.Description, tt.returnValue, "description should mention return value: %q", tt.returnValue)
			})
		}
	})
}