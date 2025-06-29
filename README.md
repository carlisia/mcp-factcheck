# MCP Fact-Check MCP Server

An MCP Server for validating code or content against the official **Model Context Protocol (MCP)** specification to ensure technical accuracy and prevent the spread of misinformation.

## Overview

The MCP Fact-Check MCP Server helps ensure technical accuracy when coding or writing about MCP by comparing content against official specifications. It uses:

- **Semantic search** with OpenAI embeddings to find relevant specification sections
- **AI-powered validation** to detect inaccuracies and suggest corrections
- **Multiple spec versions** support (draft, 2025-06-18, 2025-03-26, 2024-11-05)

## Features

### MCP Tools Exposed

1. **`validate_content`** - Validates text content against MCP specification

   - Provides corrected versions when content is inaccurate
   - Shows relevant specification references
   - Returns confidence scores

2. **`validate_code`** - Validates code implementations against MCP patterns

   - Detects MCP protocol usage patterns
   - Validates against specification requirements
   - Supports multiple programming languages

3. **`search_spec`** - Searches MCP specifications using semantic similarity

   - Returns most relevant specification sections
   - Supports all specification versions

4. **`list_spec_versions`** - Lists available MCP specification versions
   - Shows version dates and descriptions
   - Indicates which version is current

## Installation

### Client Integration

1. Build the server:

```bash
go build -o bin/mcp-factcheck-server ./cmd/mcp-factcheck-server
```

2. Add to the Host config

Example for Claude Code: (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "mcp-factcheck": {
      "command": "/path/to/bin/mcp-factcheck-server",
      "args": ["--data-dir", "/path/to/data/embeddings"],
      "env": {
        "OPENAI_API_KEY": "your-api-key"
      }
    }
  }
}
```

### Observability

#### Visual Tracing with Arize Phoenix

For a beautiful, AI-focused trace visualization UI, set up Arize Phoenix:

1. **Install and start Phoenix:**

```bash
# Install Phoenix
pipx install arize-phoenix

# Start Phoenix server
phoenix serve
```

2. **Update the Host config to send traces to Phoenix:**

```json
{
  "mcpServers": {
    "mcp-factcheck": {
      "command": "/path/to/bin/mcp-factcheck-server",
      "args": [
        "--data-dir",
        "/path/to/data/embeddings",
        "--telemetry",
        "--otlp-endpoint",
        "http://localhost:6006"
      ],
      "env": {
        "OPENAI_API_KEY": "your-api-key"
      }
    }
  }
}
```

3. **View traces at:** http://localhost:6006

**What you'll see in Phoenix:**

- Beautiful AI-focused interface designed for LLM applications
- Complete validation pipeline timeline with clear visual hierarchy
- Embedding generation performance and OpenAI API call tracking
- Vector search visualization with similarity scores
- Per-chunk validation confidence levels and quality metrics
- Cost tracking for OpenAI API usage (or whichever llm is being used for embedding the input content/code)
- Clean, intuitive navigation focused on AI workflows

Phoenix is specifically designed for AI/ML observability and provides a much more user-friendly experience than traditional tracing tools.

## Development

### Building

```bash
# Build all components
go build -o bin/mcp-factcheck-server ./cmd/mcp-factcheck-server
go build -o bin/specloader ./utils/cmd

# Run tests
go test ./...
```

### Updating Specifications

The project includes pre-extracted MCP specifications and embeddings for all versions up to 2025-06-18, plus the draft specification as of 2025-06-26.

**To update the draft specification:**

```bash
./bin/specloader spec --version draft
./bin/specloader embed --version draft
```

**To add a new specification version:**

```bash
./bin/specloader spec --version 2025-12-15
./bin/specloader embed --version 2025-12-15
```

### Testing Tools

Test the server using the included test client:

```bash
# Build test client
go build -o bin/factcheck-curl ./cmd/factcheck-curl

# Test tools
./bin/factcheck-curl --cmd ./bin/mcp-factcheck-server --data-dir ./data/embeddings tools/list
./bin/factcheck-curl --cmd ./bin/mcp-factcheck-server --data-dir ./data/embeddings tools/call validate_content '{"content":"MCP is a protocol"}'
```

## Architecture

```text
cmd/
├── mcp-factcheck-server/   # Main MCP server
└── factcheck-curl/         # Test client

utils/
└── cmd/                    # Specification extraction tool

pkg/
├── spec/                   # MCP specification tools
│   ├── list.go            # list_spec_versions implementation
│   └── search.go          # search_spec implementation
├── validator/             # Content/code validation
│   ├── content.go         # validate_content implementation
│   └── code.go            # validate_code implementation
└── telemetry/             # Clean telemetry abstractions
    ├── interfaces.go      # Provider, Middleware interfaces
    └── builder.go         # Fluent span builder

internal/
└── integrations/
    └── arizephoenix/      # Phoenix telemetry implementation
        ├── config.go      # Phoenix configuration
        ├── provider.go    # Phoenix provider
        ├── middleware.go  # Phoenix middleware
        └── init.go        # Initialization helpers

data/
├── specs/                 # Extracted MCP specifications
└── embeddings/            # Pre-generated embeddings
```

## Environment Variables

- `OPENAI_API_KEY` - Required for embedding generation and content validation
- `GITHUB_TOKEN` - Optional, for higher GitHub API rate limits when extracting specs

## License

MIT License. See [LICENSE](LICENSE) for details.
