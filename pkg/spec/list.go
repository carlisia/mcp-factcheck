package spec

import (
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// VersionLister is an interface for listing available MCP spec versions
type VersionLister interface {
	ListVersions() ([]string, error)
}

const ListSpecVersionsToolName = "list_spec_versions"

func GetListSpecVersionsTool() mcp.Tool {
	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
	schemaBytes, _ := json.Marshal(schema)
	return mcp.NewToolWithRawSchema(ListSpecVersionsToolName, "List available MCP specification versions. Use this when users ask about MCP specs, what MCP versions exist, what specifications are available, or want to know which MCP versions they can validate against.", schemaBytes)
}

func HandleListSpecVersions(versionLister VersionLister, args any) ([]mcp.Content, error) {
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
