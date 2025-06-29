package logger

import (
	"context"
	"os"

	"github.com/carlisia/mcp-factcheck/pkg/telemetry"
	"go.uber.org/zap"
)

var (
	globalLogger *zap.Logger
	sugar        *zap.SugaredLogger
)

// Initialize sets up the global logger with appropriate configuration
func Initialize(isDevelopment bool) error {
	var config zap.Config
	
	if isDevelopment {
		config = zap.NewDevelopmentConfig()
		config.Development = true
	} else {
		config = zap.NewProductionConfig()
		config.Development = false
	}
	
	// Always log to stderr to avoid interfering with MCP stdio communication
	config.OutputPaths = []string{"stderr"}
	config.ErrorOutputPaths = []string{"stderr"}
	
	logger, err := config.Build()
	if err != nil {
		return err
	}
	
	globalLogger = logger
	sugar = logger.Sugar()
	
	return nil
}

// Get returns the global logger instance
func Get() *zap.Logger {
	if globalLogger == nil {
		// Fallback to no-op logger if not initialized
		globalLogger = zap.NewNop()
	}
	return globalLogger
}

// Sugar returns the global sugared logger instance
func Sugar() *zap.SugaredLogger {
	if sugar == nil {
		// Fallback to no-op logger if not initialized
		sugar = zap.NewNop().Sugar()
	}
	return sugar
}

// WithRequestID returns a logger with the request ID from context
func WithRequestID(ctx context.Context) *zap.Logger {
	logger := Get()
	
	if requestID := telemetry.GetRequestID(ctx); requestID != "" {
		return logger.With(zap.String("request_id", requestID))
	}
	
	return logger
}

// WithRequestIDSugar returns a sugared logger with the request ID from context
func WithRequestIDSugar(ctx context.Context) *zap.SugaredLogger {
	return WithRequestID(ctx).Sugar()
}

// Sync flushes any buffered log entries
func Sync() {
	if globalLogger != nil {
		globalLogger.Sync()
	}
}

// IsDevMode checks if we're in development mode based on environment
func IsDevMode() bool {
	return os.Getenv("ENVIRONMENT") == "development" || 
		   os.Getenv("ENV") == "dev" ||
		   os.Getenv("DEBUG") == "true"
}