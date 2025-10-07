# Installation Guide

## For End Users

The MCP Fact-Check server is distributed through the [Model Context Protocol registry](https://github.com/modelcontextprotocol/servers).

**To install:**

1. Open your MCP client (Claude Desktop, etc.)
2. Search for "mcp-factcheck" in the server marketplace
3. Click install
4. Provide your OpenAI API key when prompted

The server will be automatically configured and ready to use.

---

## For Developers and Contributors

This section is for developers who want to build from source, contribute to the project, or run the server locally for development.

### Prerequisites

- Go 1.22 or later
- Git
- OpenAI API key

### Build from Source

```bash
# Clone the repository
git clone https://github.com/carlisia/mcp-factcheck.git
cd mcp-factcheck

# Build the binary
go build -o bin/mfc ./cmd/server

# Or use make
make build
```

### Development Setup

```bash
# Install dependencies
go mod download

# Run tests
go test ./...

# Run linter
golangci-lint run

# Build all tools
make build-all
```

### Running Locally

```bash
# Set your OpenAI API key
export OPENAI_API_KEY="your-api-key"

# Run the server
./bin/mfc

# Or with custom data directory
./bin/mfc --data-dir ./data/embeddings
```

### Testing with MCP Clients

To test your local build with Claude Desktop or other MCP clients, manually configure the server:

**Configuration file locations:**

- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
- **Linux**: `~/.config/Claude/claude_desktop_config.json`

**Example configuration:**

```json
{
  "mcpServers": {
    "mcp-factcheck-dev": {
      "command": "/path/to/mcp-factcheck/bin/mfc",
      "env": {
        "OPENAI_API_KEY": "your-openai-api-key"
      }
    }
  }
}
```

### Docker Development

Build and run using Docker:

```bash
# Build the image
docker build -t mcp-factcheck:dev .

# Run the container
docker run -i --rm -e OPENAI_API_KEY=your-key mcp-factcheck:dev
```

### Updating Specifications

The project includes pre-extracted MCP specifications. To update:

```bash
# Build the spec loader tool
go build -o bin/specloader ./utils/cmd

# Update draft specification
./bin/specloader spec --version draft
./bin/specloader embed --version draft

# Add a new version
./bin/specloader spec --version 2025-12-15
./bin/specloader embed --version 2025-12-15
```

### Testing Tools

Test the server using the included test client:

```bash
# Build test client
go build -o bin/factcheck-curl ./cmd/factcheck-curl

# Test tools
./bin/factcheck-curl --cmd ./bin/mfc tools/list

# Test validation
./bin/factcheck-curl --cmd ./bin/mfc tools/call check_mcp_claim '{"content":"MCP is a protocol"}'
```

### Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests and linting
5. Submit a pull request

See [CONTRIBUTING.md](CONTRIBUTING.md) if available, or follow standard GitHub contribution workflows.

### Release Process

Releases are automated via GoReleaser and GitHub Actions. When a new tag is pushed:

1. Binaries are built for all platforms
2. Docker images are published to GitHub Container Registry
3. The MCP registry is automatically updated

To create a release, maintainers push a version tag:

```bash
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0
```

### Troubleshooting Development

**Permission denied:**

```bash
chmod +x bin/mfc
```

**Missing dependencies:**

```bash
go mod tidy
go mod download
```

**OpenAI API errors:**
Ensure `OPENAI_API_KEY` is set in your environment or MCP client configuration.
