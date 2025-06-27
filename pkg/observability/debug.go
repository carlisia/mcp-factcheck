package observability

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

//go:embed debug.html
var debugHTML string

//go:embed debug.js
var debugJS string

// DebugObserver implements Observer with an embedded HTTP debug server
type DebugObserver struct {
	mu           sync.RWMutex
	interactions []ToolInteraction
	server       *http.Server
	port         int
	startTime    time.Time
}

// NewDebugObserver creates a new debug observer with embedded HTTP server
func NewDebugObserver(port int) *DebugObserver {
	return &DebugObserver{
		port:      port,
		startTime: time.Now(),
	}
}

// Start starts the embedded debug HTTP server
func (d *DebugObserver) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/interactions", d.handleInteractions)
	mux.HandleFunc("/api/stats", d.handleStats)
	mux.HandleFunc("/debug.js", d.handleJS)
	mux.HandleFunc("/", d.handleUI)

	d.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", d.port),
		Handler: mux,
	}

	log.Printf("Starting debug server on http://localhost:%d", d.port)
	
	// Start server in goroutine
	go func() {
		if err := d.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Debug server error: %v", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the debug server
func (d *DebugObserver) Stop() error {
	if d.server == nil {
		return nil
	}
	
	log.Println("Shutting down debug server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return d.server.Shutdown(ctx)
}

// RecordInteraction implements Observer interface
func (d *DebugObserver) RecordInteraction(interaction ToolInteraction) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Set ID and timestamp if not provided
	if interaction.ID == "" {
		interaction.ID = uuid.New().String()
	}
	if interaction.Timestamp.IsZero() {
		interaction.Timestamp = time.Now()
	}

	// Add to interactions (keep only last 100)
	d.interactions = append(d.interactions, interaction)
	if len(d.interactions) > 100 {
		d.interactions = d.interactions[1:]
	}
}

func (d *DebugObserver) handleInteractions(w http.ResponseWriter, r *http.Request) {
	d.mu.RLock()
	interactions := append([]ToolInteraction{}, d.interactions...)
	d.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(interactions)
}

func (d *DebugObserver) handleStats(w http.ResponseWriter, r *http.Request) {
	d.mu.RLock()
	totalInteractions := len(d.interactions)
	uptime := time.Since(d.startTime).Seconds()
	d.mu.RUnlock()

	stats := map[string]interface{}{
		"total_interactions": totalInteractions,
		"uptime_seconds":     uptime,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (d *DebugObserver) handleUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(debugHTML))
}

func (d *DebugObserver) handleJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Write([]byte(debugJS))
}