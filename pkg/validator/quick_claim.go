package validator

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/carlisia/mcp-factcheck/embedding"
	mcpembedding "github.com/carlisia/mcp-factcheck/internal/embedding"
	"github.com/carlisia/mcp-factcheck/internal/specs"
	"github.com/carlisia/mcp-factcheck/internal/utils"
	"github.com/carlisia/mcp-factcheck/pkg/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
)

const (
	checkMCPQuickFactToolName = "check_mcp_quick_fact"
	quickSearchTopK           = 15 // More results for better coverage
)

// Import aliases for compile-time checks
type (
	mcpVectorDB = mcpembedding.VectorDB
)

// Compile-time interface implementation checks
var (
	_ QuickFactVectorDB           = (*mcpVectorDB)(nil)
	_ QuickFactEmbeddingGenerator = (*embedding.Generator)(nil)
)

// CheckMCPClaimArgs contains arguments for checking a single MCP claim.
// It includes the claim text and optionally the spec version to validate against.
type CheckMCPClaimArgs struct {
	Claim       string `json:"claim"`
	SpecVersion string `json:"spec_version,omitempty"`
}

// GetCheckMCPQuickFactTool returns the MCP tool definition for quick fact checking.
// This tool is optimized for validating single sentences or quick questions about MCP,
// returning concise results with a ✓/✗ verdict.
func GetCheckMCPQuickFactTool() mcp.Tool {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"claim": map[string]any{
				"type":        "string",
				"description": "A single, specific claim about MCP to fact-check (e.g., 'MCP enforces rate limits', 'MCP uses JSON-RPC')",
			},
			"specVersion": map[string]any{
				"type":        "string",
				"description": "MCP specification version to check against",
				"enum":        specs.ValidSpecVersions,
				"default":     specs.DefaultSpecVersion,
			},
		},
		"required": []string{"claim"},
	}
	schemaBytes := utils.MustMarshalSchema(schema, checkMCPQuickFactToolName)

	description := `Quickly fact-check a single sentence about MCP against the official specification.

Perfect for:
- Quick yes/no questions: "Does MCP enforce rate limits?"
- Single fact verification: "MCP uses JSON-RPC"
- Clarifying specific requirements: "Servers must implement all tools"

Returns a concise answer with:
- ✓/✗ Whether the fact is accurate
- What the spec actually says (with quotes)
- Brief explanation of the distinction

This tool is optimized for single sentences. For comprehensive content validation with multiple claims, use check_mcp_claim instead.`

	return mcp.NewToolWithRawSchema(checkMCPQuickFactToolName, description, schemaBytes)
}

// QuickFactVectorDB defines the interface for vector database operations needed by quick fact checking
type QuickFactVectorDB interface {
	Search(version string, queryEmbedding []float64, topK int) ([]embedding.SearchResult, error)
}

// QuickFactEmbeddingGenerator defines the interface for embedding and fact-checking operations
type QuickFactEmbeddingGenerator interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float64, error)
	FactCheckAgainstSpec(ctx context.Context, claim string, specSections []string, compoundEvidence map[string]string) (*embedding.FactCheckResult, error)
}

// HandleCheckMCPQuickFact handles quick fact-checking requests for single MCP claims.
// It uses aggressive search strategies to find relevant spec sections and returns
// a concise verdict on whether the claim is accurate according to the MCP specification.
func HandleCheckMCPQuickFact(ctx context.Context, vectorDB QuickFactVectorDB, generator QuickFactEmbeddingGenerator, args any) ([]mcp.Content, error) {
	log := logger.WithRequestID(ctx)

	params, ok := args.(map[string]any)
	if !ok {
		return nil, errArgumentsNotMap
	}

	claim, ok := params["claim"].(string)
	if !ok || claim == "" {
		return nil, fmt.Errorf("claim must be a non-empty string")
	}

	specVersion, ok := params["specVersion"].(string)
	if !ok {
		specVersion = specs.DefaultSpecVersion
	}

	if !specs.IsValidSpecVersion(specVersion) {
		return nil, fmt.Errorf("invalid spec version: %s", specVersion)
	}

	log.Info("Checking MCP claim",
		zap.String("claim", claim),
		zap.String("spec_version", specVersion))

	// Perform aggressive search for the claim
	results, err := performAggressiveClaimSearch(ctx, vectorDB, generator, claim, specVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to search for claim: %w", err)
	}

	// Extract spec sections
	var specSections []string
	for _, result := range results {
		specSections = append(specSections, result.Chunk.Content)
	}

	// Quick fact-check (no compound evidence for single claims)
	factCheckResult, err := generator.FactCheckAgainstSpec(ctx, claim, specSections, nil)
	if err != nil {
		log.Error("Failed to fact-check claim", zap.Error(err))
		// Fallback response
		return []mcp.Content{mcp.NewTextContent(fmt.Sprintf(
			"Unable to verify claim: %s\n\nPlease try rephrasing or use validate_content for comprehensive analysis.",
			claim,
		))}, nil
	}

	// Format concise response
	response := formatQuickClaimResponse(claim, factCheckResult, specVersion)
	return []mcp.Content{mcp.NewTextContent(response)}, nil
}

func performAggressiveClaimSearch(ctx context.Context, vectorDB QuickFactVectorDB, generator QuickFactEmbeddingGenerator, claim string, specVersion string) ([]embedding.SearchResult, error) {
	log := logger.WithRequestID(ctx)

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
				"implementations should must "+topic,
				"clients servers requirements "+topic,
				"protocol specification "+topic,
				"security best practices "+topic,
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
				"restrictions limitations "+topic,
				"security considerations "+topic,
				"must not should not "+topic,
			)
		}
	}

	// If claim is about capabilities (supports, provides, handles)
	if strings.Contains(lowerClaim, "support") ||
		strings.Contains(lowerClaim, "provide") ||
		strings.Contains(lowerClaim, "handle") ||
		strings.Contains(lowerClaim, "implement") {
		// Try both protocol-level and implementation-level searches
		allQueries = append(allQueries,
			strings.Replace(claim, "MCP ", "protocol ", 1),
			strings.Replace(claim, "MCP ", "implementations ", 1),
			strings.Replace(claim, "MCP ", "specification ", 1),
		)
	}

	// Collect all results
	resultMap := make(map[string]embedding.SearchResult)

	for _, query := range allQueries {
		log.Debug("Aggressive claim search",
			zap.String("original_claim", claim),
			zap.String("search_query", query))

		queryEmbedding, err := generator.GenerateEmbedding(ctx, query)
		if err != nil {
			log.Warn("Failed to generate embedding for query",
				zap.String("query", query),
				zap.Error(err))
			continue
		}

		results, err := vectorDB.Search(specVersion, queryEmbedding, 10)
		if err != nil {
			log.Warn("Failed to search for query",
				zap.String("query", query),
				zap.Error(err))
			continue
		}

		for _, result := range results {
			key := result.Chunk.Content
			if existing, exists := resultMap[key]; !exists || result.Similarity > existing.Similarity {
				resultMap[key] = result
			}
		}
	}

	// Convert to slice and sort
	var finalResults []embedding.SearchResult
	for _, result := range resultMap {
		finalResults = append(finalResults, result)
	}

	// Sort by similarity
	sort.Slice(finalResults, func(i, j int) bool {
		return finalResults[i].Similarity > finalResults[j].Similarity
	})

	// Limit results
	if len(finalResults) > quickSearchTopK {
		finalResults = finalResults[:quickSearchTopK]
	}

	log.Debug("Aggressive search completed",
		zap.Int("unique_results", len(finalResults)))

	return finalResults, nil
}

// extractTopicFromClaim extracts the main topic/object from a claim
func extractTopicFromClaim(claim string) string {
	// Remove common prefixes
	topic := strings.TrimPrefix(strings.ToLower(claim), "mcp ")

	// Common verb patterns to remove
	verbs := []string{"enforces ", "never ", "doesn't ", "does not ", "provides ",
		"supports ", "handles ", "validates ", "implements "}

	for _, verb := range verbs {
		if strings.HasPrefix(topic, verb) {
			topic = strings.TrimPrefix(topic, verb)
			break
		}
	}

	// Clean up the topic
	topic = strings.TrimSpace(topic)
	topic = strings.TrimSuffix(topic, ".")

	return topic
}

func formatQuickClaimResponse(claim string, factCheck *embedding.FactCheckResult, specVersion string) string {
	var response strings.Builder

	// Status icon and claim
	if factCheck.IsAccurate {
		response.WriteString("✓ **Accurate**\n\n")
	} else {
		response.WriteString("✗ **Inaccurate**\n\n")
	}

	response.WriteString(fmt.Sprintf("**Claim:** %s\n\n", claim))

	// What the spec actually says
	response.WriteString("**What the spec says:**\n")

	if !factCheck.IsAccurate && len(factCheck.Corrections) > 0 {
		// Show corrections
		for _, correction := range factCheck.Corrections {
			response.WriteString(fmt.Sprintf("- %s\n", correction))
		}
	} else if factCheck.IsAccurate {
		response.WriteString("- The claim accurately reflects the MCP specification.\n")
	} else {
		response.WriteString("- The specification does not support this claim.\n")
	}

	// Add explanation if available
	if len(factCheck.Claims) > 0 && factCheck.Claims[0].Explanation != "" {
		response.WriteString(fmt.Sprintf("\n**Explanation:** %s\n", factCheck.Claims[0].Explanation))
	}

	// Modal verb clarification if needed
	if len(factCheck.AdvisoryLanguageIssues) > 0 {
		response.WriteString("\n**Important distinction:**\n")
		for _, issue := range factCheck.AdvisoryLanguageIssues {
			response.WriteString(fmt.Sprintf("- %s\n", issue))
		}
	}

	response.WriteString(fmt.Sprintf("\n*Checked against MCP spec %s*", specVersion))

	return response.String()
}
