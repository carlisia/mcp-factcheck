# MCP Server Roadmap 🚀

## Core MCP Features ⚙️

| Task                                                                     | Status         |
| ------------------------------------------------------------------------ | -------------- |
| MCP protocol compliance                                                  | ✅ Completed   |
| JSON-RPC 2.0 stdio transport                                             | ✅ Completed   |
| Multi-version MCP spec support                                           | ✅ Completed   |
| Semantic search using OpenAI embeddings                                  | ✅ Completed   |
| Prompt templates system                                                  | ✅ Completed   |
| Pre-generated embeddings for all spec versions                           | ✅ Completed   |
| Code pattern detection (basic)                                           | ✅ Completed   |
| Tool: validate_code (pattern detection completed, schema validation WIP) | ⚠️ In Progress |
| User prompt: Content migration between spec versions                     | ✅ Completed   |
| Test client/CLI tool (factcheck-curl)                                    | ✅ Completed   |
| Spec extraction utilities (specloader)                                   | ✅ Completed   |
| Automatic spec version metadata tracking                                 | ✅ Completed   |
| Improved validation accuracy for claims/statements                       | ✅ Completed   |
| Fine-grained chunking for better semantic search                         | ✅ Completed   |
| Tool: check_mcp_claim (comprehensive validation)                         | ✅ Completed   |
| Tool: check_mcp_quick_fact (quick fact-checking)                         | ✅ Completed   |
| Tool: list_spec_versions                                                 | ✅ Completed   |
| Tool: search_spec                                                        | ✅ Completed   |
| Template-based formatting system                                         | ✅ Completed   |
| Aggressive search strategies for short queries                           | ✅ Completed   |
| Context-aware validation (full statement understanding)                  | ✅ Completed   |
| Validation explanation/reasoning output                                  | ✅ Completed   |
| User prompt: Spec diff to compare curr vs draft (topic-based or full)    | ❌ Planned     |
| Support for validating partial/incomplete content                        | ✅ Completed   |
| Schema-based code validation                                             | ❌ Planned     |
| Migrate to official Go MCP SDK (modelcontextprotocol/go-sdk)             | ❌ Planned     |

---

## Infrastructure 📊

| Task                                      | Status       |
| ----------------------------------------- | ------------ |
| Structured logging with Zap               | ✅ Completed |
| JSON log output with ordered fields       | ✅ Completed |
| MCP message request/response logging      | ✅ Completed |
| OpenTelemetry tracing integration         | ✅ Completed |
| Request tracking and correlation          | ✅ Completed |
| Arize Phoenix telemetry integration       | ✅ Completed |
| In-memory validation result caching       | ❌ Planned   |
| Embedding search performance optimization | ❌ Planned   |
| Support for multiple embedding providers  | ❌ Planned   |
| CI pipeline for testing and quality checks   | ❌ Planned |
| Semantic versioning and changelog generation | ❌ Planned |
| Release process with GitHub releases         | ❌ Planned |

---

## Security & Data Protection 🔒

| Task                                    | Status     |
| --------------------------------------- | ---------- |
| Public security issue reporting channel | ❌ Planned |
| Continuous vulnerability scanning in CI | ❌ Planned |
| Dependency audit & automated updates    | ❌ Planned |

---

## Code Quality & Testing 🧑‍💻

| Task                                         | Status     |
| -------------------------------------------- | ---------- |
| Unit test coverage > 80%        | ❌ Planned |
| Integration test suite          | ❌ Planned |
| Code linting with golangci-lint | ❌ Planned |

---

## Documentation 📖

| Task                                             | Status       |
| ------------------------------------------------ | ------------ |
| Initial project documentation and usage examples | ✅ Completed |
| CLAUDE.md development guide                      | ✅ Completed |
| Technical design documentation (DESIGN.md)       | ✅ Completed |
| Detailed API documentation                       | ❌ Planned   |
| Contributor onboarding documentation             | ❌ Planned   |
| Example integrations and use cases               | ❌ Planned   |
| Performance benchmarks for validation            | ❌ Planned   |
