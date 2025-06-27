package specs

// MCP GitHub repository constants
const (
	MCPRepoOwner    = "modelcontextprotocol"
	MCPRepoName     = "modelcontextprotocol"
	MCPRepoBranch   = "main"
	MCPSpecBasePath = "docs/specification"
)

// BuildSpecPath creates the repository path for a given spec version
func BuildSpecPath(version string) string {
	return MCPSpecBasePath + "/" + version
}