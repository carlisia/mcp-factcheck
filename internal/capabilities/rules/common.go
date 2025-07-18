package rules

// CommonValidationRules contains validation rules that apply to both full claim checking
// and quick fact checking. These rules ensure consistent interpretation across all validators.
const CommonValidationRules = `CORE VALIDATION PRINCIPLES:
- A claim is ACCURATE only if it is explicitly supported by the specification provided
- A claim is INACCURATE if:
  * The spec does NOT explicitly state or support it
  * It contradicts what the spec says
  * It makes assertions beyond what the spec states
  * The spec does NOT mention it (silence = MCP doesn't do it)
  * It conflates MCP with other protocols or technologies

MODAL VERB DISTINCTION (CRITICAL):
- "MCP enforces/provides/implements X" means the protocol itself requires X
- "Implementations SHOULD/MAY/MIGHT do X" is a recommendation, NOT enforcement by MCP
- "Implementations MUST do X" is a requirement for implementations, NOT something MCP does
- Therefore:
  * "MCP enforces X" is INACCURATE when spec only says "implementations SHOULD do X"
  * "MCP does not enforce X" is ACCURATE when spec says "implementations SHOULD do X"

NEGATIVE CLAIMS (e.g., "MCP does not X" or "MCP never Y"):
- These are ACCURATE if the spec doesn't mention MCP doing X or Y
- They are also ACCURATE if the spec only mentions implementations SHOULD/MAY/MIGHT do X or Y
- IMPORTANT: If the spec doesn't mention something, that means MCP doesn't do it
- For compound negative claims with "or": If BOTH parts are not done by MCP, the entire claim is ACCURATE
- Example: "MCP never forwards raw model traffic or enforces rate limits" is ACCURATE if:
  * The spec doesn't mention MCP forwarding raw model traffic (silence = doesn't do it)
  * The spec only says implementations SHOULD implement rate limiting (recommendation ≠ enforcement)

EVIDENCE REQUIREMENTS:
- Use ONLY the specification sections provided as evidence
- Do not rely on assumptions or general knowledge
- Focus on what the spec explicitly states, not what might be implied
- Be strict: If the spec doesn't explicitly support a positive claim, mark it as inaccurate`

// ProtocolVsImplementationDistinction clarifies the critical difference between
// what MCP as a protocol does versus what implementations should do.
const ProtocolVsImplementationDistinction = `PROTOCOL vs IMPLEMENTATION:
The MCP specification carefully distinguishes between:
1. What the PROTOCOL provides/enforces (described as "MCP does X")
2. What IMPLEMENTATIONS should/must do (described as "implementations SHOULD/MUST do X")

This distinction is CRITICAL for accuracy:
- If the spec says "implementations SHOULD validate", MCP does NOT validate
- If the spec says "servers MUST authenticate", MCP does NOT authenticate
- Only features explicitly attributed to "MCP" or "the protocol" are things MCP does`
