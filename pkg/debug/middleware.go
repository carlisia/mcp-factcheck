package debug

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
)

// ToolWrapper wraps MCP tool handlers to capture debug information
type ToolWrapper struct {
	debugServer *DebugServer
	ipcClient   *IPCClient
	enabled     bool
}

// NewToolWrapper creates a new tool wrapper for debug capture
func NewToolWrapper(debugServer *DebugServer, enabled bool) *ToolWrapper {
	return &ToolWrapper{
		debugServer: debugServer,
		enabled:     enabled,
	}
}

// NewToolWrapperWithIPC creates a new tool wrapper using IPC for debug capture
func NewToolWrapperWithIPC(ipcClient *IPCClient, enabled bool) *ToolWrapper {
	return &ToolWrapper{
		ipcClient: ipcClient,
		enabled:   enabled,
	}
}

// WrapHandler wraps an MCP tool handler to capture debug information
func (tw *ToolWrapper) WrapHandler(toolName string, originalHandler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !tw.enabled {
		return originalHandler
	}
	

	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTime := time.Now()
		
		// Create interaction record
		interaction := DebugInteraction{
			ID:        uuid.New().String(),
			Timestamp: startTime,
			ToolName:  toolName,
			Arguments: req.Params.Arguments,
		}

		// Call original handler
		result, err := originalHandler(ctx, req)
		
		// Calculate processing time
		interaction.ProcessingMs = time.Since(startTime).Milliseconds()

		// Handle error case
		if err != nil {
			interaction.Error = err.Error()
			if tw.debugServer != nil {
				tw.debugServer.AddInteraction(interaction)
			} else if tw.ipcClient != nil {
				tw.ipcClient.SendInteraction(interaction)
			}
			return result, err
		}

		// Capture response
		if result != nil {
			interaction.Response = tw.convertContentToSerializable(result.Content)
			interaction.TokenCount = tw.calculateTokens(req.Params.Arguments, result.Content)
		}

		// Record interaction
		if tw.debugServer != nil {
			tw.debugServer.AddInteraction(interaction)
		} else if tw.ipcClient != nil {
			tw.ipcClient.SendInteraction(interaction)
		}

		return result, err
	}
}

// calculateTokens estimates token usage (rough approximation)
func (tw *ToolWrapper) calculateTokens(args any, response []mcp.Content) TokenInfo {
	// Rough token estimation: ~4 characters per token
	inputTokens := tw.estimateTokens(args)
	outputTokens := 0
	
	for _, content := range response {
		if textContent, ok := content.(*mcp.TextContent); ok {
			outputTokens += len(textContent.Text) / 4
		}
	}

	totalTokens := inputTokens + outputTokens

	// Estimate savings (this would be much higher with old verbose responses)
	// Conservative estimate: old responses were 5-10x larger
	estimatedOldTokens := outputTokens * 7 // Conservative 7x multiplier
	savings := estimatedOldTokens - outputTokens
	if savings < 0 {
		savings = 0
	}

	return TokenInfo{
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  totalTokens,
		Savings:      savings,
	}
}

// estimateTokens roughly estimates tokens from any data structure
func (tw *ToolWrapper) estimateTokens(data any) int {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return 0
	}
	return len(jsonBytes) / 4 // Rough approximation
}

// convertContentToSerializable converts mcp.Content interface to serializable data
func (tw *ToolWrapper) convertContentToSerializable(content []mcp.Content) []any {
	var result []any
	
	for _, c := range content {
		if textContent, ok := c.(*mcp.TextContent); ok {
			result = append(result, map[string]any{
				"type": "text",
				"text": textContent.Text,
			})
		} else {
			// Convert any other content type to JSON-serializable form
			jsonBytes, _ := json.Marshal(c)
			var genericContent map[string]any
			json.Unmarshal(jsonBytes, &genericContent)
			result = append(result, genericContent)
		}
	}
	
	return result
}

// FormatResponsePreview creates a human-readable preview of what's sent to LLM
func FormatResponsePreview(response []mcp.Content) string {
	var parts []string
	
	for i, content := range response {
		if textContent, ok := content.(*mcp.TextContent); ok {
			preview := textContent.Text
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			parts = append(parts, fmt.Sprintf("Content %d: %s", i+1, preview))
		} else {
			parts = append(parts, fmt.Sprintf("Content %d: [Non-text content]", i+1))
		}
	}
	
	return strings.Join(parts, "\n\n")
}