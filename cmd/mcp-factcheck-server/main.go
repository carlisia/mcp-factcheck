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

	"github.com/carlisia/mcp-factcheck/pkg"
	"github.com/carlisia/mcp-factcheck/internal/integrations/arizephoenix"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists
	_ = godotenv.Load()

	// Parse command line flags
	dataDir := flag.String("data-dir", "/Users/carlisiacampos/code/src/github.com/carlisia/mcp-factcheck/data/embeddings", "Directory containing vector database")
	telemetry := flag.Bool("telemetry", false, "Enable OpenTelemetry tracing")
	otlpEndpoint := flag.String("otlp-endpoint", "http://localhost:4318", "OTLP endpoint for traces")
	flag.Parse()

	// Convert to absolute path if relative
	absDataDir, err := filepath.Abs(*dataDir)
	if err != nil {
		log.Fatalf("Failed to resolve data directory path: %v", err)
	}

	// Initialize telemetry if enabled
	var provider any
	var middleware any
	
	if *telemetry {
		ctx := context.Background()
		
		// Check if endpoint looks like Phoenix and use specialized integration
		if strings.Contains(*otlpEndpoint, "6006") || strings.Contains(*otlpEndpoint, "phoenix") {
			log.Println("Detected Phoenix endpoint, using clean Phoenix integration")
			config := arizephoenix.DefaultConfig()
			config.Endpoint = strings.TrimPrefix(*otlpEndpoint, "http://")
			
			phoenixProvider, phoenixMiddleware, err := arizephoenix.Initialize(ctx, config)
			if err != nil {
				log.Printf("Failed to initialize Phoenix telemetry: %v. Using no-op provider.", err)
				provider = nil
				middleware = nil
			} else {
				provider = phoenixProvider
				middleware = phoenixMiddleware
				log.Println("Phoenix telemetry provider initialized successfully")
			}
		} else {
			log.Println("Non-Phoenix endpoint detected, using no-op provider")
			provider = nil
			middleware = nil
		}
		
		// Setup graceful shutdown for telemetry
		if provider != nil {
			go func() {
				sigChan := make(chan os.Signal, 1)
				signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
				<-sigChan
				log.Println("Shutting down telemetry...")
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if p, ok := provider.(interface{ Shutdown(context.Context) error }); ok {
					p.Shutdown(ctx)
				}
			}()
		}
		
		log.Println("Clean telemetry architecture enabled")
	}

	// Create MCP fact-check server with clean telemetry
	server, err := pkg.NewFactCheckServer(absDataDir, provider, middleware)
	if err != nil {
		log.Fatalf("Failed to create MCP fact-check server: %v", err)
	}

	// Run MCP server (blocks until shutdown)
	if err := server.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}