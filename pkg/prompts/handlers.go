package prompts

import (
	"context"
	"fmt"

	"github.com/carlisia/mcp-factcheck/pkg/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
)

// HandleListPrompts handles the prompts/list request
func HandleListPrompts(ctx context.Context, args any) (*mcp.ListPromptsResult, error) {
	// Get structured logger with request ID
	log := logger.WithRequestID(ctx)
	
	log.Info("Processing prompts/list request")
	
	result, err := GetPromptsListResponse()
	if err != nil {
		log.Error("Failed to get prompts list", zap.Error(err))
		return nil, fmt.Errorf("failed to get prompts list: %w", err)
	}
	
	log.Info("Prompts list retrieved successfully", 
		zap.Int("prompt_count", len(result.Prompts)))
	
	return result, nil
}

// HandleGetPrompt handles the prompts/get request
func HandleGetPrompt(ctx context.Context, args any) (*mcp.GetPromptResult, error) {
	// Get structured logger with request ID
	log := logger.WithRequestID(ctx)
	
	params, ok := args.(map[string]any)
	if !ok {
		log.Error("Invalid arguments type for prompts/get", 
			zap.String("expected", "map[string]any"),
			zap.String("actual", fmt.Sprintf("%T", args)))
		return nil, fmt.Errorf("arguments must be a map")
	}
	
	// Extract prompt name
	name, ok := params["name"].(string)
	if !ok {
		log.Error("Invalid or missing name parameter", 
			zap.String("expected", "string"),
			zap.Any("value", params["name"]))
		return nil, fmt.Errorf("name parameter must be a string")
	}
	
	// Extract arguments (optional)
	var promptArgs map[string]interface{}
	if args, exists := params["arguments"]; exists {
		// Handle different argument types from MCP library
		switch argsMap := args.(type) {
		case map[string]interface{}:
			promptArgs = argsMap
		case map[string]string:
			// Convert map[string]string to map[string]interface{}
			promptArgs = make(map[string]interface{})
			for k, v := range argsMap {
				promptArgs[k] = v
			}
		default:
			log.Error("Invalid arguments parameter type", 
				zap.String("expected", "map[string]interface{} or map[string]string"),
				zap.String("actual", fmt.Sprintf("%T", args)))
			return nil, fmt.Errorf("arguments parameter must be a map")
		}
	} else {
		promptArgs = make(map[string]interface{})
	}
	
	log.Info("Processing prompts/get request", 
		zap.String("prompt_name", name),
		zap.Int("arg_count", len(promptArgs)))
	
	// Render the prompt
	result, err := RenderPrompt(name, promptArgs)
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