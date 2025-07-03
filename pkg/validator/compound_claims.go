package validator

import (
	"fmt"
	"strings"

	"github.com/carlisia/mcp-factcheck/embedding"
	mcpembedding "github.com/carlisia/mcp-factcheck/internal/embedding"
	"github.com/carlisia/mcp-factcheck/pkg/logger"
	"go.uber.org/zap"
)

// CompoundClaim represents a claim that has been decomposed into subclaims
type CompoundClaim struct {
	OriginalClaim string
	SubClaims     []SubClaim
	IsCompound    bool
}

// SubClaim represents a single part of a compound claim
type SubClaim struct {
	Text           string
	SearchQueries  []string
	SearchResults  []embedding.SearchResult
	HasEvidence    bool
	EvidenceQuotes []string
}

// DecomposeCompoundClaim splits a claim into subclaims if it contains conjunctions
func DecomposeCompoundClaim(claim string) CompoundClaim {
	result := CompoundClaim{
		OriginalClaim: claim,
		SubClaims:     []SubClaim{},
		IsCompound:    false,
	}

	normalized := strings.ToLower(claim)

	// Check for compound indicators
	if strings.Contains(normalized, " and ") {
		result.IsCompound = true
		parts := strings.Split(claim, " and ")

		// Extract the subject/verb pattern from the beginning
		subjectVerb := extractSubjectVerb(claim)

		for i, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			// For parts after the first, prepend the subject/verb if missing
			if i > 0 && subjectVerb != "" && !hasSubjectVerb(part) {
				part = subjectVerb + " " + part
			}

			subClaim := SubClaim{
				Text:          part,
				SearchQueries: expandClaimForSearch(part),
			}
			result.SubClaims = append(result.SubClaims, subClaim)
		}
	} else {
		// Not a compound claim, treat as single subclaim
		result.SubClaims = []SubClaim{{
			Text:          claim,
			SearchQueries: expandClaimForSearch(claim),
		}}
	}

	return result
}

// extractSubjectVerb attempts to extract the subject and verb from a claim
func extractSubjectVerb(claim string) string {
	// Simple heuristic: take words before the first object/complement
	words := strings.Fields(claim)
	if len(words) < 2 {
		return ""
	}

	// Look for common patterns like "MCP provides", "Servers implement", etc.
	for i := 1; i < len(words) && i < 4; i++ {
		candidate := strings.Join(words[:i+1], " ")
		if looksLikeSubjectVerb(candidate) {
			return candidate
		}
	}

	return ""
}

// hasSubjectVerb checks if a text fragment already has a subject and verb
func hasSubjectVerb(text string) bool {
	words := strings.Fields(strings.ToLower(text))
	if len(words) < 2 {
		return false
	}

	// Check for common subjects
	subjects := []string{"mcp", "servers", "clients", "implementations", "the protocol"}
	for _, subject := range subjects {
		if strings.Contains(strings.ToLower(text), subject) {
			return true
		}
	}

	return false
}

// looksLikeSubjectVerb checks if a phrase looks like subject + verb
func looksLikeSubjectVerb(text string) bool {
	lower := strings.ToLower(text)

	// Common subject-verb patterns in MCP context
	patterns := []string{
		"mcp provides",
		"mcp supports",
		"mcp enables",
		"mcp implements",
		"mcp recommends",
		"servers implement",
		"servers should",
		"clients can",
		"implementations should",
	}

	for _, pattern := range patterns {
		if strings.HasPrefix(lower, pattern) {
			return true
		}
	}

	return false
}

// SearchEvidenceForSubClaims performs independent searches for each subclaim
func SearchEvidenceForSubClaims(
	compound *CompoundClaim,
	vectorDB *mcpembedding.VectorDB,
	generator *embedding.Generator,
	specVersion string,
	topK int,
) error {
	log := logger.Get()

	for i := range compound.SubClaims {
		subClaim := &compound.SubClaims[i]

		log.Debug("Searching evidence for subclaim",
			zap.String("subclaim", subClaim.Text),
			zap.Int("query_count", len(subClaim.SearchQueries)))

		// Collect all results from all queries
		var allResults []embedding.SearchResult
		seenChunks := make(map[string]bool)

		for _, query := range subClaim.SearchQueries {
			// Generate embedding for the query
			queryEmbedding, err := generator.GenerateEmbedding(query)
			if err != nil {
				log.Warn("Failed to generate embedding for query",
					zap.String("query", query),
					zap.Error(err))
				continue
			}

			// Search with this query
			results, err := vectorDB.Search(specVersion, queryEmbedding, topK)
			if err != nil {
				log.Warn("Failed to search for query",
					zap.String("query", query),
					zap.Error(err))
				continue
			}

			// Deduplicate and collect results
			for _, result := range results {
				if !seenChunks[result.Chunk.ID] {
					seenChunks[result.Chunk.ID] = true
					allResults = append(allResults, result)
				}
			}
		}

		// Store results and extract evidence
		subClaim.SearchResults = allResults
		subClaim.HasEvidence = len(allResults) > 0 && allResults[0].Similarity > 0.7

		// Extract relevant quotes from top results
		for _, result := range allResults {
			if result.Similarity > 0.7 {
				quote := extractRelevantQuote(result.Chunk.Content, subClaim.Text)
				if quote != "" {
					subClaim.EvidenceQuotes = append(subClaim.EvidenceQuotes, quote)
				}
			}
		}

		log.Debug("Subclaim evidence search complete",
			zap.String("subclaim", subClaim.Text),
			zap.Bool("has_evidence", subClaim.HasEvidence),
			zap.Int("quote_count", len(subClaim.EvidenceQuotes)))
	}

	return nil
}

// extractRelevantQuote extracts the most relevant sentence from a chunk
func extractRelevantQuote(chunkContent, claimText string) string {
	// Simple implementation: find sentences containing key terms
	sentences := strings.Split(chunkContent, ". ")
	claimLower := strings.ToLower(claimText)

	// Extract key terms from the claim
	keyTerms := extractKeyTerms(claimLower)

	bestSentence := ""
	bestScore := 0

	for _, sentence := range sentences {
		sentenceLower := strings.ToLower(sentence)
		score := 0

		// Count how many key terms appear in this sentence
		for _, term := range keyTerms {
			if strings.Contains(sentenceLower, term) {
				score++
			}
		}

		if score > bestScore {
			bestScore = score
			bestSentence = strings.TrimSpace(sentence)
		}
	}

	return bestSentence
}

// extractKeyTerms extracts important terms from a claim
func extractKeyTerms(claim string) []string {
	// Remove common words and extract key terms
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"is": true, "are": true, "was": true, "were": true,
		"has": true, "have": true, "had": true, "can": true,
		"should": true, "must": true, "may": true, "will": true,
	}

	words := strings.Fields(claim)
	var terms []string

	for _, word := range words {
		word = strings.Trim(word, ".,!?;:")
		if len(word) > 2 && !stopWords[word] {
			terms = append(terms, word)
		}
	}

	return terms
}

// FormatCompoundClaimEvidence formats the evidence for a compound claim
func FormatCompoundClaimEvidence(compound CompoundClaim) string {
	if !compound.IsCompound {
		return ""
	}

	var output []string
	output = append(output, fmt.Sprintf("🦊 **Compound Claim:** %s", compound.OriginalClaim))

	allSubClaimsHaveEvidence := true

	for i, subClaim := range compound.SubClaims {
		output = append(output, fmt.Sprintf("\n📝 **Subclaim %d:** %s", i+1, subClaim.Text))

		if subClaim.HasEvidence {
			output = append(output, "   ✅ Evidence found:")
			for j, quote := range subClaim.EvidenceQuotes {
				if j < 2 { // Limit to 2 quotes per subclaim
					output = append(output, fmt.Sprintf("   - \"%s\"", quote))
				}
			}
		} else {
			output = append(output, "   ❌ No clear evidence found in spec")
			allSubClaimsHaveEvidence = false
		}
	}

	// Overall conclusion
	output = append(output, "\n🌟 **Conclusion:** ")
	if allSubClaimsHaveEvidence {
		output = append(output, "Compound claim is supported by evidence for all parts.")
	} else {
		output = append(output, "Compound claim is only partially supported. Some subclaims lack evidence.")
	}

	return strings.Join(output, "\n")
}
