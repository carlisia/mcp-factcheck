package validation_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools"
	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools/validation"
	"github.com/stretchr/testify/assert"
)

// mockDependencies holds all mock functions needed for validation testing
type mockDependencies struct {
	embedFunc  tools.EmbeddingFunc
	searchFunc tools.SearchFunc
	llmFunc    validation.LLMCompleteFunc
}

// mockConfig configures mock behavior
type mockConfig struct {
	// Embedding configuration
	embedError      error
	embedErrorOnCall []int // Fail on specific call numbers
	embedResult     []float64

	// Search configuration
	searchError   error
	searchResults []tools.SearchResult

	// LLM configuration
	llmError           error
	llmResponseBuilder func(callCount int, prompt string) string
	defaultLLMResponse string
}

// setupDefaultMocks creates standard mock functions for testing
func setupDefaultMocks() mockDependencies {
	return mockDependencies{
		embedFunc: func(ctx context.Context, content string) ([]float64, error) {
			return []float64{0.1, 0.2, 0.3}, nil
		},
		searchFunc: func(version string, queryEmbedding []float64, topK int) ([]tools.SearchResult, error) {
			return []tools.SearchResult{
				{Content: "MCP specification content", Similarity: 0.9},
			}, nil
		},
		llmFunc: func(ctx context.Context, model string, prompt string, temperature float64, maxTokens int) (string, error) {
			return `{"claims": [], "overall_is_accurate": true, "summary": "Valid"}`, nil
		},
	}
}

// setupMocksWithConfig creates mock functions with specific behavior
func setupMocksWithConfig(config mockConfig) mockDependencies {
	embedCallCount := 0
	llmCallCount := 0

	return mockDependencies{
		embedFunc: func(ctx context.Context, content string) ([]float64, error) {
			embedCallCount++
			
			// Check if we should fail on this specific call
			for _, failCall := range config.embedErrorOnCall {
				if embedCallCount == failCall {
					if config.embedError != nil {
						return nil, config.embedError
					}
					return nil, fmt.Errorf("embedding error on call %d", embedCallCount)
				}
			}
			
			if config.embedError != nil {
				return nil, config.embedError
			}
			
			if config.embedResult != nil {
				return config.embedResult, nil
			}
			
			return []float64{0.1, 0.2, 0.3}, nil
		},
		
		searchFunc: func(version string, queryEmbedding []float64, topK int) ([]tools.SearchResult, error) {
			if config.searchError != nil {
				return nil, config.searchError
			}
			
			if config.searchResults != nil {
				// Respect topK limit
				if topK < len(config.searchResults) {
					return config.searchResults[:topK], nil
				}
				return config.searchResults, nil
			}
			
			return []tools.SearchResult{
				{Content: "Default MCP spec content", Similarity: 0.9},
			}, nil
		},
		
		llmFunc: func(ctx context.Context, model string, prompt string, temperature float64, maxTokens int) (string, error) {
			llmCallCount++
			
			if config.llmError != nil {
				return "", config.llmError
			}
			
			if config.llmResponseBuilder != nil {
				return config.llmResponseBuilder(llmCallCount, prompt), nil
			}
			
			if config.defaultLLMResponse != "" {
				return config.defaultLLMResponse, nil
			}
			
			return `{"claims": [], "overall_is_accurate": true, "summary": "Valid"}`, nil
		},
	}
}

// detectChunkUsage analyzes LLM prompt to determine if chunking is being used.
// 
// This function examines the prompt structure to detect whether content has been
// split into chunks for processing. It's crucial for verifying that the chunking
// logic is activated when content exceeds the threshold.
//
// Parameters:
//   - prompt: The full LLM prompt containing the content to validate
//   - callCount: The number of times the LLM has been called (>1 indicates chunking)
//   - originalContentLength: The length of the original content before chunking
//
// Returns true if chunking is detected based on:
//   1. Multiple LLM calls (callCount > 1) - definitive indicator of chunking
//   2. Content length in prompt < original length - indicates a chunk, not full content
//
// The function looks for the specific prompt structure markers:
//   - "USER CONTENT TO CHECK:" marks the start of content
//   - "\n\nOFFICIAL MCP SPECIFICATION" marks the end of content
func detectChunkUsage(prompt string, callCount int, originalContentLength int) bool {
	if !strings.Contains(prompt, "USER CONTENT TO CHECK:") {
		return false
	}
	
	// Extract the content section from the prompt
	startIdx := strings.Index(prompt, "USER CONTENT TO CHECK:")
	if startIdx == -1 {
		return false
	}
	
	endIdx := strings.Index(prompt[startIdx:], "\n\nOFFICIAL MCP SPECIFICATION")
	if endIdx == -1 {
		return false
	}
	
	contentSection := prompt[startIdx : startIdx+endIdx]
	contentLength := len(contentSection)
	
	// Chunking is detected if:
	// 1. Multiple LLM calls (callCount > 1), OR
	// 2. Content in prompt is smaller than original (indicating a chunk)
	return callCount > 1 || contentLength < originalContentLength
}

// generateRepeatedContent creates content by repeating a base string
// repeatCount: number of times to repeat the base content
// Example: generateRepeatedContent("test ", 3) returns "test test test "
func generateRepeatedContent(baseContent string, repeatCount int) string {
	return strings.Repeat(baseContent, repeatCount)
}

// generateStructuredTestContent creates realistic MCP documentation content for chunk testing
// Returns content that exercises different chunking scenarios (paragraphs, lists, etc.)
func generateStructuredTestContent() string {
	return `MCP provides a standardized protocol for communication between clients and servers.
The protocol supports both tools and resources to enable flexible interactions.

MCP servers can expose tools that clients can invoke to perform actions. Each tool has a schema 
that defines its input parameters and expected output format. Tools are invoked using the tools/call 
method and results are returned asynchronously.

Resources in MCP are exposed by servers and can be accessed by clients. Resources have URIs that 
uniquely identify them within the server's namespace. Clients can read resources using the 
resources/read method.

The initialization phase establishes the connection between client and server. During initialization, 
both parties exchange their capabilities and negotiate the protocol version. This ensures compatibility 
and allows for feature discovery.

Security is a critical aspect of MCP. Servers should implement authentication and authorization 
mechanisms to protect sensitive resources. The protocol supports various security patterns including 
token-based authentication and capability-based access control.

MCP uses JSON-RPC for message formatting and transport. This provides a simple, standardized way 
to encode requests and responses. The protocol is transport-agnostic and can work over various 
communication channels.`
}

// Mock behavior helpers for common test scenarios

// successfulLLMMock returns a mock LLM function that validates all chunks as successful
func successfulLLMMock(t *testing.T) func(context.Context, string, string, float64, int) (string, error) {
	return func(ctx context.Context, model string, prompt string, temperature float64, maxTokens int) (string, error) {
		// Verify prompt structure
		assert.Contains(t, prompt, "USER CONTENT TO CHECK:", "Expected chunk content in prompt structure")
		return buildValidationResponse(true, []string{"Content validated successfully"}, []string{}), nil
	}
}

// failingLLMMock returns a mock LLM function that fails validation with specified issues
func failingLLMMock(t *testing.T, issues []string, missingPractices []string) func(context.Context, string, string, float64, int) (string, error) {
	return func(ctx context.Context, model string, prompt string, temperature float64, maxTokens int) (string, error) {
		assert.Contains(t, prompt, "USER CONTENT TO CHECK:", "Expected chunk content in prompt structure")
		
		// Build response with failures
		claims := []string{"Invalid claim detected"}
		response := fmt.Sprintf(`{
			"claims": [{
				"claim": "%s",
				"is_accurate": false,
				"explanation": "This claim is inaccurate"
			}],
			"missing_best_practices": %s,
			"advisory_language_issues": [],
			"overall_is_accurate": false,
			"summary": "Validation failed"
		}`, claims[0], formatStringArray(missingPractices))
		
		return response, nil
	}
}

// chunkTrackingLLMMock returns a mock that tracks chunk processing
func chunkTrackingLLMMock(t *testing.T, onChunk func(int, string)) func(context.Context, string, string, float64, int) (string, error) {
	callCount := 0
	return func(ctx context.Context, model string, prompt string, temperature float64, maxTokens int) (string, error) {
		callCount++
		if onChunk != nil {
			onChunk(callCount, prompt)
		}
		return buildValidationResponse(true, []string{fmt.Sprintf("Chunk %d processed", callCount)}, []string{}), nil
	}
}

// formatStringArray formats a string slice as a JSON array
func formatStringArray(items []string) string {
	if len(items) == 0 {
		return "[]"
	}
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = fmt.Sprintf(`"%s"`, item)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

// buildValidationResponse creates a structured LLM response for testing
func buildValidationResponse(isAccurate bool, claims []string, issues []string) string {
	claimsJSON := ""
	for i, claim := range claims {
		if i > 0 {
			claimsJSON += ","
		}
		claimsJSON += fmt.Sprintf(`{
			"claim": "%s",
			"is_accurate": %v,
			"explanation": "Test explanation"
		}`, claim, isAccurate)
	}
	
	issuesJSON := ""
	for i, issue := range issues {
		if i > 0 {
			issuesJSON += ","
		}
		issuesJSON += fmt.Sprintf(`"%s"`, issue)
	}
	
	return fmt.Sprintf(`{
		"claims": [%s],
		"missing_best_practices": [%s],
		"advisory_language_issues": [],
		"overall_is_accurate": %v,
		"summary": "Test validation result"
	}`, claimsJSON, issuesJSON, isAccurate)
}

// Deduplication helper functions

// hasDuplicates checks if a string slice contains duplicate values
func hasDuplicates(items []string) bool {
	seen := make(map[string]bool)
	for _, item := range items {
		if seen[item] {
			return true
		}
		seen[item] = true
	}
	return false
}

// findDuplicates returns all duplicate values in a string slice
func findDuplicates(items []string) []string {
	seen := make(map[string]int)
	var duplicates []string
	
	for _, item := range items {
		seen[item]++
		if seen[item] == 2 { // Only add on first duplicate occurrence
			duplicates = append(duplicates, item)
		}
	}
	
	return duplicates
}