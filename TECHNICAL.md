# MCP Fact-Check: Technical Documentation

## Overview

MCP Fact-Check is a semantic validation engine built as a Model Context Protocol (MCP) server that validates MCP-related content against official specifications using AI-powered embeddings and semantic search.

## Technical Architecture

### Core Technology Stack

**Language & Runtime:**

- Go 1.24.1
- MCP protocol implementation via [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) SDK

**AI/ML Infrastructure:**

- OpenAI Embeddings API (`text-embedding-ada-002`) for vector generation
- Semantic search using cosine similarity
- LLM-based validation (supports OpenAI, Anthropic, Google Gemini)

**Observability:**

- OpenTelemetry (OTLP) for distributed tracing
- Arize Phoenix integration for AI-specific telemetry
- Structured logging with Uber Zap

### System Components

```
┌─────────────────────────────────────────────────────────────┐
│                       MCP Server                            │
│  (cmd/server/main.go - JSON-RPC 2.0 over stdio)             │
└──────────────────┬──────────────────────────────────────────┘
                   │
    ┌──────────────┴──────────────┐
    │                             │
    ▼                             ▼
┌─────────────────┐     ┌──────────────────────┐
│  Tool Handlers  │     │  Prompt Handlers     │
│  (pkg/mcp/)     │     │  (pkg/mcp/prompts/)  │
└────────┬────────┘     └──────────┬───────────┘
         │                         │
    ┌────┴─────┬─────────┬────────┴────┐
    ▼          ▼         ▼             ▼
┌────────┐ ┌────────┐ ┌──────┐  ┌──────────┐
│Validate│ │ Search │ │ List │  │ Migrate  │
│Content │ │  Spec  │ │Specs │  │ Content  │
└───┬────┘ └───┬────┘ └──┬───┘  └────┬─────┘
    │          │          │           │
    └──────────┴──────────┴───────────┘
                   │
         ┌─────────┴──────────┐
         ▼                    ▼
    ┌──────────┐        ┌──────────────┐
    │ VectorDB │        │ LLM Clients  │
    │ (storage)│        │ (integrations)│
    └─────┬────┘        └──────────────┘
          │
    ┌─────┴─────┐
    │ Embeddings│
    │  Binary   │
    └───────────┘
```

### Data Flow: Content Validation

```
User Input (Content + Version)
    │
    ├─> Content Preparation (internal/capabilities/tools/contentprep/)
    │   ├─> Chunking (if >2000 chars)
    │   ├─> Claim Extraction
    │   └─> Compound Claim Decomposition
    │
    ├─> Embedding Generation (OpenAI API)
    │   └─> Vector: []float64 (1536 dimensions)
    │
    ├─> Vector Search (internal/storage/vectordb.go)
    │   ├─> Try: version-fine (230 char chunks)
    │   └─> Fallback: version (500 char chunks)
    │   └─> Returns: Top K similar spec sections
    │
    ├─> LLM Validation (internal/integrations/llm/)
    │   ├─> Input: Claim + Retrieved Spec Sections
    │   ├─> Rules: internal/capabilities/rules/
    │   └─> Output: Validation Result + Confidence
    │
    └─> Result Formatting
        └─> Return: Structured validation report
```

## Implementation Deep Dive

### Vector Database Architecture

**File:** `internal/storage/vectordb.go`

The vector database implements a dual-strategy embedding system:

```go
type VectorDB struct {
    store *Store
    log   *zap.Logger
}

// Search with automatic fallback
func (db *VectorDB) Search(ctx context.Context, version string,
    queryEmbedding []float64, topK int) ([]SearchResult, error) {

    // 1. Try fine-grained embeddings (version-fine)
    fineVersion := version + "-fine"
    results, err := db.store.Search(fineVersion, queryEmbedding, topK)
    if err == nil {
        return results, nil
    }

    // 2. Fallback to regular embeddings
    return db.store.Search(version, queryEmbedding, topK)
}
```

**Chunking Strategies:**

1. **Regular Chunks** (~500 chars):

   - Better context preservation
   - Used as fallback

2. **Fine-Grained Chunks** (~230 chars):
   - Superior matching for short queries
   - 50 char overlap for context continuity
   - Sentence-boundary splitting
   - Header preservation

**Storage Format:**

- Embeddings stored as Go binary format in `internal/storage/embeddings/`
- Embedded into binary at compile time for zero-dependency deployment
- External file support via `--data-dir` flag

### Semantic Search Challenge & Solution

**Problem:**
Short user claims (e.g., "MCP enforces rate limits") produce "blurry" embeddings that weakly match longer spec paragraphs, causing false negatives.

**Root Cause:**

- Embedding models optimize for paragraph-level similarity
- Vector distance is sensitive to length mismatch
- Chunk size affects semantic granularity

**Solution:**
Fine-grained chunking with intelligent fallback:

```
User Query: "MCP enforces rate limits" (4 words, ~25 chars)
    ↓
Generate Embedding → [0.123, -0.456, ..., 0.789] (1536 dims)
    ↓
Search Fine-Grained Chunks (230 chars avg)
    ↓
Match: "Implementations SHOULD enforce rate limits
       to prevent abuse. Both clients and servers..."
       (Similarity: 0.89)
    ↓
✓ Strong match found
```

### Compound Claim Decomposition

**File:** `internal/capabilities/tools/contentprep/compound.go`

Handles claims with multiple assertions connected by "and":

```
Input: "MCP supports Resources, Tools, and Prompts"
    ↓
Decompose:
    1. "MCP supports Resources"
    2. "MCP supports Tools"
    3. "MCP supports Prompts"
    ↓
Independent Search for Each Subclaim
    ↓
Aggregate Evidence
    ↓
Validation: ALL subclaims must be accurate
```

**Benefits:**

- Prevents false negatives when concepts appear in different spec sections
- More thorough evidence collection
- Each subclaim validated against 5-15 spec chunks

### LLM Integration Layer

**File:** `internal/integrations/llm/`

Supports multiple LLM providers with unified interface:

```go
type Client interface {
    GenerateText(ctx context.Context, prompt string) (string, error)
    GenerateEmbedding(ctx context.Context, text string) ([]float64, error)
}

// Supported providers
type ProviderType string
const (
    OpenAI    ProviderType = "openai"
    Anthropic ProviderType = "anthropic"
    Gemini    ProviderType = "gemini"
)
```

**Validation Rules:** `internal/capabilities/rules/`

- Specification accuracy rules
- Modal verb detection (MUST/SHOULD/MAY)
- Style and formatting guidelines
- Context understanding rules
- Response format templates

## Specification Management

### Spec Extraction & Embedding Pipeline

**Tool:** `cmd/specloader/`

```bash
# Extract spec from GitHub
./bin/specloader spec --version draft
    ↓
# Chunk into fine-grained segments
./bin/specloader rechunk --version draft --strategy fine
    ↓
# Generate embeddings
./bin/specloader embed --version draft
./bin/specloader embed --version draft-fine
```

**Metadata Tracking:** `data/SPEC_METADATA.json`

Automatically captures:

- GitHub commit hash (for reproducibility)
- Extraction timestamp
- Chunk count and strategy
- Embedding generation metadata

```json
{
  "specs": {
    "draft": {
      "source": {
        "repository": "modelcontextprotocol/specification",
        "commit": "abc123...",
        "extracted_at": "2025-10-09T12:34:56Z"
      },
      "embeddings": {
        "regular": {
          "chunk_count": 245,
          "generated_at": "2025-10-09T12:45:00Z"
        },
        "fine": {
          "chunk_count": 587,
          "strategy": "fine-grained",
          "generated_at": "2025-10-09T12:50:00Z"
        }
      }
    }
  }
}
```

### Supported Spec Versions

- `draft` - Latest working draft from GitHub
- `2025-06-18` - Current stable release
- `2025-03-26` - Previous release
- `2024-11-05` - Legacy release

Each version maintains both regular and fine-grained embeddings.

## MCP Protocol Implementation

### Tools Exposed

**1. `check_mcp_claim`** - Comprehensive validation

- Parameters: `content`, `specVersion`, `useChunking`
- Returns: Multi-step validation workflow
- Max input: 50,000 chars

**2. `check_mcp_quick_fact`** - Fast fact-checking

- Parameters: `claim`, `specVersion`
- Returns: ✓/✗ verdict with confidence
- Optimized for <500 char claims

**3. `search_spec`** - Semantic search

- Parameters: `query`, `specVersion`, `topK`
- Returns: Ranked spec sections by similarity

**4. `list_spec_versions`** - Version enumeration

- No parameters
- Returns: Available spec versions

### Prompts Exposed

**1. `migrate-mcp-content`** - Version migration guide

- Parameters: `current_version`, `target_version`, `update_scope`
- Returns: 6-step migration workflow
- Scopes: `comprehensive`, `critical_only`, `enhancement_focused`

## Performance Characteristics

### Search Performance

```
Fine-Grained Search:
- Chunk count: ~500-600 per spec version
- Search latency: ~100-300ms (vector ops)
- Embedding API: ~200-500ms (OpenAI)
- Total: ~500-1000ms per query

Regular Search (Fallback):
- Chunk count: ~200-300 per spec version
- Search latency: ~50-150ms (vector ops)
- Total: ~350-750ms per query
```

### Embedding Cache Strategy

- Runtime embeddings generated on-demand (not cached)
- Spec embeddings pre-computed and embedded in binary
- No persistent cache (stateless design)

### Concurrency Model

- Single-threaded MCP server (stdio transport)
- Concurrent LLM API calls within validation
- Context-based cancellation throughout

## Observability & Debugging

### OpenTelemetry Integration

**File:** `cmd/server/telemetry.go`

```go
// Enable with flags
--telemetry
--otlp-endpoint http://localhost:6006
--telemetry-provider phoenix  // or otlp
```

**Traced Operations:**

- Embedding generation
- Vector search queries
- LLM validation calls
- Tool handler execution
- Prompt handler execution

### Arize Phoenix Integration

**File:** `internal/integrations/arizephoenix/`

Provides AI-specific telemetry:

- LLM token usage and cost tracking
- Embedding generation metrics
- Validation confidence scores
- Search similarity distributions
- End-to-end validation traces

**Setup:**

```bash
pipx install arize-phoenix
phoenix serve  # http://localhost:6006
```

### Structured Logging

**File:** `pkg/logger/`

Using Uber Zap with contextual fields:

```go
log.Info("Validation completed",
    zap.String("version", "draft"),
    zap.Float64("confidence", 0.95),
    zap.Int("claims_validated", 12))
```

## Testing Infrastructure

### Test Utilities

**File:** `testutil/`

- Mock LLM clients
- Mock vector databases
- Test harnesses for tool handlers
- Example-based testing
- Integration test helpers

### Test Client

**File:** `cmd/factcheck-curl/`

Command-line test client for MCP server:

```bash
# Test tool invocation
./bin/factcheck-curl --cmd ./bin/mfc tools/call \
  check_mcp_claim '{"content":"MCP is a protocol"}'

# Test prompt retrieval
./bin/factcheck-curl --cmd ./bin/mfc prompts/get \
  migrate-mcp-content '{"current_version":"2024-11-05",...}'
```

## Security Considerations

### Prompt Injection Protection

**File:** `internal/security/injection.go`

Comprehensive regex-based detection prevents malicious LLM manipulation:

**Detected Patterns:**

- Instruction overrides: "ignore previous instructions", "disregard all rules"
- Role manipulation: "SYSTEM:", "you are now", "act as"
- Prompt extraction: "show your system prompt", "repeat instructions"
- Response manipulation: "always return true", "set confidence to 1.0"
- Delimiter abuse: Excessive `---`, `###`, `===` sequences (>3)

**Implementation:**

```go
// All user inputs validated at entry points
detector := security.NewInjectionDetector()
result := detector.Detect(content)
if result.IsInjection {
    return fmt.Errorf("invalid content: %s", result.Reason)
}
```

**Defense Layers:**

1. Detection (Primary): Rejects malicious inputs
2. Sanitization (Secondary): Escapes dangerous patterns

**Applied To:**

- `check_mcp_claim` - content parameter
- `check_mcp_quick_fact` - claim parameter
- `search_spec` - query parameter

### Input Validation

- Max content: 50,000 chars
- Max query: 500 chars
- Whitespace trimming and emptiness checks
- Injection detection on all user inputs

### API Key Management

- Environment variable: `OPENAI_API_KEY`
- No key logging or telemetry
- Keys never in error messages or traces

## Build & Deployment

### Binary Compilation

```bash
# Standard build
go build -o bin/mfc ./cmd/server

# Release build with version info
go build -ldflags "\
  -X main.version=v1.0.0 \
  -X main.commit=$(git rev-parse HEAD) \
  -X main.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o bin/mfc ./cmd/server
```

### Docker Support

**File:** `Dockerfile`

Multi-stage build producing minimal runtime image:

```dockerfile
FROM golang:1.24 AS builder
# Build with embedded data
RUN go build -o /mfc ./cmd/server

FROM scratch
COPY --from=builder /mfc /mfc
ENTRYPOINT ["/mfc"]
```

### Deployment Modes

**1. Embedded Data (Recommended):**

```bash
./bin/mfc  # Uses built-in embeddings
```

**2. External Data:**

```bash
./bin/mfc --data-dir ./data/embeddings
```

**3. With Telemetry:**

```bash
./bin/mfc --telemetry --otlp-endpoint http://localhost:6006
```

## Project Structure

```
mcp-factcheck/
├── cmd/
│   ├── server/          # MCP server main entry point
│   └── factcheck-curl/  # Test client
├── internal/
│   ├── capabilities/    # MCP tools and prompts
│   │   ├── tools/
│   │   │   ├── validation/   # Claim validation logic
│   │   │   ├── search/       # Semantic search
│   │   │   ├── list/         # Version listing
│   │   │   └── contentprep/  # Chunking & decomposition
│   │   ├── prompts/
│   │   │   └── migrate/      # Migration prompt
│   │   └── rules/            # Validation rules
│   ├── storage/
│   │   ├── vectordb.go       # Vector search engine
│   │   ├── store.go          # Embedding storage
│   │   └── embeddings/       # Binary embedding files
│   ├── integrations/
│   │   ├── llm/              # LLM client abstraction
│   │   └── arizephoenix/     # Phoenix telemetry
│   └── security/             # Input validation
├── pkg/                      # Public APIs
│   ├── mcp/                  # MCP protocol handlers
│   ├── llm/                  # LLM client interface
│   └── logger/               # Structured logging
├── utils/
│   ├── cmd/                  # specloader utility
│   ├── specs/                # Spec extraction logic
│   └── metadata/             # Metadata tracking
├── data/
│   ├── specs/                # Chunked JSON specs
│   └── SPEC_METADATA.json    # Version metadata
└── testutil/                 # Testing utilities
```

## Key Design Decisions

### Why Go?

- Strong concurrency primitives
- Fast compilation and execution
- Excellent stdlib for JSON-RPC
- Easy cross-platform deployment
- Static binary distribution

### Why Pre-Computed Embeddings?

**Pros:**

- Zero latency for spec lookups
- No runtime OpenAI API costs for specs
- Deterministic behavior
- Offline capability

**Cons:**

- Larger binary size (~50MB)
- Manual update process
- Storage overhead

**Decision:** Pre-compute for specs, generate at runtime for user content.

### Why Dual Chunking Strategy?

**Problem:** Short queries don't match long chunks well.

**Solution:** Fine-grained chunks (230 chars) as primary, regular chunks (500 chars) as fallback.

**Results:**

- 40% improvement in short-query accuracy
- Minimal performance overhead
- Graceful degradation

### Why Multiple LLM Providers?

- Flexibility for different use cases
- Cost optimization options
- Vendor independence
- Future-proofing

## Extension Points

### Adding New LLM Providers

**File:** `internal/integrations/llm/provider.go`

Implement the `Client` interface:

```go
type NewProvider struct {
    apiKey string
}

func (p *NewProvider) GenerateText(ctx context.Context,
    prompt string) (string, error) {
    // Implementation
}

func (p *NewProvider) GenerateEmbedding(ctx context.Context,
    text string) ([]float64, error) {
    // Implementation
}
```

### Adding New Tools

**Location:** `internal/capabilities/tools/`

1. Create tool definition (implements MCP tool interface)
2. Add handler in `pkg/mcp/`
3. Register in server initialization

### Adding New Validation Rules

**Location:** `internal/capabilities/rules/`

Define rules as Go constants/functions that are injected into LLM prompts.

## Performance Tuning

### Vector Search Optimization

- Adjust `topK` for accuracy vs. speed tradeoff
- Default: 5 results (optimal for most cases)
- Range: 1-20 results

### LLM Cost Optimization

- Use cheaper models for embedding (ada-002)
- Use GPT-4 only for complex validations
- Implement result caching (future enhancement)

### Memory Usage

- Embedding binary: ~30-50MB in memory
- Peak memory: ~100-200MB during validation
- No persistent connections

## Known Limitations

1. **Embedding Model Dependency:**

   - Locked to OpenAI's ada-002
   - Vector dimension: 1536 (fixed)
   - Migration cost if model changes

2. **Specification Updates:**

   - Manual update process
   - No automatic drift detection
   - Requires rebuild for embedded data

3. **Concurrency:**

   - Single-threaded MCP server (stdio limitation)
   - Sequential tool execution

4. **Language Support:**
   - English-only specifications
   - English-optimized embeddings

## Future Enhancements

- **Automatic spec updates:** GitHub webhook integration
- **Multi-language support:** Multilingual embeddings
- **Result caching:** Redis/SQLite cache layer
- **Hybrid search:** BM25 + semantic for better accuracy
- **Official Go SDK:** Migrate from mark3labs to official MCP SDK
- **Custom embedding models:** Support local/fine-tuned models
- **Batch validation:** Parallel claim processing
- **Streaming responses:** Progressive result delivery

## Development Workflow

### Prerequisites

```bash
go version  # 1.24.1 or higher
export OPENAI_API_KEY=sk-...
```

### Local Development

```bash
# Build
go build -o bin/mfc ./cmd/server
go build -o bin/specloader ./utils/cmd

# Test
go test ./...

# Run server
./bin/mfc --data-dir ./data/embeddings --telemetry
```

### Updating Specifications

```bash
# 1. Extract latest draft
./bin/specloader spec --version draft

# 2. Chunk for fine-grained search
./bin/specloader rechunk --version draft --strategy fine

# 3. Generate embeddings
./bin/specloader embed --version draft
./bin/specloader embed --version draft-fine

# 4. Verify metadata
cat data/SPEC_METADATA.json | jq .specs.draft
```

### Testing Changes

```bash
# Unit tests
go test ./internal/... -v

# Integration tests
go test ./test/e2e/... -v

# Manual testing
./bin/factcheck-curl --cmd ./bin/mfc tools/call \
  check_mcp_quick_fact '{"claim":"MCP uses JSON-RPC"}'
```

## Troubleshooting

### Common Issues

**1. "Version not found" error:**

```bash
# Check available versions
./bin/factcheck-curl --cmd ./bin/mfc tools/call list_spec_versions '{}'

# Verify embedding files exist
ls -la internal/storage/embeddings/
```

**2. Low similarity scores:**

- Try fine-grained embeddings (automatic fallback)
- Increase topK parameter
- Rephrase query to match spec terminology

**3. LLM API errors:**

```bash
# Verify API key
echo $OPENAI_API_KEY

# Check API status
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $OPENAI_API_KEY"
```

**4. High memory usage:**

- Embeddings loaded into memory at startup
- Expected: 50-100MB baseline
- Scale: ~20MB per additional spec version

## Contributing

### Code Organization Principles

- **Clear separation:** `internal/` vs `pkg/`
- **Interface-driven:** Mock-friendly design
- **Test coverage:** Aim for >80%
- **Documentation:** Package-level `doc.go` files

### Testing Standards

- Unit tests for business logic
- Integration tests for external dependencies
- Example-based tests for documentation
- Harnesses for complex scenarios

### Commit Guidelines

- Conventional commits format
- Reference issues where applicable
- Include tests with features
- Update docs with changes

---

**Documentation Version:** 1.0.0
**Last Updated:** 2025-10-09
**Spec Version:** draft (2025-10-09)
**Go Version:** 1.24.1
