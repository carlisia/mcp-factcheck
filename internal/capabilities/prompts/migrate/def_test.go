package migrate_test

import (
	"strings"
	"testing"

	"github.com/carlisia/mcp-factcheck/internal/capabilities/prompts/migrate"
)

const (
	// Expected prompt name constant for testing
	expectedMigratePromptName = "migrate-mcp-content"

	// Expected spec versions
	expectedVersionLatest  = "2025-06-18"
	expectedVersionMarch   = "2025-03-26"
	expectedVersionNov     = "2024-11-05"
	expectedVersionDraft   = "draft"
	
	// Expected update scopes
	expectedScopeComprehensive = "comprehensive"
	expectedScopeCritical      = "critical_only"
	expectedScopeEnhancement   = "enhancement_focused"
)

func TestPromptDefinition(t *testing.T) {
	def := migrate.PromptDefinition()

	t.Run("basic properties", func(t *testing.T) {
		tests := []struct {
			name     string
			got      any
			want     any
			checkFunc func(got, want any) bool
		}{
			{
				name: "prompt name",
				got:  def.Name,
				want: expectedMigratePromptName,
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
				name: "has arguments",
				got:  len(def.Arguments),
				want: 3,
				checkFunc: func(got, want any) bool {
					return got.(int) == want.(int)
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

	t.Run("description content", func(t *testing.T) {
		tests := []struct {
			phrase string
		}{
			{"Update MCP documentation"},
			{"tutorials"},
			{"content"},
			{"specification version"},
		}

		for _, tt := range tests {
			t.Run(tt.phrase, func(t *testing.T) {
				if !strings.Contains(def.Description, tt.phrase) {
					t.Errorf("description missing expected phrase: %q", tt.phrase)
				}
			})
		}
	})

	t.Run("arguments", func(t *testing.T) {
		tests := []struct {
			name          string
			argIndex      int
			expectedName  string
			expectedReq   bool
			checkDesc     []string
		}{
			{
				name:         "current_version argument",
				argIndex:     0,
				expectedName: "current_version",
				expectedReq:  true,
				checkDesc:    []string{expectedVersionLatest, expectedVersionDraft, "current MCP specification"},
			},
			{
				name:         "target_version argument",
				argIndex:     1,
				expectedName: "target_version",
				expectedReq:  true,
				checkDesc:    []string{expectedVersionLatest, expectedVersionDraft, "target MCP specification"},
			},
			{
				name:         "update_scope argument",
				argIndex:     2,
				expectedName: "update_scope",
				expectedReq:  false,
				checkDesc:    []string{expectedScopeComprehensive, expectedScopeCritical, expectedScopeEnhancement},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.argIndex >= len(def.Arguments) {
					t.Fatalf("argument index %d out of range", tt.argIndex)
				}
				
				arg := def.Arguments[tt.argIndex]
				
				if arg.Name != tt.expectedName {
					t.Errorf("argument name = %q, want %q", arg.Name, tt.expectedName)
				}
				
				if arg.Required != tt.expectedReq {
					t.Errorf("argument required = %v, want %v", arg.Required, tt.expectedReq)
				}
				
				if arg.Description == "" {
					t.Error("argument should have a description")
				}
				
				for _, phrase := range tt.checkDesc {
					if !strings.Contains(arg.Description, phrase) {
						t.Errorf("argument description missing phrase: %q", phrase)
					}
				}
			})
		}
	})
}

func TestMCPContentPromptNameConstant(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		want     string
	}{
		{
			name: "constant matches expected value",
			got:  migrate.MCPContentPromptName,
			want: expectedMigratePromptName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("MCPContentPromptName = %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestArgumentStructure(t *testing.T) {
	def := migrate.PromptDefinition()

	t.Run("all required arguments listed first", func(t *testing.T) {
		// Check that required arguments come before optional ones
		foundOptional := false
		for i, arg := range def.Arguments {
			if !arg.Required {
				foundOptional = true
			} else if foundOptional {
				t.Errorf("required argument %q at index %d comes after optional argument", arg.Name, i)
			}
		}
	})

	t.Run("version arguments list all valid versions", func(t *testing.T) {
		versionArgs := []migrate.Argument{def.Arguments[0], def.Arguments[1]}
		expectedVersions := []string{
			expectedVersionLatest,
			expectedVersionMarch,
			expectedVersionNov,
			expectedVersionDraft,
		}

		for _, arg := range versionArgs {
			for _, version := range expectedVersions {
				if !strings.Contains(arg.Description, version) {
					t.Errorf("version argument %q missing version %q in description", arg.Name, version)
				}
			}
		}
	})

	t.Run("update scope lists all valid scopes", func(t *testing.T) {
		scopeArg := def.Arguments[2]
		expectedScopes := []string{
			expectedScopeComprehensive + " (default)",
			expectedScopeCritical,
			expectedScopeEnhancement,
		}

		for _, scope := range expectedScopes {
			if !strings.Contains(scopeArg.Description, scope) {
				t.Errorf("update_scope argument missing scope %q in description", scope)
			}
		}
	})
}