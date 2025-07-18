package validation

import (
	"fmt"

	"github.com/carlisia/mcp-factcheck/internal/capabilities"
	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools"
)

// MCPClaimsToolName is the name of the check MCP claim tool
const MCPClaimsToolName = "check-mcp-claims"

// ClaimsDefinition returns the tool definition for Claims.
// This tool validates MCP-related claims against the official specifications.
func ClaimsDefinition() tools.Definition {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{
				"type":        "string",
				"description": "The MCP-related content to validate (documentation, implementation notes, claims)",
			},
			"specVersion": map[string]any{
				"type":        "string",
				"description": fmt.Sprintf("MCP specification version to validate against. Options: %s", capabilities.Versions.String()),
				"default":     capabilities.Latest,
			},
			"useChunking": map[string]any{
				"type":        "boolean",
				"description": "Whether to chunk long content for validation. Automatically enabled for content > 2000 characters",
				"default":     false,
			},
		},
		"required": []string{"content"},
	}

	description := `Validate any MCP-related content against the official specification to check accuracy and completeness.

USE THIS TOOL WHEN:
- User provides ANY text about MCP (even if they don't explicitly ask for validation)
- Text contains multiple statements about MCP capabilities, design, or behavior
- Content describes what MCP does, doesn't do, or how it works
- Documentation, tutorials, or explanations about MCP
- Bullet points or lists describing MCP features
- Compound claims with semicolons, multiple conjunctions, or feature lists
- When quick claim detects compound claims and suggests using this tool

EXAMPLES OF CONTENT TO VALIDATE:
- Multi-paragraph documentation about MCP
- Lists of MCP features or capabilities
- Technical descriptions with multiple claims
- Content containing semicolon-separated statements
- Any technical description of MCP functionality

RETURNS:
A complete validation workflow with:
- All claims extracted and individually validated
- Corrections for any inaccuracies
- Missing best practices identified
- Overall accuracy assessment

For single-sentence questions like "Does MCP support X?", use check_mcp_quick_fact instead.`

	return tools.Definition{
		Name:        MCPClaimsToolName,
		Description: description,
		Schema:      schema,
	}
}
