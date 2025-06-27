package embedding

// EmbeddedChunk represents a chunk of text with its embedding
type EmbeddedChunk struct {
	ID        string                 `json:"id"`
	Version   string                 `json:"version"`
	FilePath  string                 `json:"file_path,omitempty"`
	Section   string                 `json:"section,omitempty"`
	Content   string                 `json:"content"`
	Embedding []float64              `json:"embedding"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// SpecEmbedding represents all embeddings for a specific MCP spec version
type SpecEmbedding struct {
	Version string          `json:"version"`
	Chunks  []EmbeddedChunk `json:"chunks"`
	Count   int             `json:"count"`
}

// SearchResult represents a similarity search result
type SearchResult struct {
	Chunk      EmbeddedChunk `json:"chunk"`
	Similarity float64       `json:"similarity"`
	Rank       int           `json:"rank"`
}