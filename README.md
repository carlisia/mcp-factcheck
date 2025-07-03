# MCP Fact-Check MCP Server

An MCP Server for validating code or content against the official **Model Context Protocol (MCP)** specification to ensure technical accuracy and prevent the spread of misinformation.

📋 **[View Project Roadmap](ROADMAP.md)** - See planned features and development progress

🏗️ **[Design Documentation](DESIGN.md)** - Technical design and implementation details

## Overview

The MCP Fact-Check MCP Server helps ensure technical accuracy when coding or writing about MCP by comparing content against official specifications. It uses:

- **Semantic search** with OpenAI embeddings to find relevant specification sections
- **AI-powered validation** to detect inaccuracies and suggest corrections
- **Compound claim decomposition** to validate complex statements with multiple assertions
- **Multiple spec versions** support (draft, 2025-06-18, 2025-03-26, 2024-11-05)

## Features

### MCP Tools Exposed

1. **`check_mcp_claim`** - Comprehensive validation of MCP-related content

   - Validates multi-claim content (documentation, tutorials, bullet points)
   - Automatically decomposes compound claims (e.g., "X and Y") for accurate validation
   - Provides step-by-step validation workflow
   - Identifies missing best practices and modal verb issues
   - Returns corrected content with confidence scores

2. **`check_mcp_quick_fact`** - Quick fact-checking for single MCP claims

   - Validates single sentences or quick questions
   - Returns concise ✓/✗ verdict with explanation
   - Uses aggressive search strategies for accuracy
   - Perfect for "Does MCP support X?" questions

3. **`validate_code`** - Validates code implementations against MCP patterns (⚠️ WIP)

   - Detects MCP protocol usage patterns
   - Validates against specification requirements (not yet implemented)
   - Supports multiple programming languages

4. **`search_spec`** - Searches MCP specifications using semantic similarity

   - Returns most relevant specification sections
   - Supports all specification versions

5. **`list_spec_versions`** - Lists available MCP specification versions
   - Shows version dates and descriptions
   - Indicates which version is current

### MCP Prompts Available

1. **`migrate-mcp-content`** - Guides content migration between MCP specification versions

   - Validates content against source specification first
   - Identifies changes between specification versions
   - Provides step-by-step migration guidance
   - Works with any type of MCP-related content
   - Preserves the original tone, style, and voice when making corrections or suggestions

   **Parameters:**

   - `current_version` (required): Source MCP specification version (e.g., "2024-11-05", "2025-06-18")
   - `target_version` (required): Target MCP specification version to migrate to (e.g., "draft")
   - `update_scope` (optional): Determines how aggressive the migration should be
     - `critical_only`: Fix only critical inaccuracies and breaking changes (minimal changes)
     - `enhancement_focused`: Fix issues and improve clarity, align with best practices
     - `comprehensive`: Complete review with all improvements and enhanced clarity
     - Default: `comprehensive`

## Installation

### Client Integration

1. Build the server:

```bash
go build -o bin/mcp-factcheck-server ./cmd/mcp-factcheck-server
```

2. Configure your MCP client

**For Claude Desktop App:**

- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
- **Linux**: `~/.config/Claude/claude_desktop_config.json`

Example configuration:

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

The project includes pre-extracted MCP specifications and embeddings for all versions. To check when the draft specification was last updated, see `data/SPEC_METADATA.json`:

```bash
# View draft update information
cat data/SPEC_METADATA.json | jq '.specs.draft'
```

**To update the draft specification:**

```bash
./bin/specloader spec --version draft
./bin/specloader embed --version draft
./bin/specloader embed --version draft-fine
```

**To add a new specification version:**

```bash
./bin/specloader spec --version 2025-12-15
./bin/specloader embed --version 2025-12-15
./bin/specloader embed --version 2025-12-15-fine
```

All specification extraction dates and source commits are automatically tracked in `data/SPEC_METADATA.json`.

### Testing Tools

Test the server using the included test client:

```bash
# Build test client
go build -o bin/factcheck-curl ./cmd/factcheck-curl

# Test tools
./bin/factcheck-curl --cmd ./bin/mcp-factcheck-server --data-dir ./data/embeddings tools/list
./bin/factcheck-curl --cmd ./bin/mcp-factcheck-server --data-dir ./data/embeddings tools/call validate_content '{"content":"MCP is a protocol"}'

# Test prompts
./bin/factcheck-curl --cmd ./bin/mcp-factcheck-server --data-dir ./data/embeddings prompts/list

# Get migration prompt with minimal parameters
./bin/factcheck-curl --cmd ./bin/mcp-factcheck-server --data-dir ./data/embeddings prompts/get migrate-mcp-content '{"current_version":"2024-11-05","target_version":"draft"}'

# Get migration prompt with all parameters
./bin/factcheck-curl --cmd ./bin/mcp-factcheck-server --data-dir ./data/embeddings prompts/get migrate-mcp-content '{
  "current_version": "2024-11-05",
  "target_version": "2025-06-18",
  "update_scope": "critical_only"
}'
```

## Architecture

See [DESIGN.md](DESIGN.md#architecture-overview) for the complete architecture documentation.

## Environment Variables

- `OPENAI_API_KEY` - Required for embedding generation and content validation
- `GITHUB_TOKEN` - Optional, for higher GitHub API rate limits when extracting specs

## License

MIT License. See [LICENSE](LICENSE) for details.
