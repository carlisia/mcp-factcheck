package pkg

import (
	"context"
	"fmt"

	"github.com/carlisia/mcp-factcheck/embedding"
	mcpembedding "github.com/carlisia/mcp-factcheck/internal/embedding"
	"github.com/carlisia/mcp-factcheck/pkg/spec"
	"github.com/carlisia/mcp-factcheck/pkg/telemetry"
	"github.com/carlisia/mcp-factcheck/pkg/validator"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// FactCheckServer wraps the actual MCP server with fact-check specific functionality
type FactCheckServer struct {
	vectorDB   *mcpembedding.VectorDB
	generator  *embedding.Generator
	mcpServer  *server.MCPServer
	provider   any
	middleware any
}

// NewFactCheckServer creates a new fact-check server instance using clean telemetry abstractions
func NewFactCheckServer(dataDir string, provider any, middleware any) (*FactCheckServer, error) {
	vectorDB := mcpembedding.NewVectorDB(dataDir)

	generator, err := embedding.NewGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding generator: %w", err)
	}

	// Create the actual MCP server
	mcpServer := server.NewMCPServer(
		"mcp-factcheck-server",
		"0.1.0",
	)

	// Store provider and middleware as-is (can be nil)

	factCheckServer := &FactCheckServer{
		vectorDB:   vectorDB,
		generator:  generator,
		mcpServer:  mcpServer,
		provider:   provider,
		middleware: middleware,
	}

	// Register tools with the MCP server
	factCheckServer.registerTools()

	return factCheckServer, nil
}

// wrapToolHandler wraps a tool handler with telemetry if middleware is available
func (s *FactCheckServer) wrapToolHandler(toolName string, handler telemetry.ToolHandler) telemetry.ToolHandler {
	if s.middleware != nil {
		if mw, ok := s.middleware.(interface {
			WrapToolHandler(string, telemetry.ToolHandler) telemetry.ToolHandler
		}); ok {
			return mw.WrapToolHandler(toolName, handler)
		}
	}
	return handler
}

// registerTools registers all fact-check tools with the MCP server
func (s *FactCheckServer) registerTools() {
	// Create base tool handlers
	validateContentHandler := telemetry.ToolHandler(func(ctx context.Context, req any) (any, error) {
		return validator.HandleValidateContent(ctx, s.vectorDB, s.generator, req)
	})

	validateCodeHandler := telemetry.ToolHandler(func(ctx context.Context, req any) (any, error) {
		return validator.HandleValidateCode(ctx, s.vectorDB, s.generator, req)
	})

	searchSpecHandler := telemetry.ToolHandler(func(ctx context.Context, req any) (any, error) {
		return spec.HandleSearchSpec(s.vectorDB, s.generator, req)
	})

	listVersionsHandler := telemetry.ToolHandler(func(ctx context.Context, req any) (any, error) {
		return spec.HandleListSpecVersions(s.vectorDB, req)
	})

	// Wrap handlers with telemetry middleware
	validateContentHandler = s.wrapToolHandler("validate_content", validateContentHandler)
	validateCodeHandler = s.wrapToolHandler("validate_code", validateCodeHandler)
	searchSpecHandler = s.wrapToolHandler("search_spec", searchSpecHandler)
	listVersionsHandler = s.wrapToolHandler("list_spec_versions", listVersionsHandler)

	// Convert to MCP-compatible handlers
	mcpValidateContentHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := validateContentHandler(ctx, req.Params.Arguments)
		if err != nil {
			return nil, err
		}
		if content, ok := result.([]mcp.Content); ok {
			return &mcp.CallToolResult{Content: content}, nil
		}
		return nil, fmt.Errorf("unexpected result type from validate_content")
	}

	mcpValidateCodeHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := validateCodeHandler(ctx, req.Params.Arguments)
		if err != nil {
			return nil, err
		}
		if content, ok := result.([]mcp.Content); ok {
			return &mcp.CallToolResult{Content: content}, nil
		}
		return nil, fmt.Errorf("unexpected result type from validate_code")
	}

	mcpSearchSpecHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := searchSpecHandler(ctx, req.Params.Arguments)
		if err != nil {
			return nil, err
		}
		if content, ok := result.([]mcp.Content); ok {
			return &mcp.CallToolResult{Content: content}, nil
		}
		return nil, fmt.Errorf("unexpected result type from search_spec")
	}

	mcpListVersionsHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := listVersionsHandler(ctx, req.Params.Arguments)
		if err != nil {
			return nil, err
		}
		if content, ok := result.([]mcp.Content); ok {
			return &mcp.CallToolResult{Content: content}, nil
		}
		return nil, fmt.Errorf("unexpected result type from list_spec_versions")
	}

	// Register tools with the MCP server
	s.mcpServer.AddTool(validator.GetValidateContentTool(), mcpValidateContentHandler)
	s.mcpServer.AddTool(validator.GetValidateCodeTool(), mcpValidateCodeHandler)
	s.mcpServer.AddTool(spec.GetSearchSpecTool(), mcpSearchSpecHandler)
	s.mcpServer.AddTool(spec.GetListSpecVersionsTool(), mcpListVersionsHandler)
}

// Run starts the MCP server using stdio transport
func (s *FactCheckServer) Run() error {
	return server.ServeStdio(s.mcpServer)
}

// GetVectorDB returns the vector database instance
func (s *FactCheckServer) GetVectorDB() *mcpembedding.VectorDB {
	return s.vectorDB
}

// GetGenerator returns the embedding generator instance
func (s *FactCheckServer) GetGenerator() *embedding.Generator {
	return s.generator
}

