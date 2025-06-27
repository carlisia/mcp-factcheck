package main

import (
	"flag"
	"log"
	"path/filepath"

	"github.com/carlisia/mcp-factcheck/pkg"
	"github.com/carlisia/mcp-factcheck/pkg/debug"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists
	_ = godotenv.Load()

	// Parse command line flags
	dataDir := flag.String("data-dir", "/Users/carlisiacampos/code/src/github.com/carlisia/mcp-factcheck/data/embeddings", "Directory containing vector database")
	debugMode := flag.Bool("debug", false, "Enable debug mode with IPC communication")
	debugSocket := flag.String("debug-socket", "/tmp/mcp-factcheck-debug.sock", "Unix socket path for debug IPC")
	flag.Parse()

	// Convert to absolute path if relative
	absDataDir, err := filepath.Abs(*dataDir)
	if err != nil {
		log.Fatalf("Failed to resolve data directory path: %v", err)
	}

	// Create debug IPC client if enabled
	var debugClient *debug.IPCClient
	if *debugMode {
		debugClient = debug.NewIPCClient(*debugSocket)
		log.Printf("Debug mode enabled - sending data to socket %s", *debugSocket)
	} else {
		log.Printf("Debug mode disabled")
	}

	// Create and run MCP fact-check server
	server, err := pkg.NewFactCheckServer(absDataDir, debugClient)
	if err != nil {
		log.Fatalf("Failed to create MCP fact-check server: %v", err)
	}

	if err := server.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}