package main

import (
	"context"
	"fmt"
	stdlog "log"
	"os"

	"github.com/carlisia/mcp-factcheck/pkg/llm"
	"github.com/carlisia/mcp-factcheck/pkg/logger"
	"github.com/carlisia/mcp-factcheck/pkg/mcp"
	"go.uber.org/zap"
)

// Version information - can be overridden at build time
var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

// providerStrings converts provider types to string slice for logging
func providerStrings() []string {
	providers := llm.ValidProviders()
	result := make([]string, len(providers))
	for i, p := range providers {
		result[i] = string(p)
	}
	return result
}


func main() {
	// Initialize structured logging with Zap
	if err := logger.Initialize(logger.IsDevMode()); err != nil {
		// Can't use structured logging yet since it failed to initialize
		stdlog.Fatalf("Failed to initialize logger: %v", err)
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			// Use standard error output since logger might not be available
			stdlog.Printf("Failed to sync logger: %v\n", err)
		}
	}()

	// Get structured logger for use in main
	log := logger.Get()

	// Parse configuration
	cfg, err := parseConfig()
	if err != nil {
		log.Fatal("Failed to parse configuration", zap.Error(err))
	}

	// Show version information if requested
	if cfg.ShowVersion {
		fmt.Printf("MCP Fact-check Server\nVersion: %s\nCommit: %s\nBuilt: %s\n",
			version, commit, buildDate)
		os.Exit(0)
	}

	// Create LLM configuration
	llmConfig := llm.Config{
		Type:   llm.ProviderType(cfg.LLMProvider),
		APIKey: cfg.APIKey,
	}

	// Initialize telemetry if enabled
	var telemetryProvider TelemetryProvider
	var middleware Middleware

	if cfg.Telemetry.Enabled {
		ctx := context.Background()
		telemetryProvider, middleware = setupTelemetry(ctx, cfg.Telemetry, log)
		setupGracefulShutdown(telemetryProvider, log)
		log.Info("Telemetry configuration completed",
			zap.String("provider_type", cfg.Telemetry.ProviderType))
	}

	// Create MCP fact-check server with LLM configuration
	srv, err := mcp.NewFactCheckServerWithConfig(cfg.DataDir, llmConfig, telemetryProvider, middleware)
	if err != nil {
		log.Fatal("Failed to create MCP fact-check server", zap.Error(err))
	}

	log.Info("Starting MCP fact-check server",
		zap.String("version", version),
		zap.String("commit", commit),
		zap.String("build_date", buildDate),
		zap.String("provider", cfg.LLMProvider),
		zap.Bool("telemetry", cfg.Telemetry.Enabled))

	// Run MCP server (blocks until shutdown)
	if err := srv.Run(); err != nil {
		log.Fatal("Server error", zap.Error(err))
	}
}
