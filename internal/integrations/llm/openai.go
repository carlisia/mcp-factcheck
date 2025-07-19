package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// Default models for OpenAI
const (
	defaultEmbeddingModel  = openai.LargeEmbedding3
	defaultCompletionModel = openai.GPT4oMini
)

// openAIProvider implements the Provider interface for OpenAI
type openAIProvider struct {
	client          *openai.Client
	embeddingModel  openai.EmbeddingModel
	completionModel string
}

// newOpenAIProvider creates a new OpenAI provider
func newOpenAIProvider(apiKey string) (Provider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key not provided")
	}

	client := openai.NewClient(apiKey)
	return &openAIProvider{
		client:          client,
		embeddingModel:  defaultEmbeddingModel,
		completionModel: defaultCompletionModel,
	}, nil
}

// CreateEmbedding generates embeddings using OpenAI
func (p *openAIProvider) CreateEmbedding(ctx context.Context, text string) ([]float64, error) {
	resp, err := p.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: p.embeddingModel,
	})
	if err != nil {
		return nil, NewAPIError(OpenAI, "failed to create embedding", err, map[string]any{
			"input_length": len(text),
			"model":        string(p.embeddingModel),
		})
	}

	if len(resp.Data) == 0 {
		return nil, NewAPIError(OpenAI, "no embedding data returned", nil, nil)
	}

	// Convert []float32 to []float64
	embedding := make([]float64, len(resp.Data[0].Embedding))
	for i, v := range resp.Data[0].Embedding {
		embedding[i] = float64(v)
	}

	return embedding, nil
}

// CreateEmbeddingsBatch generates embeddings for multiple texts in a single API call
func (p *openAIProvider) CreateEmbeddingsBatch(ctx context.Context, texts []string) ([][]float64, error) {
	resp, err := p.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: texts,
		Model: p.embeddingModel,
	})
	if err != nil {
		return nil, NewAPIError(OpenAI, "failed to create embeddings batch", err, map[string]any{
			"batch_size": len(texts),
			"model":      string(p.embeddingModel),
		})
	}

	if len(resp.Data) != len(texts) {
		return nil, NewAPIError(OpenAI, "embedding count mismatch", nil, map[string]any{
			"expected": len(texts),
			"received": len(resp.Data),
		})
	}

	// Convert all embeddings from []float32 to []float64
	embeddings := make([][]float64, len(resp.Data))
	for i, data := range resp.Data {
		embedding := make([]float64, len(data.Embedding))
		for j, v := range data.Embedding {
			embedding[j] = float64(v)
		}
		embeddings[i] = embedding
	}

	return embeddings, nil
}

// CompleteJSON performs completion and unmarshals the response
func (p *openAIProvider) CompleteJSON(ctx context.Context, prompt string, opts CompletionOptions, result any) error {
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		},
	}

	if opts.SystemPrompt != "" {
		messages = append([]openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: opts.SystemPrompt,
			},
		}, messages...)
	}

	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       p.completionModel,
		Messages:    messages,
		Temperature: opts.Temperature,
		MaxTokens:   opts.MaxTokens,
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	})
	if err != nil {
		return NewAPIError(OpenAI, "chat completion failed", err, map[string]any{
			"model":         p.completionModel,
			"temperature":   opts.Temperature,
			"max_tokens":    opts.MaxTokens,
			"prompt_length": len(prompt),
		})
	}

	if len(resp.Choices) == 0 {
		return NewAPIError(OpenAI, "no completion choices returned", nil, nil)
	}

	content := resp.Choices[0].Message.Content
	if err := json.Unmarshal([]byte(content), result); err != nil {
		// Include a portion of the content in error for debugging
		preview := content
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		return NewAPIError(OpenAI, "failed to parse JSON response", err, map[string]any{
			"response_preview": preview,
			"prompt":           prompt,
			"model":            p.completionModel,
		})
	}

	return nil
}

// Complete performs a standard text completion
func (p *openAIProvider) Complete(ctx context.Context, prompt string, opts CompletionOptions) (string, error) {
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		},
	}

	if opts.SystemPrompt != "" {
		messages = append([]openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: opts.SystemPrompt,
			},
		}, messages...)
	}

	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       p.completionModel,
		Messages:    messages,
		Temperature: opts.Temperature,
		MaxTokens:   opts.MaxTokens,
	})
	if err != nil {
		return "", NewAPIError(OpenAI, "chat completion failed", err, map[string]any{
			"model":         p.completionModel,
			"temperature":   opts.Temperature,
			"max_tokens":    opts.MaxTokens,
			"prompt_length": len(prompt),
		})
	}

	if len(resp.Choices) == 0 {
		return "", NewAPIError(OpenAI, "no completion choices returned", nil, nil)
	}

	return resp.Choices[0].Message.Content, nil
}

// EmbeddingModel returns the embedding model being used
func (p *openAIProvider) EmbeddingModel() string {
	return string(p.embeddingModel)
}

// CompletionModel returns the completion model being used
func (p *openAIProvider) CompletionModel() string {
	return p.completionModel
}
