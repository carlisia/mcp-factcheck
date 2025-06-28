package arizephoenix

import (
	"context"
	"fmt"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// Provider implements the telemetry.Provider interface for Arize Phoenix
type Provider struct {
	config         Config
	tracerProvider *sdktrace.TracerProvider
	tracer         trace.Tracer
}

// NewProvider creates a new Phoenix telemetry provider
func NewProvider(ctx context.Context, config Config) (*Provider, error) {
	// Create resource with Phoenix-specific attributes
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			attribute.String("phoenix.project.name", config.ProjectName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create OTLP trace exporter for Phoenix
	// Note: WithEndpoint expects just host:port, not full URL
	
	var opts []otlptracehttp.Option
	opts = append(opts, otlptracehttp.WithEndpoint(config.Endpoint))
	opts = append(opts, otlptracehttp.WithHeaders(map[string]string{
		"Content-Type": "application/x-protobuf",
	}))
	
	if config.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	
	traceExporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	log.Printf("Phoenix OTLP exporter created for endpoint: %s", config.Endpoint)

	// Configure trace provider based on config
	var processor sdktrace.SpanProcessor
	if config.BatchTimeout > 0 {
		processor = sdktrace.NewBatchSpanProcessor(traceExporter,
			sdktrace.WithBatchTimeout(config.BatchTimeout),
			sdktrace.WithExportTimeout(config.ExportTimeout),
		)
	} else {
		processor = sdktrace.NewSimpleSpanProcessor(traceExporter)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(processor),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(config.SampleRate)),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tracerProvider)

	// Get tracer
	tracer := otel.Tracer(config.ServiceName)

	return &Provider{
		config:         config,
		tracerProvider: tracerProvider,
		tracer:         tracer,
	}, nil
}

// StartSpan implements telemetry.Provider
func (p *Provider) StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	// Apply content length limits for Phoenix compatibility
	filteredAttrs := make([]attribute.KeyValue, 0, len(attrs))
	for _, attr := range attrs {
		if attr.Key == "input.value" || attr.Key == "output.value" {
			if len(attr.Value.AsString()) > p.config.MaxContentLength {
				truncated := attr.Value.AsString()[:p.config.MaxContentLength] + "..."
				filteredAttrs = append(filteredAttrs, attribute.String(string(attr.Key), truncated))
			} else {
				filteredAttrs = append(filteredAttrs, attr)
			}
		} else if attr.Key == "retrieval.query" {
			if len(attr.Value.AsString()) > p.config.MaxAttributeLength {
				truncated := attr.Value.AsString()[:p.config.MaxAttributeLength] + "..."
				filteredAttrs = append(filteredAttrs, attribute.String(string(attr.Key), truncated))
			} else {
				filteredAttrs = append(filteredAttrs, attr)
			}
		} else {
			filteredAttrs = append(filteredAttrs, attr)
		}
	}

	return p.tracer.Start(ctx, name, trace.WithAttributes(filteredAttrs...))
}

// Shutdown implements telemetry.Provider
func (p *Provider) Shutdown(ctx context.Context) error {
	return p.tracerProvider.Shutdown(ctx)
}