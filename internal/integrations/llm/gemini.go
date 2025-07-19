package llm

import (
	"context"
	"fmt"
)

// geminiProvider implements the Provider interface for Google Gemini
type geminiProvider struct {
	apiKey string
}

// newGeminiProvider creates a new Gemini provider
func newGeminiProvider(apiKey string) (Provider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Gemini API key not provided") //nolint:staticcheck
	}

	return &geminiProvider{
		apiKey: apiKey,
	}, nil
}

// CreateEmbedding generates embeddings using Gemini
func (p *geminiProvider) CreateEmbedding(ctx context.Context, text string) ([]float64, error) {
	// Stub implementation
	return nil, &APIError{
		Provider: Gemini,
		Message:  "Gemini provider not yet implemented",
		Err:      fmt.Errorf("not implemented"),
		Context: map[string]any{
			"method": "CreateEmbedding",
		},
	}
}

// CreateEmbeddingsBatch generates embeddings for multiple texts
func (p *geminiProvider) CreateEmbeddingsBatch(ctx context.Context, texts []string) ([][]float64, error) {
	// Stub implementation
	return nil, &APIError{
		Provider: Gemini,
		Message:  "Gemini provider not yet implemented",
		Err:      fmt.Errorf("not implemented"),
		Context: map[string]any{
			"method": "CreateEmbeddingsBatch",
		},
	}
}

// CompleteJSON performs completion and unmarshals the response
func (p *geminiProvider) CompleteJSON(ctx context.Context, prompt string, opts CompletionOptions, result any) error {
	// Stub implementation
	return &APIError{
		Provider: Gemini,
		Message:  "Gemini provider not yet implemented",
		Err:      fmt.Errorf("not implemented"),
		Context: map[string]any{
			"method": "CompleteJSON",
		},
	}
}

// Complete performs a standard text completion
func (p *geminiProvider) Complete(ctx context.Context, prompt string, opts CompletionOptions) (string, error) {
	// Stub implementation
	return "", &APIError{
		Provider: Gemini,
		Message:  "Gemini provider not yet implemented",
		Err:      fmt.Errorf("not implemented"),
		Context: map[string]any{
			"method": "Complete",
		},
	}
}

// EmbeddingModel returns the embedding model being used
func (p *geminiProvider) EmbeddingModel() string {
	return "not-implemented"
}

// CompletionModel returns the completion model being used
func (p *geminiProvider) CompletionModel() string {
	return "not-implemented"
}
