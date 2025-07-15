package mcp

import (
	"context"

	"github.com/carlisia/mcp-factcheck/internal/storage"
	"github.com/carlisia/mcp-factcheck/pkg/llm"
	"github.com/carlisia/mcp-factcheck/pkg/mcp/tools"
	"github.com/mark3labs/mcp-go/mcp"
)

// ToolHandlerSpec creates tool handlers with bound dependencies
type ToolHandlerSpec struct {
	vectorDB  *storage.VectorDB
	llmClient *llm.Client
}

// handlerSpec creates a new handler factory with the given dependencies
func handlerSpec(vectorDB *storage.VectorDB, llmClient *llm.Client) *ToolHandlerSpec {
	return &ToolHandlerSpec{
		vectorDB:  vectorDB,
		llmClient: llmClient,
	}
}

// claimsHandlerSpec returns a handler for claims validation with bound dependencies
func (f *ToolHandlerSpec) claimsHandlerSpec() func(context.Context, any) ([]mcp.Content, error) {
	return func(ctx context.Context, args any) ([]mcp.Content, error) {
		return tools.HandleClaimsValidation(ctx, f.vectorDB, f.llmClient, args)
	}
}

// quickClaimHandlerSpec returns a handler for quick claim validation with bound dependencies
func (f *ToolHandlerSpec) quickClaimHandlerSpec() func(context.Context, any) ([]mcp.Content, error) {
	return func(ctx context.Context, args any) ([]mcp.Content, error) {
		return tools.HandleQuickClaimValidation(ctx, f.vectorDB, f.llmClient, args)
	}
}

// searchSpecHandlerSpec returns a handler for spec search with bound dependencies
func (f *ToolHandlerSpec) searchSpecHandlerSpec() func(context.Context, any) ([]mcp.Content, error) {
	return func(ctx context.Context, args any) ([]mcp.Content, error) {
		return tools.HandleSearchSpec(ctx, f.vectorDB, f.llmClient, args)
	}
}

// listSpecVersionsHandlerSpec returns a handler for listing spec versions with bound dependencies
func (f *ToolHandlerSpec) listSpecVersionsHandlerSpec() func(context.Context, any) ([]mcp.Content, error) {
	return func(ctx context.Context, args any) ([]mcp.Content, error) {
		// This handler doesn't use args, but we accept it for interface consistency
		return tools.HandleListSpecVersions(ctx, f.vectorDB)
	}
}
