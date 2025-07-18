package llm

import (
	"context"
	"fmt"
)

// anthropicProvider implements the Provider interface for Anthropic
type anthropicProvider struct {
	apiKey string
}

// newAnthropicProvider creates a new Anthropic provider
func newAnthropicProvider(apiKey string) (Provider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Anthropic API key not provided") //nolint:staticcheck
	}

	return &anthropicProvider{
		apiKey: apiKey,
	}, nil
}

// CreateEmbedding generates embeddings using Anthropic
func (p *anthropicProvider) CreateEmbedding(ctx context.Context, text string) ([]float64, error) {
	// Stub implementation
	return nil, &APIError{
		Provider: Anthropic,
		Message:  "Anthropic provider not yet implemented",
		Err:      fmt.Errorf("not implemented"),
		Context: map[string]any{
			"method": "CreateEmbedding",
		},
	}
}

// CompleteJSON performs completion and unmarshals the response
func (p *anthropicProvider) CompleteJSON(ctx context.Context, prompt string, opts CompletionOptions, result any) error {
	// Stub implementation
	return &APIError{
		Provider: Anthropic,
		Message:  "Anthropic provider not yet implemented",
		Err:      fmt.Errorf("not implemented"),
		Context: map[string]any{
			"method": "CompleteJSON",
		},
	}
}

// Complete performs a standard text completion
func (p *anthropicProvider) Complete(ctx context.Context, prompt string, opts CompletionOptions) (string, error) {
	// Stub implementation
	return "", &APIError{
		Provider: Anthropic,
		Message:  "Anthropic provider not yet implemented",
		Err:      fmt.Errorf("not implemented"),
		Context: map[string]any{
			"method": "Complete",
		},
	}
}
