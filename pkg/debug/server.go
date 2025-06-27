package debug

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// DebugServer provides HTTP endpoints for inspecting MCP interactions
type DebugServer struct {
	mu           sync.RWMutex
	interactions []DebugInteraction
	stats        DebugStats
	startTime    time.Time
	upgrader     websocket.Upgrader
	clients      map[*websocket.Conn]bool
}

// NewDebugServer creates a new debug server instance
func NewDebugServer() *DebugServer {
	return &DebugServer{
		interactions: make([]DebugInteraction, 0, 100), // Keep last 100 interactions
		stats: DebugStats{
			ToolUsage: make(map[string]int),
		},
		startTime: time.Now(),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for debugging
			},
		},
		clients: make(map[*websocket.Conn]bool),
	}
}

// AddInteraction records a new tool interaction
func (ds *DebugServer) AddInteraction(interaction DebugInteraction) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Add to interactions (keep only last 100)
	ds.interactions = append(ds.interactions, interaction)
	if len(ds.interactions) > 100 {
		ds.interactions = ds.interactions[1:]
	}

	// Update stats
	ds.stats.TotalInteractions++
	ds.stats.ToolUsage[interaction.ToolName]++
	ds.stats.TotalSavings += interaction.TokenCount.Savings
	
	// Calculate average tokens
	totalTokens := 0
	for _, inter := range ds.interactions {
		totalTokens += inter.TokenCount.TotalTokens
	}
	ds.stats.AverageTokens = float64(totalTokens) / float64(len(ds.interactions))
	ds.stats.UptimeSeconds = int64(time.Since(ds.startTime).Seconds())

	// Broadcast to WebSocket clients
	ds.broadcastInteraction(interaction)
}

// broadcastInteraction sends interaction to all connected WebSocket clients
func (ds *DebugServer) broadcastInteraction(interaction DebugInteraction) {
	message, _ := json.Marshal(interaction)
	for client := range ds.clients {
		err := client.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			delete(ds.clients, client)
			client.Close()
		}
	}
}

// Start starts the debug HTTP server
func (ds *DebugServer) Start(port int) error {
	mux := http.NewServeMux()
	
	// API endpoints
	mux.HandleFunc("/api/interactions", ds.handleInteractions)
	mux.HandleFunc("/api/stats", ds.handleStats)
	mux.HandleFunc("/api/latest", ds.handleLatest)
	mux.HandleFunc("/ws", ds.handleWebSocket)
	
	// UI endpoints
	mux.HandleFunc("/", ds.handleUI)
	mux.HandleFunc("/interaction/", ds.handleInteractionDetail)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Debug server starting on http://localhost%s", addr)
	return http.ListenAndServe(addr, mux)
}

// handleInteractions returns all interactions as JSON
func (ds *DebugServer) handleInteractions(w http.ResponseWriter, r *http.Request) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ds.interactions)
}

// handleStats returns statistics as JSON
func (ds *DebugServer) handleStats(w http.ResponseWriter, r *http.Request) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ds.stats)
}

// handleLatest returns the most recent interaction
func (ds *DebugServer) handleLatest(w http.ResponseWriter, r *http.Request) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	
	if len(ds.interactions) == 0 {
		http.Error(w, "No interactions yet", http.StatusNotFound)
		return
	}
	
	latest := ds.interactions[len(ds.interactions)-1]
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(latest)
}

// handleWebSocket handles WebSocket connections for live updates
func (ds *DebugServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := ds.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	ds.mu.Lock()
	ds.clients[conn] = true
	ds.mu.Unlock()

	// Send existing interactions
	for _, interaction := range ds.interactions {
		message, _ := json.Marshal(interaction)
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			break
		}
	}

	// Keep connection alive
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			ds.mu.Lock()
			delete(ds.clients, conn)
			ds.mu.Unlock()
			break
		}
	}
}

// handleUI serves the main debug interface
func (ds *DebugServer) handleUI(w http.ResponseWriter, r *http.Request) {
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>MCP Fact-Check Debug Interface</title>
    <style>
        body { font-family: monospace; margin: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; }
        .stats { background: white; padding: 15px; border-radius: 5px; margin-bottom: 20px; }
        .interaction { background: white; padding: 15px; border-radius: 5px; margin-bottom: 10px; border-left: 4px solid #007acc; }
        .error { border-left-color: #e74c3c; }
        .timestamp { color: #666; font-size: 0.9em; }
        .tokens { background: #e8f4f8; padding: 5px; border-radius: 3px; display: inline-block; }
        .savings { color: #27ae60; font-weight: bold; }
        pre { background: #f8f8f8; padding: 10px; border-radius: 3px; overflow-x: auto; }
        .live-indicator { color: #27ae60; }
        .tool-name { color: #8e44ad; font-weight: bold; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔍 MCP Fact-Check Debug Interface</h1>
        
        <div class="stats">
            <h3>Statistics</h3>
            <div id="stats">Loading...</div>
        </div>

        <div>
            <h3>Recent Interactions <span class="live-indicator" id="live">● LIVE</span></h3>
            <div id="interactions">Loading...</div>
        </div>
    </div>

    <script>
        const ws = new WebSocket('ws://localhost:8080/ws');
        let interactions = [];

        // Load initial stats
        fetch('/api/stats')
            .then(r => r.json())
            .then(stats => displayStats(stats));

        // Load initial interactions
        fetch('/api/interactions')
            .then(r => r.json())
            .then(data => {
                interactions = data;
                displayInteractions();
            });

        // WebSocket for live updates
        ws.onmessage = function(event) {
            const interaction = JSON.parse(event.data);
            interactions.push(interaction);
            if (interactions.length > 50) interactions.shift(); // Keep last 50
            displayInteractions();
            
            // Update live indicator
            const liveIndicator = document.getElementById('live');
            liveIndicator.style.color = '#27ae60';
            setTimeout(() => { liveIndicator.style.color = '#ccc'; }, 1000);
        };

        function displayStats(stats) {
            document.getElementById('stats').innerHTML = 
                'Total: ' + stats.total_interactions + ' | ' +
                'Avg Tokens: ' + Math.round(stats.average_tokens) + ' | ' +
                'Total Savings: ' + stats.total_savings + ' tokens | ' +
                'Uptime: ' + Math.round(stats.uptime_seconds) + 's';
        }

        function displayInteractions() {
            const container = document.getElementById('interactions');
            container.innerHTML = interactions.slice(-10).reverse().map(interaction => 
                '<div class="interaction ' + (interaction.error ? 'error' : '') + '">' +
                    '<div class="timestamp">' + new Date(interaction.timestamp).toLocaleTimeString() + '</div>' +
                    '<div><span class="tool-name">' + interaction.tool_name + '</span> ' +
                    '<span class="tokens">🎯 ' + interaction.token_count.total_tokens + ' tokens' +
                    (interaction.token_count.savings > 0 ? ' <span class="savings">(-' + interaction.token_count.savings + ')</span>' : '') +
                    '</span></div>' +
                    '<details><summary>Arguments</summary><pre>' + JSON.stringify(interaction.arguments, null, 2) + '</pre></details>' +
                    '<details><summary>Response to LLM</summary><pre>' + JSON.stringify(interaction.response, null, 2) + '</pre></details>' +
                    (interaction.error ? '<div style="color: red;">Error: ' + interaction.error + '</div>' : '') +
                '</div>'
            ).join('');
        }

        // Refresh stats every 5 seconds
        setInterval(() => {
            fetch('/api/stats').then(r => r.json()).then(displayStats);
        }, 5000);
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	t, _ := template.New("debug").Parse(tmpl)
	t.Execute(w, nil)
}

// handleInteractionDetail shows detailed view of a specific interaction
func (ds *DebugServer) handleInteractionDetail(w http.ResponseWriter, r *http.Request) {
	indexStr := r.URL.Path[len("/interaction/"):]
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		http.Error(w, "Invalid interaction index", http.StatusBadRequest)
		return
	}

	ds.mu.RLock()
	defer ds.mu.RUnlock()

	if index < 0 || index >= len(ds.interactions) {
		http.Error(w, "Interaction not found", http.StatusNotFound)
		return
	}

	interaction := ds.interactions[index]
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(interaction)
}