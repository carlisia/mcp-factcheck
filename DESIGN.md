# MCP Fact-Check Design Document (Enhanced)

## Table of Contents

1. [Overview & Objectives](#overview--objectives)
2. [Terminology](#terminology)
3. [Capabilities](#capabilities)
   - [Tools](#tools)
     - [Comprehensive Content Validation (`check-mcp-claims`)](#comprehensive-content-validation-check-mcp-claims)
     - [Quick Content Validation (`check-mcp-quick-claim`)](#quick-content-validation-check-mcp-quick-claim)
     - [Search (`search-spec`)](#search-search-spec)
     - [List Spec Versions (`list-spec-versions`)](#list-spec-versions-list-spec-versions)
   - [Prompts](#prompts)
     - [Content Migration (`migrate-mcp-content`)](#content-migration-migrate-mcp-content)
4. [System Architecture](#system-architecture)
   - [Semantic Validation Engine](#semantic-validation-engine)
   - [Core Components](#core-components)
   - [Data Flow](#data-flow)
   - [External Dependencies](#external-dependencies)
   - [Project Structure](#project-structure)
5. [Spec Maintenance and Updates](#spec-maintenance-and-updates)
6. [Implementation Strategies](#implementation-strategies)
   - [Semantic Search Challenge](#semantic-search-challenge)
   - [Chunking Strategy](#chunking-strategy)
   - [Claim Extraction](#claim-extraction)
   - [Compound Claim Decomposition](#compound-claim-decomposition)
   - [Runtime Content Chunking](#runtime-content-chunking)
   - [Template & Error Handling](#template--error-handling)
7. [Trade-offs & Rationale](#trade-offs--rationale)
   - [Alternatives Considered](#alternatives-considered)
   - [Key Trade-offs](#key-trade-offs)
8. [Conclusion](#conclusion)

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

## Capabilities

### Tools

#### Comprehensive Content Validation (`check-mcp-claims`)

**Purpose:**

Validates MCP-related content against official specifications to check accuracy and completeness. This tool performs comprehensive multi-claim validation with detailed analysis.

**Use Cases:**

- Validating MCP documentation, tutorials, or blog posts
- Checking technical descriptions of MCP functionality
- Verifying implementation notes or design documents
- Analyzing bullet points or lists describing MCP features
- Any text containing multiple statements about MCP capabilities, design, or behavior

**Tool Parameters:**

| Parameter   | Type    | Required | Default        | Description                                                                                                               |
| ----------- | ------- | -------- | -------------- | ------------------------------------------------------------------------------------------------------------------------- |
| content     | string  | Yes      | -              | The MCP-related content to validate. Maximum 50,000 characters.                                                           |
| specVersion | string  | No       | Latest version | MCP specification version to validate against. Valid options: `draft`, `2025-06-18` (latest), `2025-03-26`, `2024-11-05`. |
| useChunking | boolean | No       | false          | Whether to chunk long content for validation. Automatically enabled for content > 2000 characters.                        |

**Process:**

1. Content Processing: If content > 2000 chars or useChunking=true, splits by paragraphs
2. Embedding Generation: Creates vector embeddings of the content
3. Semantic Search: Finds relevant spec sections using fine-grained embeddings with fallback
4. Claim Extraction: Extracts individual claims from the content
5. Validation: Each claim is validated against retrieved spec sections
6. Aggregation: Results are combined into a comprehensive report

**Tool Output:**

The tool returns a structured validation workflow with multiple sections:

    1. Step 1: Claim Extraction
        - Lists all individual claims found in the content
        - Handles compound claims by decomposing them
    2. Step 2: Validation Results
        - Each claim's validation status (✓ ACCURATE, ✗ INCORRECT, ⚠️ PARTIALLY CORRECT)
        - Supporting evidence from specs
        - Corrections for inaccuracies
    3. Step 3: Missing Best Practices
        - Identifies SHOULD-level requirements not mentioned
        - Provides recommendations for improvement
    4. Step 4: Modal Verb Analysis
        - Checks proper use of MUST/SHALL, SHOULD, MAY
        - Identifies any misrepresented requirement levels
    5. Summary Section
        - Overall confidence score (0.0 to 1.0)
        - Count of valid vs invalid claims
        - Aggregated issues and suggestions
    6. Corrected Version (if applicable)
        - Provides a corrected version of the content with all inaccuracies fixed

**Example Input:**

```json
{
  "content": "MCP servers expose Resources and Tools to clients. They enforce ACLs, rate limits, and provenance tracking. Servers must implement JSON-RPC 2.0.",
  "specVersion": "2025-06-18"
}
```

**Key Differentiators from check-mcp-quick-claim:**

- Handles long-form content (up to 50,000 chars)
- Provides detailed step-by-step analysis
- Extracts and validates multiple claims
- Identifies missing best practices
- Offers comprehensive corrections
- Suitable for documentation validation

#### Quick Content Validation (`check-mcp-quick-claim`)

**Purpose:**

Quick fact-checking for single MCP claims or questions. Optimized for fast validation of concise statements about MCP functionality with clear ✓/✗ verdicts.

**Use Cases:**

- Single-sentence claims like "MCP enforces strict typing"
- Yes/no questions like "Does MCP support bidirectional communication?"
- Quick verification of MCP features and capabilities
- Short questions about MCP behavior or requirements
- Fast fact-checking during conversations

**Tool Parameters:**

| Parameter   | Type   | Required | Default        | Description                                                                                                                  |
| ----------- | ------ | -------- | -------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| claim       | string | Yes      | -              | A single sentence or question about MCP to fact-check. Maximum 50,000 characters (recommended <500 for optimal performance). |
| specVersion | string | No       | Latest version | MCP specification version to check against. Valid options: `draft`, `2025-06-18` (latest), `2025-03-26`, `2024-11-05`.       |

**Process:**

1. Aggressive Multi-Strategy Search:

   - Primary search retrieves 15 results (3x normal) for maximum coverage
   - Falls back to 5 results if insufficient relevant content found
   - Uses pattern transformations to improve search (e.g., "enforces" → "implementations must")

2. Embedding Generation: Creates vector embedding of the claim
3. Semantic Search: Finds relevant spec sections using aggressive strategies
4. Quick Fact-Check: LLM validates claim against retrieved sections
5. Concise Formatting: Returns verdict with explanation

**Tool Output:**

The tool returns a concise validation result with:

6. Verdict Line:

   - ✓ ACCURATE: [original claim] - if the claim is correct
   - ✗ INACCURATE: [original claim] - if the claim is incorrect

7. Explanation:

   - Brief explanation of why the claim is accurate or inaccurate
   - References specific spec sections when relevant

8. Correct Information (if inaccurate):

   - Provides the correct statement based on the specification
   - Example: "Correct information: MCP uses JSON-RPC 2.0, not REST"

9. Confidence Score:

   - Numerical confidence (0.0 to 1.0) in the validation result
   - Only shown when relevant

**Example Input:**

```json
{
  "claim": "Does MCP enforce rate limits?",
  "specVersion": "2025-06-18"
}
```

**Example Output:**

```markdown
✗ INACCURATE: Does MCP enforce rate limits?

No, MCP specifications does not enforce rate limiting, but does state that implementations SHOULD enforce rate limits to prevent abuse. The spec specifically mentions that "Both clients and servers SHOULD implement rate limiting" as a security best practice.

Confidence: 0.95
```

**Key Differentiators from check-mcp-claims:**

- Optimized for speed
- Single claim/question only
- Concise output with clear verdict
- Aggressive search strategy for better coverage
- No content chunking
- No detailed workflow steps.
- Focused on quick yes/no answers

**Performance Characteristics:**

- Uses aggressive search (15 results initially, fallback to 5)
- Prioritizes speed over completeness
- Designed for interactive use during conversations
- Early cancellation support for fast failure

#### Search (search-spec)

**Purpose:**

Search MCP specification using semantic similarity to find specific sections, features, implementation details, and discover related concepts across the spec.

Use Cases:

- Finding specific sections in the MCP specification
- Searching for information about MCP features
- Locating implementation details and requirements
- Discovering related concepts across the spec
- Exploring MCP capabilities by topic
- Quick reference lookup during development

**Tool Parameters:**

| Parameter   | Type    | Required | Default        | Description                                                                                                                         |
| ----------- | ------- | -------- | -------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| query       | string  | Yes      | -              | Search query to find relevant specification content. Maximum 500 characters.                                                        |
| specVersion | string  | No       | Latest version | MCP specification version to search. Valid options: `current` (latest released), `draft`, `2025-06-18`, `2025-03-26`, `2024-11-05`. |
| topK        | integer | No       | 5              | Number of top results to return. Range: 1-20.                                                                                       |

**Process:**

1. Query Validation: Ensures query is within 500 character limit
2. Embedding Generation: Creates vector embedding of the search query
3. Vector Similarity Search: Finds most similar spec sections using cosine similarity
4. Result Ranking: Returns top K results ordered by similarity score
5. Formatting: Presents results with rank and similarity scores

**Tool Output:**

The tool returns formatted search results containing:

1. Header:

   - Search results for '[query]' in MCP [version]:

2. Ranked Results:
   Each result includes:
   - Rank: Position in results (1 = most relevant)
   - Similarity Score: Decimal value (0.0-1.0) indicating relevance
   - Content: The actual spec text that matched

**Example Input:**

```json
{
  "query": "How does MCP handle authentication?",
  "specVersion": "draft",
  "topK": 3
}
```

**Example Output:**

Search results for 'How does MCP handle authentication?' in MCP draft:

```markdown
Rank 1 (similarity: 0.8234):
Authentication in MCP is handled through transport-layer security. The protocol itself does not define authentication mechanisms, as these are expected to be implemented at the transport level (e.g., HTTP
authentication, TLS client certificates).

Rank 2 (similarity: 0.7891):
Security Considerations: MCP implementations MUST ensure proper authentication and authorization at the transport layer. The protocol provides extension points for custom auth schemes but does not mandate
specific mechanisms.

Rank 3 (similarity: 0.7456):
Clients and servers SHOULD implement appropriate security measures including authentication, encryption, and access control based on their deployment context.
```

**Key Characteristics:**

- Uses AI-powered semantic search (not keyword matching)
- Searches against pre-embedded spec chunks
- Returns results ordered by semantic relevance
- Supports all available spec versions
- Fast response time (typically <2 seconds)
- Configurable result count (1-20)

**Performance Notes:**

- Default 5 results balances relevance and brevity
- Maximum 20 results prevents overwhelming output
- Query length limited to 500 chars for optimal embedding quality
- Uses same embedding model as validation tools for consistency

**Differences from Validation Tools:**

- Pure search functionality - no validation or fact-checking
- Returns raw spec content without interpretation
- No LLM analysis of results
- Designed for exploration and reference
- Simpler output format focused on content discovery

#### List Spec Versions ( list-spec-versions)

**Purpose:**

List all available MCP specification versions that can be used with validation and search tools.

**Use Cases:**

- When users ask about available MCP specs or versions
- To discover which specifications are available for validation
- When users need to know which MCP versions they can validate against
- When users are unsure which spec version to use
- Before using other tools that require a spec version parameter
- For understanding the evolution of MCP specifications

**Tool Parameters:**

| Parameter | Type | Required | Default | Description                      |
| --------- | ---- | -------- | ------- | -------------------------------- |
| _(none)_  | -    | -        | -       | This tool accepts no parameters. |

**Process:**

1. Context Check: Verifies the operation hasn't been cancelled
2. List Retrieval: Queries the vector database for available versions
3. Formatting: Formats versions as a bullet list with usage instructions

**Tool Output:**

The tool returns a formatted list containing:

1. Header:

   - Available MCP specification versions:

2. Version List:

   - Each version as a bullet point
   - Includes both official releases and draft versions

3. Usage Instructions:

   - Information about using these versions with other tools

**Example Input:**

```json
{}
```

**Example Output:**

```markdown
Available MCP specification versions:

- draft
- 2025-06-18
- 2025-03-26
- 2024-11-05
```

You can use these versions with other tools like check_mcp_claim, check_mcp_quick_fact, and search_spec.

**Key Characteristics:**

- No parameters required - simple enumeration tool
- Fast operation (typically < 100ms)
- Returns all versions with embedded data available
- Lightweight local operation (no external API calls)
- Results include both release versions and draft
- Versions are returned in the order they appear in the database

**Performance Notes:**

- This is a fast, local operation reading from the vector database
- No embedding generation or search required
- Primarily used during startup or when users need version information
- Context cancellation mainly for clean shutdown scenarios

**Version Types:**

- draft: The latest working draft of the MCP specification
- Date-based versions (e.g., 2025-06-18): Official released versions
- All versions have both regular and fine-grained embeddings available

**Integration with Other Tools:**
The versions returned by this tool can be used as the specVersion parameter in:

- check-mcp-claims - for comprehensive validation
- check-mcp-quick-claim - for quick fact-checking
- search-spec - for semantic search

### Prompts

#### Content Migration (`migrate-mcp-content`)

**Purpose:**

Update MCP documentation, tutorials, or content to align with a target specification version. This prompt guides users through a systematic migration process to ensure their MCP-related content remains accurate when specifications change.

**Use Cases:**

- Updating documentation when a new MCP version is released
- Migrating tutorials from older to newer spec versions
- Ensuring existing content aligns with latest best practices
- Fixing outdated terminology or deprecated features
- Upgrading examples to match current specifications
- Maintaining content accuracy across version changes

**Prompt Parameters:**

| Parameter       | Type   | Required | Default         | Description                                                                                                            |
| --------------- | ------ | -------- | --------------- | ---------------------------------------------------------------------------------------------------------------------- |
| current_version | string | Yes      | -               | Current MCP specification version the content is based on. Options: `2025-06-18`, `2025-03-26`, `2024-11-04`, `draft`. |
| target_version  | string | Yes      | -               | Target MCP specification version to update content for. Options: `2025-06-18`, `2025-03-26`, `2024-11-04`, `draft`.    |
| update_scope    | string | No       | `comprehensive` | Controls the type of updates to make. Options: `comprehensive` (default), `critical_only`, `enhancement_focused`.      |

**Update Scopes Explained:**

1. **comprehensive** (default):

   - All improvements including better explanations and enhanced clarity
   - Full review of content against target specification
   - Updates for accuracy, completeness, and best practices

2. **critical_only**:

   - Only breaking changes and critical fixes
   - Minimal disruption to existing structure
   - Focus on must-fix inaccuracies

3. **enhancement_focused**:

   - Focus on improvements and new features
   - Clarifying explanations within existing content
   - Aligning with target version best practices

**Process Overview:**

The prompt guides through a 6-step migration workflow:

1. **Step 1:** Validate Current Content

   - Uses validate_content tool to check against current version
   - Identifies existing inaccuracies and missing information
   - Establishes baseline quality assessment

2. **Step 2:** Analyze Specification Changes

   - Uses search_spec tool to compare versions
   - Lists relevant terminology and concept changes
   - Highlights changes in requirements (MUST/SHOULD/MAY)

3. **Step 3:** Identify Update Requirements

   - Maps specification changes to content updates needed
   - Groups by: pre-existing issues, required updates, recommended updates
   - Identifies deprecated content and enhancement opportunities

4. **Step 4:** Create Update Strategy

   - Develops high-level plan based on update_scope
   - Strictly scoped to revising only provided content
   - No addition of new sections or features

5. **Step 5:** Generate Updated Content

   - Revises content according to strategy
   - Updates terminology, examples, and version-specific info
   - Preserves original tone and style

6. **Step 6:** Validate Updated Content

   - Uses validate_content to verify accuracy
   - Ensures correct terminology and examples
   - Final quality check against target version

Key Principles:

1. Scope Limitation: Only updates content that was provided - no adding new sections or features
2. Style Preservation: Maintains the tone, style, and formality of the original content
3. Interactive Process: Outputs results at each step and waits for confirmation
4. Accuracy Focus: Follows strict accuracy rules throughout the migration

**Example Input:**

```json
{
  "current_version": "2024-11-04",
  "target_version": "2025-06-18",
  "update_scope": "comprehensive"
}
```

**Output Format:**

Each step produces structured output:

- Step 1: JSON/markdown list of validation findings
- Step 2: Table/bullet list of relevant spec changes
- Step 3: Categorized list of update requirements
- Step 4: High-level update strategy
- Step 5: Revised content with highlighted changes
- Step 6: Validation summary

**Integration with Tools:**

The prompt instructs the LLM to use:

- check-mcp-claims tool for validation
- search_spec tool for comparing specifications

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

- **MCP Go SDK**: Currently community-maintained [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go), planned migration to official [go-sdk](https://github.com/modelcontextprotocol/go-sdk) SDK.
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
