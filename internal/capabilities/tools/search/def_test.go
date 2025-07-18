package search_test

import (
	"strings"
	"testing"

	"github.com/carlisia/mcp-factcheck/internal/capabilities"
	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools/search"
)

const (
	// Expected tool name constant for testing
	expectedSearchToolName = "search-spec"
)

func TestSearchSpecDefinition(t *testing.T) {
	def := search.SearchSpecDefinition()

	t.Run("basic properties", func(t *testing.T) {
		tests := []struct {
			name      string
			got       any
			want      any
			checkFunc func(got, want any) bool
		}{
			{
				name: "tool name",
				got:  def.Name,
				want: expectedSearchToolName,
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
			{"Search MCP specification"},
			{"semantic similarity"},
			{"Find specific sections"},
			{"AI-powered"},
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

	// Test properties
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties should be a map[string]any")
	}

	// Test query property
	queryProp, ok := properties["query"].(map[string]any)
	if !ok {
		t.Fatal("query property should be a map")
	}
	if queryProp["type"] != "string" {
		t.Errorf("query type should be string, got %v", queryProp["type"])
	}
	if queryProp["description"] == "" {
		t.Error("query should have a description")
	}

	// Test specVersion property
	specVersionProp, ok := properties["specVersion"].(map[string]any)
	if !ok {
		t.Fatal("specVersion property should be a map")
	}
	if specVersionProp["type"] != "string" {
		t.Errorf("specVersion type should be string, got %v", specVersionProp["type"])
	}
	if specVersionProp["default"] != capabilities.Latest {
		t.Errorf("specVersion default should be capabilities.Latest, got %v", specVersionProp["default"])
	}

	// Check enum values
	enumValues, ok := specVersionProp["enum"].([]string)
	if !ok {
		t.Fatal("specVersion enum should be []string")
	}
	if len(enumValues) == 0 {
		t.Error("specVersion enum should not be empty")
	}

	// Test topK property
	topKProp, ok := properties["topK"].(map[string]any)
	if !ok {
		t.Fatal("topK property should be a map")
	}
	if topKProp["type"] != "integer" {
		t.Errorf("topK type should be integer, got %v", topKProp["type"])
	}
	if topKProp["default"] != 5 { // defaultTopK value
		t.Errorf("topK default should be 5, got %v", topKProp["default"])
	}
	if topKProp["minimum"] != 1 { // minTopK value
		t.Errorf("topK minimum should be 1, got %v", topKProp["minimum"])
	}
	if topKProp["maximum"] != 20 { // maxTopK value
		t.Errorf("topK maximum should be 20, got %v", topKProp["maximum"])
	}

	// Test required fields
	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("required should be []string")
	}
	if len(required) != 1 || required[0] != "query" {
		t.Errorf("only 'query' should be required, got %v", required)
	}
}

func TestSearchToolNameConstant(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{
			name: "constant matches expected value",
			got:  search.SpecDBToolName,
			want: expectedSearchToolName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("SpecDBToolName = %q, want %q", tt.got, tt.want)
			}
		})
	}
}
