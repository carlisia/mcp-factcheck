package validation

import (
	"context"
	"fmt"
	"strings"
	
	"github.com/carlisia/mcp-factcheck/internal/tools"
)

// ClaimsRequest represents a request to validate MCP-related claims or content.
// It supports validation of documentation, tutorials, implementation notes, or
// any text that makes assertions about MCP capabilities or behavior.
type ClaimsRequest struct {
	// Content is the MCP-related text to validate against the specification.
	// This can include documentation, tutorials, blog posts, or any content
	// that describes MCP functionality.
	// Required: true
	// MaxLength: 50000 characters
	// Example: "MCP provides automatic retry logic for failed requests"
	Content string `json:"content"`

	// SpecVersion specifies which MCP specification version to validate against.
	// Required: false (defaults to "draft")
	// Valid values: "draft", "2025-06-18", "2025-03-26", "2024-11-05"
	// Example: "draft"
	SpecVersion string `json:"specVersion,omitempty"`

	// UseChunking enables content chunking for large texts.
	// When enabled, content is split into smaller chunks for more accurate validation.
	// Required: false (auto-enabled for content > 2000 characters)
	// Example: true
	UseChunking bool `json:"useChunking,omitempty"`
}

const (
	// ChunkSizeThreshold defines the maximum size for a single content chunk.
	// Set to 2000 characters based on:
	// - LLM context window efficiency (smaller chunks = more focused analysis)
	// - Embedding quality (embeddings work better on coherent, focused text)
	// - Balance between granularity and processing overhead
	// This threshold ensures each chunk contains roughly 1-3 paragraphs of text,
	// maintaining semantic coherence while enabling accurate validation.
	ChunkSizeThreshold = 2000
)

// Claims validates MCP spec related claims against specifications.
//
// Context handling:
//   - Cancellation: Stops validation immediately to prevent unnecessary API calls
//   - This function orchestrates multiple expensive operations:
//   - Content chunking (if enabled) - CPU intensive for large documents
//   - Embedding generation - External API call to OpenAI
//   - Specification search - Vector similarity computation
//   - LLM fact-checking - External API call, potentially slow
//   - When cancelled:
//   - No partial results are returned (all-or-nothing operation)
//   - Any in-progress API calls are abandoned
//   - Chunked validations stop at the current chunk
//
// The context should have reasonable timeouts set by the caller, as
// validation of large content can take 10-30 seconds.
func Claims(ctx context.Context, req ClaimsRequest, embedFunc tools.EmbeddingFunc, searchFunc tools.SearchFunc, llmFunc LLMCompleteFunc) (*Result, error) {
	// Check for early cancellation before starting validation
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("claims validation cancelled before execution: %w", err)
	}

	// If chunking is enabled, process chunks
	if req.UseChunking {
		return validateWithChunking(ctx, req, embedFunc, searchFunc, llmFunc)
	}

	// Otherwise, validate as single content
	return validateSingleContent(ctx, req.Content, req.SpecVersion, embedFunc, searchFunc, llmFunc)
}

// ParseClaimsArgs parses raw arguments into a validated ClaimsRequest
func ParseClaimsArgs(args any) (*ClaimsRequest, error) {
	// Parse arguments
	params, ok := args.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("claims arguments must be a map")
	}

	// Extract parameters
	content, ok := params["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content parameter is required")
	}

	specVersion, _ := params["specVersion"].(string)
	useChunking, _ := params["useChunking"].(bool)

	// Validate and build request
	return validateClaimsRequest(content, specVersion, useChunking)
}

// newDefaultClaimsRequest creates a ClaimsRequest with default values
func newDefaultClaimsRequest() ClaimsRequest {
	return ClaimsRequest{
		SpecVersion: tools.Current,
		UseChunking: false,
	}
}

// claimsRequestBuilder builds and validates ClaimsRequest instances
type claimsRequestBuilder struct {
	request ClaimsRequest
	errors  []error
}

// newClaimsRequestBuilder creates a new builder for ClaimsRequest
func newClaimsRequestBuilder() *claimsRequestBuilder {
	return &claimsRequestBuilder{
		request: newDefaultClaimsRequest(),
	}
}

// WithContent sets the content to validate
func (b *claimsRequestBuilder) WithContent(content string) *claimsRequestBuilder {
	validated, err := tools.ValidateContentLength(content, "content", MaxContentLength)
	if err != nil {
		b.errors = append(b.errors, err)
	}
	b.request.Content = validated

	// Auto-enable chunking for long content
	if len(validated) > ChunkSizeThreshold {
		b.request.UseChunking = true
	}
	return b
}

// WithSpecVersion sets the specification version
func (b *claimsRequestBuilder) WithSpecVersion(version string) *claimsRequestBuilder {
	validated, err := tools.ValidateSpecVersion(version)
	if err != nil {
		b.errors = append(b.errors, err)
	}
	b.request.SpecVersion = validated
	return b
}

// WithChunking explicitly sets chunking behavior
func (b *claimsRequestBuilder) WithChunking(useChunking bool) *claimsRequestBuilder {
	b.request.UseChunking = useChunking
	return b
}

// Build returns the built request or an error if validation failed
func (b *claimsRequestBuilder) Build() (*ClaimsRequest, error) {
	if len(b.errors) > 0 {
		return nil, tools.ValidationErrors{Errors: b.errors}
	}
	return &b.request, nil
}

// validateClaimsRequest validates and normalizes a claims validation request
func validateClaimsRequest(content string, version string, useChunking bool) (*ClaimsRequest, error) {
	return newClaimsRequestBuilder().
		WithContent(content).
		WithSpecVersion(version).
		WithChunking(useChunking).
		Build()
}

// validateSingleContent validates a single piece of content
func validateSingleContent(ctx context.Context, content, specVersion string, embedFunc tools.EmbeddingFunc, searchFunc tools.SearchFunc, llmFunc LLMCompleteFunc) (*Result, error) {
	// Search for relevant spec sections using validation builder
	searchResults, err := tools.NewValidationBuilder(content, specVersion).
		WithFunctions(embedFunc, searchFunc).
		WithSearchTopK(tools.DefaultSearchTopK).
		Search(ctx)
	if err != nil {
		return nil, err
	}

	// Perform fact-checking
	factCheckResult, err := performClaimCheck(ctx, llmFunc, content, searchResults)
	if err != nil {
		return nil, fmt.Errorf("failed to perform fact-check: %w", err)
	}

	// Build validation result
	result := &Result{
		IsValid:          factCheckResult.IsAccurate,
		Confidence:       factCheckResult.Confidence,
		ParsedClaims:     factCheckResult.ParsedClaims,
		Issues:           factCheckResult.Inaccuracies,
		Suggestions:      factCheckResult.Suggestions,
		CorrectedVersion: factCheckResult.CorrectedVersion,
		SpecVersion:      specVersion,
		FactCheckResult:  factCheckResult,
	}

	return result, nil
}

// validateWithChunking validates content by splitting it into chunks
func validateWithChunking(ctx context.Context, req ClaimsRequest, embedFunc tools.EmbeddingFunc, searchFunc tools.SearchFunc, llmFunc LLMCompleteFunc) (*Result, error) {
	// Split content into chunks
	chunks := chunkContent(req.Content)

	// Validate each chunk
	var allClaims []Claim
	var allIssues []string
	var allSuggestions []string
	var chunkErrors []error
	totalConfidence := 0.0
	validChunks := 0
	processedChunks := 0

	for i, chunk := range chunks {
		result, err := validateSingleContent(ctx, chunk, req.SpecVersion, embedFunc, searchFunc, llmFunc)
		if err != nil {
			// Collect error for reporting but continue processing
			chunkErrors = append(chunkErrors, fmt.Errorf("chunk %d validation failed: %w", i+1, err))
			continue
		}

		processedChunks++
		if result.FactCheckResult != nil {
			allClaims = append(allClaims, result.FactCheckResult.Claims...)
			allIssues = append(allIssues, result.FactCheckResult.Inaccuracies...)
			allSuggestions = append(allSuggestions, result.FactCheckResult.Suggestions...)
			totalConfidence += result.Confidence
			if result.IsValid {
				validChunks++
			}
		}
	}

	// Add chunk error information to issues if any chunks failed
	if len(chunkErrors) > 0 {
		errorSummary := fmt.Sprintf("Warning: %d of %d chunks failed validation", len(chunkErrors), len(chunks))
		allIssues = append([]string{errorSummary}, allIssues...)
	}

	// Calculate confidence only from processed chunks
	avgConfidence := 0.0
	if processedChunks > 0 {
		avgConfidence = totalConfidence / float64(processedChunks)
	}

	// Aggregate results
	isValid := validChunks == len(chunks) && len(chunkErrors) == 0

	// Create corrected version if needed
	correctedVersion := ""
	if !isValid && len(allClaims) > 0 {
		correctedVersion = buildCorrectedVersion(req.Content, allClaims)
	}

	return &Result{
		IsValid:          isValid,
		Confidence:       avgConfidence,
		Issues:           deduplicate(allIssues),
		Suggestions:      deduplicate(allSuggestions),
		CorrectedVersion: correctedVersion,
		SpecVersion:      req.SpecVersion,
		FactCheckResult: &FactCheckResult{
			IsAccurate:       isValid,
			Confidence:       avgConfidence,
			Inaccuracies:     allIssues,
			Suggestions:      allSuggestions,
			CorrectedVersion: correctedVersion,
			Claims:           allClaims,
		},
	}, nil
}

// Helper functions

func chunkContent(content string) []string {
	// Simple chunking by paragraphs or sentences
	var chunks []string
	currentChunk := ""

	for para := range strings.SplitSeq(content, "\n\n") {
		if len(currentChunk)+len(para) > ChunkSizeThreshold && currentChunk != "" {
			chunks = append(chunks, strings.TrimSpace(currentChunk))
			currentChunk = para
		} else {
			if currentChunk != "" {
				currentChunk += "\n\n"
			}
			currentChunk += para
		}
	}

	if currentChunk != "" {
		chunks = append(chunks, strings.TrimSpace(currentChunk))
	}

	return chunks
}

func deduplicate(items []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

func buildCorrectedVersion(original string, claims []Claim) string {
	corrected := original
	for _, claim := range claims {
		if !claim.IsAccurate && claim.Correction != "" {
			// Simple replacement - in production this would be more sophisticated
			corrected = strings.ReplaceAll(corrected, claim.Claim, claim.Correction)
		}
	}
	return corrected
}

// FormatClaimsResult formats validation results for claims
func FormatClaimsResult(result *Result) []string {
	return tools.NewResultFormatter().
		WithConfidence(result.Confidence).
		WithParsedClaims(result.ParsedClaims).
		WithIssues(result.Issues).
		WithSuggestions(result.Suggestions).
		WithCorrectedVersion(result.CorrectedVersion).
		BuildSections()
}
