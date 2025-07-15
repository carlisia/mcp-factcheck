package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/carlisia/mcp-factcheck/internal/integrations/arizephoenix"
	"go.uber.org/zap"
)

// TelemetryProvider defines the interface for telemetry providers
type TelemetryProvider interface {
	// Shutdown gracefully shuts down the telemetry provider
	Shutdown(ctx context.Context) error
}

// Middleware defines the interface for telemetry middleware
type Middleware interface {
	// This interface can be extended with middleware-specific methods as needed
	// For now, it serves as a marker interface to replace the use of 'any'
}

// setupTelemetry initializes telemetry providers based on the configuration.
// It returns provider and middleware instances, or nil if telemetry initialization fails.
func setupTelemetry(ctx context.Context, cfg TelemetryConfig, log *zap.Logger) (provider TelemetryProvider, middleware Middleware) {
	switch cfg.ProviderType {
	case "phoenix":
		log.Info("Initializing Phoenix telemetry provider")
		config := arizephoenix.DefaultConfig()
		config.Endpoint = strings.TrimPrefix(cfg.Endpoint, "http://")

		phoenixProvider, phoenixMiddleware, err := arizephoenix.Initialize(ctx, config)
		if err != nil {
			log.Warn("Failed to initialize Phoenix telemetry, telemetry disabled", zap.Error(err))
			return nil, nil
		}
		
		log.Info("Phoenix telemetry provider initialized successfully", 
			zap.String("provider_type", fmt.Sprintf("%T", phoenixProvider)),
			zap.String("middleware_type", fmt.Sprintf("%T", phoenixMiddleware)))
		return phoenixProvider, phoenixMiddleware

	case "otlp":
		log.Info("OTLP telemetry provider not yet implemented, using no-op")
		return nil, nil

	case "noop":
		log.Info("Using no-op telemetry provider")
		return nil, nil

	default:
		log.Warn("Unknown telemetry provider type, using no-op",
			zap.String("provider_type", cfg.ProviderType))
		return nil, nil
	}
}

