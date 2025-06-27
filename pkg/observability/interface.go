package observability

import (
	"context"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// ToolInteraction represents a single tool invocation
type ToolInteraction struct {
	ID           string        `json:"id"`
	Timestamp    time.Time     `json:"timestamp"`
	ToolName     string        `json:"tool_name"`
	Arguments    any           `json:"arguments"`
	Response     []mcp.Content `json:"response"`
	ProcessingMs int64         `json:"processing_ms"`
	Error        string        `json:"error,omitempty"`
}

// Observer is an optional interface for observing MCP tool interactions
type Observer interface {
	// RecordInteraction records a tool interaction
	RecordInteraction(interaction ToolInteraction)
}

// ToolWrapper wraps MCP tool handlers to capture interactions
type ToolWrapper interface {
	// WrapHandler wraps an MCP tool handler to record interactions
	WrapHandler(toolName string, handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
}

// NoOpObserver implements Observer but does nothing (null object pattern)
type NoOpObserver struct{}

func (n NoOpObserver) RecordInteraction(interaction ToolInteraction) {
	// Do nothing
}

// NoOpWrapper implements ToolWrapper but does nothing
type NoOpWrapper struct{}

func (n NoOpWrapper) WrapHandler(toolName string, handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return handler // Return unwrapped handler
}