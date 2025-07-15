package list

import "github.com/carlisia/mcp-factcheck/internal/tools"

// SpecVersionsToolName is the name of the list spec versions tool
const SpecVersionsToolName = "list-spec-versions"

// ListSpecVersionsDefinition returns the tool definition for listing available
// MCP specification versions. This tool helps users discover which spec versions
// are available for validation and search operations.
func ListSpecVersionsDefinition() tools.Definition {
	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}

	description := `List available MCP specification versions.

Use this tool when users:
- Ask about MCP specs or versions
- Want to know what specifications are available
- Need to know which MCP versions they can validate against
- Are unsure which spec version to use

Returns a list of all available specification versions that can be used with other tools.`

	return tools.Definition{
		Name:        SpecVersionsToolName,
		Description: description,
		Schema:      schema,
	}
}
