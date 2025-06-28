package arizephoenix

import (
	"context"
	"log"

	"github.com/carlisia/mcp-factcheck/pkg/telemetry"
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

	log.Printf("Arize Phoenix telemetry initialized with endpoint: %s", config.Endpoint)
	
	return provider, middleware, nil
}

// MustInitialize is like Initialize but panics on error (for development)
func MustInitialize(ctx context.Context, config Config) (telemetry.Provider, telemetry.Middleware) {
	provider, middleware, err := Initialize(ctx, config)
	if err != nil {
		log.Fatalf("Failed to initialize Phoenix telemetry: %v", err)
	}
	return provider, middleware
}