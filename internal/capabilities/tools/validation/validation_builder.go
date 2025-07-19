package validation

import (
	"context"
	"fmt"

	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools"
	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools/contentprep"
)

// Constants
const (
	defaultSearchTopK = 10
)

// ValidationBuilder builds and executes validation pipelines
type ValidationBuilder struct {
	// Required fields
	content     string
	specVersion string

	// Functions
	embedFunc  tools.EmbeddingFunc
	searchFunc tools.SearchFunc
	llmFunc    LLMCompleteFunc

	// Options
	searchStrategy SearchStrategy
	validationType ValidationType
	chunks         []contentprep.Chunk

	// Error tracking
	errors []error
}

// SearchStrategy defines how to search for relevant specs
type SearchStrategy interface {
	Search(ctx context.Context, content, specVersion string, embedFunc tools.EmbeddingFunc, searchFunc tools.SearchFunc) ([]tools.SearchResult, error)
}

// ValidationType defines how to validate content
type ValidationType int

const (
	StandardValidation ValidationType = iota
	QuickClaimValidation
	ChunkValidation
)

// NewValidationBuilder creates a new validation builder
func NewValidationBuilder(content, specVersion string) *ValidationBuilder {
	return &ValidationBuilder{
		content:        content,
		specVersion:    specVersion,
		validationType: StandardValidation,
	}
}

// WithFunctions sets the required functions
func (vb *ValidationBuilder) WithFunctions(embedFunc tools.EmbeddingFunc, searchFunc tools.SearchFunc, llmFunc LLMCompleteFunc) *ValidationBuilder {
	vb.embedFunc = embedFunc
	vb.searchFunc = searchFunc
	vb.llmFunc = llmFunc
	return vb
}

// WithSearchStrategy sets a custom search strategy
func (vb *ValidationBuilder) WithSearchStrategy(strategy SearchStrategy) *ValidationBuilder {
	vb.searchStrategy = strategy
	return vb
}

// AsQuickClaim configures for quick claim validation
func (vb *ValidationBuilder) AsQuickClaim() *ValidationBuilder {
	vb.validationType = QuickClaimValidation
	vb.searchStrategy = &AggressiveSearchStrategy{
		PrimaryTopK:  quickSearchTopK,
		FallbackTopK: fallbackTopK,
	}
	return vb
}

// AsChunkValidation configures for chunk validation
func (vb *ValidationBuilder) AsChunkValidation() *ValidationBuilder {
	vb.validationType = ChunkValidation
	return vb
}

// WithChunks sets the chunks for validation
func (vb *ValidationBuilder) WithChunks(chunks []contentprep.Chunk) *ValidationBuilder {
	vb.chunks = chunks
	return vb
}

// Build validates the configuration and returns errors if any
func (vb *ValidationBuilder) Build() error {
	if vb.embedFunc == nil {
		vb.errors = append(vb.errors, fmt.Errorf("embedFunc is required"))
	}
	if vb.searchFunc == nil {
		vb.errors = append(vb.errors, fmt.Errorf("searchFunc is required"))
	}
	if vb.llmFunc == nil {
		vb.errors = append(vb.errors, fmt.Errorf("llmFunc is required"))
	}
	if vb.content == "" {
		vb.errors = append(vb.errors, fmt.Errorf("content is required"))
	}

	if len(vb.errors) > 0 {
		return fmt.Errorf("validation builder errors: %v", vb.errors)
	}
	return nil
}

// Execute runs the validation pipeline
func (vb *ValidationBuilder) Execute(ctx context.Context) (*Result, error) {
	// Validate configuration
	if err := vb.Build(); err != nil {
		return nil, err
	}

	// Search for relevant content
	searchResults, err := vb.performSearch(ctx)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Perform validation based on type
	var factCheckResult *FactCheckResult
	switch vb.validationType {
	case QuickClaimValidation:
		factCheckResult, err = performQuickClaimCheck(ctx, vb.llmFunc, vb.content, searchResults)
	default:
		factCheckResult, err = performClaimCheck(ctx, vb.llmFunc, vb.content, searchResults)
	}

	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Build result
	result := vb.buildResult(factCheckResult)

	// Apply type-specific formatting
	if vb.validationType == QuickClaimValidation {
		vb.formatQuickClaimResult(result, vb.content)
	}

	return result, nil
}

// performSearch executes the search strategy
func (vb *ValidationBuilder) performSearch(ctx context.Context) ([]tools.SearchResult, error) {
	// Use custom strategy if provided
	if vb.searchStrategy != nil {
		return vb.searchStrategy.Search(ctx, vb.content, vb.specVersion, vb.embedFunc, vb.searchFunc)
	}

	// Default strategy based on content type
	if isNegativeClaim(vb.content) {
		return searchForNegativeClaim(ctx, vb.content, vb.specVersion, vb.embedFunc, vb.searchFunc)
	}

	// Standard search
	return tools.EmbedAndSearch(ctx, vb.content, vb.specVersion, defaultSearchTopK, vb.embedFunc, vb.searchFunc)
}

// buildResult creates a Result from FactCheckResult
func (vb *ValidationBuilder) buildResult(factCheckResult *FactCheckResult) *Result {
	return &Result{
		IsValid:          factCheckResult.IsAccurate,
		Confidence:       factCheckResult.Confidence,
		ParsedClaims:     factCheckResult.ParsedClaims,
		Issues:           factCheckResult.Inaccuracies,
		Suggestions:      factCheckResult.Suggestions,
		CorrectedVersion: factCheckResult.CorrectedVersion,
		SpecVersion:      vb.specVersion,
		FactCheckResult:  factCheckResult,
	}
}

// formatQuickClaimResult applies quick claim specific formatting
func (vb *ValidationBuilder) formatQuickClaimResult(result *Result, claim string) {
	if result.IsValid {
		result.ParsedClaims = []string{fmt.Sprintf("✓ ACCURATE: %s", claim)}
		if result.FactCheckResult.Explanation != "" {
			result.Issues = []string{result.FactCheckResult.Explanation}
		}
	} else {
		result.ParsedClaims = []string{fmt.Sprintf("✗ INACCURATE: %s", claim)}
		result.Issues = []string{result.FactCheckResult.Explanation}
		if len(result.FactCheckResult.Corrections) > 0 {
			result.Suggestions = result.FactCheckResult.Corrections
			result.CorrectedVersion = result.FactCheckResult.Corrections[0]
		}
	}
}

// StandardSearchStrategy implements default search behavior
type StandardSearchStrategy struct {
	TopK int
}

func (s *StandardSearchStrategy) Search(ctx context.Context, content, specVersion string, embedFunc tools.EmbeddingFunc, searchFunc tools.SearchFunc) ([]tools.SearchResult, error) {
	topK := s.TopK
	if topK == 0 {
		topK = defaultSearchTopK
	}
	return tools.EmbedAndSearch(ctx, content, specVersion, topK, embedFunc, searchFunc)
}

// AggressiveSearchStrategy implements aggressive search with fallback
type AggressiveSearchStrategy struct {
	PrimaryTopK  int
	FallbackTopK int
}

func (s *AggressiveSearchStrategy) Search(ctx context.Context, content, specVersion string, embedFunc tools.EmbeddingFunc, searchFunc tools.SearchFunc) ([]tools.SearchResult, error) {
	// Use the primary topK for initial search
	searchResults, err := tools.EmbedAndSearch(ctx, content, specVersion, s.PrimaryTopK, embedFunc, searchFunc)
	if err != nil {
		return nil, err
	}

	// If we don't have enough results, try with fallback
	if len(searchResults) < s.FallbackTopK {
		return tools.EmbedAndSearch(ctx, content, specVersion, s.FallbackTopK, embedFunc, searchFunc)
	}

	return searchResults, nil
}
