package list

import (
	"context"
	"fmt"

	"github.com/carlisia/mcp-factcheck/internal/tools"
)

// ListFunc is a function that returns available MCP spec versions
type ListFunc func() ([]string, error)

// ListVersions returns all available MCP specification versions.
//
// Context handling:
//   - This is typically a fast, local operation (reading from vector DB)
//   - Cancellation usually indicates:
//   - Client disconnection during server startup
//   - Server shutdown in progress
//   - The operation is lightweight, so cancellation is mainly
//     for clean shutdown rather than performance
//
// The listFunc is expected to be fast (< 100ms) as it only
// reads metadata from the local vector database.
func ListVersions(ctx context.Context, listFunc ListFunc) ([]string, error) {
	// Check context - mainly for clean shutdown scenarios
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("list operation cancelled: %w", err)
	}

	versions, err := listFunc()
	if err != nil {
		return nil, fmt.Errorf("failed to list spec versions: %w", err)
	}

	return versions, nil
}

// FormatResults formats the list of versions for display
func FormatResults(versions []string) string {
	formatter := tools.NewResultFormatter().
		WithText("Available MCP specification versions:")

	// Format versions as a bullet list
	for _, version := range versions {
		formatter.WithText(fmt.Sprintf("- %s", version))
	}

	formatter.WithText("You can use these versions with other tools like check_mcp_claim, check_mcp_quick_fact, and search_spec.")

	return formatter.BuildSection()
}
