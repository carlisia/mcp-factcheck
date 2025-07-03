# MCP Fact-Check Design Document

## Table of Contents

1. [Background & Context](#background--context)
2. [Terminology](#terminology)
3. [Tool Design & Prompt Flow](#tool-design--prompt-flow)
4. [User Prompt Design](#user-prompt-design)
5. [System Architecture](#system-architecture)
   - [Core Components](#core-components)
   - [Data Flow](#data-flow)
   - [External Dependencies](#external-dependencies)
   - [Project Structure](#project-structure)
6. [Implementation Details](#implementation-details)
   - [Semantic Search Challenge](#semantic-search-challenge)
   - [Chunking Strategy](#chunking-strategy)
   - [Spec Maintenance and Updates](#spec-maintenance-and-updates)
   - [Content Chunking (Runtime)](#content-chunking-runtime)
   - [Claim Extraction](#claim-extraction)
   - [Template System](#template-system)
   - [Error Handling](#error-handling)
7. [Trade-offs and Design Decisions](#trade-offs-and-design-decisions)
8. [Conclusion](#conclusion)

## Background & Context

The MCP Fact-Check server validates content and code against official Model Context Protocol (MCP) specifications using semantic search with vector embeddings. The system needs to handle various types of validation requests:

- Quick single-claim fact-checking (e.g., "Does MCP enforce rate limits?")
- Comprehensive multi-claim content validation (e.g., documentation, tutorials)
- Bullet-point technical descriptions
- Code validation against protocol requirements

## Terminology

### Key Terms

**Claim**: A single, atomic statement about MCP that can be evaluated as true or false. Examples:

- "MCP enforces rate limits" (single claim)
- "MCP uses JSON-RPC" (single claim)
- "MCP enforces ACLs, rate limits, and provenance" (compound claim containing 3 individual claims)

**Spec Chunking** (Maintenance Task): The process of dividing the MCP specification documents into smaller sections before generating embeddings. This is performed manually:

- **Purpose**: Create searchable units of the specification
- **When**: When updating draft specs or adding new official versions
- **Who**: Manually
- **Strategies**: Regular (~500 chars) or fine-grained (~230 chars)
- **Output**: Pre-generated embedding files committed to `data/embeddings/`

**Content Chunking** (Runtime): The optional process of dividing user input into smaller sections for validation when the content is very long. This is ONLY used by the `check_mcp_claim` tool:

- **Purpose**: Handle large documents that exceed context limits
- **When**: During validation if content > 2000 chars or explicitly requested
- **Where**: Only in `check_mcp_claim` tool (NOT used by `check_mcp_quick_fact`)
- **Strategy**: Split by paragraphs, validate each chunk separately
- **Output**: Aggregated validation results

**Embedding**: A numerical vector representation of text that captures semantic meaning, enabling similarity comparisons between different pieces of text.

**Modal Verbs**: Requirements level indicators in specifications:

- **MUST/SHALL**: Absolute requirements
- **SHOULD**: Recommended practices that may be ignored with justification
- **MAY**: Optional features

**Best Practices**: SHOULD-level requirements from the MCP specification that represent recommended but not mandatory implementation patterns.

## Tool Design & Prompt Flow

### Tool 1: check_mcp_claim

**Purpose**: Comprehensive validation of multi-claim content

**Prompt Flow**:

```text
1. LLM receives content from user (paragraphs, bullets, documentation)
    ↓
2. Full Content Embedding Generation
    ↓
3. Vector Search (fine embeddings with fallback)
    ↓
4. Retrieve Top-K Spec Sections
    ↓
5. LLM Fact-Checking Against Spec Sections
    ↓
6. Extract Individual Claims from Results
    ↓
7. Template-Based Formatting
    ↓
8. Step-by-Step Workflow Output
```

**Output Structure**:

- Step 1: Claim Extraction
- Step 2: Validation Results (per claim)
- Step 3: Missing Best Practices (SHOULD requirements)
- Step 4: Modal Verb Issues (MUST/SHOULD/MAY)
- Summary with confidence score
- Corrected content suggestions

### Tool 2: check_mcp_quick_fact

**Purpose**: Quick validation of single claims/questions

**Prompt Flow**:

```text
1. LLM receives single claim/question from user
    ↓
2. Aggressive Multi-Strategy Search:
   - Direct claim embedding
   - Pattern-based expansion ("enforces" → "implementations should")
   - Negative patterns ("never" → "must not")
    ↓
3. Aggregate and Deduplicate Results
    ↓
4. LLM Fact-Check Against Best Matches
    ↓
5. Concise Formatting
    ↓
6. ✓/✗ Verdict with Explanation
```

**Search Strategies**:

```go
// Pattern transformations in performAggressiveClaimSearch
"MCP enforces X" → [
    "implementations should must X",
    "clients servers should X",
    "security requirements X"
]

"MCP never X" → [
    "must not should not X",
    "restrictions limitations X",
    "security considerations X"
]
```

## User Prompt Design

### migrate-mcp-content Prompt

The `migrate-mcp-content` prompt is a user-facing MCP prompt that guides content migration between MCP specification versions.

**Purpose**: Helps the LLM guide users in updating their MCP-related content when specifications change between versions.

**Key Design Decisions**:

1. **Validation-First Approach**: Always validates content against the source specification before suggesting migrations
2. **Scope Control**: Three update scopes allow control over migration aggressiveness:
   - `critical_only`: Minimal changes for breaking issues only
   - `enhancement_focused`: Fixes plus clarity improvements
   - `comprehensive`: Full review with all enhancements
3. **Tone Preservation**: Explicitly preserves the original content's voice and style
4. **Structured Output**: Returns step-by-step migration guidance

**Prompt Template Structure**:

```
1. Validate content against source spec
2. Identify spec differences between versions
3. Apply changes based on update_scope
4. Preserve original tone and style
5. Provide migration recommendations
```

**Parameters**:

- `current_version` (required): Source MCP spec version
- `target_version` (required): Target MCP spec version
- `update_scope` (optional): Migration aggressiveness level

This design ensures the LLM can provide users with controlled, accurate, specification-compliant migration guidance.

## System Architecture

### Core Components

The MCP Fact-Check server consists of several key components:

1. **MCP Server** (`cmd/mcp-factcheck-server/`): Implements the MCP protocol and exposes specs and validation tools, and user prompts
2. **Vector Database** (`internal/embedding/vectordb.go`): Manages embeddings with fine-grained fallback
3. **Embedding Generator** (`embedding/generator.go`): OpenAI-based embedding generation
4. **Validator Package** (`pkg/validator/`): Implements validation related MCP tools
   - `check_mcp_claim` (content.go) - Comprehensive content validation
   - `check_mcp_quick_fact` (quick_claim.go) - Quick fact-checking
   - `validate_code` (code.go) - Code validation (WIP)
5. **Specification Tools** (`pkg/spec/`): Implements spec-related MCP tools
   - `list_spec_versions` (list.go) - Lists available MCP versions
   - `search_spec` (search.go) - Semantic search in specifications
6. **Prompt Service** (`pkg/prompts/`): MCP user prompts
   - `migrate-mcp-content`
7. **Telemetry System** (`pkg/telemetry/`): Clean abstraction for observability

### Data Flow

1. **Specification Extraction**: GitHub → `utils/cmd/spec.go` → `data/specs/`
2. **Embedding Generation**: Specs → `utils/cmd/embed.go` → `data/embeddings/`
3. **Validation Request**: User → LLM → MCP Tool → Validator → Vector Search → Response
4. **Metadata Tracking**: Automatic updates to `data/SPEC_METADATA.json` during spec/embed operations

### External Dependencies

#### MCP SDK

- **Current Library**: https://github.com/mark3labs/mcp-go - A community Go SDK for Model Context Protocol
- **Purpose**: Provides the core MCP server implementation, protocol handling, and tool/prompt interfaces
- **Version**: Latest stable version (see go.mod for current version)
- **Migration Plan**: Once the official Go SDK at https://github.com/modelcontextprotocol/go-sdk reaches stable status, the plan is to migrate from the community SDK to ensure long-term compatibility and support

#### OpenAI API Key Requirement

- **Current State**: The server requires an OpenAI API key (`OPENAI_API_KEY`) for generating embeddings during runtime search and validation
- **Purpose**: Used to embed user-provided content for semantic search against pre-generated spec embeddings
- **Configuration**: Set via environment variable in the MCP client configuration

  **Future Plans**:

  - Support for multiple embedding providers (e.g., Anthropic, Cohere, local models)
  - Provider-agnostic embedding interface to reduce dependency on a single service
  - Potential for self-hosted embedding models to eliminate external API requirements

### Project Structure

```text
cmd/
├── factcheck-curl/         # Test client
└── mcp-factcheck-server/   # Main MCP server

data/
├── embeddings/            # Pre-generated embeddings
│   ├── *-fine.json        # Fine-grained embeddings (~230 char chunks)
│   └── *.json             # Regular embeddings (~500 char chunks)
├── specs/                 # Extracted MCP specifications
└── SPEC_METADATA.json     # Automatic tracking of spec versions

embedding/
├── generator.go           # OpenAI embedding generation
└── types.go               # Embedding types and interfaces

internal/
├── embedding/
│   └── vectordb.go        # Vector database with fine-grained fallback
├── integrations/
│   └── arizephoenix/      # Phoenix telemetry implementation
│       ├── config.go      # Phoenix configuration
│       ├── init.go        # Initialization helpers
│       ├── middleware.go  # Phoenix middleware
│       └── provider.go    # Phoenix provider
└── prompts/
    ├── factcheck.go       # Fact-checking prompt logic
    └── templates/         # Internal prompt templates

pkg/
├── prompts/               # MCP prompt templates
│   ├── service.go         # Prompt service implementation
│   └── templates/         # Prompt template files
├── spec/                  # MCP specification tools
│   ├── list.go            # list_spec_versions implementation
│   └── search.go          # search_spec implementation
├── telemetry/             # Clean telemetry abstractions
│   ├── builder.go         # Fluent span builder
│   └── interfaces.go      # Provider, Middleware interfaces
└── validator/             # Content/code validation
    ├── claim_expansion.go # Query expansion for better search
    ├── claim_search.go    # Claim extraction and search enhancement
    ├── code.go            # validate_code implementation
    ├── compound_claims.go # Compound claim decomposition
    ├── content.go         # check_mcp_claim implementation
    ├── format_workflow.go # Step-by-step workflow formatting
    ├── internal_formatter.go # Template-based formatting
    ├── quick_claim.go     # check_mcp_quick_fact implementation
    └── stability.go       # Content stability checking

utils/
├── cmd/                   # CLI tools
│   ├── embed.go           # Embedding generation command
│   ├── main.go            # Specloader CLI entry point
│   └── spec.go            # Spec extraction with metadata tracking
├── metadata/              # Automatic metadata management
│   ├── metadata.go        # Metadata types and operations
│   └── github.go          # GitHub commit hash retrieval
└── specs/                 # Specification processing
    ├── chunking_v2.go     # Fine-grained chunking strategies
    └── loader.go          # GitHub spec extraction
```

## Implementation Details

### Semantic Search Challenge

**Problem Context:**

- Short queries or claims (like "enforces ACLs, rate limits, and provenance") often lack enough context for the embedding model to match them strongly with longer, more detailed spec sections
- Embeddings of short phrases are often "blurry" or less semantically rich
- Spec sections might be much longer and include multiple requirements, examples, and context, making their embeddings drift from short-form queries

**Why This Happens:**

1. **Embedding Models Work Best on Paragraphs**: Most embedding models (including OpenAI's text-embedding-ada-002) are trained for paragraph-level similarity, not ultra-short phrases vs. long context
2. **Vector Distance Is Sensitive**: When you embed "enforces ACLs" and compare to a whole paragraph about security, the resulting similarity may be weak even if both discuss the same topic
3. **Chunk Size Mismatch**: The spec might be chunked into medium/large sections (500+ chars), but claims are often small atomic statements (20-50 chars)

**Specific Challenges Encountered:**

1. **False Negatives**: The system claimed "rate limit is not mentioned in the spec" when the spec actually states "Both parties SHOULD implement rate limiting"
2. **Context Loss**: Short claims like "MCP enforces X" couldn't find spec sections saying "implementations should enforce X"
3. **Compound Claims**: Bullet points with multiple claims (e.g., "enforces ACLs, rate limits, and provenance") needed to be split for accurate validation
   - **Solution**: Implemented compound claim decomposition that splits "X and Y" into separate subclaims for independent validation

### Chunking Strategy

**Approach**: Create smaller chunks (~230 chars) with overlap during spec preprocessing to solve the semantic search challenge.

**Spec Chunking Implementation:**

```go
// Chunking strategies defined in utils/specs/chunking_v2.go
FineGrainedStrategy = ChunkingStrategy{
    Name:            "fine",
    ChunkSize:       230,      // Small chunks for spec documents
    ChunkOverlap:    50,       // Overlap to preserve context
    SplitBySentence: true,     // Natural sentence breaks
    SplitByBullet:   true,     // Bullet points as boundaries
    KeepHeaders:     true,     // Headers stay with content
}
```

**Key Decisions for Spec Chunking:**

1. **Chunk Size**: 230 characters (optimized for 2-3 sentence spec chunks)
2. **Overlap**: 50 characters (preserves context across chunk boundaries)
3. **Sentence Boundaries**: Respects natural language breaks in specifications
4. **Bullet Awareness**: Treats spec bullet points as natural chunk boundaries
5. **Header Preservation**: Keeps spec section headers with their content

**Dual Embedding System**:

- **Regular Embeddings**: Original spec chunks (~500 chars) for backward compatibility
- **Fine Embeddings**: New fine-grained spec chunks (~230 chars) with `-fine` suffix
- **Automatic Fallback**: System searches fine embeddings first, falls back to regular

**Why This Matters**: The fine-grained chunking of specifications enables better matching when the LLM processes short claims or questions. A claim like "MCP enforces rate limits" can now match against a small spec chunk that specifically mentions rate limiting, rather than getting lost in a larger paragraph about general security considerations.

### Spec Maintenance and Updates

**Process**: This is performed manually when updating the draft specification or when a new official MCP version is released. All generated embeddings and metadata are committed to the repository in the `data/` directory.

**When to Update**:

- When the draft specification changes significantly
- When a new official MCP version is released
- Periodically to keep draft spec current

**Update Workflow**:

```bash
# Step 1: Extract specification from GitHub
# This automatically captures the commit hash and updates metadata
./bin/specloader spec --version draft

# Step 2: Generate embeddings (both regular and fine-grained)
# This automatically updates metadata with chunk counts
./bin/specloader embed --version draft
./bin/specloader embed --version draft-fine

# Step 3: Verify metadata was updated
cat data/SPEC_METADATA.json | jq .specs.draft
```

**Automatic Metadata Tracking**:

The `data/SPEC_METADATA.json` file is automatically updated by the tools:

- **During `spec` command**:
  - Captures GitHub commit hash
  - Records extraction timestamp
  - Stores repository and branch/tag information
- **During `embed` command**:
  - Records embedding generation timestamp
  - Stores actual chunk count
  - Tracks chunking strategy used

This ensures complete traceability without manual bookkeeping. The metadata file provides:

- When each spec version was last extracted
- Exact source commit for reproducibility
- Embedding generation details for each strategy
- Easy verification of spec freshness

### Content Chunking (Runtime)

**Tool**: Used exclusively by `check_mcp_claim` (NOT by `check_mcp_quick_fact`)

**When Applied**: Automatically when content exceeds 2000 characters.

**How It Works**:

1. LLM receives large document from user and invokes `check_mcp_claim`
2. Tool detects content length > 2000 chars and enables chunking
3. System splits content into manageable paragraphs
4. Each paragraph is validated independently
5. Results are aggregated into final report

**Important**: This is different from spec chunking. Content chunking handles large user inputs at runtime in the `check_mcp_claim` tool, while spec chunking prepares the MCP specifications during preprocessing.

### Claim Extraction

**Handles**:

- Bullet points (-, \*, •)
- Compound sentences with semicolons
- List patterns ("enforces X, Y, and Z")
- Maintains original claim for reference

### Compound Claim Decomposition

**Purpose**: Improve validation accuracy for claims containing "and" by searching for evidence independently for each subclaim.

**How It Works**:

1. **Detection**: Identifies claims containing " and " as potential compound claims
2. **Decomposition**: Splits into subclaims while preserving subject/verb:
   - "Servers implement validation and timeouts" → 
     - "Servers implement validation"
     - "Servers implement timeouts"
3. **Independent Search**: Each subclaim is searched separately:
   - Generates multiple search queries per subclaim
   - Collects evidence from different spec sections
   - Deduplicates results across queries
4. **Evidence Aggregation**: Combines evidence for all subclaims
5. **Validation**: Claim is accurate only if ALL subclaims have supporting evidence

**Example Flow**:

```text
Original: "MCP supports request validation and timeouts"
    ↓
Decompose: ["MCP supports request validation", "MCP supports timeouts"]
    ↓
Search Each: 
    - Subclaim 1 → Find validation mentions in spec
    - Subclaim 2 → Find timeout mentions in spec
    ↓
Aggregate: Both found → Compound claim is ACCURATE
```

**Benefits**:

- Prevents false negatives when concepts appear in different spec sections
- More thorough evidence collection
- Better handling of multi-part requirements

### Template System

**Internal Template** (not user-facing):

- Embedded in `internal_formatter.go`
- Consistent formatting across all validations
- Easy to modify without changing logic
- Supports conditional sections (best practices, modal verbs)

### Error Handling

**Graceful Degradation**:

1. Try fine embeddings → fallback to regular
2. Try fact-checking → fallback to similarity scoring
3. Try template formatting → fallback to direct formatting

## Trade-offs and Design Decisions

### Alternatives Considered

1. **Large Chunk Strategy (Original Approach)**

   - **Chunk Size**: 500-1000 characters
   - **Pros**: Good context preservation, fewer chunks to manage
   - **Cons**: Poor matching with short queries, missed specific requirements

2. **Keyword-Based Search**

   - **Approach**: Traditional keyword matching before semantic search
   - **Pros**: Exact matches for specific terms
   - **Cons**: Misses semantic relationships, brittle with synonyms

3. **Hybrid Search (Keyword + Semantic)**

   - **Approach**: Combine BM25 with vector search
   - **Pros**: Best of both worlds
   - **Cons**: Complex implementation, tuning required

4. **Query Expansion**

   - **Approach**: Expand short queries with context before embedding
   - **Pros**: Improves semantic richness
   - **Cons**: Risk of drift, requires careful prompt engineering

5. **Fine-Grained Chunking (Chosen)**
   - **Approach**: Create smaller chunks (~230 chars) with overlap
   - **Pros**: Better matching with short queries, maintains context with overlap
   - **Cons**: More chunks to store and search

### Key Trade-offs

#### 1. Matching Accuracy

**Fine-Grained Chunks**:

- ✅ Dramatically improved matching for short queries
- ✅ Found "rate limiting" mentions that were previously missed
- ❌ Slightly reduced context per chunk
- **Mitigation**: 50-char overlap preserves context

#### 2. Performance

**Storage**:

- Regular embeddings: ~500KB per spec version
- Fine embeddings: ~1.5MB per spec version
- **Decision**: 3x storage acceptable for accuracy gains

**Search Latency**:

- More chunks to search through
- **Mitigation**: Still using same top-K (10-15 results)
- **Reality**: OpenAI API latency dominates, not vector search

#### 3. Explainability

**Benefits**:

- Smaller chunks = more precise references
- Easier to show exactly which spec section supports/refutes a claim
- Better alignment between claim size and evidence size

#### 4. Implementation Complexity

**Added Complexity**:

- Dual embedding system
- Fallback logic
- Multiple search strategies

**Simplification**:

- Kept original validation flow intact
- Template-based formatting separates concerns
- Clear tool separation (comprehensive vs. quick)

## Conclusion

The fine-grained chunking strategy successfully addresses the semantic search granularity problem while maintaining system simplicity. By creating specialized tools for different use cases and implementing smart search strategies, the system now accurately validates both quick claims and comprehensive content against MCP specifications.

The key insight: **Match the granularity of the search corpus to the expected query size** - when users ask about specific claims, search against claim-sized chunks.
