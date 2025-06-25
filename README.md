# MCP Fact-Check Tool

An MCP Server for validating content about the **Model Context Protocol (MCP)** against the official specification to ensure technical accuracy and prevent the spread of misinformation.

## Overview

The MCP Fact-Check Tool is a specialized MCP Server designed to combat misinformation in the rapidly evolving MCP ecosystem. As MCP gains adoption, ensuring accurate technical documentation and content becomes critical for maintaining ecosystem integrity and preventing vulnerabilities that could arise from incorrect implementations based on inaccurate information.

This tool helps technical writers, developers, and content creators validate their MCP-related content for accuracy by using AI-powered analysis to compare user content against the official MCP specification. It provides detailed feedback on potential inaccuracies, ambiguities, or missing information that could lead to incorrect implementations or issues.

**Current Status**: HTTP-based prototype with plans to migrate to full MCP protocol compliance.

## Features

### ✅ Current Capabilities

- **CLI Interface**: Command-line tool for easy content validation
- **Multiple Input Methods**: Validate content from files (`--file`) or direct text (`--blurb`)
- **AI-Powered Analysis**: Uses OpenAI GPT-4 for intelligent content comparison
- **Client-Server Architecture**: Separate client and server components
- **Configurable**: Environment-based configuration for API keys and server settings

### 🚧 Planned Features

- **MCP Protocol Compliance**: Full JSON-RPC MCP client/server implementation
- **Embedding-Based Analysis**: Semantic content comparison using text embeddings
- **Batch Processing**: Validate multiple files simultaneously
- **Interactive Mode**: Real-time content validation session
- **Structured Feedback**: Detailed, section-specific guidance and suggestions

## Installation

### Prerequisites

- Go 1.24.1 or later
- OpenAI API key

### Build from Source

```bash
# Clone the repository
git clone https://github.com/carlisia/mcp-factcheck.git
cd mcp-factcheck

# Build the binaries
go build -o bin/factcheck-client ./cmd/factcheck-client
go build -o bin/factcheck-server ./cmd/factcheck-server
```

## Usage

### 1. Start the Server

```bash
export OPENAI_API_KEY="your-api-key-here"
./bin/factcheck-server
```

The server will start on port 8080 by default (configurable via `PORT` environment variable).

### 2. Use the Client

**Validate a file:**

```bash
./bin/factcheck-client verify --file content.md
```

**Validate text directly:**

```bash
./bin/factcheck-client verify --blurb "Your MCP content here"
```

**Use custom server URL:**

```bash
./bin/factcheck-client verify --server http://localhost:9000 --file content.md
```

### Command Reference

```bash
# Show help
./bin/factcheck-client --help
./bin/factcheck-client verify --help

# Validate content
./bin/factcheck-client verify --file path/to/content.md
./bin/factcheck-client verify --blurb "MCP content to validate"
./bin/factcheck-client verify --server http://custom-server:8080 --file content.md
```

## Configuration

### Environment Variables

- `OPENAI_API_KEY`: Required for server operation
- `PORT`: Server port (default: 8080)

## Architecture

```text
┌─────────────────┐    HTTP/JSON    ┌─────────────────┐
│                 │ ────────────────► │                 │
│ factcheck-client│                  │ factcheck-server│
│                 │ ◄──────────────── │                 │
└─────────────────┘                  └─────────────────┘
                                             │
                                             ▼
                                      ┌─────────────┐
                                      │ OpenAI API  │
                                      │ (GPT-4)     │
                                      └─────────────┘
```

## Roadmap

This project is actively being developed toward full MCP protocol compliance. See our detailed [ROADMAP.md](ROADMAP.md) for:

- **Current implementation status** (17 features completed ✅)
- **Planned features** (49 features in development ❌)
- **Migration path** from HTTP to MCP protocol
- **Development phases** and milestones

### Key Milestones

1. **Enhanced Prototype**: Complete fact-checking functionality
2. **MCP Protocol Migration**: Replace HTTP with JSON-RPC MCP protocol
3. **Production Ready**: Add testing, security, and monitoring
4. **Ecosystem Integration**: Publish as official MCP validation tool

## Contributing

This project is in active development. Contributions are welcome!

### Development Setup

```bash
# Install dependencies
go mod download

# Run tests (when available)
go test ./...

# Build and test locally
go build ./...
./bin/factcheck-server &
./bin/factcheck-client verify --blurb "Test content"
```

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.

## Support

For issues, questions, or contributions, please visit the project repository or open an issue.

---

**Note**: This is a prototype implementation. The tool currently uses HTTP for communication but will be refactored to use the native MCP protocol as outlined in the [roadmap](ROADMAP.md).

