package specs

// SpecSource represents a source for MCP specification content
type SpecSource struct {
	Type string `json:"type"` // "local_dir" or "github_repo"
	Path string `json:"path"` // Directory path or repository path
}