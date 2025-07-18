package arizephoenix

import (
	"context"
	"os"
)

// Config holds Arize Phoenix specific configuration
type Config struct {
	Endpoint         string
	MaxContentLength int
	Enabled          bool
}

// DefaultConfig returns default Phoenix configuration
func DefaultConfig() Config {
	endpoint := os.Getenv("PHOENIX_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:6006"
	}

	return Config{
		Endpoint:         endpoint,
		MaxContentLength: 8192,
		Enabled:          os.Getenv("PHOENIX_ENABLED") == "true",
	}
}

// Setup initializes Arize Phoenix specific configuration
// This is called by pkg/logger to set up Phoenix telemetry
func Setup(ctx context.Context) (Config, error) {
	config := DefaultConfig()

	// Any Phoenix-specific initialization can go here
	// For now, we just return the config

	return config, nil
}
