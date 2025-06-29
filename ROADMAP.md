# ROADMAP

## **Project Overview**

Transform the HTTP-based fact-checking prototype into a full MCP-compliant server implementation that validates content about the Model Context Protocol against the official specification.

---

## **✅ Completed Features**

### **MCP Server Implementation**

- ✅ Full MCP server using the mark3labs/mcp-go library
- ✅ JSON-RPC 2.0 stdio transport implementation
- ✅ Four MCP tools: validate_content, validate_code, search_spec, list_spec_versions
- ✅ Support for multiple MCP spec versions (draft, 2025-06-18, 2025-03-26, 2024-11-05)
- ✅ Semantic search using OpenAI embeddings
- ✅ Tool interaction monitoring and visualization

### **Infrastructure & Tools**

- ✅ Spec extraction from GitHub repositories
- ✅ Embedding generation utilities
- ✅ Vector database for semantic search (json files for now)
- ✅ Test client for MCP server validation
- ✅ Comprehensive project documentation

### **Observability & Monitoring**

- ✅ OpenTelemetry tracing with Arize Phoenix integration
- ✅ Structured logging with Zap (JSON format)
- ✅ Request ID tracking and correlation
- ✅ Clean telemetry architecture with abstraction layers
- ✅ Performance and similarity score tracking

---

## **🚧 Future Enhancements**

### **Phase 1: Content Chunking & Long-Form Validation** ✅ COMPLETED

| **Task**                  | **Status** | **Description**                                                    |
| ------------------------- | ---------- | ------------------------------------------------------------------ |
| Content chunking strategy | ✅         | Split long-form content into logical chunks (paragraphs, sections) |
| Chunk-level validation    | ✅         | Validate each chunk individually against MCP specs                 |
| Enhanced coverage mapping | ✅         | Detailed ValidationError types with context and suggestions        |
| Progressive processing    | ✅         | Request ID tracking and structured logging for monitoring          |

### **Phase 2: Code Validation Improvements**

| **Task**                   | **Status** | **Description**                                                  |
| -------------------------- | ---------- | ---------------------------------------------------------------- |
| Schema-based validation    | ❌         | Validate code against MCP JSON schemas rather than documentation |
| Language-specific patterns | ❌         | Add pattern detection for Python, TypeScript, etc.               |
| Implementation examples    | ❌         | Return working code examples for common patterns                 |
| Error recovery suggestions | ❌         | Provide specific fixes for detected issues                       |

### **Phase 3: Enhanced Features**

| **Task**              | **Status** | **Description**                                |
| --------------------- | ---------- | ---------------------------------------------- |
| Batch validation      | ❌         | Validate multiple files/content in one request |
| Diff-based validation | ❌         | Validate changes between versions              |
| Custom rule sets      | ❌         | Allow users to define validation rules         |
| Validation reports    | ❌         | Generate detailed validation reports           |
| CI/CD integration     | ❌         | GitHub Actions for automated validation        |

### **Phase 4: Advanced Capabilities**

| **Task**                 | **Status** | **Description**                                  |
| ------------------------ | ---------- | ------------------------------------------------ |
| Fine-tuned models        | ❌         | Train specialized models for MCP validation      |
| Multi-language support   | ❌         | Validate content in languages other than English |
| MCP registry integration | ❌         | Integrate with an MCP tool registry              |

---

## **📋 Technical Debt & Improvements**

### **Code Quality**

- ❌ Add comprehensive test coverage
- ✅ Implement proper error handling throughout
- ✅ Add structured logging with levels
- ❌ Create integration test suite

### **Performance**

- ❌ Implement caching for repeated validations
- ❌ Optimize embedding search algorithms
- ❌ Implement request queuing and batching?

### **Configuration & Flexibility**

- ❌ Support multiple embedding model options (OpenAI, local models, etc.)
- ❌ Configurable model parameters and providers
- ❌ Runtime model switching capabilities?

### **Security**

- ❌ Add rate limiting per API key
- ❌ Implement request validation and sanitization
- ❌ Add authentication for debug interface
- ❌ Restrict debug server to localhost only

---

## **🎯 Milestones**

### **Milestone 1: Basic Functionality** ✅ COMPLETED

- ✅ Full MCP protocol implementation
- ✅ All core validation tools working
- ✅ Debug interface for development
- ✅ Documentation and examples

### **Milestone 2: Enhanced Validation** 🚧 IN PROGRESS

- ❌ Schema-based code validation
- ✅ Improved error messages and suggestions
- ✅ Performance optimizations (request tracking, structured logging)
- ❌ Test coverage > 80%

### **Milestone 3: Automation**

- ❌ Batch processing capabilities
- ❌ CI/CD integrations

### **Milestone 4: Ecosystem Integration**

- ❌ MCP registry listing
- ❌ Community contribution framework

---

**Status Legend:**

- ✅ **Completed** - Feature is implemented and working
- ❌ **Planned** - Feature is planned for future implementation
- 🚧 **In Progress** - Feature is currently being developed
