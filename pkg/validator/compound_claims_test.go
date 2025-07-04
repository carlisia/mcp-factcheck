package validator

import (
	"testing"
)

func TestDecomposeCompoundClaim(t *testing.T) {
	tests := []struct {
		name             string
		claim            string
		expectedSubCount int
		expectedFirst    string
		expectedSecond   string
		isCompound       bool
	}{
		{
			name:             "Simple compound with and",
			claim:            "Servers implement request validation and timeouts",
			expectedSubCount: 2,
			expectedFirst:    "Servers implement request validation",
			expectedSecond:   "Servers implement timeouts",
			isCompound:       true,
		},
		{
			name:             "Not a compound claim",
			claim:            "Servers implement request validation",
			expectedSubCount: 1,
			expectedFirst:    "Servers implement request validation",
			isCompound:       false,
		},
		{
			name:             "Complex compound with multiple parts",
			claim:            "MCP provides tools and resources and prompts",
			expectedSubCount: 3,
			expectedFirst:    "MCP provides tools",
			isCompound:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DecomposeCompoundClaim(tt.claim)

			if result.IsCompound != tt.isCompound {
				t.Errorf("Expected IsCompound=%v, got %v", tt.isCompound, result.IsCompound)
			}

			if len(result.SubClaims) != tt.expectedSubCount {
				t.Errorf("Expected %d subclaims, got %d", tt.expectedSubCount, len(result.SubClaims))
			}

			if tt.expectedSubCount > 0 && result.SubClaims[0].Text != tt.expectedFirst {
				t.Errorf("Expected first subclaim '%s', got '%s'", tt.expectedFirst, result.SubClaims[0].Text)
			}

			if tt.expectedSubCount > 1 && tt.expectedSecond != "" && result.SubClaims[1].Text != tt.expectedSecond {
				t.Errorf("Expected second subclaim '%s', got '%s'", tt.expectedSecond, result.SubClaims[1].Text)
			}
		})
	}
}

func TestExtractKeyTerms(t *testing.T) {
	tests := []struct {
		name     string
		claim    string
		expected []string
	}{
		{
			name:     "Simple claim",
			claim:    "servers implement validation",
			expected: []string{"servers", "implement", "validation"},
		},
		{
			name:     "Claim with stop words",
			claim:    "the server should validate all requests",
			expected: []string{"server", "validate", "all", "requests"},
		},
		{
			name:     "Claim with punctuation",
			claim:    "MCP provides tools, resources, and prompts.",
			expected: []string{"MCP", "provides", "tools", "resources", "prompts"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractKeyTerms(tt.claim)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d terms, got %d: %v", len(tt.expected), len(result), result)
				return
			}

			for i, term := range result {
				if term != tt.expected[i] {
					t.Errorf("Expected term[%d]='%s', got '%s'", i, tt.expected[i], term)
				}
			}
		})
	}
}
