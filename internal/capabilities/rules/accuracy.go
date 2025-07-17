package rules

// accuracyChecking defines how to check claims against the MCP specification.
// It provides strict criteria for determining whether claims are supported by
// the specification, including important distinctions between protocol-level
// and implementation-level requirements.
const accuracyChecking = `- A claim is inaccurate if:
    - The spec does NOT explicitly state that MCP provides, enforces, or implements the claimed feature or behavior.
    - The specification assigns responsibility or permission (e.g., "implementations SHOULD", "servers MAY", or "clients MAY/MUST") but does not state that "MCP" itself provides, enforces, or implements a feature. In this case, mark as inaccurate any claim that assigns that feature directly to MCP.
    - The spec only recommends (e.g., SHOULD, MAY) or suggests an implementation, but does not require or provide it as part of MCP itself.
    - The spec does NOT mention it.
    - It contradicts the spec.
    - It misrepresents MCP's purpose/capabilities.
- Be strict: If the spec doesn't explicitly support a claim, mark it as inaccurate.
- Use only the supplied spec sections—no prior knowledge or assumptions.
- Important distinction: If the specification says "Implementations SHOULD validate message content," but does not say "MCP validates message content," then the claim "MCP validates message content" is inaccurate. The only accurate claim is one that matches the language & subject of the specification exactly.
- Additionally, check for:
    - Missing best practices (SHOULD requirements not mentioned)
    - Ignored advisory language (MAY options not acknowledged)
    - Incomplete requirements (partial MUST/MUST NOT coverage)`

// AccuracyCheckingShort provides a condensed version of accuracy checking rules
// for use in prompt templates. It summarizes the key principles for validating
// claims against MCP specifications.
const AccuracyCheckingShort = `Check each claim against the MCP specification:
- Claims must be explicitly supported by the spec
- Distinguish between "MCP provides" vs "implementations SHOULD"
- Mark unsupported claims as inaccurate
- Be strict—no assumptions beyond the provided spec`