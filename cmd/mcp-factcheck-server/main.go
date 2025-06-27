package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/carlisia/mcp-factcheck/pkg"
	"github.com/carlisia/mcp-factcheck/pkg/observability"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists
	_ = godotenv.Load()

	// Parse command line flags
	dataDir := flag.String("data-dir", "/Users/carlisiacampos/code/src/github.com/carlisia/mcp-factcheck/data/embeddings", "Directory containing vector database")
	debug := flag.Bool("debug", false, "Enable debug server on port 8083")
	debugPort := flag.Int("debug-port", 8083, "Debug server port")
	flag.Parse()

	// Convert to absolute path if relative
	absDataDir, err := filepath.Abs(*dataDir)
	if err != nil {
		log.Fatalf("Failed to resolve data directory path: %v", err)
	}

	// Create observer (debug or no-op)
	var observer observability.Observer
	if *debug {
		debugObserver := observability.NewDebugObserver(*debugPort)
		if err := debugObserver.Start(); err != nil {
			log.Fatalf("Failed to start debug server: %v", err)
		}
		observer = debugObserver
		
		// Setup graceful shutdown for debug server
		go func() {
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			<-sigChan
			log.Println("Shutting down debug server...")
			debugObserver.Stop()
		}()
	}

	// Create MCP fact-check server with observer
	server, err := pkg.NewFactCheckServer(absDataDir, observer)
	if err != nil {
		log.Fatalf("Failed to create MCP fact-check server: %v", err)
	}

	// Run MCP server (blocks until shutdown)
	if err := server.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}