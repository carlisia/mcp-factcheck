package prompts

import "slices"

// Spec version constants
const (
	// Draft represents the draft/development version of the MCP specification
	Draft = "draft"

	// Current represents the current stable version of the MCP specification
	Current = "2025-06-18"
)

// ValidSpecVersions lists all supported MCP specification versions.
// This is used across all prompts that work with MCP specifications.
var ValidSpecVersions = []string{Draft, Current, "2025-03-26", "2024-11-05"}

// IsValidSpecVersion checks if the provided version string is a supported
// MCP specification version. Returns true if the version is in ValidSpecVersions.
func IsValidSpecVersion(version string) bool {
	return slices.Contains(ValidSpecVersions, version)
}