# MCP Fact-Check Tool

An MCP Server that validates content and code against the official Model Context Protocol specifications using semantic search and AI-powered analysis.

## Overview

The MCP Fact-Check Tool helps ensure technical accuracy when writing about MCP by comparing content against official specifications. It uses:

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

### Claude Desktop Integration

1. Build the server:

```bash
go build -o bin/mcp-factcheck-server ./cmd/mcp-factcheck-server
```

2. Add to Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

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

### Debug Mode

To see what data is being passed to the LLM, add the `--debug` flag:

```json
{
  "mcpServers": {
    "mcp-factcheck": {
      "command": "/path/to/bin/mcp-factcheck-server",
      "args": ["--data-dir", "/path/to/data/embeddings", "--debug"],
      "env": {
        "OPENAI_API_KEY": "your-api-key"
      }
    }
  }
}
```

The debug server will automatically start on port 8083 when the MCP server starts and shut down when it stops. Access the debug interface at `http://localhost:8083`

Optional: Use `--debug-port 8084` to run the debug server on a different port.

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
└── observability/         # Debug interface
    ├── interface.go       # Observer interface
    └── debug.go           # HTTP debug server

data/
├── specs/                 # Extracted MCP specifications
└── embeddings/            # Pre-generated embeddings
```

## Environment Variables

- `OPENAI_API_KEY` - Required for embedding generation and content validation
- `GITHUB_TOKEN` - Optional, for higher GitHub API rate limits when extracting specs

## License

MIT License. See [LICENSE](LICENSE) for details.

