package tools

import (
	"context"
)

const (
	// DefaultSearchTopK is the default number of results for standard searches
	DefaultSearchTopK = 10
)

// ValidationBuilder provides a fluent interface for building validation workflows
// that can be used across different tool types (search, validation, etc.)
type ValidationBuilder struct {
	content     string
	specVersion string
	searchTopK  int
	embedFunc   EmbeddingFunc
	searchFunc  SearchFunc

	// Optional configurations
	searchStrategy SearchStrategy
	preprocessFunc func(string) string
}

// NewValidationBuilder creates a new validation builder
func NewValidationBuilder(content string, specVersion string) *ValidationBuilder {
	return &ValidationBuilder{
		content:     content,
		specVersion: specVersion,
		searchTopK:  DefaultSearchTopK,
	}
}

// WithFunctions sets the required functions
func (vb *ValidationBuilder) WithFunctions(embedFunc EmbeddingFunc, searchFunc SearchFunc) *ValidationBuilder {
	vb.embedFunc = embedFunc
	vb.searchFunc = searchFunc
	return vb
}

// WithSearchTopK sets the number of search results
func (vb *ValidationBuilder) WithSearchTopK(topK int) *ValidationBuilder {
	vb.searchTopK = topK
	return vb
}

// WithSearchStrategy sets a custom search strategy
func (vb *ValidationBuilder) WithSearchStrategy(strategy SearchStrategy) *ValidationBuilder {
	vb.searchStrategy = strategy
	return vb
}

// WithPreprocessing sets a preprocessing function for content
func (vb *ValidationBuilder) WithPreprocessing(fn func(string) string) *ValidationBuilder {
	vb.preprocessFunc = fn
	return vb
}

// Search performs the search operation using the configured strategy
func (vb *ValidationBuilder) Search(ctx context.Context) ([]SearchResult, error) {
	// Apply preprocessing if configured
	content := vb.content
	if vb.preprocessFunc != nil {
		content = vb.preprocessFunc(content)
	}

	// Use default strategy if not provided
	if vb.searchStrategy == nil {
		vb.searchStrategy = &DefaultSearchStrategy{topK: vb.searchTopK}
	}

	// Perform search
	return vb.searchStrategy.Search(ctx, content, vb.specVersion, vb.embedFunc, vb.searchFunc)
}
