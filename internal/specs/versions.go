// Package specs provides utilities for managing MCP specification versions.
// It defines valid spec versions and helper functions for version validation.
package specs

import "slices"

// Valid MCP spec versions
var ValidSpecVersions = []string{"draft", "2025-06-18", "2025-03-26", "2024-11-05"}

// Default spec version
const DefaultSpecVersion = "2025-06-18"

// IsValidSpecVersion checks if the provided version is supported
func IsValidSpecVersion(version string) bool {
	return slices.Contains(ValidSpecVersions, version)
}
