package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/carlisia/mcp-factcheck/pkg/llm"
)

// Config holds all configuration for the MCP fact-check server
type Config struct {
	// Data directory containing vector database
	DataDir string

	// Telemetry configuration
	Telemetry TelemetryConfig

	// LLM provider configuration
	LLMProvider string
	APIKey      string

	// Version display flag
	ShowVersion bool
}

// TelemetryConfig holds telemetry-specific configuration
type TelemetryConfig struct {
	Enabled      bool
	ProviderType string // "phoenix", "otlp", "noop"
	Endpoint     string
}

// parseConfig parses configuration from command-line flags and environment variables
func parseConfig() (*Config, error) {
	cfg := &Config{}

	// Define command-line flags
	llmProvider := flag.String("provider", "openai", "LLM provider to use, one of: openai (default), anthropic (not implemented), gemini (not implemented)")
	telemetry := flag.Bool("telemetry", false, "Enable OpenTelemetry tracing")
	otlpEndpoint := flag.String("otlp-endpoint", "http://localhost:4318", "OTLP endpoint for traces")
	apiKey := flag.String("api-key", "", "API key for the LLM provider (if not set via environment)")
	showVersion := flag.Bool("version", false, "Show version information")

	flag.Parse()

	// Basic configuration
	cfg.LLMProvider = *llmProvider
	cfg.ShowVersion = *showVersion

	// Set data directory to empty string to indicate embedded data should be used
	// In the future, we could add a flag to allow external data directory
	cfg.DataDir = ""

	// Validate LLM provider
	if !llm.IsValidProvider(cfg.LLMProvider) {
		return nil, fmt.Errorf("invalid LLM provider %q, valid providers: %v",
			cfg.LLMProvider, providerStrings())
	}

	// Resolve API key
	apiKeyVal, err := resolveAPIKey(cfg.LLMProvider, *apiKey)
	if err != nil {
		return nil, err
	}
	cfg.APIKey = apiKeyVal

	// Configure telemetry
	cfg.Telemetry.Enabled = *telemetry
	cfg.Telemetry.Endpoint = *otlpEndpoint
	cfg.Telemetry.ProviderType = detectTelemetryProvider(*otlpEndpoint)

	return cfg, nil
}

// resolveAPIKey resolves the API key from flag or environment variable
func resolveAPIKey(provider, flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}

	// Use the centralized GetAPIKeyFromEnv from pkg/llm
	// This function already knows about all provider environment variables
	key := llm.GetAPIKeyFromEnv(llm.ProviderType(provider))
	if key == "" {
		// Provide helpful error message with the correct environment variable
		envVars := map[string]string{
			"openai":    "OPENAI_API_KEY",
			"anthropic": "ANTHROPIC_API_KEY",
			"gemini":    "GEMINI_API_KEY",
		}
		envVar := envVars[provider]
		return "", fmt.Errorf("API key required for provider %q not set (use -api-key flag or %s env var)",
			provider, envVar)
	}

	return key, nil
}

// detectTelemetryProvider determines the telemetry provider type from the endpoint
func detectTelemetryProvider(endpoint string) string {
	// This is a simple detection for now
	// Could be enhanced with explicit flags or configuration
	if endpoint == "" {
		return "noop"
	}

	// Phoenix detection - could be made more robust
	if strings.Contains(endpoint, "6006") || strings.Contains(endpoint, "phoenix") {
		return "phoenix"
	}

	return "otlp"
}

