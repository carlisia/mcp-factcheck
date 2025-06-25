package main

import (
	"log"
	"net/http"
	"os"

	httphandlers "github.com/carlisia/mcp-factcheck/internal/http"
)

func main() {
	// Get port from environment or default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/verify", httphandlers.HandleVerify)
	log.Printf("🔌 MCP Fact-check prototype server running on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
