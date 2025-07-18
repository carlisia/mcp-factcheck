package rules

// ImportantContextUnderstanding provides guidance on understanding claims in context,
// including initialization flows, protocol capabilities, and implementation patterns.
const ImportantContextUnderstanding = `IMPORTANT CONTEXT UNDERSTANDING:
- When content says "MCP enables servers to expose X" - this IS accurate if the spec shows servers can expose X
- When content says "During initialization, X happens" - this IS accurate if the spec shows X happens in initialization
- When content says "Clients can invoke tools through tools/call" - this IS accurate if the spec shows tools/call messages
- When content says "Tools with JSON Schema definitions" - this IS accurate if spec mentions "inputSchema: JSON Schema"
- Look for the CONCEPT being described, not just exact wording
- Consider protocol flow diagrams, examples, and message formats as valid evidence`

// CriticalExposureRule explains how to validate claims about what MCP "exposes"
// or "provides" based on what the protocol enables servers to do.
const CriticalExposureRule = `CRITICAL: When evaluating "MCP exposes Resources and Tools":
- If spec says "servers expose resources and tools" via MCP, the claim is ACCURATE
- If spec mentions tools have "inputSchema" with JSON Schema, then "with JSON Schema definitions" is ACCURATE
- The protocol enables/allows servers to expose these - that means MCP exposes them`

// InitializationFlowUnderstanding explains how to validate claims about initialization
// processes by checking various parts of the specification.
const InitializationFlowUnderstanding = `INITIALIZATION FLOW UNDERSTANDING:
- Any description of what happens "during initialization" should be checked against:
  - The initialization section of the spec
  - Any protocol flow diagrams showing initialization
  - Request/response structures for initialization
- If the spec shows something happens as part of the initialization process, then saying it happens "during initialization" is ACCURATE

Example initialization checks:
- Content: "During startup, the system establishes connections"
  - If spec shows connection establishment in init sequence → ACCURATE
- Content: "Initial handshake includes version negotiation"  
  - If spec shows version info in initial messages → ACCURATE
- Content: "Configuration is loaded at initialization"
  - If spec shows config loading in startup flow → ACCURATE`

// PatternRecognition provides guidance on recognizing important patterns in
// specifications, particularly around modal verbs and inherited requirements.
const PatternRecognition = `Critical Pattern Recognition:
When analyzing the specification sections, pay special attention to:
1. Headers with modal verbs like "Implementations MUST:", "Servers SHOULD:", "Clients MAY:", "Implementations MUST NOT:", etc.
2. ALL bullet points under such headers inherit that modal verb
3. Example: Under "Implementations SHOULD:", a bullet "- Monitor for sensitive content" means "Implementations SHOULD monitor for sensitive content"
4. Check if the content addresses each bullet point under modal verb headers`
