package pkg

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/carlisia/mcp-factcheck/embedding"
	mcpembedding "github.com/carlisia/mcp-factcheck/internal/embedding"
	"github.com/carlisia/mcp-factcheck/pkg/logger"
	"github.com/carlisia/mcp-factcheck/pkg/prompts"
	"github.com/carlisia/mcp-factcheck/pkg/spec"
	"github.com/carlisia/mcp-factcheck/pkg/telemetry"
	"github.com/carlisia/mcp-factcheck/pkg/validator"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// FactCheckServer wraps the actual MCP server with fact-check specific functionality
type FactCheckServer struct {
	vectorDB      *mcpembedding.VectorDB
	generator     *embedding.Generator
	mcpServer     *server.MCPServer
	provider      any
	middleware    any
	promptService *prompts.Service
}

// zapError implements zap.ObjectMarshaler for structured error logging
type zapError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MarshalLogObject implements zap.ObjectMarshaler
func (e zapError) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddInt("code", e.Code)
	enc.AddString("message", e.Message)
	return nil
}

// ServerOption is a function that configures the FactCheckServer
type ServerOption func(*serverConfig)

// serverConfig holds configuration options for the server
type serverConfig struct {
	logMCPMessages bool
	logMCPPayloads bool
}

// WithMCPLogging enables MCP message logging
func WithMCPLogging(logMessages, logPayloads bool) ServerOption {
	return func(cfg *serverConfig) {
		cfg.logMCPMessages = logMessages
		cfg.logMCPPayloads = logPayloads
	}
}

// NewFactCheckServer creates a new fact-check server instance using clean telemetry abstractions
func NewFactCheckServer(dataDir string, provider any, middleware any, opts ...ServerOption) (*FactCheckServer, error) {
	// Apply options
	cfg := &serverConfig{
		logMCPMessages: true,  // Default to true
		logMCPPayloads: false, // Default to false for security
	}
	for _, opt := range opts {
		opt(cfg)
	}
	vectorDB := mcpembedding.NewVectorDB(dataDir)

	generator, err := embedding.NewGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding generator: %w", err)
	}

	// Create prompt service
	promptService, err := prompts.NewService()
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt service: %w", err)
	}

	// Create hooks for MCP message logging
	hooks := &server.Hooks{}

	if cfg.logMCPMessages {
		// Log all incoming requests
		hooks.AddBeforeAny(func(ctx context.Context, id any, method mcp.MCPMethod, message any) {
			log := logger.WithRequestID(ctx)
			fields := []zap.Field{
				zap.String("component", "mcp-factcheck"),
				zap.String("direction", "Client->Server"),
				zap.String("type", "REQUEST"),
				zap.Any("id", id),
				zap.String("jsonrpc", "2.0"),
				zap.String("method", string(method)),
			}
			if cfg.logMCPPayloads {
				fields = append(fields, zap.Any("params", message))
			}
			log.Info("MCP message", fields...)
		})

		// Log all successful responses
		hooks.AddOnSuccess(func(ctx context.Context, id any, method mcp.MCPMethod, message any, result any) {
			log := logger.WithRequestID(ctx)
			fields := []zap.Field{
				zap.String("component", "mcp-factcheck"),
				zap.String("direction", "Server->Client"),
				zap.String("type", "RESPONSE"),
				zap.Any("id", id),
				zap.String("jsonrpc", "2.0"),
				zap.String("method", string(method)),
			}
			if cfg.logMCPPayloads {
				fields = append(fields, zap.Any("result", result))
			}
			log.Info("MCP message", fields...)
		})

		// Log all error responses
		hooks.AddOnError(func(ctx context.Context, id any, method mcp.MCPMethod, message any, err error) {
			log := logger.WithRequestID(ctx)

			// Extract error code if available
			var errorCode int
			var errorMessage string
			// Default to internal error
			errorCode = -32603
			errorMessage = err.Error()

			log.Error("MCP message",
				zap.String("component", "mcp-factcheck"),
				zap.String("direction", "Server->Client"),
				zap.String("type", "ERROR"),
				zap.Any("id", id),
				zap.String("jsonrpc", "2.0"),
				zap.String("method", string(method)),
				zap.Object("error", zapError{Code: errorCode, Message: errorMessage}),
			)
		})

		// Log session lifecycle
		hooks.AddOnRegisterSession(func(ctx context.Context, session server.ClientSession) {
			log := logger.Get()
			log.Info("MCP session",
				zap.String("component", "mcp-factcheck"),
				zap.String("event", "session_start"),
				zap.String("session_id", session.SessionID()),
			)
		})

		hooks.AddOnUnregisterSession(func(ctx context.Context, session server.ClientSession) {
			log := logger.Get()
			log.Info("MCP session",
				zap.String("component", "mcp-factcheck"),
				zap.String("event", "session_end"),
				zap.String("session_id", session.SessionID()),
			)
		})
	}

	// Create the actual MCP server with hooks
	mcpServer := server.NewMCPServer(
		"mcp-factcheck-server",
		"0.1.0",
		server.WithHooks(hooks),
	)

	factCheckServer := &FactCheckServer{
		vectorDB:      vectorDB,
		generator:     generator,
		mcpServer:     mcpServer,
		provider:      provider,
		middleware:    middleware,
		promptService: promptService,
	}

	// Register tools and prompts with the MCP server
	factCheckServer.registerTools()
	factCheckServer.registerPrompts()

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
	// Create base tool handlers with request ID tracking and logging
	checkMCPClaimHandler := telemetry.ToolHandler(func(ctx context.Context, req any) (any, error) {
		// Add request ID to context
		ctx = telemetry.WithRequestID(ctx)

		result, err := validator.HandleCheckMCPClaim(ctx, s.vectorDB, s.generator, req)

		return result, err
	})

	validateCodeHandler := telemetry.ToolHandler(func(ctx context.Context, req any) (any, error) {
		// Add request ID to context
		ctx = telemetry.WithRequestID(ctx)

		result, err := validator.HandleValidateCode(ctx, s.vectorDB, s.generator, req)

		return result, err
	})

	searchSpecHandler := telemetry.ToolHandler(func(ctx context.Context, req any) (any, error) {
		// Add request ID to context
		ctx = telemetry.WithRequestID(ctx)

		result, err := spec.HandleSearchSpecMCP(ctx, s.vectorDB, s.generator, req)

		return result, err
	})

	listVersionsHandler := telemetry.ToolHandler(func(ctx context.Context, req any) (any, error) {
		// Add request ID to context
		ctx = telemetry.WithRequestID(ctx)

		result, err := spec.HandleListSpecVersions(ctx, s.vectorDB, req)

		return result, err
	})

	checkMCPQuickFactHandler := telemetry.ToolHandler(func(ctx context.Context, req any) (any, error) {
		// Add request ID to context
		ctx = telemetry.WithRequestID(ctx)

		result, err := validator.HandleCheckMCPQuickFact(ctx, s.vectorDB, s.generator, req)

		return result, err
	})

	// Wrap handlers with telemetry middleware
	checkMCPClaimHandler = s.wrapToolHandler("check_mcp_claim", checkMCPClaimHandler)
	validateCodeHandler = s.wrapToolHandler("validate_code", validateCodeHandler)
	searchSpecHandler = s.wrapToolHandler("search_spec", searchSpecHandler)
	listVersionsHandler = s.wrapToolHandler("list_spec_versions", listVersionsHandler)
	checkMCPQuickFactHandler = s.wrapToolHandler("check_mcp_quick_fact", checkMCPQuickFactHandler)

	// Convert to MCP-compatible handlers
	mcpCheckMCPClaimHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := checkMCPClaimHandler(ctx, req.Params.Arguments)
		if err != nil {
			return nil, err
		}
		if content, ok := result.([]mcp.Content); ok {
			return &mcp.CallToolResult{Content: content}, nil
		}
		return nil, fmt.Errorf("unexpected result type from check_mcp_claim")
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

	mcpCheckMCPQuickFactHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := checkMCPQuickFactHandler(ctx, req.Params.Arguments)
		if err != nil {
			return nil, err
		}
		if content, ok := result.([]mcp.Content); ok {
			return &mcp.CallToolResult{Content: content}, nil
		}
		return nil, fmt.Errorf("unexpected result type from check_mcp_quick_fact")
	}

	// Register tools with the MCP server
	s.mcpServer.AddTool(validator.GetCheckMCPClaimTool(), mcpCheckMCPClaimHandler)
	s.mcpServer.AddTool(validator.GetValidateCodeTool(), mcpValidateCodeHandler)
	s.mcpServer.AddTool(validator.GetCheckMCPQuickFactTool(), mcpCheckMCPQuickFactHandler)
	s.mcpServer.AddTool(spec.GetSearchSpecTool(), mcpSearchSpecHandler)
	s.mcpServer.AddTool(spec.GetListSpecVersionsTool(), mcpListVersionsHandler)

	// Register prompts with the MCP server
	s.registerPrompts()
}

// registerPrompts registers all prompts with the MCP server using the new service
func (s *FactCheckServer) registerPrompts() {
	// Get all prompts from the service
	listResult, err := s.promptService.ListPrompts(context.Background())
	if err != nil {
		// Log error but don't fail server startup
		log := logger.Get()
		log.Error("Failed to list prompts during registration", zap.Error(err))
		return
	}

	// Register each prompt with the MCP server
	for _, prompt := range listResult.Prompts {
		// Convert our arguments back to MCP format
		var mcpArgs []mcp.PromptOption
		mcpArgs = append(mcpArgs, mcp.WithPromptDescription(prompt.Description))

		for _, arg := range prompt.Arguments {
			argOptions := []mcp.ArgumentOption{
				mcp.ArgumentDescription(arg.Description),
			}
			if arg.Required {
				argOptions = append(argOptions, mcp.RequiredArgument())
			}
			mcpArgs = append(mcpArgs, mcp.WithArgument(arg.Name, argOptions...))
		}

		// Create the MCP prompt and handler
		mcpPrompt := mcp.NewPrompt(prompt.Name, mcpArgs...)
		handler := s.createPromptHandler(prompt.Name)

		// Register with the MCP server
		s.mcpServer.AddPrompt(mcpPrompt, handler)
	}
}

// createPromptHandler creates a prompt handler with logging and request ID tracking
func (s *FactCheckServer) createPromptHandler(promptName string) func(context.Context, mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		// Add request ID to context
		ctx = telemetry.WithRequestID(ctx)

		// Delegate to the prompt service
		result, err := s.promptService.GetPrompt(ctx, promptName, request.Params.Arguments)

		return result, err
	}
}

// Run starts the MCP server using stdio transport
func (s *FactCheckServer) Run() error {
	// Create a logger that discards all output to prevent legacy format logs
	discardLogger := log.New(io.Discard, "", 0)
	return server.ServeStdio(s.mcpServer, server.WithErrorLogger(discardLogger))
}

// GetVectorDB returns the vector database instance
func (s *FactCheckServer) GetVectorDB() *mcpembedding.VectorDB {
	return s.vectorDB
}

// GetGenerator returns the embedding generator instance
func (s *FactCheckServer) GetGenerator() *embedding.Generator {
	return s.generator
}
