package spec

import (
	"context"
	"fmt"

	"github.com/carlisia/mcp-factcheck/pkg/utils"
	"github.com/mark3labs/mcp-go/mcp"
)

// VersionLister is an interface for listing available MCP spec versions.
// Implementations should return all supported specification versions
// that can be used for validation and search operations.
type VersionLister interface {
	ListVersions() ([]string, error)
}

// ListSpecVersionsToolName is the name of the MCP tool for listing specification versions.
const ListSpecVersionsToolName = "list_spec_versions"

// GetListSpecVersionsTool returns the MCP tool definition for listing available
// MCP specification versions. This tool helps users discover which spec versions
// are available for validation and search operations.
func GetListSpecVersionsTool() mcp.Tool {
	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
	schemaBytes := utils.MustMarshalSchema(schema, ListSpecVersionsToolName)
	return mcp.NewToolWithRawSchema(ListSpecVersionsToolName, "List available MCP specification versions. Use this when users ask about MCP specs, what MCP versions exist, what specifications are available, or want to know which MCP versions they can validate against.", schemaBytes)
}

// HandleListSpecVersions processes requests to list available MCP specification versions.
// It returns a formatted list of all versions that can be used with other tools
// like validate_content and search_spec.
func HandleListSpecVersions(ctx context.Context, versionLister VersionLister, args any) ([]mcp.Content, error) {
	versions, err := versionLister.ListVersions()
	if err != nil {
		return nil, fmt.Errorf("failed to list spec versions: %w", err)
	}

	// Build response content
	var contentParts []mcp.Content
	contentParts = append(contentParts, mcp.NewTextContent(
		"Available MCP specification versions:\n\n"))

	for _, version := range versions {
		contentParts = append(contentParts, mcp.NewTextContent(
			fmt.Sprintf("- %s\n", version)))
	}

	return contentParts, nil
}
