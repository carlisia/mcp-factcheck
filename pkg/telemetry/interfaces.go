package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Provider abstracts telemetry provider implementations
type Provider interface {
	// StartSpan starts a new span with the given name and attributes
	StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span)
	
	// Shutdown gracefully shuts down the provider
	Shutdown(ctx context.Context) error
}

// Middleware represents telemetry middleware for MCP tool calls
type Middleware interface {
	// WrapToolHandler wraps an MCP tool handler with telemetry
	WrapToolHandler(toolName string, handler ToolHandler) ToolHandler
}

// ToolHandler represents an MCP tool handler function
type ToolHandler func(context.Context, any) (any, error)

// SpanBuilder provides a fluent interface for creating spans with telemetry
type SpanBuilder interface {
	// WithKind sets the OpenInference span kind (LLM, RETRIEVER, EMBEDDING, TOOL, CHAIN)
	WithKind(kind string) SpanBuilder
	
	// WithInput sets input content and metadata
	WithInput(content string, mimeType string) SpanBuilder
	
	// WithOutput sets output content and metadata  
	WithOutput(content string, mimeType string) SpanBuilder
	
	// WithTokens sets token count information
	WithTokens(prompt, completion, total int) SpanBuilder
	
	// WithModel sets model information
	WithModel(name, system, provider string) SpanBuilder
	
	// WithRetrieval sets retrieval-specific attributes
	WithRetrieval(query string, topK int, documents []RetrievalDocument) SpanBuilder
	
	// WithTool sets tool-specific attributes
	WithTool(name, description string, parameters any) SpanBuilder
	
	// WithCustom adds custom attributes
	WithCustom(attrs ...attribute.KeyValue) SpanBuilder
	
	// Start creates and starts the span
	Start(ctx context.Context, name string) (context.Context, trace.Span)
}

// RetrievalDocument represents a document returned from retrieval
type RetrievalDocument struct {
	ID       string         `json:"id"`
	Score    float64        `json:"score"`
	Content  string         `json:"content"`
	Metadata map[string]any `json:"metadata"`
}