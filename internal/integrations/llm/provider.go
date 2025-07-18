package llm

import (
	"context"
	"fmt"
	"os"
)

// ProviderType represents the type of LLM provider
type ProviderType string

const (
	OpenAI    ProviderType = "openai"
	Anthropic ProviderType = "anthropic"
	Gemini    ProviderType = "gemini"
)

// validProvidersMap is used for O(1) provider validation
var validProvidersMap = map[ProviderType]struct{}{
	OpenAI:    {},
	Anthropic: {},
	Gemini:    {},
}

// ValidProviders returns all valid provider types
func ValidProviders() []ProviderType {
	providers := make([]ProviderType, 0, len(validProvidersMap))
	for p := range validProvidersMap {
		providers = append(providers, p)
	}
	return providers
}

// IsValidProvider checks if a provider type is valid
func IsValidProvider(provider string) bool {
	_, exists := validProvidersMap[ProviderType(provider)]
	return exists
}

// GetAPIKeyFromEnv retrieves the API key from the appropriate environment variable
// based on the provider type. Returns empty string if not found.
func GetAPIKeyFromEnv(provider ProviderType) string {
	switch provider {
	case OpenAI:
		return os.Getenv("OPENAI_API_KEY")
	case Anthropic:
		return os.Getenv("ANTHROPIC_API_KEY")
	case Gemini:
		return os.Getenv("GEMINI_API_KEY")
	default:
		return ""
	}
}

// Provider defines a generic LLM provider interface
type Provider interface {
	// CreateEmbedding generates embeddings for the given text
	CreateEmbedding(ctx context.Context, text string) ([]float64, error)

	// CompleteJSON performs completion and unmarshals the response into result
	CompleteJSON(ctx context.Context, prompt string, opts CompletionOptions, result any) error

	// Complete performs a standard text completion
	Complete(ctx context.Context, prompt string, opts CompletionOptions) (string, error)
}

// CompletionOptions contains options for LLM completion requests
type CompletionOptions struct {
	Model        string
	Temperature  float32
	MaxTokens    int
	SystemPrompt string
}

// Config holds configuration needed to instantiate providers
type Config struct {
	Type   ProviderType
	APIKey string
}

// NewProvider creates a new LLM provider based on the configuration.
// If APIKey is empty in the config, it will attempt to retrieve it from
// the appropriate environment variable for the provider type.
func NewProvider(cfg Config) (Provider, error) {
	// If no API key provided, try to get from environment
	if cfg.APIKey == "" {
		cfg.APIKey = GetAPIKeyFromEnv(cfg.Type)
	}

	switch cfg.Type {
	case OpenAI:
		return newOpenAIProvider(cfg.APIKey)
	case Anthropic:
		return newAnthropicProvider(cfg.APIKey)
	case Gemini:
		return newGeminiProvider(cfg.APIKey)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", cfg.Type)
	}
}
