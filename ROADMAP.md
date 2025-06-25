# ROADMAP

## **Project Overview**

Transform the current HTTP-based fact-checking prototype into a proper MCP-compliant client and server implementation that validates content about the Model Context Protocol against the official specification.

---

## **📦 MCP Server Implementation**

### **Phase 1: Foundation & Core Functionality**

| **Task**                    | **Status** | **Description**                                 |
| --------------------------- | ---------- | ----------------------------------------------- |
| Basic HTTP server setup     | ✅         | HTTP server running on configurable port        |
| Request/response types      | ✅         | `VerifyRequest` and `VerifyResponse` structures |
| OpenAI integration          | ✅         | GPT-4 client with API key validation            |
| Content validation endpoint | ✅         | `/verify` endpoint accepting JSON requests      |
| Basic input validation      | ✅         | JSON schema validation and request sanitization |
| Environment configuration   | ✅         | PORT and OPENAI_API_KEY environment variables   |

### **Phase 2: Enhanced Validation & Processing**

| **Task**                       | **Status** | **Description**                                       |
| ------------------------------ | ---------- | ----------------------------------------------------- |
| MCP specification loader       | ❌         | Load and parse official MCP spec markdown files       |
| Embedding-based comparison     | ❌         | Use Ada v2 embeddings for semantic content analysis   |
| Structured feedback generation | ❌         | Generate detailed, section-specific feedback          |
| Content type detection         | ❌         | Identify and handle different content formats         |
| MCP spec version selection     | ❌         | Allow users to specify which MCP spec version to validate against |
| Spec version listing           | ❌         | Implement backend support for listing available MCP versions |
| Spec section retrieval         | ❌         | Backend support for retrieving specific MCP spec sections |
| Code example validation        | ❌         | Validate MCP implementation code snippets |
| Fact-checking processor        | ❌         | Replace placeholder with actual AI-powered validation |

### **Phase 3: Server MCP Protocol Compliance**

| **Task**                    | **Status** | **Description**                                    |
| --------------------------- | ---------- | -------------------------------------------------- |
| JSON-RPC 2.0 implementation | ❌         | Replace HTTP with JSON-RPC protocol                |
| MCP tool definition         | ❌         | Define `validate_content` tool with proper schema  |
| Additional MCP tools        | ❌         | Implement `list_spec_versions`, `get_spec_section`, `validate_code_examples` tools |
| Tool parameter validation   | ❌         | Validate tool calls against MCP tool schema        |
| MCP resource advertisement  | ❌         | Advertise available validation tools and resources |
| Stdio transport layer       | ❌         | Implement MCP stdio communication protocol         |

### **Phase 4: Server Features & Observability**

| **Task**              | **Status** | **Description**                                 |
| --------------------- | ---------- | ----------------------------------------------- |
| Comprehensive logging | ❌         | Request/response logging with correlation IDs   |
| Error handling        | ❌         | MCP-compliant error responses and handling      |
| Tool execution gating | ❌         | Auth/consent validation before tool execution   |
| Registry endpoint     | ❌         | `/registry` endpoint describing available tools |
| Rate limiting         | ❌         | Prevent abuse and manage API usage              |
| Health check endpoint | ❌         | Server health and readiness monitoring          |

---

## **🖥️ MCP Client Implementation**

### **Phase 1: Foundation & HTTP Client**

| **Task**                 | **Status** | **Description**                              |
| ------------------------ | ---------- | -------------------------------------------- |
| CLI framework setup      | ✅         | Cobra-based command structure                |
| Basic HTTP client        | ✅         | HTTP POST requests to server endpoint        |
| File input handling      | ✅         | `--file` flag for reading content from files |
| Text input handling      | ✅         | `--blurb` flag for direct text input         |
| Server URL configuration | ✅         | `--server` flag with default localhost:8080  |
| Response parsing         | ✅         | Parse JSON responses and display feedback    |

### **Phase 2: Enhanced Client Features**

| **Task**                     | **Status** | **Description**                               |
| ---------------------------- | ---------- | --------------------------------------------- |
| Content type detection       | ❌         | Auto-detect file types (markdown, text, etc.) |
| Input validation             | ❌         | Validate content before sending to server     |
| MCP spec version flag        | ❌         | `--spec-version` flag to specify target MCP version |
| Structured output formatting | ❌         | Pretty-print feedback in readable format      |
| Configuration file support   | ❌         | Support for client configuration files        |
| Multiple file processing     | ❌         | Batch processing of multiple files            |

### **Phase 3: Client MCP Protocol Compliance**

| **Task**                    | **Status** | **Description**                                |
| --------------------------- | ---------- | ---------------------------------------------- |
| JSON-RPC client             | ❌         | Replace HTTP with JSON-RPC MCP client          |
| MCP tool call structure     | ❌         | Build proper `tools/call` requests             |
| Tool parameter construction | ❌         | Create structured tool parameters              |
| MCP response handling       | ❌         | Parse `tool_result` responses properly         |
| Server capability discovery | ❌         | Query server for available tools and resources |

### **Phase 4: User Experience & Security**

| **Task**                | **Status** | **Description**                               |
| ----------------------- | ---------- | --------------------------------------------- |
| Pre-execution consent   | ❌         | Ask user permission before tool execution     |
| Authentication handling | ❌         | Support for MCP server authentication         |
| Interactive mode        | ❌         | Interactive content validation session        |
| Progress indicators     | ❌         | Show progress for long-running validations    |
| Result caching          | ❌         | Cache validation results for repeated content |
| Offline mode            | ❌         | Basic validation without server connection    |

---

## **🔧 Infrastructure & DevOps**

### **Development & Testing**

| **Task**          | **Status** | **Description**                                   |
| ----------------- | ---------- | ------------------------------------------------- |
| Go modules setup  | ✅         | Proper go.mod with dependencies                   |
| Build system      | ❌         | Makefile or build scripts for binaries            |
| Unit tests        | ❌         | Test coverage for core functionality              |
| Integration tests | ❌         | End-to-end testing of client-server communication |
| CI/CD pipeline    | ❌         | Automated testing and building                    |
| Docker support    | ❌         | Containerization for easy deployment              |

### **Documentation & Examples**

| **Task**             | **Status** | **Description**                          |
| -------------------- | ---------- | ---------------------------------------- |
| README documentation | ✅         | Comprehensive project documentation      |
| License file         | ✅         | LICENSE file added                       |
| Project roadmap      | ✅         | ROADMAP.md with implementation plan      |
| Usage examples       | ❌         | Example content and validation results   |
| API documentation    | ❌         | MCP tool and resource specifications     |
| Developer guide      | ❌         | Contributing and development setup guide |
| MCP compliance guide | ❌         | Documentation of MCP protocol adherence  |

---

## **🎯 Milestones**

### **Milestone 1: Enhanced Prototype** (Current → Functional)

- ✅ Complete Phase 1 tasks for both client and server
- ❌ Implement actual fact-checking processor
- ❌ Add MCP specification loading and comparison

### **Milestone 2: MCP Protocol Migration** (Functional → MCP-Compliant)

- ❌ Replace HTTP with JSON-RPC MCP protocol
- ❌ Implement proper MCP tool definitions and calls
- ❌ Add stdio transport layer

### **Milestone 3: Production Ready** (MCP-Compliant → Production)

- ❌ Add comprehensive testing and documentation
- ❌ Implement security, logging, and monitoring
- ❌ Add advanced features like batch processing and caching

### **Milestone 4: Ecosystem Integration** (Production → MCP Ecosystem)

- ❌ Publish as official MCP validation tool
- ❌ Integration with MCP registries and toolchains
- ❌ Community adoption and feedback incorporation

---

## **📋 Dependencies & Prerequisites**

### **Current Dependencies**

- ✅ Go 1.24.1+
- ✅ OpenAI API access and key
- ✅ Cobra CLI framework
- ✅ Standard HTTP libraries

### **Future Dependencies**

- ❌ MCP Go SDK (or custom JSON-RPC implementation)
- ❌ MCP specification files (markdown)
- ❌ Vector database for embeddings (optional)
- ❌ Testing frameworks (testify, etc.)

---

**Status Legend:**

- ✅ **Completed** - Feature is implemented and working
- ❌ **Not Started** - Feature needs to be implemented
- ⚠️ **In Progress** - Feature is partially implemented or being worked on

