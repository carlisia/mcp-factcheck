package llm

import (
	"context"
	"fmt"
	"os"
)

// Client wraps a Provider to provide domain-specific operations
type Client struct {
	provider Provider
}

// New creates a new client with the default provider (OpenAI) using environment variables
func New() (*Client, error) {
	// Default to OpenAI for backward compatibility
	return NewWithProvider(Config{
		Type: OpenAI,
	})
}

// NewWithKey creates a new OpenAI client with the provided API key (backward compatibility)
func NewWithKey(apiKey string) (*Client, error) {
	return NewWithProvider(Config{
		Type:   OpenAI,
		APIKey: apiKey,
	})
}

// NewWithProvider creates a new client with the specified provider
func NewWithProvider(cfg Config) (*Client, error) {
	provider, err := NewProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating provider: %w", err)
	}

	return &Client{
		provider: provider,
	}, nil
}

// CreateEmbedding generates an embedding for a single text
func (c *Client) CreateEmbedding(ctx context.Context, text string) ([]float64, error) {
	return c.provider.CreateEmbedding(ctx, text)
}

// CompleteJSON sends a chat completion request expecting a JSON response
func (c *Client) CompleteJSON(ctx context.Context, prompt string, opts CompletionOptions, result any) error {
	return c.provider.CompleteJSON(ctx, prompt, opts, result)
}

// Complete sends a chat completion request expecting a plain text response
func (c *Client) Complete(ctx context.Context, prompt string, opts CompletionOptions) (string, error) {
	return c.provider.Complete(ctx, prompt, opts)
}

// GetProviderFromEnv returns the provider type from environment variable
func GetProviderFromEnv() ProviderType {
	provider := os.Getenv("LLM_PROVIDER")
	if provider == "" {
		return OpenAI // Default to OpenAI
	}

	if IsValidProvider(provider) {
		return ProviderType(provider)
	}

	return OpenAI // Default to OpenAI if invalid
}
