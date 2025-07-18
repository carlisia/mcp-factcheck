package llm

import "fmt"

// APIError represents an error from an LLM provider with additional context
type APIError struct {
	Provider ProviderType
	Message  string
	Err      error
	Context  map[string]any // Additional debugging context
}

func (e *APIError) Error() string {
	if len(e.Context) > 0 {
		return fmt.Sprintf("[%s]: %s - %v (context: %+v)", e.Provider, e.Message, e.Err, e.Context)
	}
	return fmt.Sprintf("[%s]: %s - %v", e.Provider, e.Message, e.Err)
}

func (e *APIError) Unwrap() error { return e.Err }

// NewAPIError creates a new APIError with the given parameters
func NewAPIError(provider ProviderType, message string, err error, context map[string]any) *APIError {
	return &APIError{
		Provider: provider,
		Message:  message,
		Err:      err,
		Context:  context,
	}
}

// LogFields returns structured logging fields for the error
// Compatible with slog and similar structured logging libraries
func (e *APIError) LogFields() []any {
	fields := []any{
		"provider", e.Provider,
		"message", e.Message,
		"error", e.Err,
	}
	for k, v := range e.Context {
		fields = append(fields, k, v)
	}
	return fields
}
