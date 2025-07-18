package tools

import (
	"context"
	"encoding/json"

	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools/list"
	"github.com/carlisia/mcp-factcheck/internal/storage"
	"github.com/mark3labs/mcp-go/mcp"
)

// ListSpecVersionsTool creates the MCP tool for list_spec_versions from the tool definition
func ListSpecVersionsTool() mcp.Tool {
	def := list.ListSpecVersionsDefinition()

	schemaBytes, err := json.Marshal(def.Schema)
	if err != nil {
		// This should never happen with our static schema
		panic("failed to marshal list spec versions schema: " + err.Error())
	}

	return mcp.NewToolWithRawSchema(def.Name, def.Description, schemaBytes)
}

// HandleListSpecVersions handles the list_spec_versions MCP tool call
func HandleListSpecVersions(ctx context.Context, vectorDB *storage.VectorDB) ([]mcp.Content, error) {
	// Create list function adapter
	listFunc := func() ([]string, error) {
		return vectorDB.ListVersions()
	}

	// Get versions
	versions, err := list.ListVersions(ctx, listFunc)
	if err != nil {
		return nil, err
	}

	// Format results
	formatted := list.FormatResults(versions)

	// Return MCP content
	return []mcp.Content{
		mcp.NewTextContent(formatted),
	}, nil
}
