package debug

import (
	"time"
)

// DebugInteraction represents a complete tool interaction
type DebugInteraction struct {
	ID           string            `json:"id"`
	Timestamp    time.Time         `json:"timestamp"`
	ToolName     string            `json:"tool_name"`
	Arguments    any               `json:"arguments"`
	Response     []any             `json:"response"`
	TokenCount   TokenInfo         `json:"token_count"`
	ProcessingMs int64             `json:"processing_ms"`
	Error        string            `json:"error,omitempty"`
}

// TokenInfo provides token usage information
type TokenInfo struct {
	InputTokens   int `json:"input_tokens"`
	OutputTokens  int `json:"output_tokens"`
	TotalTokens   int `json:"total_tokens"`
	Savings       int `json:"savings_vs_old"` // Estimated savings from optimization
}

// DebugStats provides overall statistics
type DebugStats struct {
	TotalInteractions int           `json:"total_interactions"`
	AverageTokens     float64       `json:"average_tokens"`
	TotalSavings      int           `json:"total_savings"`
	UptimeSeconds     int64         `json:"uptime_seconds"`
	ToolUsage         map[string]int `json:"tool_usage"`
}