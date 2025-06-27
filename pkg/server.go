package pkg

import (
	"context"
	"fmt"
	"time"

	"github.com/carlisia/mcp-factcheck/embedding"
	mcpembedding "github.com/carlisia/mcp-factcheck/internal/embedding"
	"github.com/carlisia/mcp-factcheck/pkg/observability"
	"github.com/carlisia/mcp-factcheck/pkg/spec"
	"github.com/carlisia/mcp-factcheck/pkg/validator"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// FactCheckServer wraps the official MCP server with fact-check specific functionality
type FactCheckServer struct {
	vectorDB  *mcpembedding.VectorDB
	generator *embedding.Generator
	mcpServer *server.MCPServer
	observer  observability.Observer
	wrapper   observability.ToolWrapper
}

// NewFactCheckServer creates a new fact-check server instance using the official MCP library
func NewFactCheckServer(dataDir string, observer observability.Observer) (*FactCheckServer, error) {
	vectorDB := mcpembedding.NewVectorDB(dataDir)
	
	generator, err := embedding.NewGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding generator: %w", err)
	}

	// Create official MCP server
	mcpServer := server.NewMCPServer(
		"mcp-factcheck-server",
		"0.1.0",
	)

	// Use no-op observer if none provided
	if observer == nil {
		observer = observability.NoOpObserver{}
	}

	// Create tool wrapper for observability
	var wrapper observability.ToolWrapper = observability.NoOpWrapper{}
	if observer != nil {
		wrapper = NewObservabilityWrapper(observer)
	}

	factCheckServer := &FactCheckServer{
		vectorDB:  vectorDB,
		generator: generator,
		mcpServer: mcpServer,
		observer:  observer,
		wrapper:   wrapper,
	}

	// Register tools with the official MCP server
	factCheckServer.registerTools()

	return factCheckServer, nil
}

// NewObservabilityWrapper creates a tool wrapper that records interactions
func NewObservabilityWrapper(observer observability.Observer) observability.ToolWrapper {
	return &observabilityWrapper{observer: observer}
}

type observabilityWrapper struct {
	observer observability.Observer
}

func (w *observabilityWrapper) WrapHandler(toolName string, originalHandler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()
		
		// Call original handler and record interaction
		result, err := originalHandler(ctx, req)
		
		// Record the interaction
		interaction := observability.ToolInteraction{
			ToolName:     toolName,
			Arguments:    req.Params.Arguments,
			Timestamp:    start,
			ProcessingMs: time.Since(start).Milliseconds(),
		}
		
		if err != nil {
			interaction.Error = err.Error()
		} else if result != nil {
			interaction.Response = result.Content
		}
		
		w.observer.RecordInteraction(interaction)
		
		return result, err
	}
}

// registerTools registers all fact-check tools with the MCP server
func (s *FactCheckServer) registerTools() {
	// Create handlers with optional debug wrapping
	validateContentHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		content, err := validator.HandleValidateContent(s.vectorDB, s.generator, req.Params.Arguments)
		if err != nil {
			return nil, err
		}
		return &mcp.CallToolResult{Content: content}, nil
	}

	validateCodeHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		content, err := validator.HandleValidateCode(s.vectorDB, s.generator, req.Params.Arguments)
		if err != nil {
			return nil, err
		}
		return &mcp.CallToolResult{Content: content}, nil
	}

	searchSpecHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		content, err := spec.HandleSearchSpec(s.vectorDB, s.generator, req.Params.Arguments)
		if err != nil {
			return nil, err
		}
		return &mcp.CallToolResult{Content: content}, nil
	}

	listVersionsHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		content, err := spec.HandleListSpecVersions(s.vectorDB, req.Params.Arguments)
		if err != nil {
			return nil, err
		}
		return &mcp.CallToolResult{Content: content}, nil
	}

	// Wrap handlers with observability
	validateContentHandler = s.wrapper.WrapHandler("validate_content", validateContentHandler)
	validateCodeHandler = s.wrapper.WrapHandler("validate_code", validateCodeHandler)
	searchSpecHandler = s.wrapper.WrapHandler("search_spec", searchSpecHandler)
	listVersionsHandler = s.wrapper.WrapHandler("list_spec_versions", listVersionsHandler)

	// Register tools with wrapped handlers
	s.mcpServer.AddTool(validator.GetValidateContentTool(), validateContentHandler)
	s.mcpServer.AddTool(validator.GetValidateCodeTool(), validateCodeHandler)
	s.mcpServer.AddTool(spec.GetSearchSpecTool(), searchSpecHandler)
	s.mcpServer.AddTool(spec.GetListSpecVersionsTool(), listVersionsHandler)
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