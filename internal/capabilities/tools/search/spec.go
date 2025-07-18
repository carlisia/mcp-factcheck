package search

import (
	"context"
	"fmt"
	"github.com/carlisia/mcp-factcheck/internal/capabilities"

	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools"
)

// Constants for search request validation
// These are defined in def.go but documented here for visibility:
// - defaultTopK = 5    // Default number of results to return
// - minTopK = 1        // Minimum allowed value for topK
// - maxTopK = 20       // Maximum allowed value for topK
// - maxQueryLength = 500 // Maximum allowed query length in characters

// SearchRequest represents a semantic search request against MCP specifications.
// It allows querying specific versions of the MCP specification using AI-powered
// semantic similarity search.
type SearchRequest struct {
	// Query is the search text used to find relevant specification content.
	// Required: true
	// MaxLength: 500 characters
	// Example: "How does MCP handle authentication?"
	Query string `json:"query"`

	// SpecVersion specifies which MCP specification version to search.
	// Default: "current" (latest released version)
	// Valid values: "current", "draft", "2025-06-18", "2025-03-26", "2024-11-05"
	SpecVersion string `json:"specVersion,omitempty"`

	// TopK controls the number of top results to return.
	// Default: 5
	// Range: 1-20
	// Example: 10
	TopK int `json:"topK,omitempty"`
}

// Search performs semantic search against MCP specifications.
//
// Context handling:
//   - Cancellation: Immediately stops the search operation. This typically happens when:
//   - The MCP client disconnects or cancels the request
//   - The server is shutting down gracefully
//   - A timeout is reached (if configured by the caller)
//   - The search involves two potentially slow operations:
//   - Embedding generation (API call to OpenAI)
//   - Vector similarity search (CPU-intensive)
//   - Early cancellation prevents wasted computation and API costs
//
// Returns an error if the context is cancelled or if any step fails.
func Search(ctx context.Context, req *SearchRequest, embedFunc tools.EmbeddingFunc, searchFunc tools.SearchFunc) ([]tools.SearchResult, error) {
	// Check for early cancellation before starting expensive operations
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("search cancelled before execution: %w", err)
	}

	// Use the validation builder for consistent search behavior
	return tools.NewValidationBuilder(req.Query, req.SpecVersion).
		WithFunctions(embedFunc, searchFunc).
		WithSearchTopK(req.TopK).
		Search(ctx)
}

// ParseSearchArgs parses raw arguments into a validated SearchRequest
func ParseSearchArgs(args any) (*SearchRequest, error) {
	// Parse arguments
	params, ok := args.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("arguments must be a map")
	}

	// Extract parameters
	query, ok := params["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query must be a string")
	}

	specVersion, _ := params["specVersion"].(string)

	// Extract topK - could be nil
	var topK any
	if v, exists := params["topK"]; exists {
		topK = v
	}

	// Validate and build request
	return validateSearchRequest(query, specVersion, topK)
}

// newDefaultSearchRequest creates a SearchRequest with default values
func newDefaultSearchRequest() SearchRequest {
	return SearchRequest{
		SpecVersion: capabilities.Latest,
		TopK:        defaultTopK,
	}
}

// searchRequestBuilder builds and validates SearchRequest instances
type searchRequestBuilder struct {
	request SearchRequest
	errors  []error
}

// newSearchRequestBuilder creates a new builder for SearchRequest
func newSearchRequestBuilder() *searchRequestBuilder {
	return &searchRequestBuilder{
		request: newDefaultSearchRequest(),
	}
}

// WithQuery sets the search query
func (b *searchRequestBuilder) WithQuery(query string) *searchRequestBuilder {
	validated, err := tools.ValidateContentLength(query, "query", maxQueryLength)
	if err != nil {
		b.errors = append(b.errors, err)
	}
	b.request.Query = validated
	return b
}

// WithSpecVersion sets the specification version
func (b *searchRequestBuilder) WithSpecVersion(version string) *searchRequestBuilder {
	validated, err := tools.ValidateSpecVersion(version)
	if err != nil {
		b.errors = append(b.errors, err)
	}
	b.request.SpecVersion = validated
	return b
}

// WithTopK sets the number of results to return
func (b *searchRequestBuilder) WithTopK(topK int) *searchRequestBuilder {
	validated, err := tools.ValidateTopK(topK, minTopK, maxTopK)
	if err != nil {
		b.errors = append(b.errors, err)
	}
	b.request.TopK = validated
	return b
}

// Build returns the built request or an error if validation failed
func (b *searchRequestBuilder) Build() (*SearchRequest, error) {
	if len(b.errors) > 0 {
		return nil, tools.ValidationErrors{Errors: b.errors}
	}
	return &b.request, nil
}

// validateSearchRequest validates and normalizes a search request
func validateSearchRequest(query string, version string, topK any) (*SearchRequest, error) {
	builder := newSearchRequestBuilder().
		WithQuery(query).
		WithSpecVersion(version)

	// Handle topK conversion
	if topK != nil {
		switch v := topK.(type) {
		case float64:
			builder.WithTopK(int(v))
		case int:
			builder.WithTopK(v)
		default:
			// no-op - builder already has default
		}
	}

	return builder.Build()
}

// FormatResults formats search results for display
func FormatResults(query string, version string, results []tools.SearchResult) string {
	formatter := tools.NewResultFormatter().
		WithText(fmt.Sprintf("Search results for '%s' in MCP %s:", query, version))

	for _, match := range results {
		formatter.WithText(fmt.Sprintf("Rank %d (similarity: %.4f):\n%s",
			match.Rank, match.Similarity, match.Content))
	}

	return formatter.BuildSection()
}
