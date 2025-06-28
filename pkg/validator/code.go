package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/carlisia/mcp-factcheck/embedding"
	mcpembedding "github.com/carlisia/mcp-factcheck/internal/embedding"
	"github.com/carlisia/mcp-factcheck/internal/specs"
	"github.com/mark3labs/mcp-go/mcp"
)

const ValidateCodeToolName = "validate_code"

type ValidateCodeArgs struct {
	Code        string `json:"code"`
	SpecVersion string `json:"spec_version,omitempty"`
	Language    string `json:"language,omitempty"`
}

func GetValidateCodeTool() mcp.Tool {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"code": map[string]any{
				"type":        "string",
				"description": "Code to validate against MCP specification",
			},
			"specVersion": map[string]any{
				"type":        "string",
				"description": "MCP specification version to validate against",
				"enum":        specs.ValidSpecVersions,
				"default":     specs.DefaultSpecVersion,
			},
			"language": map[string]any{
				"type":        "string",
				"description": "Programming language of the code",
				"default":     "go",
			},
		},
		"required": []string{"code"},
	}
	schemaBytes, _ := json.Marshal(schema)
	return mcp.NewToolWithRawSchema(ValidateCodeToolName, "Validate code against MCP specification and protocol requirements. Uses the most current spec version by default. On first use, inform the user that other versions (2025-03-26, 2024-11-05, draft) are available by specifying specVersion parameter.", schemaBytes)
}

func HandleValidateCode(ctx context.Context, vectorDB *mcpembedding.VectorDB, generator *embedding.Generator, args any) ([]mcp.Content, error) {
	params, ok := args.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("arguments must be a map")
	}
	code, ok := params["code"].(string)
	if !ok {
		return nil, fmt.Errorf("code must be a string")
	}

	specVersion, ok := params["specVersion"].(string)
	if !ok {
		specVersion = specs.DefaultSpecVersion
	}

	language, ok := params["language"].(string)
	if !ok {
		language = "go"
	}

	if !specs.IsValidSpecVersion(specVersion) {
		return nil, fmt.Errorf("invalid spec version: %s", specVersion)
	}

	// Analyze code to extract MCP-relevant patterns and concepts
	codeAnalysis := analyzeCodeForMCPPatterns(code, language)
	
	// Generate embedding for the code analysis
	codeEmbedding, err := generator.GenerateEmbedding(codeAnalysis)
	if err != nil {
		return nil, fmt.Errorf("failed to generate code embedding: %w", err)
	}

	// Search for relevant spec sections
	results, err := vectorDB.Search(specVersion, codeEmbedding, 8)
	if err != nil {
		return nil, fmt.Errorf("failed to search specifications: %w", err)
	}

	// Analyze code validation results
	validationResult := analyzeCodeValidation(code, codeAnalysis, results, specVersion)
	matches := summarizeCodeMatches(results, 3)
	
	// Create optimized response
	response := FormatValidationResult(validationResult, matches)
	
	return []mcp.Content{mcp.NewTextContent(response)}, nil
}

// analyzeCodeValidation determines if code follows MCP patterns
func analyzeCodeValidation(code, codeAnalysis string, results []embedding.SearchResult, specVersion string) ValidationResult {
	if len(results) == 0 {
		return ValidationResult{
			IsValid:     false,
			Confidence:  0.1,
			Issues:      []string{"No MCP-related patterns found in code"},
			SpecVersion: specVersion,
		}
	}

	// Calculate similarity score
	var totalSimilarity float64
	for _, result := range results {
		totalSimilarity += result.Similarity
	}
	avgSimilarity := totalSimilarity / float64(len(results))

	// Extract detected patterns from analysis
	var detectedPatterns []string
	if strings.Contains(codeAnalysis, "JSON-RPC") {
		detectedPatterns = append(detectedPatterns, "JSON-RPC")
	}
	if strings.Contains(codeAnalysis, "MCP tools") {
		detectedPatterns = append(detectedPatterns, "MCP tools")
	}
	if strings.Contains(codeAnalysis, "MCP server") {
		detectedPatterns = append(detectedPatterns, "MCP server")
	}

	// Determine validation
	isValid := avgSimilarity > 0.6 && len(detectedPatterns) > 0
	confidence := avgSimilarity * (float64(len(detectedPatterns)) / 3.0) // Boost confidence with pattern detection

	var issues []string
	var suggestions []string

	if !isValid {
		if len(detectedPatterns) == 0 {
			issues = append(issues, "No MCP patterns detected in code")
			suggestions = append(suggestions, "Ensure code implements MCP protocol patterns")
		}
		if avgSimilarity < 0.5 {
			issues = append(issues, "Code structure doesn't match MCP specification patterns")
			suggestions = append(suggestions, "Review MCP specification for proper implementation patterns")
		}
	}

	result := ValidationResult{
		IsValid:     isValid,
		Confidence:  confidence,
		Issues:      issues,
		Suggestions: suggestions,
		SpecVersion: specVersion,
	}

	// Add detected patterns to suggestions if valid
	if isValid && len(detectedPatterns) > 0 {
		result.Suggestions = append(result.Suggestions, fmt.Sprintf("Detected MCP patterns: %s", strings.Join(detectedPatterns, ", ")))
	}

	return result
}

// summarizeCodeMatches creates concise summaries from search results
func summarizeCodeMatches(results []embedding.SearchResult, maxMatches int) []ValidationMatch {
	if maxMatches > len(results) {
		maxMatches = len(results)
	}

	var matches []ValidationMatch
	for i := 0; i < maxMatches; i++ {
		result := results[i]
		
		// Extract topic from content
		lines := strings.Split(result.Chunk.Content, "\n")
		topic := "MCP Implementation"
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if len(line) > 0 && (strings.Contains(line, "server") || strings.Contains(line, "client") || strings.Contains(line, "tool")) {
				if len(line) > 50 {
					topic = line[:50] + "..."
				} else {
					topic = line
				}
				break
			}
		}

		// Create brief summary (much shorter for code)
		summary := result.Chunk.Content
		if len(summary) > 150 {
			summary = summary[:150] + "..."
		}

		matches = append(matches, ValidationMatch{
			Topic:     topic,
			Relevance: result.Similarity,
			Summary:   summary,
		})
	}
	return matches
}

// analyzeCodeForMCPPatterns extracts MCP-relevant information from code
func analyzeCodeForMCPPatterns(code, language string) string {
	var analysis []string
	
	// Convert to lowercase for pattern matching
	lowerCode := strings.ToLower(code)
	
	// Check for common MCP patterns
	patterns := map[string]string{
		"json-rpc":     "JSON-RPC protocol implementation",
		"mcp":          "Model Context Protocol usage",
		"tools":        "MCP tools implementation",
		"resources":    "MCP resources implementation", 
		"server":       "MCP server implementation",
		"client":       "MCP client implementation",
		"stdio":        "Standard I/O transport",
		"initialize":   "MCP initialization process",
		"notification": "MCP notifications",
		"request":      "MCP requests handling",
		"response":     "MCP responses handling",
		"error":        "Error handling patterns",
		"schema":       "Schema validation",
		"params":       "Parameter handling",
		"result":       "Result processing",
	}
	
	analysis = append(analysis, fmt.Sprintf("Language: %s", language))
	
	var foundPatterns []string
	for pattern, desc := range patterns {
		if strings.Contains(lowerCode, pattern) {
			foundPatterns = append(foundPatterns, desc)
		}
	}
	
	if len(foundPatterns) > 0 {
		analysis = append(analysis, "Detected MCP patterns:")
		for _, pattern := range foundPatterns {
			analysis = append(analysis, fmt.Sprintf("- %s", pattern))
		}
	} else {
		analysis = append(analysis, "No obvious MCP patterns detected in the code")
	}
	
	// Add code structure info
	lines := strings.Split(code, "\n")
	analysis = append(analysis, fmt.Sprintf("Code contains %d lines", len(lines)))
	
	return strings.Join(analysis, "\n")
}