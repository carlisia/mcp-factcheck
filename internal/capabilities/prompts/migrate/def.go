package migrate

import (
	"github.com/carlisia/mcp-factcheck/internal/capabilities"
)

// MCPContentPromptName is the name of the migrate content prompt
const MCPContentPromptName = "migrate-mcp-content"

// Definition represents the migrate content prompt definition
type Definition struct {
	Name        string
	Description string
	Arguments   []Argument
}

// Argument represents a prompt argument
type Argument struct {
	Name        string
	Description string
	Required    bool
}

// PromptDefinition returns the prompt definition for migrating MCP content
// between different specification versions. This prompt helps users update
// their MCP documentation to align with newer specification versions.
func PromptDefinition() Definition {
	return Definition{
		Name:        MCPContentPromptName,
		Description: "Update MCP documentation, tutorials, or content to align with a target specification version",
		Arguments: []Argument{
			{
				Name:        "current_version",
				Description: capabilities.Versions.ForDescription() + " (current MCP specification version the content is based on)",
				Required:    true,
			},
			{
				Name:        "target_version",
				Description: capabilities.Versions.ForDescription() + " (target MCP specification version to update content for)",
				Required:    true,
			},
			{
				Name:        "update_scope",
				Description: "comprehensive (default), critical_only, enhancement_focused",
				Required:    false,
			},
		},
	}
}

