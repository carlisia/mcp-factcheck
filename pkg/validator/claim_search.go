package validator

import (
	"context"
	"strings"

	"github.com/carlisia/mcp-factcheck/embedding"
	mcpembedding "github.com/carlisia/mcp-factcheck/internal/embedding"
	"github.com/carlisia/mcp-factcheck/pkg/logger"
	"go.uber.org/zap"
)

// extractIndividualClaims splits content into individual claims
func extractIndividualClaims(content string) []string {
	var claims []string

	// First, check for bullet points
	lines := strings.Split(content, "\n")
	var nonBulletContent []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if this is a bullet point
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") || strings.HasPrefix(line, "•") {
			// Extract the bullet content
			bulletContent := strings.TrimSpace(strings.TrimPrefix(line, "-"))
			bulletContent = strings.TrimSpace(strings.TrimPrefix(bulletContent, "*"))
			bulletContent = strings.TrimSpace(strings.TrimPrefix(bulletContent, "•"))

			// Process the bullet content for lists
			processClaim(bulletContent, &claims)
		} else {
			// Not a bullet, save for later processing
			nonBulletContent = append(nonBulletContent, line)
		}
	}

	// Process non-bullet content
	if len(nonBulletContent) > 0 {
		remainingContent := strings.Join(nonBulletContent, " ")

		// Split by semicolons to handle compound sentences
		sentences := strings.Split(remainingContent, ";")

		for _, sentence := range sentences {
			sentence = strings.TrimSpace(sentence)
			if sentence == "" {
				continue
			}

			processClaim(sentence, &claims)
		}
	}

	// If no claims were extracted, use the whole content
	if len(claims) == 0 && content != "" {
		claims = []string{content}
	}

	return claims
}

// processClaim handles a single claim, potentially splitting it if it contains lists
func processClaim(sentence string, claims *[]string) {
	sentence = strings.TrimSpace(sentence)
	if sentence == "" {
		return
	}

	// First check if sentence contains semicolon and split there
	if strings.Contains(sentence, ";") {
		parts := strings.Split(sentence, ";")
		for _, part := range parts {
			processClaim(part, claims)
		}
		return
	}

	// Check if this sentence contains a verb followed by a list
	lowerSentence := strings.ToLower(sentence)

	// Look for patterns like "enforces X, Y, and Z" or "supports A, B, C"
	verbPatterns := []string{"enforces", "supports", "implements", "provides", "enables", "executes", "exposes", "shares", "validates", "handles"}

	var foundVerb string
	var verbEnd int
	for _, verb := range verbPatterns {
		if idx := strings.Index(lowerSentence, verb); idx != -1 {
			foundVerb = verb
			verbEnd = idx + len(verb)
			break
		}
	}

	if foundVerb != "" && verbEnd < len(sentence) {
		// Extract the subject part (before the verb)
		subjectPart := strings.TrimSpace(sentence[:strings.Index(lowerSentence, foundVerb)])
		if subjectPart == "" {
			subjectPart = "MCP"
		}

		// Extract the object part (after the verb)
		objectPart := strings.TrimSpace(sentence[verbEnd:])

		// Split the object part by commas to handle lists
		// Handle "X, Y, and Z" patterns
		objectPart = strings.ReplaceAll(objectPart, ", and ", ", ")
		objectPart = strings.ReplaceAll(objectPart, " and ", ", ")

		objects := strings.Split(objectPart, ",")

		for _, obj := range objects {
			obj = strings.TrimSpace(obj)
			if obj != "" {
				claim := subjectPart + " " + foundVerb + " " + obj
				*claims = append(*claims, claim)
			}
		}
	} else {
		// No list pattern found, treat as a single claim
		*claims = append(*claims, sentence)
	}
}

// performClaimBasedSearch searches for each claim individually and aggregates results
func performClaimBasedSearch(
	ctx context.Context,
	vectorDB *mcpembedding.VectorDB,
	generator *embedding.Generator,
	claims []string,
	specVersion string,
	topKPerClaim int,
) ([]embedding.SearchResult, error) {
	log := logger.WithRequestID(ctx)

	// Map to track unique results
	resultMap := make(map[string]embedding.SearchResult)

	for _, claim := range claims {
		log.Debug("Searching for individual claim",
			zap.String("claim", claim))

		// Expand short claims with context for better matching
		expandedClaim := expandClaimContext(claim)

		// Generate embedding for the expanded claim
		claimEmbedding, err := generator.GenerateEmbedding(expandedClaim)
		if err != nil {
			log.Warn("Failed to generate claim embedding",
				zap.String("claim", claim),
				zap.String("expanded", expandedClaim),
				zap.Error(err))
			continue
		}

		// Search for this specific claim
		claimResults, err := vectorDB.Search(specVersion, claimEmbedding, topKPerClaim)
		if err != nil {
			log.Warn("Failed to search for claim",
				zap.String("claim", claim),
				zap.Error(err))
			continue
		}

		// Add unique results to our map
		for _, result := range claimResults {
			key := result.Chunk.Content
			if existing, exists := resultMap[key]; !exists || result.Similarity > existing.Similarity {
				resultMap[key] = result
			}
		}
	}

	// Convert map back to slice
	var aggregatedResults []embedding.SearchResult
	for _, result := range resultMap {
		aggregatedResults = append(aggregatedResults, result)
	}

	// Sort by similarity
	// Note: You might want to implement sorting here

	log.Debug("Claim-based search completed",
		zap.Int("total_claims", len(claims)),
		zap.Int("unique_results", len(aggregatedResults)))

	return aggregatedResults, nil
}

// expandClaimContext adds contextual information to short claims for better semantic matching
func expandClaimContext(claim string) string {
	// For very short claims, add generic topic context
	if len(claim) < 50 {
		// Default expansion adds general protocol context
		return "Model Context Protocol (MCP) specification and requirements: " + claim
	}

	// Longer claims likely have enough context
	return claim
}
