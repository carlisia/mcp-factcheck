package mcp

import (
	"context"

	"github.com/carlisia/mcp-factcheck/pkg/mcp/prompts"
	"github.com/mark3labs/mcp-go/mcp"
)

// PromptHandlerSpec creates prompt handlers with bound dependencies
type PromptHandlerSpec struct{}

// promptHandlerSpec creates a new handler factory with the given dependencies
func promptHandlerSpec() *PromptHandlerSpec {
	return &PromptHandlerSpec{}
}

// migrateContentHandlerSpec returns a handler for migrate content prompt
func (f *PromptHandlerSpec) migrateContentHandlerSpec() func(context.Context, map[string]string) (*mcp.GetPromptResult, error) {
	return func(ctx context.Context, args map[string]string) (*mcp.GetPromptResult, error) {
		// For now, we import prompts package, but this could be refactored
		// to avoid circular dependencies if needed
		return prompts.HandleMigrateContent(ctx, args)
	}
}

