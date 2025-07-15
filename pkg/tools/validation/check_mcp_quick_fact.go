package validation

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/carlisia/mcp-factcheck/internal/storage"
	"github.com/carlisia/mcp-factcheck/pkg/llm"
	"github.com/carlisia/mcp-factcheck/pkg/logger"
	"github.com/carlisia/mcp-factcheck/pkg/tools/spec"
	"github.com/mark3labs/mcp-go/mcp"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

const (
	quickSearchTopK  = 15  // More results for better coverage
	maxSectionLength = 200 // Maximum length for displayed spec sections
)

// HandleCheckMCPQuickFact handles quick fact-checking requests for single MCP claims.
// It uses aggressive search strategies to find relevant spec sections and returns
// a concise verdict on whether the claim is accurate according to the MCP specification.
func HandleCheckMCPQuickFact(ctx context.Context, vectorDB *storage.VectorDB, generator *llm.Client, args any) ([]mcp.Content, error) {
	params, ok := args.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("arguments must be a map")
	}

	claim, ok := params["claim"].(string)
	if !ok || claim == "" {
		return nil, fmt.Errorf("claim must be a non-empty string")
	}

	specVersion, ok := params["specVersion"].(string)
	if !ok {
		specVersion = spec.DefaultSpecVersion
	}

	if !spec.IsValidSpecVersion(specVersion) {
		return nil, fmt.Errorf("invalid spec version: %s", specVersion)
	}

	log := logger.Get()
	log.Debug("Checking MCP claim",
		zap.String("claim", claim),
		zap.String("spec_version", specVersion))

	// Perform aggressive search for the claim
	results, err := performAggressiveClaimSearch(ctx, vectorDB, generator, claim, specVersion)
	if err != nil {
		// Create a fallback result when search fails
		fallbackResult := &FactCheckResult{
			IsAccurate:   false,
			Confidence:   0.1,
			Inaccuracies: []string{"Unable to verify claim due to search error"},
		}
		response := formatQuickFactResult(claim, fallbackResult, specVersion, "")
		return []mcp.Content{mcp.NewTextContent(response)}, nil
	}

	// Extract spec sections
	var specSections []string
	for _, result := range results {
		specSections = append(specSections, result.Content)
	}

	// Quick fact-check (no compound evidence for single claims)
	factCheckResult, err := CheckFacts(ctx, generator, claim, specSections, nil)
	if err != nil || factCheckResult == nil {
		if err != nil {
			log.Warn("Failed to fact-check claim", zap.Error(err))
		} else {
			log.Warn("Fact-check returned nil result")
		}
		// Fallback response
		return []mcp.Content{mcp.NewTextContent(fmt.Sprintf(
			"Unable to verify claim: %s\n\nPlease try rephrasing or use check_mcp_claim for comprehensive analysis.",
			claim,
		))}, nil
	}

	// Get the most relevant section
	relevantSection := ""
	if len(results) > 0 {
		relevantSection = truncateSection(results[0].Content)
	}

	// Format response
	response := formatQuickFactResult(claim, factCheckResult, specVersion, relevantSection)
	return []mcp.Content{mcp.NewTextContent(response)}, nil
}

// performAggressiveClaimSearch performs an aggressive multi-query search to find relevant spec sections.
func performAggressiveClaimSearch(ctx context.Context, vectorDB *storage.VectorDB, generator *llm.Client, claim string, specVersion string) ([]storage.SearchResult, error) {
	log := logger.Get()

	var allQueries []string

	// Always search with the original claim
	allQueries = append(allQueries, claim)

	// Extract key terms from the claim
	lowerClaim := strings.ToLower(claim)

	// If claim is about enforcement/requirements, search for implementation guidance
	if strings.Contains(lowerClaim, "enforce") ||
		strings.Contains(lowerClaim, "require") ||
		strings.Contains(lowerClaim, "must") ||
		strings.Contains(lowerClaim, "guarantee") {
		// Extract what's being enforced/required
		topic := extractTopicFromClaim(claim)
		if topic != "" {
			allQueries = append(allQueries,
				"implementation should must "+topic,
				"client server requirement "+topic,
				"protocol specification "+topic,
				"security best practice "+topic,
				"MUST SHOULD requirement "+topic,
			)
		}
	}

	// If claim is negative (never, doesn't, can't)
	if strings.Contains(lowerClaim, "never") ||
		strings.Contains(lowerClaim, "doesn't") ||
		strings.Contains(lowerClaim, "does not") ||
		strings.Contains(lowerClaim, "cannot") {
		topic := extractTopicFromClaim(claim)
		if topic != "" {
			allQueries = append(allQueries,
				"restriction limitation "+topic,
				"security consideration "+topic,
				"must not should not "+topic,
				"MUST NOT requirement "+topic,
			)
		}
	}

	// Extract specific MCP concepts mentioned
	concepts := extractMCPConcepts(claim)
	for _, concept := range concepts {
		allQueries = append(allQueries,
			fmt.Sprintf("%s specification", concept),
			fmt.Sprintf("%s requirements and behavior", concept),
		)
	}

	// Perform searches with all queries
	resultMap := make(map[string]storage.SearchResult)
	var allResults []storage.SearchResult

	for queryIdx, query := range allQueries {
		// Check context cancellation before each query
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		log.Debug("Searching with query", zap.String("query", query))

		// Generate embedding for query
		embedding, err := generator.CreateEmbedding(ctx, query)
		if err != nil {
			log.Warn("Failed to generate embedding for query", zap.String("query", query), zap.Error(err))
			continue
		}

		results, err := vectorDB.Search(specVersion, embedding, quickSearchTopK)
		if err != nil {
			log.Warn("Search failed for query", zap.String("query", query), zap.Error(err))
			continue
		}

		// Record search metrics on the main span
		if len(results) > 0 {
			logger.SetSpanAttributes(ctx,
				attribute.String(fmt.Sprintf("retrieval.query_%d", queryIdx), query),
				attribute.Float64(fmt.Sprintf("retrieval.query_%d.top_similarity", queryIdx), results[0].Similarity),
				attribute.Int(fmt.Sprintf("retrieval.query_%d.results_count", queryIdx), len(results)),
			)
		}

		// Deduplicate results
		for _, result := range results {
			if existing, exists := resultMap[result.ChunkID]; !exists || result.Similarity > existing.Similarity {
				resultMap[result.ChunkID] = storage.SearchResult{
					Content:    result.Content,
					ChunkID:    result.ChunkID,
					Similarity: result.Similarity,
					Version:    result.Version,
					Rank:       result.Rank,
				}
			}
		}
	}

	// Convert map to slice
	for _, result := range resultMap {
		allResults = append(allResults, result)
	}

	// Sort by similarity
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Similarity > allResults[j].Similarity
	})

	// Take top results
	if len(allResults) > quickSearchTopK {
		allResults = allResults[:quickSearchTopK]
	}

	log.Debug("Aggressive search completed",
		zap.Int("unique_results", len(allResults)),
		zap.Int("queries_used", len(allQueries)))

	// Add final search metrics to span
	logger.SetSpanAttributes(ctx,
		attribute.Int("retrieval.total_queries", len(allQueries)),
		attribute.Int("retrieval.unique_results", len(allResults)),
		attribute.String("retrieval.spec_version", specVersion),
	)

	return allResults, nil
}

// extractTopicFromClaim extracts the main topic from a claim by removing common prefixes.
func extractTopicFromClaim(claim string) string {
	// Simple extraction of the main topic
	// Common prefixes to remove (case-insensitive)
	prefixes := []string{
		"MCP", "The MCP",
		"Does", "Can", "Will", "Should", "Must",
		"Is", "Are", "Was", "Were",
		"enforce", "enforces", "require", "requires",
		"guarantee", "guarantees", "ensure", "ensures",
	}

	topic := claim
	topicLower := strings.ToLower(topic)

	for _, prefix := range prefixes {
		// Check with optional space after prefix
		prefixWithSpace := strings.ToLower(prefix + " ")
		prefixAlone := strings.ToLower(prefix)

		if strings.HasPrefix(topicLower, prefixWithSpace) {
			// Remove prefix with space
			topic = topic[len(prefix)+1:] // +1 for the space
			break
		} else if strings.HasPrefix(topicLower, prefixAlone) && len(topic) == len(prefix) {
			// Exact match of the whole string
			topic = ""
			break
		}
	}

	// Remove trailing punctuation
	topic = strings.TrimRight(topic, ".?!")

	return strings.TrimSpace(topic)
}

// extractMCPConcepts extracts MCP-specific concepts mentioned in the claim.
func extractMCPConcepts(claim string) []string {
	var concepts []string
	lowerClaim := strings.ToLower(claim)

	// MCP-specific concepts to look for
	mcpConcepts := []string{
		"transport", "stdio", "http", "sse",
		"jsonrpc", "json-rpc", "rpc",
		"tool", "tools", "prompt", "prompts",
		"resource", "resources", "sampling",
		"server", "client", "protocol",
		"request", "response", "notification",
		"capability", "capabilities", "negotiation",
		"initialization", "shutdown",
		"error", "errors", "result",
		"content", "text", "image",
		"argument", "parameter", "schema",
		"metadata", "annotation",
		"security", "authentication", "authorization",
		"session", "connection", "lifecycle",
	}

	for _, concept := range mcpConcepts {
		if strings.Contains(lowerClaim, concept) {
			concepts = append(concepts, concept)
		}
	}

	return concepts
}

// truncateSection truncates content to a reasonable length for display.
func truncateSection(content string) string {
	if len(content) <= maxSectionLength {
		return content
	}

	// Try to truncate at a sentence boundary
	truncated := content[:maxSectionLength]
	lastPeriod := strings.LastIndex(truncated, ". ")
	if lastPeriod > maxSectionLength/2 {
		return truncated[:lastPeriod+1]
	}

	return truncated + "..."
}

// formatQuickFactResult formats the quick fact check result into a concise response.
func formatQuickFactResult(claim string, factCheckResult *FactCheckResult, specVersion string, relevantSection string) string {
	verdict := "✗ INACCURATE"
	if factCheckResult.IsAccurate {
		verdict = "✓ ACCURATE"
	}

	response := fmt.Sprintf("%s\n\nClaim: %s\n", verdict, claim)

	// Add explanation
	if !factCheckResult.IsAccurate && len(factCheckResult.Inaccuracies) > 0 {
		response += fmt.Sprintf("\n%s\n", factCheckResult.Inaccuracies[0])
		if factCheckResult.CorrectedVersion != "" {
			response += fmt.Sprintf("\nCorrect statement: %s\n", factCheckResult.CorrectedVersion)
		}
	} else if factCheckResult.IsAccurate {
		// For accurate claims, use explanation from the fact check result
		if factCheckResult.Explanation != "" {
			response += fmt.Sprintf("\n%s\n", factCheckResult.Explanation)
		} else if len(factCheckResult.Claims) > 0 {
			// Use the explanation from the first claim if available
			for _, claim := range factCheckResult.Claims {
				if claim.Explanation != "" {
					response += fmt.Sprintf("\n%s\n", claim.Explanation)
					break
				}
			}
		}
	}

	// Add relevant section if available
	if relevantSection != "" {
		response += fmt.Sprintf("\nRelevant spec section:\n%s\n", relevantSection)
	}

	// Use a default confidence if not set
	confidence := factCheckResult.Confidence
	if confidence == 0 {
		confidence = 0.9 // Default high confidence
	}

	response += fmt.Sprintf("\nConfidence: %.0f%%", confidence*100)
	response += fmt.Sprintf("\nSpec version: %s", specVersion)

	return response
}
