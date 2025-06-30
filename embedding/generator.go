package embedding

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/carlisia/mcp-factcheck/internal/prompts"
	"github.com/sashabaranov/go-openai"
)

const (
	// LLM configuration for fact-checking
	factCheckModel       = openai.GPT4oMini
	factCheckTemperature = 0.0   // Zero temperature for deterministic fact-checking
	factCheckMaxTokens   = 1000  // Max tokens for fact-check response
)

// Generator handles embedding generation and LLM operations using OpenAI
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

// ClaimResult represents a single claim's fact-check result
type ClaimResult struct {
	Claim       string `json:"claim"`
	IsAccurate  bool   `json:"is_accurate"`
	Correction  string `json:"correction"`
	Explanation string `json:"explanation"`
}

// FactCheckResponse represents the new format from the fact-checking prompt
type FactCheckResponse struct {
	Claims            []ClaimResult `json:"claims"`
	OverallIsAccurate bool          `json:"overall_is_accurate"`
	Summary           string        `json:"summary"`
}

// FactCheckResult represents the result of fact-checking content against spec
type FactCheckResult struct {
	IsAccurate   bool     `json:"is_accurate"`
	Inaccuracies []string `json:"inaccuracies"`
	Corrections  []string `json:"corrections"`
	Explanation  string   `json:"explanation"`
	ParsedClaims []string `json:"parsed_claims"` // All claims extracted from content
}

// FactCheckAgainstSpec validates content claims against MCP specification sections
func (g *Generator) FactCheckAgainstSpec(content string, specSections []string) (*FactCheckResult, error) {
	// Create the fact-check prompt renderer
	promptRenderer, err := prompts.NewFactCheckPrompt()
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt renderer: %w", err)
	}

	// Render the prompt with the data
	prompt, err := promptRenderer.Render(prompts.FactCheckData{
		Content:      content,
		SpecSections: specSections,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to render prompt: %w", err)
	}

	resp, err := g.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: factCheckModel,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: prompts.FactCheckSystemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			Temperature: factCheckTemperature,
			MaxTokens:   factCheckMaxTokens,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fact-check content: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from fact-checking")
	}

	// Parse the JSON response in the new format
	var response FactCheckResponse
	content = resp.Choices[0].Message.Content
	
	if err := json.Unmarshal([]byte(content), &response); err != nil {
		// Try parsing the old format as fallback
		var result FactCheckResult
		if err2 := json.Unmarshal([]byte(content), &result); err2 == nil {
			return &result, nil
		}
		// If both parsing attempts fail, return a generic error result
		return &FactCheckResult{
			IsAccurate:   false,
			Inaccuracies: []string{"Failed to parse fact-checking response"},
			Explanation:  fmt.Sprintf("Raw response: %s", content),
		}, nil
	}

	// Convert new format to old format for compatibility
	result := &FactCheckResult{
		IsAccurate:   response.OverallIsAccurate,
		Inaccuracies: []string{},
		Corrections:  []string{},
		Explanation:  response.Summary,
		ParsedClaims: []string{},  // Add all parsed claims
	}

	// Extract all claims and track inaccuracies
	for _, claim := range response.Claims {
		result.ParsedClaims = append(result.ParsedClaims, claim.Claim)
		if !claim.IsAccurate {
			result.Inaccuracies = append(result.Inaccuracies, claim.Claim)
			result.Corrections = append(result.Corrections, claim.Correction)
		}
	}

	return result, nil
}