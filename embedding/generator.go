package embedding

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/carlisia/mcp-factcheck/internal/prompts"
	"github.com/carlisia/mcp-factcheck/pkg/logger"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

const (
	// LLM configuration for fact-checking
	factCheckModel       = openai.GPT4oMini
	factCheckTemperature = 0.0  // Zero temperature for deterministic fact-checking
	factCheckMaxTokens   = 2500 // Increased to handle longer validation responses with many claims
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

// max returns the larger of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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
	Claims                 []ClaimResult `json:"claims"`
	OverallIsAccurate      bool          `json:"overall_is_accurate"`
	Summary                string        `json:"summary"`
	MissingBestPractices   []string      `json:"missing_best_practices,omitempty"`
	AdvisoryLanguageIssues []string      `json:"advisory_language_issues,omitempty"`
}

// FactCheckResult represents the result of fact-checking content against spec
type FactCheckResult struct {
	IsAccurate             bool     `json:"is_accurate"`
	Inaccuracies           []string `json:"inaccuracies"`
	Corrections            []string `json:"corrections"`
	Explanation            string   `json:"explanation"`
	ParsedClaims           []string `json:"parsed_claims"`            // All claims extracted from content
	MissingBestPractices   []string `json:"missing_best_practices"`   // SHOULD requirements not mentioned
	AdvisoryLanguageIssues []string `json:"advisory_language_issues"` // MAY/CAN confusion
	Claims                 []Claim  `json:"claims"`                   // Detailed claim analysis
	RawResponse            string   `json:"-"`                        // Raw LLM response for debugging
}

// Claim represents a single claim with its validation details
type Claim struct {
	Claim       string `json:"claim"`
	IsAccurate  bool   `json:"is_accurate"`
	Correction  string `json:"correction,omitempty"`
	Explanation string `json:"explanation,omitempty"`
}

// FactCheckAgainstSpec validates content claims against MCP specification sections
func (g *Generator) FactCheckAgainstSpec(content string, specSections []string, compoundEvidence map[string]string) (*FactCheckResult, error) {
	// Create the fact-check prompt renderer
	promptRenderer, err := prompts.NewFactCheckPrompt()
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt renderer: %w", err)
	}

	// Render the prompt with the data
	data := prompts.FactCheckData{
		Content:      content,
		SpecSections: specSections,
	}

	// Add compound evidence if provided
	if len(compoundEvidence) > 0 {
		data.CompoundEvidence = compoundEvidence
	}

	prompt, err := promptRenderer.Render(data)
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
			ResponseFormat: &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONObject,
			},
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

	// Store raw response for debugging
	rawResponse := content

	// Add debug logging to help diagnose parsing issues
	log := logger.Get()
	if log != nil {
		log.Debug("Raw LLM response for fact-checking",
			zap.String("response", content),
			zap.Int("length", len(content)))
	}

	if err := json.Unmarshal([]byte(content), &response); err != nil {
		// Check if this might be a truncation issue
		isTruncated := false
		if jsonErr, ok := err.(*json.SyntaxError); ok {
			// Check if error is at the end of the content (likely truncation)
			if jsonErr.Offset == int64(len(content)) || jsonErr.Offset == int64(len(content)-1) {
				isTruncated = true
			}
		} else if err.Error() == "unexpected end of JSON input" {
			isTruncated = true
		}

		// Log the parsing error details
		if log != nil {
			if isTruncated {
				log.Warn("LLM response appears to be truncated",
					zap.Error(err),
					zap.Int("response_length", len(content)),
					zap.String("last_100_chars", content[max(0, len(content)-100):]))
			} else {
				log.Error("Failed to parse fact-checking response",
					zap.Error(err),
					zap.String("raw_content", content))
			}
		}

		// Try parsing the old format as fallback
		var result FactCheckResult
		if err2 := json.Unmarshal([]byte(content), &result); err2 == nil {
			result.RawResponse = rawResponse
			return &result, nil
		}

		// Return appropriate error message
		errorMsg := "Failed to parse fact-checking response"
		if isTruncated {
			errorMsg = "Response was truncated - content too long for analysis. Consider using chunked validation for large documents."
		}

		return &FactCheckResult{
			IsAccurate:   false,
			Inaccuracies: []string{errorMsg},
			Explanation:  fmt.Sprintf("The validation could not be completed. %s", errorMsg),
			RawResponse:  rawResponse,
		}, nil
	}

	// Convert new format to FactCheckResult
	result := &FactCheckResult{
		IsAccurate:             response.OverallIsAccurate,
		Inaccuracies:           []string{},
		Corrections:            []string{},
		Explanation:            response.Summary,
		ParsedClaims:           []string{},
		MissingBestPractices:   response.MissingBestPractices,
		AdvisoryLanguageIssues: response.AdvisoryLanguageIssues,
		Claims:                 []Claim{},
	}

	// Extract all claims and track inaccuracies
	for _, claim := range response.Claims {
		result.ParsedClaims = append(result.ParsedClaims, claim.Claim)

		// Add to detailed claims
		result.Claims = append(result.Claims, Claim(claim))

		if !claim.IsAccurate {
			result.Inaccuracies = append(result.Inaccuracies, claim.Claim)
			result.Corrections = append(result.Corrections, claim.Correction)
		}
	}

	// Add raw response for debugging
	result.RawResponse = rawResponse

	return result, nil
}
