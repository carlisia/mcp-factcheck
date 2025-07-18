package prompts

import (
	"context"

	"github.com/carlisia/mcp-factcheck/internal/capabilities/prompts/migrate"
	"github.com/mark3labs/mcp-go/mcp"
)

// MigrateMCPContentPrompt creates the MCP prompt for migrate-mcp-content from the prompt definition
func MigrateMCPContentPrompt() mcp.Prompt {
	def := migrate.PromptDefinition()

	// Convert internal arguments to MCP format
	var mcpArgs []mcp.PromptArgument
	for _, arg := range def.Arguments {
		mcpArgs = append(mcpArgs, mcp.PromptArgument{
			Name:        arg.Name,
			Description: arg.Description,
			Required:    arg.Required,
		})
	}

	return mcp.Prompt{
		Name:        def.Name,
		Description: def.Description,
		Arguments:   mcpArgs,
	}
}

// HandleMigrateContent handles the migrate-mcp-content prompt
func HandleMigrateContent(ctx context.Context, args map[string]string) (*mcp.GetPromptResult, error) {
	// Parse and validate arguments
	req, err := migrate.ParseMigrateArgs(args)
	if err != nil {
		return nil, err
	}

	// Render the prompt with the validated request
	return migrate.Render(req)
}
