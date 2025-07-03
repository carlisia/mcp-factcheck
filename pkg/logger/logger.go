package logger

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/carlisia/mcp-factcheck/pkg/telemetry"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)


var (
	globalLogger *zap.Logger
	sugar        *zap.SugaredLogger
)

// orderedJSONEncoder ensures timestamp comes first in JSON output
type orderedJSONEncoder struct {
	zapcore.Encoder
	pool buffer.Pool
}

// Clone implements zapcore.Encoder
func (e *orderedJSONEncoder) Clone() zapcore.Encoder {
	return &orderedJSONEncoder{
		Encoder: e.Encoder.Clone(),
		pool:    e.pool,
	}
}

// EncodeEntry ensures timestamp is the first field
func (e *orderedJSONEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	// Get a buffer from the pool
	buf := e.pool.Get()

	// Start JSON object
	buf.AppendByte('{')

	// Write timestamp first
	buf.AppendString(`"timestamp":"`)
	buf.AppendString(entry.Time.Format("2006-01-02T15:04:05.000-0700"))
	buf.AppendString(`"`)

	// Write level second
	buf.AppendString(`,"level":"`)
	buf.AppendString(entry.Level.String())
	buf.AppendString(`"`)

	// Write caller if present
	if entry.Caller.Defined {
		buf.AppendString(`,"caller":"`)
		buf.AppendString(entry.Caller.TrimmedPath())
		buf.AppendString(`"`)
	}

	// Write message
	buf.AppendString(`,"msg":"`)
	// Escape the message for JSON
	escaped, err := json.Marshal(entry.Message)
	if err != nil {
		// Fallback to raw message with basic escaping
		buf.AppendString(strings.ReplaceAll(entry.Message, `"`, `\"`))
	} else if len(escaped) >= 2 {
		buf.AppendBytes(escaped[1 : len(escaped)-1]) // Remove quotes
	}
	buf.AppendString(`"`)

	// Add custom fields
	enc := zapcore.NewMapObjectEncoder()
	for _, field := range fields {
		field.AddTo(enc)
	}

	// Write remaining fields
	for k, v := range enc.Fields {
		data, err := json.Marshal(v)
		if err != nil {
			// If marshaling fails, write error as string
			buf.AppendString(`,"`)
			buf.AppendString(k)
			buf.AppendString(`":"[marshal error: `)
			buf.AppendString(err.Error())
			buf.AppendString(`]"`)
		} else {
			buf.AppendString(`,"`)
			buf.AppendString(k)
			buf.AppendString(`":`)
			buf.AppendBytes(data)
		}
	}

	// Add stack trace if present
	if entry.Stack != "" {
		buf.AppendString(`,"stacktrace":`)
		data, err := json.Marshal(entry.Stack)
		if err != nil {
			// Fallback to escaped string
			buf.AppendString(`"`)
			buf.AppendString(strings.ReplaceAll(entry.Stack, `"`, `\"`))
			buf.AppendString(`"`)
		} else {
			buf.AppendBytes(data)
		}
	}

	// Close JSON object
	buf.AppendByte('}')
	buf.AppendByte('\n')

	return buf, nil
}

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

	// Configure encoder settings
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.MessageKey = "msg"
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.CallerKey = "caller"

	// Create buffer pool
	pool := buffer.NewPool()

	// Build encoder
	encoder := zapcore.NewJSONEncoder(config.EncoderConfig)

	// Wrap with our ordered encoder
	orderedEncoder := &orderedJSONEncoder{
		Encoder: encoder,
		pool:    pool,
	}

	// Build the logger with custom encoder
	core := zapcore.NewCore(
		orderedEncoder,
		zapcore.AddSync(os.Stderr),
		config.Level,
	)

	logger := zap.New(core, zap.AddCaller())

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

// Sync flushes any buffered log entries. Returns error if sync fails.
func Sync() error {
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
}

// IsDevMode checks if we're in development mode based on environment
func IsDevMode() bool {
	return os.Getenv("ENVIRONMENT") == "development" ||
		os.Getenv("ENV") == "dev" ||
		os.Getenv("DEBUG") == "true"
}
