package arizephoenix

import "time"

// Config holds Arize Phoenix specific configuration
type Config struct {
	// OTLP endpoint for Phoenix
	Endpoint string
	
	// Whether to use insecure connection (for local development)
	Insecure bool
	
	// Project name in Phoenix
	ProjectName string
	
	// Service identification
	ServiceName    string
	ServiceVersion string
	
	// Sampling configuration
	SampleRate float64
	
	// Export configuration
	BatchTimeout time.Duration
	ExportTimeout time.Duration
	
	// Phoenix-specific features
	AutoCreateProject bool
	EnableCostTracking bool
	
	// OpenInference semantic conventions
	OpenInferenceCompliant bool
	
	// Content limits for attributes (to avoid Phoenix UI issues)
	MaxContentLength     int
	MaxDocumentLength    int
	MaxAttributeLength   int
}

// DefaultConfig returns sensible defaults for Phoenix integration
func DefaultConfig() Config {
	return Config{
		Endpoint:               "localhost:6006",
		Insecure:               true,
		ProjectName:            "mcp-factcheck",
		ServiceName:            "mcp-factcheck-server",
		ServiceVersion:         "0.1.0",
		SampleRate:             1.0, // Sample all traces in development
		BatchTimeout:           time.Second * 5,
		ExportTimeout:          time.Second * 30,
		AutoCreateProject:      true,
		EnableCostTracking:     true,
		OpenInferenceCompliant: true,
		MaxContentLength:       500,   // Max content in attributes
		MaxDocumentLength:      200,   // Max document content
		MaxAttributeLength:     1000,  // Max any single attribute
	}
}