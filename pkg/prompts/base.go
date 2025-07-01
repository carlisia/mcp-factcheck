package prompts

import (
	"bytes"
	"context"
	"text/template"

	"github.com/mark3labs/mcp-go/mcp"
)

// BasePrompt provides a standard implementation of the Prompt interface
type BasePrompt struct {
	name        string
	description string
	arguments   []Argument
	template    *template.Template
}

// NewBasePrompt creates a new base prompt with the given template
func NewBasePrompt(name, description, templateText string, args []Argument) (*BasePrompt, error) {
	tmpl, err := template.New(name).Parse(templateText)
	if err != nil {
		return nil, err
	}
	
	return &BasePrompt{
		name:        name,
		description: description,
		arguments:   args,
		template:    tmpl,
	}, nil
}

// NewBasePromptWithFuncs creates a new base prompt with template functions
func NewBasePromptWithFuncs(name, description, templateText string, args []Argument, funcs map[string]any) (*BasePrompt, error) {
	tmpl, err := template.New(name).Funcs(template.FuncMap(funcs)).Parse(templateText)
	if err != nil {
		return nil, err
	}
	
	return &BasePrompt{
		name:        name,
		description: description,
		arguments:   args,
		template:    tmpl,
	}, nil
}

// Name returns the prompt name
func (p *BasePrompt) Name() string {
	return p.name
}

// Description returns the prompt description
func (p *BasePrompt) Description() string {
	return p.description
}

// Arguments returns the prompt arguments
func (p *BasePrompt) Arguments() []Argument {
	return p.arguments
}

// Render generates the prompt content with the given arguments
func (p *BasePrompt) Render(ctx context.Context, args Arguments) (*mcp.GetPromptResult, error) {
	// Create template data with defaults
	templateData := make(map[string]any)
	
	// Set defaults first
	for _, arg := range p.arguments {
		if arg.Default != "" {
			templateData[arg.Name] = arg.Default
		}
	}
	
	// Override with provided arguments
	for key, value := range args {
		templateData[key] = value
	}
	
	// Execute template
	var buf bytes.Buffer
	if err := p.template.Execute(&buf, templateData); err != nil {
		return nil, err
	}
	
	// Create the MCP result
	return &mcp.GetPromptResult{
		Description: p.description,
		Messages: []mcp.PromptMessage{
			{
				Role:    "user",
				Content: mcp.NewTextContent(buf.String()),
			},
		},
	}, nil
}