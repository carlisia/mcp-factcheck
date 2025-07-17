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

The MCP Fact-Check server validates content against official Model Context Protocol (MCP) specifications using semantic search with vector embeddings. The system handles various types of validation requests:

- Quick single-claim fact-checking (e.g., "Does MCP enforce rate limits?")
- Comprehensive multi-claim content validation (e.g., documentation, tutorials)
- Bullet-point technical descriptions
- Content migration between MCP specification versions

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
- **Intermediate Output**: Chunked spec JSON files in `data/specs/` (e.g., `draft-spec.json`, `draft-spec-fine.json`)
- **Final Output**: Pre-generated embedding files in `internal/storage/embeddings/` that are embedded into the binary

**Content Chunking** (Runtime): The optional process of dividing user input into smaller sections for validation when the content is very long. This is ONLY used by the `check-mcp-claims` tool:

- **Purpose**: Handle large documents that exceed context limits
- **When**: During validation if content > 2000 chars or explicitly requested
- **Where**: Only in `check-mcp-claims` tool (NOT used by `check-mcp-quick-claim`)
- **Strategy**: Split by paragraphs (`\n\n`), validate each chunk separately
- **Fallback**: If no paragraph breaks exist, processes entire content as single chunk
- **Error Handling**: Failed chunks are tracked but don't stop validation of other chunks
- **Output**: Aggregated validation results with warnings for any failed chunks

**Embedding**: A numerical vector representation of text that captures semantic meaning, enabling similarity comparisons between different pieces of text.

**Modal Verbs**: Requirements level indicators in specifications:

- **MUST/SHALL**: Absolute requirements
- **SHOULD**: Recommended practices that may be ignored with justification
- **MAY**: Optional features

**Best Practices**: SHOULD-level requirements from the MCP specification that represent recommended but not mandatory implementation patterns.

## Tool Design & Prompt Flow

# MCP Fact-Check Design Document (Enhanced)

## Table of Contents

---

## Overview & Objectives

The MCP Fact-Check server performs semantic validation against official Model Context Protocol (MCP) specifications using embeddings and semantic search to:

- Quickly verify individual MCP claims.
- Comprehensively validate multi-claim documentation.
- Guide content migration across MCP specification versions.

---

## Terminology

**Claim:** Atomic MCP-related statement evaluated for accuracy.

| Type           | Example                                          |
| -------------- | ------------------------------------------------ |
| Single Claim   | "MCP enforces rate limits"                       |
| Compound Claim | "MCP enforces ACLs, rate limits, and provenance" |

**Spec Chunking:** Pre-processing of MCP specification documents into searchable segments (\~500 or fine-grained \~230 chars).

**Content Chunking:** Runtime division of long user inputs (>2000 chars) into manageable segments.

**Embedding:** Vector representation capturing semantic meaning for similarity comparisons.

**Modal Verbs:** Requirements in MCP specifications:

- **MUST/SHALL:** Mandatory
- **SHOULD:** Recommended
- **MAY:** Optional

---

## Tool & Prompt Workflow

### Comprehensive Validation (`check-mcp-claims`)

Workflow:

```
User Content → Embedding Generation → Semantic Search → Claim Extraction → Validation → Structured Output
```

Output:

- Claim-specific validation
- Missing best practices
- Modal verb compliance
- Summary confidence score
- Content improvement suggestions

### Quick Validation (`check-mcp-quick-claim`)

Workflow:

```
Single Claim → Aggressive Multi-Strategy Search → Semantic Validation → Concise Verdict (✓/✗)
```

Search Patterns:

```go
"MCP enforces X" → [
    "implementations must X",
    "clients should X",
    "security requirements X"
]
```

### Content Migration (`migrate-mcp-content`)

Workflow:

```
Validate Source Content → Identify Spec Differences → Scope-based Updates → Tone Preservation → Step-by-Step Recommendations
```

Update Scopes:

- **critical_only:** Minimal, breaking changes only
- **enhancement_focused:** Clarity improvements
- **comprehensive:** Full review

---

## System Architecture

### Semantic Validation Engine

Characteristics:

- Vector embeddings for semantic similarity.
- Specialized MCP domain knowledge.
- Intelligent semantic fact-checking and contextual reasoning.

### Core Components

- **MCP Server:** Exposes tools and prompts (`cmd/server/`).
- **Vector Database:** Embeddings management (`internal/storage/vectordb.go`).
- **Embedding Generator:** Runtime embeddings via OpenAI (`internal/embedding/generator.go`).
- **Validation Tools:** Claims and quick claims validation (`internal/tools/validation/`).
- **Spec Tools:** Spec listing and search (`internal/tools/list/`, `internal/tools/search/`).
- **Prompt Service:** Migration prompts (`internal/prompts/migrate/`).
- **Handlers:** MCP protocol handlers (`pkg/mcp/`).

### Data Flow

```
GitHub Spec Extraction → Chunked JSON → Embedding Generation → Embedded Binary → Runtime Validation → User Response
```

### External Dependencies

- **MCP Go SDK**: Currently community-maintained, planned migration to official SDK.
- **OpenAI API**: Runtime embeddings, future plans for multiple providers.

### Project Structure

Structured with clear separation of concerns:

- **cmd/**: Server and CLI utilities
- **data/**: Chunked specs and metadata
- **internal/**: Core logic for vector db, embeddings, integrations (llm, telemetry), tools (spec search/list, validation), user prompts
- **pkg/**: Public interfaces (prompt and tool handlers, LLM clients, structure logging and telemetry)
- **utils/**: Maintenance and utility scripts

---

## Spec Maintenance and Updates

**Process**: This is performed manually when updating the draft specification or when a new official MCP version is released. The chunked specifications are stored in `data/specs/`, embeddings are generated into `internal/storage/embeddings/`, and metadata is tracked in `data/SPEC_METADATA.json`.

**When to Update**:

- When the draft specification changes significantly
- When a new official MCP version is released
- Periodically to keep draft spec current

**Update Workflow**:

```bash
# Step 1: Extract specification from GitHub
# This automatically captures the commit hash and updates metadata
./bin/specloader spec --version draft

# Step 2: Re-chunk for fine-grained embeddings (if needed)
./bin/specloader rechunk --version draft --strategy fine

# Step 3: Generate embeddings (both regular and fine-grained)
# This writes to internal/storage/embeddings/ and updates metadata
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

---

## Implementation Strategies

### Semantic Search Challenge

**Problem Context:**

- Short queries or claims (like "enforces ACLs, rate limits, and provenance") often lack enough context for the embedding model to match them strongly with longer, more detailed spec sections
- Embeddings of short phrases are often "blurry" or less semantically rich
- Spec sections might be much longer and include multiple requirements, examples, and context, making their embeddings drift from short-form queries

**Why This Happens:**

1. **Embedding Models Work Best on Paragraphs**: Most embedding models (including OpenAI's text-embedding-ada-002) are trained for paragraph-level similarity, not ultra-short phrases vs. long context
2. **Vector Distance Is Sensitive**: When you embed "enforces ACLs" and compare to a whole paragraph about security, the resulting similarity may be weak even if both discuss the same topic
3. **Chunk Size Mismatch**: The spec might be chunked into medium/large sections (500+ chars), but claims are often small atomic statements (20-50 chars)

### Chunking Strategy

**Approach**: Create smaller chunks (~230 chars) with overlap during spec preprocessing to solve the semantic search challenge.

Fine-Grained Spec Chunking:

```go
ChunkSize: 230 chars
ChunkOverlap: 50 chars
SplitBySentence: true
KeepHeaders: true
```

- Dual embedding system with automatic fallback (fine → regular).

**Why This Matters**: The fine-grained chunking of specifications enables better matching when the LLM processes short claims or questions. A claim like "MCP enforces rate limits" can now match against a small spec chunk that specifically mentions rate limiting, rather than getting lost in a larger paragraph about general security considerations.

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

### Runtime Content Chunking

**When Applied**: Automatically when content exceeds 2000 characters.

**How It Works**:

1. LLM receives large document from user and invokes `check-mcp-claims`
2. Tool detects content length > 2000 chars and enables chunking
3. System splits content into manageable paragraphs
4. Each paragraph is validated independently
5. Results are aggregated into final report

**Important**: This is different from spec chunking. Content chunking handles large user inputs at runtime in the `check-mcp-claims` tool, while spec chunking prepares the MCP specifications during preprocessing.

### Template & Error Handling

- Centralized templates for uniform validation output.
- Graceful fallbacks for embeddings, validation, and formatting.

---

## Trade-offs & Rationale

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

| Decision                | Pros                                   | Cons                      | Mitigation                  |
| ----------------------- | -------------------------------------- | ------------------------- | --------------------------- |
| Fine-Grained Chunking   | Improved short-query matching accuracy | Increased storage         | Acceptable storage increase |
| Semantic Search         | Rich semantic matching                 | "Blurry" short embeddings | Fine-grained chunking       |
| Compound Claim Handling | Improved validation accuracy           | Slight complexity         | Clear decomposition logic   |

---

## Conclusion

The enhanced MCP Fact-Check design addresses semantic validation challenges by aligning chunk granularity with query size. This results in accurate, contextually relevant validations and comprehensive user guidance, ensuring robust adherence to MCP specifications.

### Tool 1: check-mcp-claims
