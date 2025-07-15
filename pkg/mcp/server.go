// Package mcp provides the MCP fact-check server implementation.
// It coordinates the MCP tools for validating content and code against MCP specifications.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	stdlog "log"
	"os"

	"github.com/carlisia/mcp-factcheck/internal/prompts/migrate"
	"github.com/carlisia/mcp-factcheck/internal/storage"
	"github.com/carlisia/mcp-factcheck/internal/tools/list"
	"github.com/carlisia/mcp-factcheck/internal/tools/search"
	"github.com/carlisia/mcp-factcheck/internal/tools/validation"
	"github.com/carlisia/mcp-factcheck/pkg/llm"
	"github.com/carlisia/mcp-factcheck/pkg/logger"
	"github.com/carlisia/mcp-factcheck/pkg/mcp/prompts"
	"github.com/carlisia/mcp-factcheck/pkg/mcp/tools"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const (
	// serverName is the name of the MCP server
	serverName = "mcp-factcheck-server"
	// serverVersion is the version of the MCP server
	serverVersion = "0.1.0"
	// logPrefix is the prefix for log messages
	logPrefix = "[mcp-factcheck] "
)

// TelemetryProvider is an alias for the logger telemetry provider
type TelemetryProvider = logger.TelemetryProvider

// Telemetry explicitly defines the telemetry methods required
type Telemetry interface {
	StartToolSpan(ctx context.Context, name string) (context.Context, trace.Span)
}

// FactCheckServer wraps the actual MCP server with fact-check specific functionality.
// It provides MCP tools for validating content and code against MCP specifications
// using semantic search and LLM-based fact-checking.
type FactCheckServer struct {
	mcpServer     *server.MCPServer
	toolHandlers  map[string]ToolHandler
	promptHandler PromptHandler
	telemetry     Telemetry
}

// ToolHandler is a function that handles tool invocations
type ToolHandler func(ctx context.Context, args any) ([]mcp.Content, error)

// PromptHandler is a function that handles prompt requests
type PromptHandler func(ctx context.Context, name string, args map[string]string) (*mcp.GetPromptResult, error)

// NewFactCheckServer creates a new fact-check server instance.
// It initializes the MCP server with all available fact-checking tools.
// The toolHandlers map should contain handlers for each tool, and promptHandler
// should handle prompt requests.
func NewFactCheckServer(toolHandlers map[string]ToolHandler, promptHandler PromptHandler, telemetry Telemetry) (*FactCheckServer, error) {

	mcpServer := server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
		server.WithRecovery(),
	)

	// Default to no-op telemetry if not provided
	if telemetry == nil {
		telemetry = logger.NewNoOpTelemetryProvider()
	}

	factCheckServer := &FactCheckServer{
		mcpServer:     mcpServer,
		toolHandlers:  toolHandlers,
		promptHandler: promptHandler,
		telemetry:     telemetry,
	}

	// Register all fact-checking tools
	if err := factCheckServer.registerTools(); err != nil {
		return nil, fmt.Errorf("failed to register tools: %w", err)
	}

	// Register all prompts
	if err := factCheckServer.registerPrompts(); err != nil {
		return nil, fmt.Errorf("failed to register prompts: %w", err)
	}

	return factCheckServer, nil
}

// registerTools registers all fact-check tools with the MCP server
func (s *FactCheckServer) registerTools() error {
	log := logger.Get()

	// Map tools to their handlers using shared constants
	toolRegistrations := []struct {
		tool mcp.Tool
		name string
	}{
		{tools.ClaimsTool(), validation.MCPClaimsToolName},
		{tools.QuickClaimTool(), validation.MCPQuickClaimToolName},
		{tools.SearchSpecTool(), search.SpecDBToolName},
		{tools.ListSpecVersionsTool(), list.SpecVersionsToolName},
	}

	for _, reg := range toolRegistrations {
		s.mcpServer.AddTool(reg.tool, s.createMCPHandler(reg.name))
		log.Debug("Registered tool", zap.String("name", reg.name))
	}

	return nil
}

// createMCPHandler creates an MCP-compatible handler from a ToolHandler
func (s *FactCheckServer) createMCPHandler(toolName string) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Start telemetry span
		ctx, span := s.telemetry.StartToolSpan(ctx, toolName)
		defer span.End()

		// Add tool arguments as span attributes
		if args, ok := req.Params.Arguments.(map[string]any); ok {
			// Set input.value as JSON representation of arguments
			inputJSON, err := json.Marshal(args)
			if err != nil {
				logger.SetSpanAttributes(ctx, logger.Attribute("input.value", fmt.Sprintf("error marshaling: %v", err)))
			} else {
				logger.SetSpanAttributes(ctx, logger.Attribute("input.value", string(inputJSON)))
			}

			// Also set individual arguments for easier filtering
			for key, value := range args {
				logger.SetSpanAttributes(ctx, logger.Attribute(fmt.Sprintf("tool.argument.%s", key), value))
			}
		}

		handler, exists := s.toolHandlers[toolName]
		if !exists {
			err := fmt.Errorf("no handler registered for tool: %s", toolName)
			logger.RecordError(ctx, err)
			return mcp.NewToolResultError(err.Error()), nil
		}

		content, err := handler(ctx, req.Params.Arguments)
		if err != nil {
			logger.RecordError(ctx, err)
			logger.SetSpanAttributes(ctx, logger.Attribute("tool.error", true))
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Add success attributes and output
		logger.SetSpanAttributes(ctx,
			logger.Attribute("tool.success", true),
			logger.Attribute("tool.response_count", len(content)),
		)

		// Set output.value as the content
		if len(content) > 0 {
			// For single content item, try to extract text
			if len(content) == 1 {
				if textContent, ok := mcp.AsTextContent(content[0]); ok {
					logger.SetSpanAttributes(ctx, logger.Attribute("output.value", textContent.Text))
				} else {
					logger.SetSpanAttributes(ctx, logger.Attribute("output.value", "[non-text content]"))
				}
			} else {
				// For multiple items, create a summary
				outputSummary := fmt.Sprintf("[%d content items returned]", len(content))
				logger.SetSpanAttributes(ctx, logger.Attribute("output.value", outputSummary))
			}
		}

		return &mcp.CallToolResult{Content: content}, nil
	}
}

// registerPrompts registers all prompts with the MCP server
func (s *FactCheckServer) registerPrompts() error {
	log := logger.Get()

	// Map prompts to their handlers using shared constants
	promptRegistrations := []struct {
		prompt mcp.Prompt
		name   string
	}{
		{prompts.MigrateMCPContentPrompt(), migrate.MCPContentPromptName},
	}

	for _, reg := range promptRegistrations {
		s.mcpServer.AddPrompt(s.createPromptFromDefinition(reg.prompt), s.createMCPPromptHandler(reg.name))
		log.Debug("Registered prompt", zap.String("name", reg.name))
	}

	return nil
}

// createPromptFromDefinition creates an MCP prompt from the prompt definition
func (s *FactCheckServer) createPromptFromDefinition(prompt mcp.Prompt) mcp.Prompt {
	opts := []mcp.PromptOption{
		mcp.WithPromptDescription(prompt.Description),
	}

	// Add arguments
	for _, arg := range prompt.Arguments {
		if arg.Required {
			opts = append(opts, mcp.WithArgument(arg.Name, mcp.ArgumentDescription(arg.Description), mcp.RequiredArgument()))
		} else {
			opts = append(opts, mcp.WithArgument(arg.Name, mcp.ArgumentDescription(arg.Description)))
		}
	}

	return mcp.NewPrompt(prompt.Name, opts...)
}

// createMCPPromptHandler creates an MCP-compatible prompt handler
func (s *FactCheckServer) createMCPPromptHandler(promptName string) func(context.Context, mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		if s.promptHandler == nil {
			return nil, fmt.Errorf("no prompt handler configured")
		}
		return s.promptHandler(ctx, promptName, request.Params.Arguments)
	}
}

// Run starts the MCP server using stdio transport
func (s *FactCheckServer) Run() error {
	log := logger.Get()
	log.Info("Starting MCP server on stdio")

	// Create stderr logger for MCP server errors (stdio transport requires no stdout output)
	stderrLogger := stdlog.New(os.Stderr, logPrefix, stdlog.LstdFlags)

	// Start MCP server on stdio transport
	err := server.ServeStdio(s.mcpServer, server.WithErrorLogger(stderrLogger))

	if err != nil {
		log.Error("Server exited with error", zap.Error(err))
	} else {
		log.Info("Server exited normally")
	}

	return err
}

// initializeTelemetry creates the appropriate telemetry provider based on input
func initializeTelemetry(telemetryProvider any) Telemetry {
	log := logger.Get()

	if telemetryProvider != nil {
		log.Debug("Using provided telemetry provider")
		return logger.NewTelemetryProvider(serverName)
	}

	log.Debug("Using no-op telemetry provider")
	return logger.NewNoOpTelemetryProvider()
}

// NewFactCheckServerWithDependencies creates a new fact-check server with all dependencies wired up.
// This is a convenience function that creates the storage, embedding service, and handlers.
// The dataDir should point to the directory containing the vector database files.
func NewFactCheckServerWithDependencies(dataDir string, telemetryProvider any, telemetryMiddleware any) (*FactCheckServer, error) {
	// Creates a server with default LLM provider (OpenAI)
	return NewFactCheckServerWithConfig(dataDir, llm.Config{Type: llm.OpenAI}, telemetryProvider, telemetryMiddleware)
}

// NewFactCheckServerWithConfig creates a new fact-check server with specific LLM configuration.
// This allows choosing between different LLM providers (OpenAI, Anthropic, Gemini).
func NewFactCheckServerWithConfig(dataDir string, llmConfig llm.Config, telemetryProvider any, telemetryMiddleware any) (*FactCheckServer, error) {
	// Create storage - use embedded data if dataDir is empty
	var vectorDB *storage.VectorDB
	if dataDir == "" {
		vectorDB = storage.NewEmbeddedVectorDB()
	} else {
		vectorDB = storage.NewVectorDB(dataDir)
	}

	// Set up telemetry
	telemetry := initializeTelemetry(telemetryProvider)

	// Get logger for this function
	log := logger.Get()

	// Create LLM client - requires concrete TelemetryProvider type
	var llmTelemetry *logger.TelemetryProvider
	if tp, ok := telemetry.(*logger.TelemetryProvider); ok {
		llmTelemetry = tp
	}
	llmClient, err := llm.NewWithProvider(llmConfig, llmTelemetry)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	// Fetch handlers with pre-defined dependencies
	handler := handlerSpec(vectorDB, llmClient)

	// Map tool names to their handler functions
	toolHandlers := map[string]ToolHandler{
		validation.MCPClaimsToolName:     handler.claimsHandlerSpec(),
		validation.MCPQuickClaimToolName: handler.quickClaimHandlerSpec(),
		search.SpecDBToolName:            handler.searchSpecHandlerSpec(),
		list.SpecVersionsToolName:        handler.listSpecVersionsHandlerSpec(),
	}

	log.Info("Created tool handlers",
		zap.Int("count", len(toolHandlers)),
		zap.String("llm_provider", string(llmConfig.Type)))

	// Create prompt handler factory
	promptHandlerFactory := promptHandlerSpec()

	// Map prompt names to their handler functions
	promptHandlers := map[string]func(context.Context, map[string]string) (*mcp.GetPromptResult, error){
		migrate.MCPContentPromptName: promptHandlerFactory.migrateContentHandlerSpec(),
	}

	// Create unified prompt handler
	promptHandler := func(ctx context.Context, name string, args map[string]string) (*mcp.GetPromptResult, error) {
		handler, exists := promptHandlers[name]
		if !exists {
			return nil, fmt.Errorf("unknown prompt: %s", name)
		}
		return handler(ctx, args)
	}

	return NewFactCheckServer(toolHandlers, promptHandler, telemetry)
}
