package spec

import (
	"context"
	"fmt"

	"github.com/carlisia/mcp-factcheck/embedding"
	mcpembedding "github.com/carlisia/mcp-factcheck/internal/embedding"
	"github.com/carlisia/mcp-factcheck/internal/specs"
	"github.com/carlisia/mcp-factcheck/internal/utils"
	"github.com/mark3labs/mcp-go/mcp"
)

// vectorDB defines the interface for vector database operations needed by search
type vectorDB interface {
	Search(version string, queryEmbedding []float64, topK int) ([]embedding.SearchResult, error)
}

// embeddingGenerator defines the interface for embedding generation needed by search
type embeddingGenerator interface {
	GenerateEmbedding(ctx context.Context, content string) ([]float64, error)
}

const SearchSpecToolName = "search_spec"

type SearchSpecArgs struct {
	Query       string `json:"query"`
	SpecVersion string `json:"spec_version,omitempty"`
	TopK        int    `json:"top_k,omitempty"`
}

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
func HandleSearchSpecMCP(ctx context.Context, vectorDB *mcpembedding.VectorDB, generator *embedding.Generator, args any) ([]mcp.Content, error) {
	return HandleSearchSpec(ctx, vectorDB, generator, args)
}
