package prompts

import (
	"context"
	"fmt"

	"github.com/carlisia/mcp-factcheck/pkg/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
)

// Service provides prompt functionality with a clean interface
type Service struct {
	registry *Registry
}

// NewService creates a new prompt service with all standard prompts registered
func NewService() (*Service, error) {
	registry := NewRegistry()
	
	// Register all standard prompts
	prompts := []func() (Prompt, error){
		NewMigrateContentPrompt,
	}
	
	for _, promptFactory := range prompts {
		prompt, err := promptFactory()
		if err != nil {
			return nil, fmt.Errorf("failed to create prompt: %w", err)
		}
		
		if err := registry.Register(prompt); err != nil {
			return nil, fmt.Errorf("failed to register prompt %s: %w", prompt.Name(), err)
		}
	}
	
	return &Service{
		registry: registry,
	}, nil
}

// ListPrompts handles the prompts/list request
func (s *Service) ListPrompts(ctx context.Context) (*mcp.ListPromptsResult, error) {
	log := logger.WithRequestID(ctx)
	log.Info("Processing prompts/list request")
	
	result, err := s.registry.ListForMCP()
	if err != nil {
		log.Error("Failed to list prompts", zap.Error(err))
		return nil, fmt.Errorf("failed to list prompts: %w", err)
	}
	
	log.Info("Prompts list retrieved successfully", 
		zap.Int("prompt_count", len(result.Prompts)))
	
	return result, nil
}

// GetPrompt handles the prompts/get request
func (s *Service) GetPrompt(ctx context.Context, name string, rawArgs any) (*mcp.GetPromptResult, error) {
	log := logger.WithRequestID(ctx)
	log.Info("Processing prompts/get request", 
		zap.String("prompt_name", name))
	
	// Convert arguments to our type-safe Arguments type
	args, err := s.convertArguments(rawArgs)
	if err != nil {
		log.Error("Failed to convert arguments", 
			zap.String("prompt_name", name),
			zap.Error(err))
		return nil, err
	}
	
	log.Debug("Prompt arguments converted", 
		zap.String("prompt_name", name),
		zap.Int("arg_count", len(args)))
	
	// Render the prompt
	result, err := s.registry.RenderPrompt(ctx, name, args)
	if err != nil {
		log.Error("Failed to render prompt", 
			zap.String("prompt_name", name),
			zap.Error(err))
		return nil, fmt.Errorf("failed to render prompt '%s': %w", name, err)
	}
	
	log.Info("Prompt rendered successfully", 
		zap.String("prompt_name", name),
		zap.Int("message_count", len(result.Messages)))
	
	return result, nil
}

// convertArguments converts raw arguments to our Arguments type
func (s *Service) convertArguments(rawArgs any) (Arguments, error) {
	if rawArgs == nil {
		return make(Arguments), nil
	}
	
	// Handle different argument types from MCP library
	switch args := rawArgs.(type) {
	case map[string]any:
		return Arguments(args), nil
	case map[string]string:
		// Convert map[string]string to map[string]any
		result := make(Arguments)
		for k, v := range args {
			result[k] = v
		}
		return result, nil
	case Arguments:
		return args, nil
	default:
		return nil, fmt.Errorf("unsupported arguments type: %T", rawArgs)
	}
}