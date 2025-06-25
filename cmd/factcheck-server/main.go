package main

import (
	"log"
	"net/http"
	"os"

	"github.com/carlisia/mcp-factcheck/internal/factcheck"
	"github.com/joho/godotenv"
	httphandlers "github.com/carlisia/mcp-factcheck/internal/http"
)

func main() {
	// Load .env file if it exists
	_ = godotenv.Load()

	// Initialize OpenAI client after loading .env
	if err := factcheck.InitOpenAI(); err != nil {
		log.Fatalf("Failed to initialize OpenAI: %v", err)
	}

	// Get port from environment or default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/verify", httphandlers.HandleVerify)
	log.Printf("🔌 MCP Fact-check prototype server running on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
