package validation

import (
	"context"
	"fmt"
	"github.com/carlisia/mcp-factcheck/internal/capabilities"

	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools"
)

const (
	// fallbackTopK defines the minimum number of results to retrieve if the
	// aggressive search fails to find sufficient relevant content.
	// Set to 5 to balance between:
	// - Ensuring enough context for accurate validation
	// - Maintaining quick response times
	// - Reducing noise from less relevant results
	fallbackTopK = 5

	// quickSearchTopK defines the aggressive number of results for initial search.
	// Set to 15 (3x the fallback) based on:
	// - Quick claims often need more context to find the specific answer
	// - Cast a wider net initially to maximize chance of finding relevant content
	// - LLMs can effectively filter through more results for focused questions
	// - Still fast enough for "quick" validation (typically <3 seconds)
	quickSearchTopK = 15
)

// QuickClaimRequest represents a request to perform quick fact-checking of a
// single MCP-related claim. This is optimized for validating short, specific
// statements about MCP functionality.
type QuickClaimRequest struct {
	// Claim is a single statement or question about MCP to validate.
	// This should be a concise claim or question, not lengthy documentation.
	// Required: true
	// MaxLength: 50000 characters (shared limit with full validation)
	// Recommended: <500 characters for optimal quick validation performance
	// Example: "Does MCP support WebSocket connections?"
	Claim string `json:"claim"`

	// SpecVersion specifies which MCP specification version to validate against.
	// Required: false (defaults to "draft")
	// Valid values: "draft", "2025-06-18", "2025-03-26", "2024-11-05"
	// Example: "2025-03-26"
	SpecVersion string `json:"specVersion,omitempty"`
}

// QuickClaim validates a single MCP claim or question.
//
// Context handling:
//   - Optimized for fast response times (typically <3 seconds)
//   - Uses aggressive search strategies to find relevant content quickly
//   - Cancellation typically indicates:
//   - User navigated away or closed the validation UI
//   - Client-side timeout (quick claims should be fast)
//   - Server shutdown or resource constraints
//   - Unlike full validation, quick claims prioritize speed over completeness
//
// The aggressive search strategy may make multiple search attempts,
// so early cancellation can save significant processing time.
//
// Compound claim handling:
//   - If the claim appears to contain multiple statements, the function
//     will return a result indicating that full validation should be used
//   - This ensures complex claims get comprehensive analysis
func QuickClaim(ctx context.Context, req QuickClaimRequest, embedFunc tools.EmbeddingFunc, searchFunc tools.SearchFunc, llmFunc LLMCompleteFunc) (*Result, error) {
	// Check for early cancellation - quick claims should fail fast
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("quick claim validation cancelled: %w", err)
	}

	// Check if this is a compound claim that should use full validation
	classification := ClassifyClaim(req.Claim)
	if classification.Type == CompoundClaim {
		// Return a special result indicating compound claim detected
		return &Result{
			IsValid: false,
			Issues: []string{
				fmt.Sprintf("Compound claim detected: %s", classification.Suggestion),
				"This claim contains multiple statements that should be validated separately.",
				fmt.Sprintf("Detected indicators: %v", classification.Indicators),
			},
			Suggestions: []string{
				"Please use the full validation tool (check_mcp_claim) for comprehensive analysis of compound claims.",
				"Alternatively, submit each statement as a separate quick claim.",
			},
			ParsedClaims: []string{req.Claim},
			Confidence:   1.0, // We're confident this is a compound claim
		}, nil
	}

	// Use aggressive search strategy for quick facts
	return validateWithAggressiveSearch(ctx, req.Claim, req.SpecVersion, embedFunc, searchFunc, llmFunc)
}

// ParseQuickClaimArgs parses raw arguments into a validated QuickClaimRequest
func ParseQuickClaimArgs(args any) (*QuickClaimRequest, error) {
	// Parse arguments
	params, ok := args.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("quick claim arguments must be a map")
	}

	// Extract parameters
	claim, ok := params["claim"].(string)
	if !ok {
		return nil, fmt.Errorf("claim parameter is required")
	}

	specVersion, _ := params["specVersion"].(string)

	// Validate and build request
	return validateQuickClaimRequest(claim, specVersion)
}

// newDefaultQuickClaimRequest creates a QuickClaimRequest with default values
func newDefaultQuickClaimRequest() QuickClaimRequest {
	return QuickClaimRequest{
		SpecVersion: capabilities.Latest,
	}
}

// quickClaimRequestBuilder builds and validates QuickClaimRequest instances
type quickClaimRequestBuilder struct {
	request QuickClaimRequest
	errors  []error
}

// newQuickClaimRequestBuilder creates a new builder for QuickClaimRequest
func newQuickClaimRequestBuilder() *quickClaimRequestBuilder {
	return &quickClaimRequestBuilder{
		request: newDefaultQuickClaimRequest(),
	}
}

// WithClaim sets the claim to validate
func (b *quickClaimRequestBuilder) WithClaim(claim string) *quickClaimRequestBuilder {
	validated, err := tools.ValidateContentLength(claim, "claim", MaxContentLength)
	if err != nil {
		b.errors = append(b.errors, err)
	}
	b.request.Claim = validated
	return b
}

// WithSpecVersion sets the specification version
func (b *quickClaimRequestBuilder) WithSpecVersion(version string) *quickClaimRequestBuilder {
	validated, err := tools.ValidateSpecVersion(version)
	if err != nil {
		b.errors = append(b.errors, err)
	}
	b.request.SpecVersion = validated
	return b
}

// Build returns the built request or an error if validation failed
func (b *quickClaimRequestBuilder) Build() (*QuickClaimRequest, error) {
	if len(b.errors) > 0 {
		return nil, tools.ValidationErrors{Errors: b.errors}
	}
	return &b.request, nil
}

// validateQuickClaimRequest validates and normalizes a quick claim request
func validateQuickClaimRequest(claim string, version string) (*QuickClaimRequest, error) {
	return newQuickClaimRequestBuilder().
		WithClaim(claim).
		WithSpecVersion(version).
		Build()
}

// validateWithAggressiveSearch performs validation with multiple search strategies.
// The aggressive search strategy works as follows:
//  1. First attempts to retrieve quickSearchTopK (15) results for maximum coverage
//  2. If insufficient relevant results are found, falls back to fallbackTopK (5)
//  3. This two-tier approach ensures quick claims get enough context while
//     maintaining fast response times even for edge cases
func validateWithAggressiveSearch(ctx context.Context, claim, specVersion string, embedFunc tools.EmbeddingFunc, searchFunc tools.SearchFunc, llmFunc LLMCompleteFunc) (*Result, error) {
	// Use aggressive search strategy with fallback
	aggressiveStrategy := &tools.AggressiveSearchStrategy{
		PrimaryTopK:  quickSearchTopK,
		FallbackTopK: fallbackTopK,
	}

	searchResults, err := tools.NewValidationBuilder(claim, specVersion).
		WithFunctions(embedFunc, searchFunc).
		WithSearchStrategy(aggressiveStrategy).
		Search(ctx)
	if err != nil {
		return nil, err
	}

	// Perform quick fact-check
	factCheckResult, err := performQuickClaimCheck(ctx, llmFunc, claim, searchResults)
	if err != nil {
		// Provide more context about the failure type
		return nil, fmt.Errorf("failed quick fact-check (LLM error): %w", err)
	}

	// Build validation result
	result := &Result{
		IsValid:         factCheckResult.IsAccurate,
		Confidence:      factCheckResult.Confidence,
		SpecVersion:     specVersion,
		FactCheckResult: factCheckResult,
	}

	// Format the response
	if factCheckResult.IsAccurate {
		result.ParsedClaims = []string{fmt.Sprintf("✓ ACCURATE: %s", claim)}
		// Include explanation for accurate results too
		if factCheckResult.Explanation != "" {
			result.Issues = []string{factCheckResult.Explanation}
		}
	} else {
		result.ParsedClaims = []string{fmt.Sprintf("✗ INACCURATE: %s", claim)}
		result.Issues = []string{factCheckResult.Explanation}
		if len(factCheckResult.Corrections) > 0 {
			result.Suggestions = factCheckResult.Corrections
			result.CorrectedVersion = factCheckResult.Corrections[0]
		}
	}

	return result, nil
}

// FormatQuickClaimResult formats validation results for quick claims
func FormatQuickClaimResult(result *Result) string {
	formatter := tools.NewResultFormatter()

	// Quick facts start with the verdict
	if len(result.ParsedClaims) > 0 {
		formatter.WithText(result.ParsedClaims[0])
	}

	// Add explanation from issues
	for _, issue := range result.Issues {
		formatter.WithText(issue)
	}

	// Add correction if available
	if result.CorrectedVersion != "" {
		formatter.WithText(fmt.Sprintf("\n**Correct information**: %s", result.CorrectedVersion))
	}

	// Add confidence if needed
	if result.Confidence > 0 {
		formatter.WithText(fmt.Sprintf("\n**Confidence**: %.2f", result.Confidence))
	}

	return formatter.BuildSection()
}
