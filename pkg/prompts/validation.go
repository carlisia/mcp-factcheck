package prompts

import (
	_ "embed"
)

// Template content embedded at compile time
var (
	//go:embed templates/migrate-content.tmpl
	migrateContentTemplate string
)

// NewMigrateContentPrompt creates the MCP content migration prompt
func NewMigrateContentPrompt() (Prompt, error) {
	args := []Argument{
		{
			Name:        "current_version",
			Description: "2025-06-18, 2025-03-26, 2024-11-04, draft (current MCP specification version the content is based on)",
			Required:    true,
			Type:        "string",
		},
		{
			Name:        "target_version",
			Description: "2025-06-18, 2025-03-26, 2024-11-04, draft (target MCP specification version to update content for)",
			Required:    true,
			Type:        "string",
		},
		{
			Name:        "update_scope",
			Description: "critical_only, comprehensive, enhancement_focused",
			Required:    false,
			Type:        "string",
			Default:     "comprehensive",
		},
	}

	return NewBasePrompt(
		"migrate-mcp-content",
		"Update MCP documentation, tutorials, or content to align with a target specification version",
		migrateContentTemplate,
		args,
	)
}
