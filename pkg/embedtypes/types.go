// Package embedtypes defines public types for embedding operations and fact-checking.
// These types are used across the MCP fact-check system for vector search,
// content validation, and claim verification.
package embedtypes

// EmbeddedChunk represents a chunk of text with its embedding vector.
// Each chunk contains a portion of MCP specification content along with
// metadata for identification and semantic search.
type EmbeddedChunk struct {
	ID        string         `json:"id"`
	Version   string         `json:"version"`
	FilePath  string         `json:"file_path,omitempty"`
	Section   string         `json:"section,omitempty"`
	Content   string         `json:"content"`
	Embedding []float64      `json:"embedding"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// SpecEmbedding represents all embeddings for a specific MCP spec version.
// It contains all chunks generated from a particular version of the MCP
// specification, enabling version-specific semantic search.
type SpecEmbedding struct {
	Version string          `json:"version"`
	Chunks  []EmbeddedChunk `json:"chunks"`
	Count   int             `json:"count"`
}

// SearchResult represents a similarity search result from vector search.
// It includes the matched chunk, similarity score (0-1), and rank position
// among all search results.
type SearchResult struct {
	Chunk      EmbeddedChunk `json:"chunk"`
	Similarity float64       `json:"similarity"`
	Rank       int           `json:"rank"`
}

// FactCheckResult contains comprehensive validation results from fact-checking content
// against MCP specifications. It includes claim-by-claim analysis, best practice
// recommendations, and advisory language clarifications.
type FactCheckResult struct {
	IsAccurate             bool     `json:"is_accurate"`
	Inaccuracies           []string `json:"inaccuracies"`
	Corrections            []string `json:"corrections"`
	Explanation            string   `json:"explanation"`
	ParsedClaims           []string `json:"parsed_claims"`            // All claims extracted from content
	MissingBestPractices   []string `json:"missing_best_practices"`   // SHOULD requirements not mentioned
	AdvisoryLanguageIssues []string `json:"advisory_language_issues"` // MAY/CAN confusion
	Claims                 []Claim  `json:"claims"`                   // Detailed claim analysis
	RawResponse            string   `json:"-"`                        // Raw LLM response for debugging
}

// Claim represents a single claim extracted from content with its validation details.
// Each claim is assessed for accuracy against the MCP specification with corrections
// and explanations provided when issues are found.
type Claim struct {
	Claim       string `json:"claim"`
	IsAccurate  bool   `json:"is_accurate"`
	Correction  string `json:"correction,omitempty"`
	Explanation string `json:"explanation,omitempty"`
}