// Package specs provides utilities for managing MCP specification versions.
// It defines valid spec versions and helper functions for version validation.
package specs

import "slices"

// ValidSpecVersions lists all supported MCP specification versions.
// These versions correspond to different releases of the MCP specification
// that the fact-check system can validate against.
var ValidSpecVersions = []string{"draft", "2025-06-18", "2025-03-26", "2024-11-05"}

// DefaultSpecVersion is the default MCP specification version used when
// no specific version is provided. This typically corresponds to the most
// recent stable release of the MCP specification.
const DefaultSpecVersion = "2025-06-18"

// IsValidSpecVersion checks if the provided version string is a supported
// MCP specification version. Returns true if the version is in ValidSpecVersions.
func IsValidSpecVersion(version string) bool {
	return slices.Contains(ValidSpecVersions, version)
}