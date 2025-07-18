package validation

import (
	"fmt"

	"github.com/carlisia/mcp-factcheck/internal/capabilities"
	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools"
)

// MCPQuickClaimToolName is the name of the check MCP quick fact tool
const MCPQuickClaimToolName = "check-mcp-quick-claim"

// QuickClaimDefinition returns the tool definition for Quick Claim.
// This tool is optimized for validating single sentences or quick questions about MCP,
// returning concise results with a ✓/✗ verdict.
func QuickClaimDefinition() tools.Definition {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"claim": map[string]any{
				"type":        "string",
				"description": "A single, specific claim about MCP to fact-check (e.g., 'MCP enforces rate limits', 'MCP uses JSON-RPC')",
			},
			"specVersion": map[string]any{
				"type":        "string",
				"description": fmt.Sprintf("MCP specification version to check against. Options: %s", capabilities.Versions.String()),
				"default":     capabilities.Latest,
			},
		},
		"required": []string{"claim"},
	}

	description := `Quick fact-checking for a single MCP claim or question.

Perfect for:
- Quick yes/no questions: "Does MCP enforce rate limits?"
- Single fact verification: "MCP uses JSON-RPC"
- Clarifying specific requirements: "Servers must implement all tools"
- Single-sentence claims like "MCP enforces strict typing"
- Yes/no questions like "Does MCP support bidirectional communication?"

NOT suitable for:
- Claims with semicolons separating multiple statements
- Claims with multiple "and" conjunctions
- Comma-separated lists of features/capabilities
- Complex multi-part claims

Returns a clear ✓ ACCURATE or ✗ INACCURATE verdict with explanation:
- ✓/✗ Whether the fact is accurate
- What the spec actually says (with quotes)
- Brief explanation of the distinction

This tool is optimized for single sentences. For comprehensive content validation with multiple claims, use check_mcp_claim instead. If a compound claim is detected, the tool will suggest using check_mcp_claim for proper analysis.`
	return tools.Definition{
		Name:        MCPQuickClaimToolName,
		Description: description,
		Schema:      schema,
	}
}
