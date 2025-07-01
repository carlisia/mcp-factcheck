package prompts

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/mark3labs/mcp-go/mcp"
)

// PromptTemplate defines a reusable validation prompt
type PromptTemplate struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Arguments   []PromptArgument       `json:"arguments,omitempty"`
	Template    string                 `json:"template"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// PromptArgument defines an argument for a prompt template
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Type        string `json:"type,omitempty"`
	Default     string `json:"default,omitempty"`
}

// GetAvailablePrompts returns all available validation prompt templates
func GetAvailablePrompts() []PromptTemplate {
	return []PromptTemplate{
		{
			Name:        "migrate-mcp-content",
			Description: "Update MCP documentation, tutorials, or content to align with a target specification version",
			Arguments: []PromptArgument{
				{
					Name:        "current_version",
					Description: "Current MCP specification version the content is based on",
					Required:    true,
					Type:        "string",
				},
				{
					Name:        "target_version",
					Description: "Target MCP specification version to update content for",
					Required:    true,
					Type:        "string",
				},
				{
					Name:        "content",
					Description: "The documentation, tutorial, or content that needs updating",
					Required:    true,
					Type:        "string",
				},
				{
					Name:        "content_type",
					Description: "Type of content being updated (documentation, tutorial, blog_post, readme, api_reference, guide)",
					Required:    false,
					Type:        "string",
					Default:     "documentation",
				},
				{
					Name:        "target_audience",
					Description: "Intended audience for the content (developers, users, beginners, advanced, technical_writers)",
					Required:    false,
					Type:        "string",
					Default:     "developers",
				},
				{
					Name:        "update_scope",
					Description: "Scope of updates needed (critical_only, comprehensive, enhancement_focused)",
					Required:    false,
					Type:        "string",
					Default:     "comprehensive",
				},
			},
			Template: `Please help me update this MCP content from {{.current_version}} to {{.target_version}}:

**Content Migration Details:**
- Current spec version: {{.current_version}}
- Target spec version: {{.target_version}}
- Content type: {{.content_type}}
- Target audience: {{.target_audience}}
- Update scope: {{.update_scope}}

**Original Content:**
{{.content}}

---

**Step 1: Analyze Specification Changes**
Use the search_spec tool to identify changes between {{.current_version}} and {{.target_version}} that affect:
- Terminology and concepts mentioned in the content
- Features, APIs, or protocols described
- Examples and code snippets referenced
- Best practices and recommendations

**Step 2: Validate Current Content Accuracy**
Use the validate_content tool to assess the current content against {{.current_version}} specification:
- Identify any existing inaccuracies
- Check for missing context or incomplete information
- Verify example accuracy and completeness

**Step 3: Identify Content Update Requirements**
Based on the specification changes, determine:
- **Critical updates**: Incorrect information that must be fixed
- **New content**: Features/concepts that should be added
- **Deprecated content**: Information that should be removed or marked as outdated
- **Enhanced explanations**: Areas that benefit from {{.target_version}} improvements

**Step 4: Create Updated Content Strategy**
Provide a content update plan that addresses:
{{if eq .update_scope "critical_only"}}
- Only critical inaccuracies and breaking changes
- Minimal disruption to existing content structure
{{else if eq .update_scope "enhancement_focused"}}
- Leverage new features and improvements in {{.target_version}}
- Enhance examples and best practices
- Improve clarity and completeness
{{else}}
- Comprehensive review and updates
- New sections for {{.target_version}} features
- Improved examples and best practices
- Enhanced clarity for {{.target_audience}}
{{end}}

**Step 5: Generate Updated Content**
Create the revised content with:
- Updated terminology consistent with {{.target_version}}
- Corrected examples and code snippets
- New sections for relevant {{.target_version}} features
- Clear indicators of version-specific information
- Maintained tone and style appropriate for {{.target_audience}}

**Step 6: Validate Updated Content**
Use the validate_content tool to verify the updated content against {{.target_version}} specification for:
- Technical accuracy and completeness
- Proper use of {{.target_version}} terminology
- Example correctness and best practices
- Appropriate level of detail for {{.target_audience}}

Please start with Step 1 by analyzing the key changes between {{.current_version}} and {{.target_version}} that are relevant to this {{.content_type}} content.`,
			Metadata: map[string]any{
				"category":    "migration",
				"use_case":    "content_update",
				"workflow":    "multi_step",
				"complexity":  "medium",
				"output_type": "content",
			},
		},
	}
}

// GetPromptByName returns a specific prompt template by name
func GetPromptByName(name string) (*PromptTemplate, error) {
	prompts := GetAvailablePrompts()
	for _, prompt := range prompts {
		if prompt.Name == name {
			return &prompt, nil
		}
	}
	return nil, fmt.Errorf("prompt not found: %s", name)
}

// GetPromptsListResponse returns the MCP prompts/list response
func GetPromptsListResponse() (*mcp.ListPromptsResult, error) {
	templates := GetAvailablePrompts()
	prompts := make([]mcp.Prompt, len(templates))
	
	for i, template := range templates {
		// Convert our arguments to MCP prompt arguments
		var mcpArgs []mcp.PromptArgument
		for _, arg := range template.Arguments {
			mcpArgs = append(mcpArgs, mcp.PromptArgument{
				Name:        arg.Name,
				Description: arg.Description,
				Required:    arg.Required,
			})
		}
		
		prompts[i] = mcp.Prompt{
			Name:        template.Name,
			Description: template.Description,
			Arguments:   mcpArgs,
		}
	}
	
	return &mcp.ListPromptsResult{
		Prompts: prompts,
	}, nil
}

// RenderPrompt renders a prompt template with the given arguments
func RenderPrompt(templateName string, args map[string]any) (*mcp.GetPromptResult, error) {
	promptTemplate, err := GetPromptByName(templateName)
	if err != nil {
		return nil, err
	}
	
	// Create a copy of args with defaults filled in
	renderArgs := make(map[string]any)
	for _, arg := range promptTemplate.Arguments {
		if value, exists := args[arg.Name]; exists {
			renderArgs[arg.Name] = value
		} else if arg.Default != "" {
			renderArgs[arg.Name] = arg.Default
		} else if arg.Required {
			return nil, fmt.Errorf("required argument missing: %s", arg.Name)
		}
	}
	
	// Parse and execute the template
	tmpl, err := template.New(templateName).Parse(promptTemplate.Template)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}
	
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, renderArgs); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}
	
	// Create the prompt result
	result := &mcp.GetPromptResult{
		Description: promptTemplate.Description,
		Messages: []mcp.PromptMessage{
			{
				Role: "user",
				Content: mcp.NewTextContent(buf.String()),
			},
		},
	}
	
	return result, nil
}