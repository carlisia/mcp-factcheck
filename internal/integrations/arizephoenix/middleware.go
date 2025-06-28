package arizephoenix

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/carlisia/mcp-factcheck/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
)

// Middleware implements the telemetry.Middleware interface for Arize Phoenix
type Middleware struct {
	provider telemetry.Provider
	config   Config
}

// NewMiddleware creates a new Phoenix telemetry middleware
func NewMiddleware(provider telemetry.Provider, config Config) *Middleware {
	return &Middleware{
		provider: provider,
		config:   config,
	}
}

// WrapToolHandler implements telemetry.Middleware
func (m *Middleware) WrapToolHandler(toolName string, handler telemetry.ToolHandler) telemetry.ToolHandler {
	return func(ctx context.Context, req any) (any, error) {
		// Convert request to JSON for better visibility
		reqJSON, _ := json.Marshal(req)
		requestContent := string(reqJSON)
		
		// Truncate request if too long for Phoenix
		if len(requestContent) > m.config.MaxContentLength {
			requestContent = requestContent[:m.config.MaxContentLength] + "..."
		}

		// Start main tool span with OpenInference attributes
		ctx, span := m.provider.StartSpan(ctx, fmt.Sprintf("mcp.tool.%s", toolName),
			attribute.String("openinference.span.kind", "TOOL"),
			attribute.String("tool.name", toolName),
			attribute.String("tool.description", fmt.Sprintf("MCP tool: %s", toolName)),
			attribute.String("tool.parameters", requestContent),
			attribute.String("input.value", requestContent),
			attribute.String("input.mime_type", "application/json"),
		)
		defer span.End()

		start := time.Now()

		// Call original handler
		result, err := handler(ctx, req)

		duration := time.Since(start)

		// Convert result to JSON for output tracking
		resultJSON, _ := json.Marshal(result)
		resultContent := string(resultJSON)
		
		// Truncate result if too long for Phoenix
		if len(resultContent) > m.config.MaxContentLength {
			resultContent = resultContent[:m.config.MaxContentLength] + "..."
		}

		// Add timing, status, and output attributes
		span.SetAttributes(
			attribute.Int64("tool.duration_ms", duration.Milliseconds()),
			attribute.Bool("tool.success", err == nil),
			attribute.String("output.value", resultContent),
			attribute.String("output.mime_type", "application/json"),
		)

		if err != nil {
			span.SetAttributes(attribute.String("tool.error", err.Error()))
			span.RecordError(err)
		}

		return result, err
	}
}