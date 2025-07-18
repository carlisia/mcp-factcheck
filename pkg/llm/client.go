// Package llm provides telemetry-aware wrappers for LLM operations.
package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/carlisia/mcp-factcheck/internal/integrations/llm"
	"github.com/carlisia/mcp-factcheck/pkg/logger"
)

// CompletionOptions re-exports the internal type for public use
type CompletionOptions = llm.CompletionOptions

// Config re-exports the internal type for public use
type Config = llm.Config

// ProviderType re-exports the internal type for public use
type ProviderType = llm.ProviderType

// Provider constants
const (
	OpenAI    = llm.OpenAI
	Anthropic = llm.Anthropic
	Gemini    = llm.Gemini
)

// GetProviderFromEnv returns the provider type from environment variable
func GetProviderFromEnv() ProviderType {
	return llm.GetProviderFromEnv()
}

// IsValidProvider checks if a provider type is valid
func IsValidProvider(provider string) bool {
	return llm.IsValidProvider(provider)
}

// ValidProviders returns all valid provider types
func ValidProviders() []ProviderType {
	return llm.ValidProviders()
}

// GetAPIKeyFromEnv retrieves the API key from the appropriate environment variable
func GetAPIKeyFromEnv(provider ProviderType) string {
	return llm.GetAPIKeyFromEnv(provider)
}

// Client wraps the internal LLM client with telemetry
type Client struct {
	internal  *llm.Client
	telemetry *logger.TelemetryProvider
}

// New creates a new telemetry-aware LLM client
func New(telemetry *logger.TelemetryProvider) (*Client, error) {
	internal, err := llm.New()
	if err != nil {
		return nil, fmt.Errorf("creating internal LLM client: %w", err)
	}

	if telemetry == nil {
		telemetry = logger.NewNoOpTelemetryProvider()
	}

	return &Client{
		internal:  internal,
		telemetry: telemetry,
	}, nil
}

// NewWithProvider creates a new telemetry-aware LLM client with a specific provider
func NewWithProvider(cfg llm.Config, telemetry *logger.TelemetryProvider) (*Client, error) {
	internal, err := llm.NewWithProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating internal LLM client with provider: %w", err)
	}

	if telemetry == nil {
		telemetry = logger.NewNoOpTelemetryProvider()
	}

	return &Client{
		internal:  internal,
		telemetry: telemetry,
	}, nil
}

// CreateEmbedding generates embeddings with telemetry
func (c *Client) CreateEmbedding(ctx context.Context, text string) ([]float64, error) {
	// Get the actual embedding model from the internal client
	embeddingModel := c.internal.EmbeddingModel()

	// Start telemetry span
	ctx, span := c.telemetry.StartEmbeddingSpan(ctx, embeddingModel, len(text))
	defer span.End()

	// Set input value
	logger.SetSpanAttributes(ctx, logger.Attribute("input.value", text))

	// Call internal client
	embedding, err := c.internal.CreateEmbedding(ctx, text)
	if err != nil {
		logger.RecordError(ctx, err)
		return nil, err
	}

	// Record success metrics
	logger.SetSpanAttributes(ctx,
		logger.Attribute("embedding.model_name", embeddingModel),
		logger.Attribute("embedding.dimensions", len(embedding)),
		logger.Attribute("output.value", fmt.Sprintf("[%d-dimensional embedding]", len(embedding))),
	)

	return embedding, nil
}

// CompleteJSON sends a chat completion request with telemetry
func (c *Client) CompleteJSON(ctx context.Context, prompt string, opts llm.CompletionOptions, result any) error {
	// Start telemetry span
	ctx, span := c.telemetry.StartLLMSpan(ctx, "chat", opts.Model)
	defer span.End()

	// Estimate input tokens (rough approximation)
	estimatedInputTokens := len(prompt) / 4

	// Set input value
	logger.SetSpanAttributes(ctx,
		logger.Attribute("input.value", prompt),
		logger.Attribute("llm.token_count.prompt", estimatedInputTokens),
		logger.Attribute("llm.invocation_parameters", fmt.Sprintf(`{"model":"%s","temperature":%f,"max_tokens":%d}`, opts.Model, opts.Temperature, opts.MaxTokens)),
	)

	// Build messages array for OpenInference
	messages := []map[string]string{
		{"role": "user", "content": prompt},
	}
	if opts.SystemPrompt != "" {
		messages = append([]map[string]string{
			{"role": "system", "content": opts.SystemPrompt},
		}, messages...)
	}
	if messagesJSON, err := json.Marshal(messages); err != nil {
		logger.SetSpanAttributes(ctx, logger.Attribute("llm.input_messages", fmt.Sprintf("error marshaling: %v", err)))
	} else {
		logger.SetSpanAttributes(ctx, logger.Attribute("llm.input_messages", string(messagesJSON)))
	}

	// Call internal client
	err := c.internal.CompleteJSON(ctx, prompt, opts, result)
	if err != nil {
		logger.RecordError(ctx, err)
		return err
	}

	// Estimate output tokens and set output
	if resultJSON, err := json.Marshal(result); err != nil {
		logger.SetSpanAttributes(ctx,
			logger.Attribute("output.value", fmt.Sprintf("error marshaling: %v", err)),
			logger.Attribute("llm.token_count.completion", 0),
			logger.Attribute("llm.token_count.total", estimatedInputTokens),
		)
	} else {
		estimatedOutputTokens := len(resultJSON) / 4
		logger.SetSpanAttributes(ctx,
			logger.Attribute("output.value", string(resultJSON)),
			logger.Attribute("llm.token_count.completion", estimatedOutputTokens),
			logger.Attribute("llm.token_count.total", estimatedInputTokens+estimatedOutputTokens),
		)

		// Set output messages
		outputMessages := []map[string]string{
			{"role": "assistant", "content": string(resultJSON)},
		}
		if outputMessagesJSON, err := json.Marshal(outputMessages); err != nil {
			logger.SetSpanAttributes(ctx, logger.Attribute("llm.output_messages", fmt.Sprintf("error marshaling: %v", err)))
		} else {
			logger.SetSpanAttributes(ctx, logger.Attribute("llm.output_messages", string(outputMessagesJSON)))
		}
	}

	return nil
}

// Complete generates a plain text completion with telemetry
func (c *Client) Complete(ctx context.Context, prompt string, opts llm.CompletionOptions) (string, error) {
	// Start telemetry span
	ctx, span := c.telemetry.StartLLMSpan(ctx, "completion", opts.Model)
	defer span.End()

	// Set prompt as input value
	logger.SetSpanAttributes(ctx,
		logger.Attribute("input.value", prompt),
		logger.Attribute("llm.token_count.prompt", len(prompt)/4)) // rough estimate

	// Call internal client
	response, err := c.internal.Complete(ctx, prompt, opts)
	if err != nil {
		span.RecordError(err)
		logger.SetSpanAttributes(ctx, logger.Attribute("error", err.Error()))
		return "", err
	}

	// Set output attributes
	logger.SetSpanAttributes(ctx,
		logger.Attribute("output.value", response),
		logger.Attribute("llm.token_count.completion", len(response)/4), // rough estimate
		logger.Attribute("llm.token_count.total", len(prompt)/4+len(response)/4))

	return response, nil
}

// EmbeddingModel returns the embedding model being used
func (c *Client) EmbeddingModel() string {
	return c.internal.EmbeddingModel()
}
