package search

import (
	"github.com/carlisia/mcp-factcheck/internal/capabilities"
	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools"
)

const (
	// SpecDBToolName is the name of the search spec tool
	SpecDBToolName = "search-spec"

	// Search parameters limits
	defaultTopK    = 5   // Default number of results to return
	minTopK        = 1   // Minimum allowed value for topK
	maxTopK        = 20  // Maximum allowed value for topK
	maxQueryLength = 500 // Maximum allowed query length in characters
)

// SearchSpecDefinition returns the tool definition for searching MCP specifications
// using semantic similarity. The tool supports querying specific spec versions and
// controlling the number of results returned.
func SearchSpecDefinition() tools.Definition {
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
				"enum":        capabilities.ValidSpecVersions,
				"default":     capabilities.Latest,
			},
			"topK": map[string]any{
				"type":        "integer",
				"description": "Number of top results to return",
				"default":     defaultTopK,
				"minimum":     minTopK,
				"maximum":     maxTopK,
			},
		},
		"required": []string{"query"},
	}

	description := `Search MCP specification using semantic similarity.

This tool helps you:
- Find specific sections in the MCP specification
- Search for information about MCP features
- Locate implementation details and requirements
- Discover related concepts across the spec

Uses AI-powered semantic search to find the most relevant content.`

	return tools.Definition{
		Name:        SpecDBToolName,
		Description: description,
		Schema:      schema,
	}
}
