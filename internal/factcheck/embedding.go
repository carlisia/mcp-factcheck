package factcheck

import (
	"context"
	"os"

	"github.com/sashabaranov/go-openai"
)

var client = openai.NewClient(os.Getenv("OPENAI_API_KEY"))

func GetEmbedding(text string) ([]float32, error) {
	resp, err := client.CreateEmbeddings(context.Background(), openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.AdaEmbeddingV2,
	})
	if err != nil {
		return nil, err
	}
	return resp.Data[0].Embedding, nil
}
