package prompts

import (
	"bytes"
	_ "embed"
	"text/template"
)

//go:embed templates/fact-check.tmpl
var factCheckTemplate string

// FactCheckData holds the data for rendering the fact-check prompt
type FactCheckData struct {
	Content      string
	SpecSections []string
}

// FactCheckPrompt handles rendering of the fact-checking prompt
type FactCheckPrompt struct {
	tmpl *template.Template
}

// NewFactCheckPrompt creates a new fact-check prompt renderer
func NewFactCheckPrompt() (*FactCheckPrompt, error) {
	// Add custom function to increment index for display
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}
	
	tmpl, err := template.New("fact-check").Funcs(funcMap).Parse(factCheckTemplate)
	if err != nil {
		return nil, err
	}
	
	return &FactCheckPrompt{tmpl: tmpl}, nil
}

// Render generates the fact-check prompt with the provided data
func (p *FactCheckPrompt) Render(data FactCheckData) (string, error) {
	var buf bytes.Buffer
	if err := p.tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}