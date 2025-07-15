package validation

import "github.com/carlisia/mcp-factcheck/internal/tools"

// MCPQuickClaimToolName is the name of the check MCP quick fact tool
const MCPQuickClaimToolName = "check-mcp-quick-fact"

// QuickClaimDefinition returns the tool definition for check_mcp_quick_fact.
// This tool is optimized for validating single sentences or quick questions about MCP,
// returning concise results with a ✓/✗ verdict.
func QuickClaimDefinition() tools.Definition {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"claim": map[string]any{
				"type":        "string",
				"description": "A single sentence or question about MCP to fact-check",
			},
			"specVersion": map[string]any{
				"type":        "string",
				"description": "MCP specification version to check against. Options: draft (latest), 2025-06-18, 2025-03-26, 2024-11-05",
				"default":     tools.Current,
			},
		},
		"required": []string{"claim"},
	}

	description := `Quick fact-checking for single MCP claims or questions.

This tool is optimized for:
- Single-sentence claims like "MCP enforces strict typing"
- Yes/no questions like "Does MCP support bidirectional communication?"
- Quick verification of MCP features and capabilities

Returns a clear ✓ ACCURATE or ✗ INACCURATE verdict with explanation.`

	return tools.Definition{
		Name:        MCPQuickClaimToolName,
		Description: description,
		Schema:      schema,
	}
}
