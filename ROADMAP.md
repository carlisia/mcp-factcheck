# MCP Server Roadmap 🚀

## Core MCP Features ⚙️

| Task                                                    | Status         |
| ------------------------------------------------------- | -------------- |
| MCP protocol compliance                                 | ✅ Completed   |
| JSON-RPC 2.0 stdio transport                            | ✅ Completed   |
| Multi-version MCP spec support                          | ✅ Completed   |
| Semantic search using OpenAI embeddings                 | ✅ Completed   |
| Prompt templates system                                 | ✅ Completed   |
| Pre-generated embeddings for all spec versions          | ✅ Completed   |
| Code pattern detection and validation                   | ✅ Completed   |
| Content migration between spec versions (via prompts)   | ✅ Completed   |
| Test client/CLI tool (factcheck-curl)                   | ✅ Completed   |
| Spec extraction utilities (specloader)                  | ✅ Completed   |
| Improved validation accuracy for claims/statements      | 🚧 In Progress |
| Context-aware validation (full statement understanding) | ❌ Planned     |
| Validation explanation/reasoning output                 | ❌ Planned     |
| Confidence scoring improvements                         | ❌ Planned     |
| Support for validating partial/incomplete content       | ❌ Planned     |
| Schema-based code validation                            | ❌ Planned     |
| Language-specific validation (Python, TypeScript, etc.) | ❌ Planned     |
| Batch validation for multiple inputs                    | ❌ Planned     |
| Diff-based validation                                   | ❌ Planned     |
| User-defined custom rulesets                            | ❌ Planned     |
| Fine-tuned model support for specialized validation     | ❌ Planned     |

---

## Infrastructure 📊

| Task                                      | Status       |
| ----------------------------------------- | ------------ |
| Structured logging with Zap               | ✅ Completed |
| OpenTelemetry tracing integration         | ✅ Completed |
| Request tracking and correlation          | ✅ Completed |
| Arize Phoenix telemetry integration       | ✅ Completed |
| In-memory validation result caching       | ❌ Planned   |
| Embedding search performance optimization | ❌ Planned   |
| Support for multiple embedding providers  | ❌ Planned   |

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
| Unit test coverage > 80%                     | ❌ Planned |
| Integration test suite                       | ❌ Planned |
| Code linting with golangci-lint              | ❌ Planned |
| CI pipeline for testing and quality checks   | ❌ Planned |
| Semantic versioning and changelog generation | ❌ Planned |

---

## Documentation 📖

| Task                                             | Status       |
| ------------------------------------------------ | ------------ |
| Initial project documentation and usage examples | ✅ Completed |
| CLAUDE.md development guide                      | ✅ Completed |
| Detailed API documentation                       | ❌ Planned   |
| Contributor onboarding documentation             | ❌ Planned   |
| Example integrations and use cases               | ❌ Planned   |
| Performance benchmarks for validation            | ❌ Planned   |
