package telemetry

import (
	"context"
	"encoding/json"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// spanBuilder implements SpanBuilder interface
type spanBuilder struct {
	attributes []attribute.KeyValue
}

// NewSpanBuilder creates a new span builder
func NewSpanBuilder() SpanBuilder {
	return &spanBuilder{
		attributes: make([]attribute.KeyValue, 0),
	}
}

func (b *spanBuilder) WithKind(kind string) SpanBuilder {
	b.attributes = append(b.attributes, attribute.String("openinference.span.kind", kind))
	return b
}

func (b *spanBuilder) WithInput(content string, mimeType string) SpanBuilder {
	b.attributes = append(b.attributes,
		attribute.String("input.value", content),
		attribute.String("input.mime_type", mimeType),
	)
	return b
}

func (b *spanBuilder) WithOutput(content string, mimeType string) SpanBuilder {
	b.attributes = append(b.attributes,
		attribute.String("output.value", content),
		attribute.String("output.mime_type", mimeType),
	)
	return b
}

func (b *spanBuilder) WithTokens(prompt, completion, total int) SpanBuilder {
	b.attributes = append(b.attributes,
		attribute.Int("llm.token_count.prompt", prompt),
		attribute.Int("llm.token_count.completion", completion),
		attribute.Int("llm.token_count.total", total),
	)
	return b
}

func (b *spanBuilder) WithModel(name, system, provider string) SpanBuilder {
	b.attributes = append(b.attributes,
		attribute.String("llm.model_name", name),
		attribute.String("llm.system", system),
		attribute.String("llm.provider", provider),
	)
	return b
}

func (b *spanBuilder) WithRetrieval(query string, topK int, documents []RetrievalDocument) SpanBuilder {
	b.attributes = append(b.attributes,
		attribute.String("retrieval.query", truncateString(query, 200)),
		attribute.Int("retrieval.top_k", topK),
	)
	
	// Format documents for OpenInference
	if len(documents) > 0 {
		var docStrings []string
		var totalSimilarity float64
		var maxSimilarity, minSimilarity float64
		
		if len(documents) > 0 {
			maxSimilarity = documents[0].Score
			minSimilarity = documents[0].Score
		}
		
		for _, doc := range documents {
			docJSON, _ := json.Marshal(doc)
			docStrings = append(docStrings, string(docJSON))
			
			totalSimilarity += doc.Score
			if doc.Score > maxSimilarity {
				maxSimilarity = doc.Score
			}
			if doc.Score < minSimilarity {
				minSimilarity = doc.Score
			}
		}
		
		avgSimilarity := totalSimilarity / float64(len(documents))
		
		b.attributes = append(b.attributes,
			attribute.StringSlice("retrieval.documents", docStrings),
			attribute.Int("retrieval.document_count", len(documents)),
			attribute.Float64("retrieval.similarity.avg", avgSimilarity),
			attribute.Float64("retrieval.similarity.max", maxSimilarity),
			attribute.Float64("retrieval.similarity.min", minSimilarity),
		)
	}
	
	return b
}

func (b *spanBuilder) WithTool(name, description string, parameters any) SpanBuilder {
	paramJSON, _ := json.Marshal(parameters)
	
	b.attributes = append(b.attributes,
		attribute.String("tool.name", name),
		attribute.String("tool.description", description),
		attribute.String("tool.parameters", string(paramJSON)),
	)
	return b
}

func (b *spanBuilder) WithCustom(attrs ...attribute.KeyValue) SpanBuilder {
	b.attributes = append(b.attributes, attrs...)
	return b
}

func (b *spanBuilder) Start(ctx context.Context, name string) (context.Context, trace.Span) {
	tracer := otel.Tracer("mcp-factcheck-server")
	return tracer.Start(ctx, name, trace.WithAttributes(b.attributes...))
}

// Helper functions
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Convenience functions for common span types

// StartValidationSpan creates a validation request span
func StartValidationSpan(ctx context.Context, content, specVersion string, useChunking bool) (context.Context, trace.Span) {
	estimatedTokens := len(content) / 4
	
	// Add request ID to span attributes if available
	builder := NewSpanBuilder().
		WithKind("CHAIN").
		WithInput(content, "text/plain").
		WithCustom(
			attribute.String("validation.spec_version", specVersion),
			attribute.Bool("validation.use_chunking", useChunking),
			attribute.Int("content.length", len(content)),
			attribute.Int("content.estimated_tokens", estimatedTokens),
		)
	
	// Add request ID if available in context
	if requestID := GetRequestID(ctx); requestID != "" {
		builder = builder.WithCustom(attribute.String("request.id", requestID))
	}
	
	return builder.Start(ctx, "validate_content_request")
}

// StartEmbeddingSpan creates an embedding generation span
func StartEmbeddingSpan(ctx context.Context, text string) (context.Context, trace.Span) {
	estimatedTokens := len(text) / 4
	
	builder := NewSpanBuilder().
		WithKind("EMBEDDING").
		WithModel("text-embedding-3-small", "openai", "openai").
		WithTokens(estimatedTokens, 0, estimatedTokens).
		WithCustom(
			attribute.String("embedding.summary", fmt.Sprintf("Generating embedding for %d chars (%d tokens)", len(text), estimatedTokens)),
			attribute.Int("embedding.content_length", len(text)),
		)
	
	// Add request ID if available in context
	if requestID := GetRequestID(ctx); requestID != "" {
		builder = builder.WithCustom(attribute.String("request.id", requestID))
	}
	
	return builder.Start(ctx, "embedding.generation")
}

// StartRetrievalSpan creates a vector search span
func StartRetrievalSpan(ctx context.Context, specVersion string, topK int) (context.Context, trace.Span) {
	builder := NewSpanBuilder().
		WithKind("RETRIEVER").
		WithCustom(
			attribute.String("spec_version", specVersion),
			attribute.Int("top_k", topK),
		)
	
	// Add request ID if available in context
	if requestID := GetRequestID(ctx); requestID != "" {
		builder = builder.WithCustom(attribute.String("request.id", requestID))
	}
	
	return builder.Start(ctx, "vector.search")
}

// StartAnalysisSpan creates a validation analysis span
func StartAnalysisSpan(ctx context.Context, numMatches int, avgSimilarity float64) (context.Context, trace.Span) {
	builder := NewSpanBuilder().
		WithKind("CHAIN").
		WithCustom(
			attribute.Int("analysis.num_matches", numMatches),
			attribute.Float64("analysis.avg_similarity", avgSimilarity),
		)
	
	// Add request ID if available in context
	if requestID := GetRequestID(ctx); requestID != "" {
		builder = builder.WithCustom(attribute.String("request.id", requestID))
	}
	
	return builder.Start(ctx, "validation.analysis")
}