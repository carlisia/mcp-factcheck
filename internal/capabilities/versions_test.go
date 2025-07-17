package capabilities_test

import (
	"strings"
	"testing"

	"github.com/carlisia/mcp-factcheck/internal/capabilities"
)

func TestSpecVersions_String(t *testing.T) {
	result := capabilities.Versions.String()
	
	// Check that all versions are present
	expectedVersions := []string{"draft", "2025-06-18", "2025-03-26", "2024-11-05"}
	for _, version := range expectedVersions {
		if !strings.Contains(result, version) {
			t.Errorf("String() missing version %q, got: %s", version, result)
		}
	}
	
	// Check format
	if result != "draft, 2025-06-18, 2025-03-26, 2024-11-05" {
		t.Errorf("String() = %q, want comma-separated list", result)
	}
}

func TestSpecVersions_ForDescription(t *testing.T) {
	result := capabilities.Versions.ForDescription()
	
	// Check that it includes the base string
	baseString := capabilities.Versions.String()
	if !strings.HasPrefix(result, baseString) {
		t.Errorf("ForDescription() should start with String(), got: %s", result)
	}
	
	// Check that it includes the draft explanation
	if !strings.Contains(result, "use 'draft' for latest development version") {
		t.Errorf("ForDescription() missing draft explanation, got: %s", result)
	}
}

func TestIsValidSpecVersion(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"draft", true},
		{"2025-06-18", true},
		{"2025-03-26", true},
		{"2024-11-05", true},
		{"invalid", false},
		{"", false},
		{"2023-01-01", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			if got := capabilities.IsValidSpecVersion(tt.version); got != tt.want {
				t.Errorf("IsValidSpecVersion(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestVersionConstants(t *testing.T) {
	// Test that constants have expected values
	if capabilities.Draft != "draft" {
		t.Errorf("Draft = %q, want %q", capabilities.Draft, "draft")
	}
	
	if capabilities.Latest != "2025-06-18" {
		t.Errorf("Latest = %q, want %q", capabilities.Latest, "2025-06-18")
	}
}