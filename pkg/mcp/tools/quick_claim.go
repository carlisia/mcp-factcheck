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

// QuickClaimTool creates the MCP tool for check_mcp_quick_claim from the tool definition
func QuickClaimTool() mcp.Tool {
	def := validation.QuickClaimDefinition()

	schemaBytes, err := json.Marshal(def.Schema)
	if err != nil {
		// This should never happen with our static schema
		panic("failed to marshal quick fact schema: " + err.Error())
	}

	return mcp.NewToolWithRawSchema(def.Name, def.Description, schemaBytes)
}

// HandleQuickClaimValidation handles the check_mcp_quick_fact MCP tool call
func HandleQuickClaimValidation(ctx context.Context, vectorDB *storage.VectorDB, generator *llm.Client, args any) ([]mcp.Content, error) {
	// Parse and validate arguments
	req, err := validation.ParseQuickClaimArgs(args)
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

		// Convert storage.SearchResult to tools.SearchResult
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
	result, err := validation.QuickClaim(ctx, *req, embedFunc, searchFunc, llmFunc)
	if err != nil {
		return nil, fmt.Errorf("quick fact validation failed: %w", err)
	}

	// Format response
	return formatQuickClaimResult(result), nil
}

// formatQuickClaimResult formats validation results for quick facts
func formatQuickClaimResult(result *validation.Result) []mcp.Content {
	// Use the quick fact formatter to build a single formatted string
	formatted := validation.FormatQuickClaimResult(result)

	// Return as a single content item for quick facts
	return []mcp.Content{
		mcp.NewTextContent(formatted),
	}
}
