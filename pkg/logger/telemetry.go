package logger

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
)

// TelemetryProvider wraps telemetry functionality
type TelemetryProvider struct {
	tracer trace.Tracer
	noop   bool
}

// NewTelemetryProvider creates a new telemetry provider
// The actual tracer comes from the global OpenTelemetry provider
// which is configured by the Phoenix integration
func NewTelemetryProvider(serviceName string) *TelemetryProvider {
	return &TelemetryProvider{
		tracer: otel.Tracer(serviceName),
		noop:   false,
	}
}

// NewNoOpTelemetryProvider creates a no-op telemetry provider
// that implements all methods but does nothing
func NewNoOpTelemetryProvider() *TelemetryProvider {
	return &TelemetryProvider{
		tracer: noop.NewTracerProvider().Tracer("noop"),
		noop:   true,
	}
}

// StartSpan starts a new span with the given name and attributes
func (t *TelemetryProvider) StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	ctx, span := t.tracer.Start(ctx, name, trace.WithAttributes(attrs...))
	// Log span creation for debugging
	if !t.noop && span.SpanContext().IsValid() {
		log := Get()
		if log != nil {
			log.Debug("Created span",
				zap.String("span.name", name),
				zap.String("span.trace_id", span.SpanContext().TraceID().String()),
				zap.String("span.span_id", span.SpanContext().SpanID().String()),
			)
		}
	}
	return ctx, span
}

// StartToolSpan creates a span for MCP tool execution
func (t *TelemetryProvider) StartToolSpan(ctx context.Context, toolName string) (context.Context, trace.Span) {
	return t.StartSpan(ctx, fmt.Sprintf("mcp.tool.%s", toolName),
		attribute.String("openinference.span.kind", "TOOL"),
		attribute.String("tool.name", toolName),
	)
}

// StartLLMSpan creates a span for LLM operations
func (t *TelemetryProvider) StartLLMSpan(ctx context.Context, operation string, model string) (context.Context, trace.Span) {
	return t.StartSpan(ctx, fmt.Sprintf("llm.%s", operation),
		attribute.String("openinference.span.kind", "LLM"),
		attribute.String("llm.model_name", model),
		attribute.String("llm.system", "openai"),
		attribute.String("llm.provider", "openai"),
		attribute.String("llm.request_type", "chat"),
	)
}

// StartEmbeddingSpan creates a span for embedding generation
func (t *TelemetryProvider) StartEmbeddingSpan(ctx context.Context, model string, inputLength int) (context.Context, trace.Span) {
	estimatedTokens := inputLength / 4 // Rough estimation

	return t.StartSpan(ctx, "embedding.generation",
		attribute.String("openinference.span.kind", "EMBEDDING"),
		attribute.String("llm.model_name", model),
		attribute.String("llm.system", "openai"),
		attribute.String("llm.provider", "openai"),
		attribute.Int("llm.token_count.prompt", estimatedTokens),
		attribute.Int("llm.token_count.total", estimatedTokens),
		attribute.Int("embedding.content_length", inputLength),
	)
}

// StartRetrievalSpan creates a span for vector search operations
func (t *TelemetryProvider) StartRetrievalSpan(ctx context.Context, specVersion string, topK int) (context.Context, trace.Span) {
	return t.StartSpan(ctx, "vector.search",
		attribute.String("openinference.span.kind", "RETRIEVER"),
		attribute.String("retrieval.spec_version", specVersion),
		attribute.Int("retrieval.top_k", topK),
	)
}

// RecordError records an error on the current span
func RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if span != nil && err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("error.message", err.Error()))
	}
}

// SetSpanAttributes sets attributes on the current span
func SetSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.SetAttributes(attrs...)
	}
}

// Attribute creates an attribute key-value pair
func Attribute(key string, value any) attribute.KeyValue {
	switch v := value.(type) {
	case string:
		return attribute.String(key, v)
	case int:
		return attribute.Int(key, v)
	case int64:
		return attribute.Int64(key, v)
	case float64:
		return attribute.Float64(key, v)
	case bool:
		return attribute.Bool(key, v)
	default:
		return attribute.String(key, fmt.Sprintf("%v", v))
	}
}
