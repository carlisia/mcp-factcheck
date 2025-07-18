package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifyClaim(t *testing.T) {
	tests := []struct {
		name               string
		claim              string
		expectedType       ClaimType
		expectedCount      int
		expectedIndicators []string
	}{
		// Single claims
		{
			name:          "simple single claim",
			claim:         "MCP uses JSON-RPC 2.0",
			expectedType:  SingleClaim,
			expectedCount: 1,
		},
		{
			name:          "single claim with one 'and'",
			claim:         "MCP provides tools and resources",
			expectedType:  SingleClaim,
			expectedCount: 1,
		},
		{
			name:          "question as single claim",
			claim:         "Does MCP support WebSocket connections?",
			expectedType:  SingleClaim,
			expectedCount: 1,
		},

		// Compound claims
		{
			name:               "semicolon separated claims",
			claim:              "MCP Never forwards raw model traffic; enforces ACLs, rate limits, and provenance",
			expectedType:       CompoundClaim,
			expectedCount:      3, // Actually has 3 due to comma list
			expectedIndicators: []string{"semicolon-separated claims", "comma-separated feature list"},
		},
		{
			name:               "multiple semicolons",
			claim:              "MCP uses JSON-RPC; supports OAuth; provides resource discovery",
			expectedType:       CompoundClaim,
			expectedCount:      3,
			expectedIndicators: []string{"semicolon-separated claims"},
		},
		{
			name:               "multiple 'and' conjunctions",
			claim:              "MCP provides tools and resources and enables authentication and supports prompts",
			expectedType:       CompoundClaim,
			expectedCount:      4,
			expectedIndicators: []string{"multiple 'and' conjunctions", "multiple distinct verbs"},
		},
		{
			name:               "comma-separated feature list",
			claim:              "MCP enforces authentication, authorization, rate limiting, and access control",
			expectedType:       CompoundClaim,
			expectedCount:      4,
			expectedIndicators: []string{"comma-separated feature list"},
		},
		{
			name:               "multiple distinct verbs",
			claim:              "MCP validates tokens then authenticates users then authorizes requests",
			expectedType:       CompoundClaim,
			expectedCount:      3,
			expectedIndicators: []string{"multiple distinct verbs"},
		},

		// Edge cases
		{
			name:          "claim with commas but not a list",
			claim:         "MCP, as a protocol, uses JSON-RPC",
			expectedType:  SingleClaim,
			expectedCount: 1,
		},
		{
			name:          "claim with 'and' in object name",
			claim:         "MCP supports the 'tools and resources' message type",
			expectedType:  SingleClaim,
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyClaim(tt.claim)

			assert.Equal(t, tt.expectedType, result.Type,
				"Classification type mismatch for: %s", tt.claim)
			assert.Equal(t, tt.expectedCount, result.ClaimCount,
				"Claim count mismatch for: %s", tt.claim)

			if tt.expectedIndicators != nil {
				assert.ElementsMatch(t, tt.expectedIndicators, result.Indicators,
					"Indicators mismatch for: %s", tt.claim)
			}

			assert.NotEmpty(t, result.Suggestion, "Should have a suggestion")
		})
	}
}

func TestShouldUseQuickClaim(t *testing.T) {
	tests := []struct {
		name     string
		claim    string
		expected bool
	}{
		{
			name:     "simple claim",
			claim:    "MCP uses JSON-RPC",
			expected: true,
		},
		{
			name:     "compound claim with semicolon",
			claim:    "MCP uses JSON-RPC; supports OAuth",
			expected: false,
		},
		{
			name:     "very long claim",
			claim:    string(make([]byte, 600)), // 600 chars
			expected: false,
		},
		{
			name:     "multi-line claim",
			claim:    "MCP provides:\n- Tools\n- Resources\n- Prompts",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldUseQuickClaim(tt.claim)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsCompoundClaim(t *testing.T) {
	tests := []struct {
		claim    string
		expected bool
	}{
		{"MCP uses JSON-RPC", false},
		{"MCP uses JSON-RPC; supports OAuth", true},
		{"MCP provides tools, resources, and prompts", true},
		{"MCP enables clients to invoke tools", false},
	}

	for _, tt := range tests {
		t.Run(tt.claim, func(t *testing.T) {
			result := IsCompoundClaim(tt.claim)
			assert.Equal(t, tt.expected, result)
		})
	}
}
