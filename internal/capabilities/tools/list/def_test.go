package list_test

import (
	"strings"
	"testing"

	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools/list"
)

const (
	// Expected tool name constant for testing
	expectedListToolName = "list-spec-versions"
)

func TestListSpecVersionsDefinition(t *testing.T) {
	def := list.ListSpecVersionsDefinition()

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
				want: expectedListToolName,
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
				if !tt.checkFunc(tt.got, tt.want) {
					t.Errorf("got %v, want %v", tt.got, tt.want)
				}
			})
		}
	})

	t.Run("description phrases", func(t *testing.T) {
		tests := []struct {
			phrase string
		}{
			{"List available MCP specification versions"},
			{"MCP specs or versions"},
			{"specifications are available"},
			{"validate against"},
			{"spec version to use"},
			{"Returns a list"},
		}

		for _, tt := range tests {
			t.Run(tt.phrase, func(t *testing.T) {
				if !strings.Contains(def.Description, tt.phrase) {
					t.Errorf("description missing expected phrase: %q", tt.phrase)
				}
			})
		}
	})

	// Test schema structure
	schema := def.Schema
	if schema == nil {
		t.Fatal("schema should not be nil")
	}

	if schema["type"] != "object" {
		t.Errorf("expected schema type 'object', got %v", schema["type"])
	}

	// Test properties - should be empty map for this tool
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties should be a map[string]any")
	}

	if len(properties) != 0 {
		t.Errorf("expected empty properties map, got %d properties", len(properties))
	}

	// Test required fields - should not exist or be empty
	if required, exists := schema["required"]; exists {
		if reqArray, ok := required.([]string); ok && len(reqArray) > 0 {
			t.Errorf("expected no required fields, got %v", reqArray)
		}
	}
}

func TestListToolNameConstant(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		want     string
	}{
		{
			name: "constant matches expected value",
			got:  list.SpecVersionsToolName,
			want: expectedListToolName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("SpecVersionsToolName = %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestDefinitionStructure(t *testing.T) {
	def := list.ListSpecVersionsDefinition()

	t.Run("use cases", func(t *testing.T) {
		tests := []struct {
			useCase string
		}{
			{"Ask about MCP specs"},
			{"know what specifications"},
			{"validate against"},
			{"unsure which spec version"},
		}

		for _, tt := range tests {
			t.Run(tt.useCase, func(t *testing.T) {
				if !strings.Contains(def.Description, tt.useCase) {
					t.Errorf("description should mention use case: %q", tt.useCase)
				}
			})
		}
	})

	t.Run("integration mention", func(t *testing.T) {
		if !strings.Contains(def.Description, "other tools") {
			t.Error("description should mention that versions can be used with other tools")
		}
	})
}

