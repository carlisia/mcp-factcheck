package tools

import (
	"context"
	"encoding/json"

	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools"
	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools/search"
	"github.com/carlisia/mcp-factcheck/internal/storage"
	"github.com/carlisia/mcp-factcheck/pkg/llm"
	"github.com/mark3labs/mcp-go/mcp"
)

// SearchSpecTool creates the MCP tool for search_spec from the tool definition
func SearchSpecTool() mcp.Tool {
	def := search.SearchSpecDefinition()

	schemaBytes, err := json.Marshal(def.Schema)
	if err != nil {
		// This should never happen with our static schema
		panic("failed to marshal search spec schema: " + err.Error())
	}

	return mcp.NewToolWithRawSchema(def.Name, def.Description, schemaBytes)
}

// HandleSearchSpec handles the search_spec MCP tool call
func HandleSearchSpec(ctx context.Context, vectorDB *storage.VectorDB, generator *llm.Client, args any) ([]mcp.Content, error) {
	// Parse and validate arguments
	req, err := search.ParseSearchArgs(args)
	if err != nil {
		return nil, err
	}

	// Create adapter functions
	embedFunc := func(ctx context.Context, content string) ([]float64, error) {
		return generator.CreateEmbedding(ctx, content)
	}

	searchFunc := func(version string, queryEmbedding []float64, topK int) ([]tools.SearchResult, error) {
		results, err := vectorDB.Search(ctx, version, queryEmbedding, topK)
		if err != nil {
			return nil, err
		}

		// Convert storage.SearchResult to tools.SearchResult
		searchResults := make([]tools.SearchResult, 0, len(results))
		for _, r := range results {
			searchResults = append(searchResults, tools.SearchResult{
				Content:    r.Content,
				ChunkID:    r.ChunkID,
				Similarity: r.Similarity,
				Version:    r.Version,
				Rank:       r.Rank,
			})
		}
		return searchResults, nil
	}

	// Perform search
	results, err := search.Search(ctx, req, embedFunc, searchFunc)
	if err != nil {
		return nil, err
	}

	// Format results
	formatted := search.FormatResults(req.Query, req.SpecVersion, results)

	// Return MCP content
	return []mcp.Content{
		mcp.NewTextContent(formatted),
	}, nil
}
