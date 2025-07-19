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

// CreateEmbeddingsBatch generates embeddings for multiple texts
func (p *anthropicProvider) CreateEmbeddingsBatch(ctx context.Context, texts []string) ([][]float64, error) {
	// Stub implementation
	return nil, &APIError{
		Provider: Anthropic,
		Message:  "Anthropic provider not yet implemented",
		Err:      fmt.Errorf("not implemented"),
		Context: map[string]any{
			"method": "CreateEmbeddingsBatch",
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

// EmbeddingModel returns the embedding model being used
func (p *anthropicProvider) EmbeddingModel() string {
	return "not-implemented"
}

// CompletionModel returns the completion model being used
func (p *anthropicProvider) CompletionModel() string {
	return "not-implemented"
}
