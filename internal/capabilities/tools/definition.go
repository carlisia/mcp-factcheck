package tools

// Definition represents schema and metadata for tools
type Definition struct {
	Name        string
	Description string
	Schema      map[string]any
}
