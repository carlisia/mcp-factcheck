package prompts

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// Prompt defines the interface for all prompt implementations
type Prompt interface {
	// Name returns the unique identifier for this prompt
	Name() string
	
	// Description returns a human-readable description
	Description() string
	
	// Arguments returns the argument definitions for this prompt
	Arguments() []Argument
	
	// Render generates the prompt content with the given arguments
	Render(ctx context.Context, args Arguments) (*mcp.GetPromptResult, error)
}

// Argument defines a prompt argument with validation
type Argument struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Type        string `json:"type,omitempty"`
	Default     string `json:"default,omitempty"`
}

// Arguments provides type-safe access to prompt arguments
type Arguments map[string]interface{}

// String safely returns a string argument with fallback to default
func (a Arguments) String(key, defaultValue string) string {
	if val, exists := a[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

// Bool safely returns a boolean argument with fallback to default
func (a Arguments) Bool(key string, defaultValue bool) bool {
	if val, exists := a[key]; exists {
		if b, ok := val.(bool); ok {
			return b
		}
		// Handle string representations of booleans
		if str, ok := val.(string); ok {
			return str == "true"
		}
	}
	return defaultValue
}

// Required returns a required string argument or error
func (a Arguments) Required(key string) (string, error) {
	if val, exists := a[key]; exists {
		if str, ok := val.(string); ok && str != "" {
			return str, nil
		}
	}
	return "", fmt.Errorf("required argument missing or empty: %s", key)
}

// Registry manages all available prompts
type Registry struct {
	prompts map[string]Prompt
}

// NewRegistry creates a new prompt registry
func NewRegistry() *Registry {
	return &Registry{
		prompts: make(map[string]Prompt),
	}
}

// Register adds a prompt to the registry
func (r *Registry) Register(prompt Prompt) error {
	name := prompt.Name()
	if name == "" {
		return fmt.Errorf("prompt name cannot be empty")
	}
	
	if _, exists := r.prompts[name]; exists {
		return fmt.Errorf("prompt already registered: %s", name)
	}
	
	r.prompts[name] = prompt
	return nil
}

// Get returns a prompt by name
func (r *Registry) Get(name string) (Prompt, error) {
	prompt, exists := r.prompts[name]
	if !exists {
		return nil, fmt.Errorf("prompt not found: %s", name)
	}
	return prompt, nil
}

// List returns all registered prompts
func (r *Registry) List() []Prompt {
	prompts := make([]Prompt, 0, len(r.prompts))
	for _, prompt := range r.prompts {
		prompts = append(prompts, prompt)
	}
	return prompts
}

// ListForMCP returns prompts in MCP format for prompts/list
func (r *Registry) ListForMCP() (*mcp.ListPromptsResult, error) {
	prompts := make([]mcp.Prompt, 0, len(r.prompts))
	
	for _, prompt := range r.prompts {
		// Convert our arguments to MCP prompt arguments
		var mcpArgs []mcp.PromptArgument
		for _, arg := range prompt.Arguments() {
			mcpArgs = append(mcpArgs, mcp.PromptArgument{
				Name:        arg.Name,
				Description: arg.Description,
				Required:    arg.Required,
			})
		}
		
		prompts = append(prompts, mcp.Prompt{
			Name:        prompt.Name(),
			Description: prompt.Description(),
			Arguments:   mcpArgs,
		})
	}
	
	return &mcp.ListPromptsResult{
		Prompts: prompts,
	}, nil
}

// RenderPrompt renders a prompt with the given arguments
func (r *Registry) RenderPrompt(ctx context.Context, name string, args Arguments) (*mcp.GetPromptResult, error) {
	prompt, err := r.Get(name)
	if err != nil {
		return nil, err
	}
	
	// Validate required arguments
	for _, arg := range prompt.Arguments() {
		if arg.Required {
			if _, err := args.Required(arg.Name); err != nil {
				return nil, err
			}
		}
	}
	
	return prompt.Render(ctx, args)
}