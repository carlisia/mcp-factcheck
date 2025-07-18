package capabilities

import (
	"slices"
	"strings"
)

// Spec version constants
const (
	// Draft represents the draft/development version of the MCP specification
	Draft = "draft"

	// Latest represents the current stable version of the MCP specification
	Latest = "2025-06-18"
)

// ValidSpecVersions lists all supported MCP specification versions.
// This is used across all tools and prompts that work with MCP specifications.
var ValidSpecVersions = []string{Draft, Latest, "2025-03-26", "2024-11-05"}

// SpecVersions provides formatted string representations of valid spec versions
type SpecVersions struct{}

// String returns a comma-separated list of all valid spec versions
func (s SpecVersions) String() string {
	return strings.Join(ValidSpecVersions, ", ")
}

// ForDescription returns a formatted string suitable for tool/prompt descriptions
func (s SpecVersions) ForDescription() string {
	return s.String() + " (use 'draft' for latest development version)"
}

// Versions is a singleton instance for accessing version formatting
var Versions = SpecVersions{}

// IsValidSpecVersion checks if the provided version string is a supported
// MCP specification version. Returns true if the version is in ValidSpecVersions.
func IsValidSpecVersion(version string) bool {
	return slices.Contains(ValidSpecVersions, version)
}
