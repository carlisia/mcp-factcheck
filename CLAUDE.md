# Claude Development Guide for MCP Fact-Check

## Project Overview

MCP Fact-Check is a Model Context Protocol (MCP) server that validates content and code against MCP specifications using semantic search with vector embeddings.

## Build Commands

```bash
# Build all binaries
go build -o bin/mcp-factcheck-server ./cmd/mcp-factcheck-server
go build -o bin/factcheck-curl ./cmd/factcheck-curl

# Build utilities
go build -o bin/specloader ./utils/cmd

# Run tests
go test ./...

# Lint and format
go fmt ./...
go vet ./...
```

## Architecture

### Package Structure

```text
pkg/
├── spec/           # MCP specification functionality
│   ├── list.go     # list_spec_versions tool
│   └── search.go   # search_spec tool
└── validator/      # Content/code validation
    ├── content.go  # validate_content tool
    └── code.go     # validate_code tool
```

### MCP Tools Exposed

1. **list_spec_versions** - Lists available MCP specification versions
2. **search_spec** - Searches MCP specifications using semantic similarity
3. **validate_content** - Validates content against MCP specification
4. **validate_code** - Validates code against MCP specification with pattern detection

### Data Management

- **Specs**: Extracted from GitHub to `data/specs/` as JSON files
- **Embeddings**: Generated from specs and stored in `data/embeddings/`
- **Vector Search**: Uses OpenAI embeddings with cosine similarity

## Development Workflow

### 1. Extract MCP Specifications

```bash
# Extract specific version from GitHub
./bin/specloader spec --version draft
./bin/specloader spec --version 2025-06-18
```

### 2. Generate Embeddings

```bash
# Generate embeddings for a specific version
./bin/specloader embed --version draft
./bin/specloader embed --version 2025-06-18
```

### 3. Test MCP Server

```bash
# Start server and test tools
./bin/factcheck-curl --cmd ./bin/mcp-factcheck-server --data-dir ./data/embeddings tools/list
./bin/factcheck-curl --cmd ./bin/mcp-factcheck-server --data-dir ./data/embeddings tools/call validate_code '{"code":"..."}'
```

## Environment Variables

- `OPENAI_API_KEY` - Required for embedding generation
- `GITHUB_TOKEN` - Optional, for higher GitHub API rate limits

## Dependencies

- **Go MCP Libraries**: `github.com/mark3labs/mcp-go`
- **OpenAI API**: `github.com/sashabaranov/go-openai`
- **GitHub API**: `github.com/google/go-github/v57`
- **CLI Framework**: `github.com/spf13/cobra`

## Key Implementation Notes

- Embedding generation separated into shared `embedding/` package
- Vector storage abstracted in `vectorstore/` package
- Clean separation: utils/ for batch processing, internal/ for runtime
- All tools support multiple MCP spec versions (draft, 2025-06-18, 2025-03-26, 2024-11-05)

## Go Code Standards

- **Variable naming**: No redundant type suffixes (use `arguments` not `argsMap`)
- **Descriptive names**: Name variables based on what they contain, not their type
  - If accessing `params["language"]` → name the map `languages` (contains languages)
  - If accessing `params["specVersion"]` → name the map `specVersions` (contains spec versions)
  - Variable name should reflect the content/domain, not the container type
- **Modern Go**: Use `any` instead of `interface{}`
- **camelCase**: Use Go camelCase conventions (`specVersion` not `spec_version`)
- **No type in variable names**: Variable names should describe purpose/content, not type

