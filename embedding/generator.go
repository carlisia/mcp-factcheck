package embedding

import (
	"context"
	"fmt"
	"os"

	"github.com/sashabaranov/go-openai"
)

// Generator handles embedding generation using OpenAI
type Generator struct {
	client *openai.Client
}

// NewGenerator creates a new embedding generator using environment variable
func NewGenerator() (*Generator, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is not set")
	}

	return NewGeneratorWithKey(apiKey)
}

// NewGeneratorWithKey creates a new embedding generator with provided API key
func NewGeneratorWithKey(apiKey string) (*Generator, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key cannot be empty")
	}

	client := openai.NewClient(apiKey)
	return &Generator{client: client}, nil
}

// GenerateEmbedding creates an embedding for a single text chunk
func (g *Generator) GenerateEmbedding(content string) ([]float64, error) {
	resp, err := g.client.CreateEmbeddings(context.Background(), openai.EmbeddingRequest{
		Input: []string{content},
		Model: openai.AdaEmbeddingV2,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned")
	}

	// Convert []float32 to []float64
	embedding := make([]float64, len(resp.Data[0].Embedding))
	for i, v := range resp.Data[0].Embedding {
		embedding[i] = float64(v)
	}

	return embedding, nil
}