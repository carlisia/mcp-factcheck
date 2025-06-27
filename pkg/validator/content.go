package validator

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/carlisia/mcp-factcheck/embedding"
	mcpembedding "github.com/carlisia/mcp-factcheck/internal/embedding"
	"github.com/carlisia/mcp-factcheck/internal/specs"
	"github.com/mark3labs/mcp-go/mcp"
)

const ValidateContentToolName = "validate_content"

type ValidateContentArgs struct {
	Content     string `json:"content"`
	SpecVersion string `json:"spec_version,omitempty"`
}

func GetValidateContentTool() mcp.Tool {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{
				"type":        "string",
				"description": "Content to validate against MCP specification",
			},
			"specVersion": map[string]any{
				"type":        "string",
				"description": "MCP specification version to validate against",
				"enum":        specs.ValidSpecVersions,
				"default":     specs.DefaultSpecVersion,
			},
		},
		"required": []string{"content"},
	}
	schemaBytes, _ := json.Marshal(schema)
	return mcp.NewToolWithRawSchema(ValidateContentToolName, "Validate content against MCP specification and provide corrected version if inaccurate. Uses the most current spec version by default. On first use, inform the user that other versions (2025-03-26, 2024-11-05, draft) are available by specifying specVersion parameter.", schemaBytes)
}

func HandleValidateContent(vectorDB *mcpembedding.VectorDB, generator *embedding.Generator, args any) ([]mcp.Content, error) {
	params, ok := args.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("arguments must be a map")
	}
	content, ok := params["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content must be a string")
	}

	specVersion, ok := params["specVersion"].(string)
	if !ok {
		specVersion = specs.DefaultSpecVersion
	}

	if !specs.IsValidSpecVersion(specVersion) {
		return nil, fmt.Errorf("invalid spec version: %s", specVersion)
	}

	// Generate embedding for content
	contentEmbedding, err := generator.GenerateEmbedding(content)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content embedding: %w", err)
	}

	// Search for relevant spec sections
	results, err := vectorDB.Search(specVersion, contentEmbedding, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to search specifications: %w", err)
	}

	// Analyze validation results
	validationResult := analyzeContentValidation(content, results, specVersion)
	matches := summarizeContentMatches(results, 3)
	
	// Create optimized response
	response := FormatValidationResult(validationResult, matches)
	
	return []mcp.Content{mcp.NewTextContent(response)}, nil
}

// analyzeContentValidation determines if content is valid and provides insights
func analyzeContentValidation(content string, results []embedding.SearchResult, specVersion string) ValidationResult {
	if len(results) == 0 {
		return ValidationResult{
			IsValid:     false,
			Confidence:  0.1,
			Issues:      []string{"No relevant MCP specification content found"},
			SpecVersion: specVersion,
		}
	}

	// Calculate average similarity
	var totalSimilarity float64
	for _, result := range results {
		totalSimilarity += result.Similarity
	}
	avgSimilarity := totalSimilarity / float64(len(results))

	// Determine validation based on similarity thresholds
	isValid := avgSimilarity > 0.7
	confidence := avgSimilarity

	var issues []string
	var suggestions []string

	if !isValid {
		issues = append(issues, "Content may not align with MCP specification")
		if avgSimilarity < 0.5 {
			issues = append(issues, "Low similarity to MCP patterns detected")
		}
		suggestions = append(suggestions, "Review content against MCP specification")
		suggestions = append(suggestions, "Consider using standard MCP terminology and patterns")
	}

	return ValidationResult{
		IsValid:     isValid,
		Confidence:  confidence,
		Issues:      issues,
		Suggestions: suggestions,
		SpecVersion: specVersion,
	}
}

// summarizeContentMatches creates concise summaries from search results
func summarizeContentMatches(results []embedding.SearchResult, maxMatches int) []ValidationMatch {
	if maxMatches > len(results) {
		maxMatches = len(results)
	}

	var matches []ValidationMatch
	for i := 0; i < maxMatches; i++ {
		result := results[i]
		
		// Extract topic from content (first meaningful line)
		lines := strings.Split(result.Chunk.Content, "\n")
		topic := "MCP Specification"
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if len(line) > 0 && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "-") {
				if len(line) > 50 {
					topic = line[:50] + "..."
				} else {
					topic = line
				}
				break
			}
		}

		// Create brief summary
		summary := result.Chunk.Content
		if len(summary) > 200 {
			summary = summary[:200] + "..."
		}

		matches = append(matches, ValidationMatch{
			Topic:     topic,
			Relevance: result.Similarity,
			Summary:   summary,
		})
	}
	return matches
}