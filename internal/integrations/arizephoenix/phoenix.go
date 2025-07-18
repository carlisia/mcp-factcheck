package arizephoenix

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
	"go.opentelemetry.io/otel/trace"
)

// Provider wraps the OpenTelemetry TracerProvider for Phoenix
type Provider struct {
	tracerProvider *sdktrace.TracerProvider
	config         Config
}

// Middleware provides Phoenix-specific middleware functionality
type Middleware struct {
	tracer trace.Tracer
	config Config
}

// Initialize creates a new Phoenix telemetry provider and middleware
func Initialize(ctx context.Context, config Config) (*Provider, *Middleware, error) {
	// Force enable for debugging
	config.Enabled = true

	if !config.Enabled {
		// Return no-op implementations
		return &Provider{config: config}, &Middleware{config: config}, nil
	}

	// Create OTLP HTTP exporter for Phoenix
	fmt.Fprintf(os.Stderr, "[Phoenix] Initializing with endpoint: %s\n", config.Endpoint)

	client := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(config.Endpoint),
		otlptracehttp.WithInsecure(), // Phoenix typically runs locally
		otlptracehttp.WithTimeout(30*time.Second),
	)

	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, nil, fmt.Errorf("creating OTLP exporter: %w", err)
	}

	fmt.Fprintf(os.Stderr, "[Phoenix] OTLP exporter created successfully\n")

	// Create resource with Phoenix-compatible attributes
	res := resource.NewWithAttributes(
		"", // Empty schema URL to avoid conflicts
		semconv.ServiceName("mcp-factcheck-server"),
		semconv.ServiceVersion("1.0.0"),
		// Phoenix-specific attributes
		attribute.String("openinference.project.name", "mcp-factcheck"),
	)

	// Create tracer provider with Phoenix-optimized settings
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // Phoenix expects all traces
	)

	// Register as global provider
	otel.SetTracerProvider(tp)
	fmt.Fprintf(os.Stderr, "[Phoenix] Tracer provider registered globally\n")

	provider := &Provider{
		tracerProvider: tp,
		config:         config,
	}

	middleware := &Middleware{
		tracer: tp.Tracer("mcp-factcheck"),
		config: config,
	}

	fmt.Fprintf(os.Stderr, "[Phoenix] Returning provider and middleware\n")
	return provider, middleware, nil
}

// Shutdown gracefully shuts down the provider
func (p *Provider) Shutdown(ctx context.Context) error {
	if p.tracerProvider != nil {
		return p.tracerProvider.Shutdown(ctx)
	}
	return nil
}

// Tracer returns a configured tracer
func (p *Provider) Tracer(name string) trace.Tracer {
	if p.tracerProvider != nil {
		return p.tracerProvider.Tracer(name)
	}
	return otel.Tracer(name)
}
