package validation

import (
	"context"
	"fmt"
	"strings"

	"github.com/carlisia/mcp-factcheck/internal/capabilities"
	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools"
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
	// ChunkSizeThreshold defines when to automatically enable chunking.
	// Set to 2000 characters to trigger chunking for longer content.
	// The actual chunk size used during chunking is 800 characters with 100 character overlap,
	// as configured in contentprep.Split() for better validation accuracy.
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
	contentRaw, exists := params["content"]
	if !exists {
		return nil, fmt.Errorf("content parameter is required")
	}

	content, ok := contentRaw.(string)
	if !ok {
		return nil, fmt.Errorf("content must be a string")
	}

	specVersion, _ := params["specVersion"].(string)
	useChunking, _ := params["useChunking"].(bool)

	// Validate and build request
	return validateClaimsRequest(content, specVersion, useChunking)
}

// newDefaultClaimsRequest creates a ClaimsRequest with default values
func newDefaultClaimsRequest() ClaimsRequest {
	return ClaimsRequest{
		SpecVersion: capabilities.Latest,
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
	// Only override auto-chunking if explicitly enabling
	// This preserves auto-enable behavior for long content
	if useChunking || !b.request.UseChunking {
		b.request.UseChunking = useChunking
	}
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
	// Use validation builder for standard validation
	builder := NewValidationBuilder(content, specVersion).
		WithFunctions(embedFunc, searchFunc, llmFunc)

	// Add compound claim strategy if needed
	if IsCompoundClaim(content) {
		builder = builder.WithSearchStrategy(&StandardSearchStrategy{TopK: 20})
	}

	return builder.Execute(ctx)
}

// validateWithChunking validates content by splitting it into chunks
func validateWithChunking(ctx context.Context, req ClaimsRequest, embedFunc tools.EmbeddingFunc, searchFunc tools.SearchFunc, llmFunc LLMCompleteFunc) (*Result, error) {
	// Use comprehensive chunking validation
	return validateWithChunkingComprehensive(ctx, req, embedFunc, searchFunc, llmFunc)
}

// Helper functions

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
	var sections []string

	// Header
	if result.IsValid {
		sections = append(sections, "✅ Content is ACCURATE")
	} else {
		sections = append(sections, "❌ Content is INACCURATE")
	}

	// Confidence
	sections = append(sections, fmt.Sprintf("Confidence: %d%%", int(result.Confidence*100)))

	// Individual claims from FactCheckResult
	if result.FactCheckResult != nil && len(result.FactCheckResult.Claims) > 0 {
		sections = append(sections, "", "Claims:")
		for _, claim := range result.FactCheckResult.Claims {
			sections = append(sections, "", claim.Claim)
			if claim.IsAccurate {
				sections = append(sections, "✓ Accurate")
				// Include explanation for accurate claims too
				if claim.Explanation != "" {
					sections = append(sections, claim.Explanation)
				}
			} else {
				sections = append(sections, "✗ Inaccurate")
				if claim.Correction != "" {
					sections = append(sections, "Correction: "+claim.Correction)
				}
				// Include explanation for inaccurate claims
				if claim.Explanation != "" {
					sections = append(sections, claim.Explanation)
				}
			}
		}
	}

	// Inaccuracies
	if result.FactCheckResult != nil && len(result.FactCheckResult.Inaccuracies) > 0 {
		sections = append(sections, "", "Inaccuracies Found:")
		for _, inaccuracy := range result.FactCheckResult.Inaccuracies {
			sections = append(sections, "• "+inaccuracy)
		}
	}

	// Missing best practices
	if result.FactCheckResult != nil && len(result.FactCheckResult.MissingBestPractices) > 0 {
		sections = append(sections, "", "Missing Best Practices:")
		for _, practice := range result.FactCheckResult.MissingBestPractices {
			sections = append(sections, "• "+practice)
		}
	}

	// Suggestions
	if len(result.Suggestions) > 0 && (result.FactCheckResult == nil || len(result.FactCheckResult.MissingBestPractices) == 0) {
		sections = append(sections, "", "Suggestions:")
		for _, suggestion := range result.Suggestions {
			sections = append(sections, "• "+suggestion)
		}
	}

	// Corrected version
	if result.CorrectedVersion != "" {
		sections = append(sections, "", "Corrected Version:", result.CorrectedVersion)
	}

	return sections
}
