package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/carlisia/mcp-factcheck/internal/integrations/arizephoenix"
	"github.com/carlisia/mcp-factcheck/pkg"
	"github.com/carlisia/mcp-factcheck/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	// Initialize structured logging with Zap
	if err := logger.Initialize(logger.IsDevMode()); err != nil {
		// Can't use structured logging yet since it failed to initialize
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			// Use standard error output since logger might not be available
			log.Printf("Failed to sync logger: %v\n", err)
		}
	}()

	// Get structured logger for use in main
	log := logger.Get()

	// Parse command line flags
	dataDir := flag.String("data-dir", "/Users/carlisiacampos/code/src/github.com/carlisia/mcp-factcheck/data/embeddings", "Directory containing vector database")
	telemetry := flag.Bool("telemetry", false, "Enable OpenTelemetry tracing")
	otlpEndpoint := flag.String("otlp-endpoint", "http://localhost:4318", "OTLP endpoint for traces")
	flag.Parse()

	// Convert to absolute path if relative
	absDataDir, err := filepath.Abs(*dataDir)
	if err != nil {
		log.Fatal("Failed to resolve data directory path", zap.Error(err))
	}

	// Initialize telemetry if enabled
	var provider any
	var middleware any

	if *telemetry {
		ctx := context.Background()

		// Check if endpoint looks like Phoenix and use specialized integration
		if strings.Contains(*otlpEndpoint, "6006") || strings.Contains(*otlpEndpoint, "phoenix") {
			log.Info("Detected Phoenix endpoint, using clean Phoenix integration")
			config := arizephoenix.DefaultConfig()
			config.Endpoint = strings.TrimPrefix(*otlpEndpoint, "http://")

			phoenixProvider, phoenixMiddleware, err := arizephoenix.Initialize(ctx, config)
			if err != nil {
				log.Warn("Failed to initialize Phoenix telemetry. Using no-op provider.", zap.Error(err))
				provider = nil
				middleware = nil
			} else {
				provider = phoenixProvider
				middleware = phoenixMiddleware
				log.Info("Phoenix telemetry provider initialized successfully")
			}
		} else {
			log.Info("Non-Phoenix endpoint detected, using no-op provider")
			provider = nil
			middleware = nil
		}

		// Setup graceful shutdown for telemetry
		if provider != nil {
			go func() {
				sigChan := make(chan os.Signal, 1)
				signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
				<-sigChan
				log.Info("Shutting down telemetry...")
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if p, ok := provider.(interface{ Shutdown(context.Context) error }); ok {
					if err := p.Shutdown(ctx); err != nil {
						log.Error("Error shutting down telemetry", zap.Error(err))
					}
				}
			}()
		}

		log.Info("Clean telemetry architecture enabled")
	}

	// Create MCP fact-check server with clean telemetry
	server, err := pkg.NewFactCheckServer(absDataDir, provider, middleware)
	if err != nil {
		log.Fatal("Failed to create MCP fact-check server", zap.Error(err))
	}

	// Run MCP server (blocks until shutdown)
	if err := server.Run(); err != nil {
		log.Fatal("Server error", zap.Error(err))
	}
}
