package core

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
