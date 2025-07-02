package arizephoenix

import (
	"context"

	"github.com/carlisia/mcp-factcheck/pkg/logger"
	"github.com/carlisia/mcp-factcheck/pkg/telemetry"
	"go.uber.org/zap"
)

// Initialize creates and configures the complete Phoenix telemetry stack
func Initialize(ctx context.Context, config Config) (telemetry.Provider, telemetry.Middleware, error) {
	// Create the Phoenix provider
	provider, err := NewProvider(ctx, config)
	if err != nil {
		return nil, nil, err
	}

	// Create the middleware
	middleware := NewMiddleware(provider, config)

	logger.Get().Info("Arize Phoenix telemetry initialized", zap.String("endpoint", config.Endpoint))

	return provider, middleware, nil
}

// MustInitialize is like Initialize but panics on error (for development)
func MustInitialize(ctx context.Context, config Config) (telemetry.Provider, telemetry.Middleware) {
	provider, middleware, err := Initialize(ctx, config)
	if err != nil {
		logger.Get().Fatal("Failed to initialize Phoenix telemetry", zap.Error(err))
	}
	return provider, middleware
}
