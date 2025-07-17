package contentprep

import (
	"strings"
)

// Compound represents a claim that has been decomposed into subclaims
type Compound struct {
	OriginalClaim string
	SubClaims     []SubClaim
	IsCompound    bool
}

// SubClaim represents a single part of a compound claim
type SubClaim struct {
	Text           string
	SearchQueries  []string
	HasEvidence    bool
	EvidenceQuotes []string
}

// Decompose splits a claim into subclaims if it contains conjunctions
func Decompose(claim string) Compound {
	result := Compound{
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
				SearchQueries: Expand(part),
			}
			result.SubClaims = append(result.SubClaims, subClaim)
		}
	} else {
		// Not a compound claim, treat as single subclaim
		result.SubClaims = []SubClaim{{
			Text:          claim,
			SearchQueries: Expand(claim),
		}}
	}

	return result
}

// ExtractQuote extracts the most relevant sentence from a chunk
func ExtractQuote(chunkContent, claimText string) string {
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

