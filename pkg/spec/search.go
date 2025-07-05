// Package spec provides MCP tools for searching and listing MCP specifications.
// It implements semantic search capabilities and version management for MCP specs.
package spec

import (
	"context"
	"fmt"

	"github.com/carlisia/mcp-factcheck/internal/embedding/core"
	"github.com/carlisia/mcp-factcheck/pkg/embedtypes"
	mcpembedding "github.com/carlisia/mcp-factcheck/internal/embedding"
	"github.com/carlisia/mcp-factcheck/pkg/specs"
	"github.com/carlisia/mcp-factcheck/pkg/utils"
	"github.com/mark3labs/mcp-go/mcp"
)

// vectorDB defines the interface for vector database operations needed by search.
// It provides semantic search capabilities over embedded specification content.
type vectorDB interface {
	Search(version string, queryEmbedding []float64, topK int) ([]embedtypes.SearchResult, error)
}

// embeddingGenerator defines the interface for embedding generation needed by search.
// It converts text queries into vector embeddings for semantic similarity matching.
type embeddingGenerator interface {
	GenerateEmbedding(ctx context.Context, content string) ([]float64, error)
}

// SearchSpecToolName is the name of the MCP tool for searching specifications.
const SearchSpecToolName = "search_spec"

// GetSearchSpecTool returns the MCP tool definition for searching MCP specifications
// using semantic similarity. The tool supports querying specific spec versions and
// controlling the number of results returned.
func GetSearchSpecTool() mcp.Tool {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query to find relevant specification content",
			},
			"specVersion": map[string]any{
				"type":        "string",
				"description": "MCP specification version to search",
				"enum":        specs.ValidSpecVersions,
				"default":     specs.DefaultSpecVersion,
			},
			"topK": map[string]any{
				"type":        "integer",
				"description": "Number of top results to return",
				"default":     5,
				"minimum":     1,
				"maximum":     20,
			},
		},
		"required": []string{"query"},
	}
	schemaBytes := utils.MustMarshalSchema(schema, SearchSpecToolName)
	return mcp.NewToolWithRawSchema(SearchSpecToolName, "Search MCP specification using semantic similarity", schemaBytes)
}

// HandleSearchSpec processes search requests against MCP specifications.
// It generates embeddings for the query and performs semantic search to find
// the most relevant specification sections.
func HandleSearchSpec(ctx context.Context, db vectorDB, gen embeddingGenerator, args any) ([]mcp.Content, error) {
	params, ok := args.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("arguments must be a map")
	}
	query, ok := params["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query must be a string")
	}

	specVersion, ok := params["specVersion"].(string)
	if !ok {
		specVersion = specs.DefaultSpecVersion
	}

	topK := 5
	if k, ok := params["topK"].(float64); ok {
		topK = int(k)
	}

	if !specs.IsValidSpecVersion(specVersion) {
		return nil, fmt.Errorf("invalid spec version: %s", specVersion)
	}

	// Generate embedding for query
	queryEmbedding, err := gen.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Search specifications
	results, err := db.Search(specVersion, queryEmbedding, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to search specifications: %w", err)
	}

	// Build response content
	var contentParts []mcp.Content
	contentParts = append(contentParts, mcp.NewTextContent(
		fmt.Sprintf("Search results for '%s' in MCP %s:\n\n", query, specVersion)))

	for _, match := range results {
		contentParts = append(contentParts, mcp.NewTextContent(
			fmt.Sprintf("Rank %d (similarity: %.4f):\n%s\n\n",
				match.Rank, match.Similarity, match.Chunk.Content)))
	}

	return contentParts, nil
}

// HandleSearchSpecMCP is the MCP-compatible wrapper that accepts concrete types
func HandleSearchSpecMCP(ctx context.Context, vectorDB *mcpembedding.VectorDB, generator *core.Generator, args any) ([]mcp.Content, error) {
	return HandleSearchSpec(ctx, vectorDB, generator, args)
}
