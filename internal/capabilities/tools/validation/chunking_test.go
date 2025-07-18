package validation_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools"
	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test constants for clarity and maintainability
const (
	// chunkThresholdChars defines the character count that triggers auto-chunking
	chunkThresholdChars = 2000 // This should match validation.ChunkSizeThreshold

	// testContentRepeatPhrase is a standard phrase used for generating test content
	testContentRepeatPhrase = "MCP provides tools and resources. "
)

// TestComprehensiveChunking validates that long content is properly chunked and validated
func TestComprehensiveChunking(t *testing.T) {
	// GIVEN: Long content that exceeds the chunking threshold
	longContent := generateStructuredTestContent()

	// GIVEN: Mock dependencies configured for successful validation
	config := mockConfig{
		searchResults: []tools.SearchResult{
			{Content: "MCP provides a standardized protocol for client-server communication", Similarity: 0.9},
			{Content: "Servers expose tools and resources that clients can interact with", Similarity: 0.85},
			{Content: "The protocol supports JSON-RPC for message formatting", Similarity: 0.8},
		},
		llmResponseBuilder: func(callCount int, prompt string) string {
			// Verify we're processing chunks by checking the prompt structure
			if !strings.Contains(prompt, "USER CONTENT TO CHECK:") {
				t.Error("Expected chunk content in prompt")
			}

			// Return successful validation for each chunk
			return buildValidationResponse(true,
				[]string{fmt.Sprintf("Chunk %d claim validated", callCount)},
				[]string{})
		},
	}
	mocks := setupMocksWithConfig(config)

	// WHEN: Validation is performed with chunking enabled
	req := validation.ClaimsRequest{
		Content:     longContent,
		SpecVersion: "draft",
		UseChunking: true,
	}

	result, err := validation.Claims(context.Background(), req, mocks.embedFunc, mocks.searchFunc, mocks.llmFunc)

	// THEN: Validation should succeed without errors
	require.NoError(t, err, "Chunked validation should not fail")

	// THEN: Content should be validated as accurate
	assert.True(t, result.IsValid, "Expected chunked content to be validated as accurate")

	// THEN: Parsed claims should be collected from all chunks
	assert.NotEmpty(t, result.ParsedClaims, "Expected parsed claims from chunks")
}

// TestChunkingAutoEnabled verifies that chunking is automatically enabled for long content
func TestChunkingAutoEnabled(t *testing.T) {
	// GIVEN: Content that exceeds the chunking threshold
	// 100 repetitions * 34 chars = ~3400 chars (well above 2000 char threshold)
	longContent := generateRepeatedContent(testContentRepeatPhrase, 100)
	t.Logf("Content length: %d chars (threshold: %d)", len(longContent), chunkThresholdChars)

	// GIVEN: Mock dependencies that track chunking behavior
	chunkingDetected := false
	llmCallCount := 0

	config := mockConfig{
		llmResponseBuilder: func(callCount int, prompt string) string {
			llmCallCount = callCount

			// Use helper to detect if chunking is active
			if detectChunkUsage(prompt, callCount, len(longContent)) {
				chunkingDetected = true
				t.Logf("Chunking detected on LLM call %d", callCount)
			}

			return buildValidationResponse(true, []string{}, []string{})
		},
	}
	mocks := setupMocksWithConfig(config)

	// WHEN: Request is parsed WITHOUT explicitly enabling chunking
	args := map[string]any{
		"content":     longContent,
		"specVersion": "draft",
		"useChunking": false, // Not explicitly enabled
	}

	parsedReq, err := validation.ParseClaimsArgs(args)
	require.NoError(t, err, "Failed to parse args")

	// THEN: Chunking should be auto-enabled by the builder
	assert.True(t, parsedReq.UseChunking, "Expected chunking to be auto-enabled by builder for long content")

	// WHEN: Validation is performed
	result, err := validation.Claims(context.Background(), *parsedReq, mocks.embedFunc, mocks.searchFunc, mocks.llmFunc)
	require.NoError(t, err, "Validation should not fail")

	// THEN: Chunking should be detected during validation
	t.Logf("LLM was called %d times", llmCallCount)
	assert.True(t, chunkingDetected, "Expected chunking to be detected")
	assert.GreaterOrEqual(t, llmCallCount, 2, "Expected multiple LLM calls for chunks")

	// THEN: Validation should succeed
	assert.True(t, result.IsValid, "Expected validation to succeed")
}

// TestChunkValidationAggregation verifies proper aggregation of chunk validation results
func TestChunkValidationAggregation(t *testing.T) {
	/*
	 * Aggregation Expectations:
	 * 1. Overall validation fails if ANY chunk is invalid
	 * 2. Issues from all chunks are collected and deduplicated
	 * 3. Missing best practices from chunks are aggregated
	 * 4. Confidence is averaged across all processed chunks
	 * 5. All claims from chunks are collected
	 */

	// GIVEN: Mock dependencies that return different results for different chunks
	chunkCount := 0
	config := mockConfig{
		llmResponseBuilder: func(callCount int, prompt string) string {
			chunkCount = callCount

			// First chunk: INVALID with issues and missing best practices
			if callCount == 1 {
				return `{
					"claims": [{
						"claim": "Invalid claim in chunk 1",
						"is_accurate": false,
						"explanation": "This is incorrect"
					}],
					"missing_best_practices": ["Use proper security"],
					"advisory_language_issues": [],
					"overall_is_accurate": false,
					"summary": "Invalid chunk"
				}`
			}

			// Subsequent chunks: VALID
			return `{
				"claims": [{
					"claim": "Valid claim in chunk ` + fmt.Sprintf("%d", callCount) + `",
					"is_accurate": true,
					"explanation": "This is correct"
				}],
				"missing_best_practices": [],
				"advisory_language_issues": [],
				"overall_is_accurate": true,
				"summary": "Valid chunk"
			}`
		},
	}
	mocks := setupMocksWithConfig(config)

	// GIVEN: Multi-chunk content
	// 100 repetitions * 27 chars = 2700 chars (exceeds 2000 threshold)
	// Expected chunks: ~4 (2700 chars / 800 chunk size)
	content := generateRepeatedContent("Test content for chunking. ", 100)

	// WHEN: Validation is performed with chunking
	req := validation.ClaimsRequest{
		Content:     content,
		SpecVersion: "draft",
		UseChunking: true,
	}

	result, err := validation.Claims(context.Background(), req, mocks.embedFunc, mocks.searchFunc, mocks.llmFunc)
	require.NoError(t, err, "Validation should not fail")

	// THEN: Overall validation should be INVALID (one chunk failed)
	assert.False(t, result.IsValid, "Expected overall validation to be invalid when one chunk is invalid")

	// THEN: Issues should be aggregated from invalid chunks
	assert.NotEmpty(t, result.Issues, "Expected issues from invalid chunk")

	// THEN: Missing best practices should be aggregated
	require.NotNil(t, result.FactCheckResult, "FactCheckResult should not be nil")
	assert.NotEmpty(t, result.FactCheckResult.MissingBestPractices, "Expected missing best practices to be aggregated from chunks")

	// THEN: All claims should be collected
	assert.GreaterOrEqual(t, len(result.FactCheckResult.Claims), chunkCount,
		"Expected at least %d claims (one per chunk), got %d", chunkCount, len(result.FactCheckResult.Claims))

	// THEN: Issues should be deduplicated
	assert.False(t, hasDuplicates(result.Issues),
		"Expected deduplication of issues, but found duplicates: %v", findDuplicates(result.Issues))

	// THEN: Parsed claims should be deduplicated
	assert.False(t, hasDuplicates(result.ParsedClaims),
		"Expected deduplication of parsed claims, but found duplicates: %v", findDuplicates(result.ParsedClaims))
}

// TestChunkingErrorHandling verifies graceful handling of chunk processing errors
func TestChunkingErrorHandling(t *testing.T) {
	/*
	 * Error Handling Expectations:
	 * 1. Processing continues even when some chunks fail
	 * 2. Failed chunks are tracked and reported
	 * 3. Overall validation fails if any chunks have errors
	 * 4. Warning message is added to issues about failed chunks
	 */

	// GIVEN: Mock dependencies that fail on specific chunks
	embedCallCount := 0
	config := mockConfig{
		embedErrorOnCall: []int{2, 4}, // Fail on chunks 2 and 4
		embedError:       nil,         // Use default error message
		llmResponseBuilder: func(callCount int, prompt string) string {
			// Return valid response for chunks that process successfully
			return buildValidationResponse(true, []string{}, []string{})
		},
	}

	// Override embed function to count calls and provide detailed error tracking
	config.embedResult = []float64{0.1}
	mocks := setupMocksWithConfig(config)

	// Wrap embedFunc to count calls and log when errors occur
	originalEmbed := mocks.embedFunc
	mocks.embedFunc = func(ctx context.Context, content string) ([]float64, error) {
		embedCallCount++
		result, err := originalEmbed(ctx, content)
		if err != nil {
			// Log when chunks fail for debugging purposes
			t.Logf("Chunk %d failed with error (expected for chunks 2 and 4): %v", embedCallCount, err)
		}
		return result, err
	}

	// GIVEN: Content that will result in multiple chunks
	// 50 repetitions * 100 chars = 5000 chars (well above threshold)
	// Expected chunks: ~7 (5000 chars / 800 chunk size)
	// Chunks 2 and 4 will fail with embedding errors
	content := generateRepeatedContent("This is test content for chunking validation. It needs to be long enough to trigger multiple chunks. ", 50)

	// WHEN: Validation is performed with chunking
	req := validation.ClaimsRequest{
		Content:     content,
		SpecVersion: "draft",
		UseChunking: true,
	}

	result, err := validation.Claims(context.Background(), req, mocks.embedFunc, mocks.searchFunc, mocks.llmFunc)

	// THEN: Should return a result even with chunk errors (no top-level error)
	require.NoError(t, err, "Validation should not fail at top level")
	require.NotNil(t, result, "Expected result even with chunk errors")

	// THEN: Overall validation should be INVALID due to chunk errors
	assert.False(t, result.IsValid, "Expected validation to be invalid due to chunk errors")

	// THEN: Issues should contain a warning about failed chunks
	hasWarning := false
	for _, issue := range result.Issues {
		if strings.Contains(issue, "Warning:") && strings.Contains(issue, "chunks failed") {
			hasWarning = true
			t.Logf("Found expected warning: %s", issue)
			break
		}
	}
	assert.True(t, hasWarning, "Expected warning about failed chunks in issues, got: %v", result.Issues)

	t.Logf("Total embed calls: %d (some failed as expected)", embedCallCount)
}

// TestChunkingDeduplication verifies that duplicate issues and claims are properly deduplicated
func TestChunkingDeduplication(t *testing.T) {
	// GIVEN: Mock that returns duplicate issues and claims across chunks
	config := mockConfig{
		llmResponseBuilder: func(callCount int, prompt string) string {
			// Return the same issues and claims for multiple chunks to test deduplication
			return `{
				"claims": [
					{
						"claim": "MCP provides authentication",
						"is_accurate": true,
						"explanation": "Correctly stated"
					},
					{
						"claim": "MCP requires TLS",
						"is_accurate": false,
						"explanation": "MCP recommends but doesn't require TLS"
					}
				],
				"missing_best_practices": ["Implement proper error handling", "Use timeouts"],
				"advisory_language_issues": [],
				"overall_is_accurate": false,
				"summary": "Some claims are inaccurate"
			}`
		},
	}
	mocks := setupMocksWithConfig(config)

	// GIVEN: Content that will produce multiple chunks with same validation results
	// This tests the deduplication logic when chunks have overlapping claims/issues
	content := generateRepeatedContent("MCP provides authentication. MCP requires TLS. ", 100)

	req := validation.ClaimsRequest{
		Content:     content,
		SpecVersion: "draft",
		UseChunking: true,
	}

	// WHEN: Validation is performed
	result, err := validation.Claims(context.Background(), req, mocks.embedFunc, mocks.searchFunc, mocks.llmFunc)
	require.NoError(t, err, "Validation should not fail")

	// THEN: Parsed claims should be deduplicated
	assert.False(t, hasDuplicates(result.ParsedClaims),
		"Expected deduplication of parsed claims, found duplicates: %v", findDuplicates(result.ParsedClaims))

	// THEN: Issues should be deduplicated
	assert.False(t, hasDuplicates(result.Issues),
		"Expected deduplication of issues, found duplicates: %v", findDuplicates(result.Issues))

	// THEN: Suggestions should be deduplicated
	assert.False(t, hasDuplicates(result.Suggestions),
		"Expected deduplication of suggestions, found duplicates: %v", findDuplicates(result.Suggestions))

	// THEN: Missing best practices should be deduplicated
	if result.FactCheckResult != nil {
		assert.False(t, hasDuplicates(result.FactCheckResult.MissingBestPractices),
			"Expected deduplication of missing best practices, found duplicates: %v",
			findDuplicates(result.FactCheckResult.MissingBestPractices))
	}

	// Log the deduplicated counts for verification
	t.Logf("Deduplicated results - Claims: %d, Issues: %d, Suggestions: %d",
		len(result.ParsedClaims), len(result.Issues), len(result.Suggestions))
}

// TestChunkingWithDifferentContentTypes validates chunking behavior with various content structures
func TestChunkingWithDifferentContentTypes(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantChunks  int // minimum expected chunks
		description string
	}{
		{
			name: "markdown_content",
			content: `# MCP Overview
			
## Tools
MCP provides tools that can be invoked by clients.

## Resources  
Resources are exposed with unique URIs.

## Security
Implementations should follow security best practices.`,
			wantChunks:  1,
			description: "Markdown content should be chunked preserving structure",
		},
		{
			name:        "code_heavy_content",
			content:     generateRepeatedContent("```json\n{\n  \"method\": \"tools/call\",\n  \"params\": {\n    \"name\": \"example\"\n  }\n}\n```\n\nThis shows how to call tools.\n", 50),
			wantChunks:  2,
			description: "Code blocks should be handled appropriately in chunks",
		},
		{
			name: "list_content",
			content: generateRepeatedContent(`
- MCP provides authentication mechanisms
- MCP supports authorization patterns  
- MCP enables secure communication
- MCP implements error handling
`, 50),
			wantChunks:  2,
			description: "List items should be chunked logically",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Track number of chunks processed
			chunkCount := 0

			config := mockConfig{
				llmResponseBuilder: func(callCount int, prompt string) string {
					chunkCount = callCount
					return buildValidationResponse(true, []string{}, []string{})
				},
			}
			mocks := setupMocksWithConfig(config)

			req := validation.ClaimsRequest{
				Content:     tt.content,
				SpecVersion: "draft",
				UseChunking: true,
			}

			result, err := validation.Claims(context.Background(), req, mocks.embedFunc, mocks.searchFunc, mocks.llmFunc)
			require.NoError(t, err, "%s: validation should not fail", tt.description)

			assert.True(t, result.IsValid, "%s: expected valid result", tt.description)

			assert.GreaterOrEqual(t, chunkCount, tt.wantChunks,
				"%s: expected at least %d chunks, got %d", tt.description, tt.wantChunks, chunkCount)
		})
	}
}
