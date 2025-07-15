package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/carlisia/mcp-factcheck/internal/storage"
	"github.com/carlisia/mcp-factcheck/internal/tools"
	"github.com/carlisia/mcp-factcheck/internal/tools/validation"
	"github.com/carlisia/mcp-factcheck/pkg/llm"
	"github.com/mark3labs/mcp-go/mcp"
)

// ClaimsTool creates the MCP tool for check_mcp_claim from the tool definition
func ClaimsTool() mcp.Tool {
	def := validation.ClaimsDefinition()

	schemaBytes, err := json.Marshal(def.Schema)
	if err != nil {
		// This should never happen with our static schema
		panic("failed to marshal claims schema: " + err.Error())
	}

	return mcp.NewToolWithRawSchema(def.Name, def.Description, schemaBytes)
}

// HandleClaimsValidation handles the check_mcp_claim MCP tool call
func HandleClaimsValidation(ctx context.Context, vectorDB *storage.VectorDB, generator *llm.Client, args any) ([]mcp.Content, error) {
	// Parse and validate arguments
	req, err := validation.ParseClaimsArgs(args)
	if err != nil {
		return nil, err
	}

	// Create adapter functions
	embedFunc := func(ctx context.Context, content string) ([]float64, error) {
		return generator.CreateEmbedding(ctx, content)
	}

	searchFunc := func(version string, queryEmbedding []float64, topK int) ([]tools.SearchResult, error) {
		results, err := vectorDB.Search(version, queryEmbedding, topK)
		if err != nil {
			return nil, err
		}

		validateResults := make([]tools.SearchResult, 0, len(results))
		for _, r := range results {
			validateResults = append(validateResults, tools.SearchResult{
				Content:    r.Content,
				ChunkID:    r.ChunkID,
				Similarity: r.Similarity,
				Version:    r.Version,
				Rank:       r.Rank,
			})
		}
		return validateResults, nil
	}

	llmFunc := func(ctx context.Context, model string, prompt string, temperature float64, maxTokens int) (string, error) {
		opts := llm.CompletionOptions{
			Model:       model,
			Temperature: float32(temperature),
			MaxTokens:   maxTokens,
		}
		return generator.Complete(ctx, prompt, opts)
	}

	// Perform validation
	result, err := validation.Claims(ctx, *req, embedFunc, searchFunc, llmFunc)
	if err != nil {
		return nil, fmt.Errorf("claims validation failed: %w", err)
	}

	// Format response
	return formatClaimsResult(result), nil
}

// formatClaimsResult formats validation results for claims
func formatClaimsResult(result *validation.Result) []mcp.Content {
	// Use the centralized formatter
	sections := validation.FormatClaimsResult(result)

	// Convert sections to MCP content
	content := make([]mcp.Content, 0, len(sections))
	for _, section := range sections {
		content = append(content, mcp.NewTextContent(section))
	}

	return content
}
