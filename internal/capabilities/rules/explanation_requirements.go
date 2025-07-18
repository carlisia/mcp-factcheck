package rules

// InaccuracyExplanation defines how validators should explain why a claim is inaccurate,
// ensuring they provide both negative and positive evidence from the specification.
const InaccuracyExplanation = `Important: When a claim is inaccurate, ALWAYS explain:
1. What the spec DOESN'T say (why the claim is wrong)
2. What the spec DOES say about this topic (if anything) - QUOTE the actual spec text
3. The correct interpretation based on the spec

Example: If content claims "MCP enforces X" but spec says "Implementations SHOULD X":
- State: "The spec doesn't say MCP enforces X"
- Add: "However, the spec DOES say 'Implementations SHOULD X' - this is a recommendation for implementations"
- Clarify: "This means X should be implemented as a best practice, but is not enforced by the protocol"`

// CriticalSearchRequirement emphasizes the importance of actively searching for
// and quoting specification evidence rather than just stating what's missing.
const CriticalSearchRequirement = `CRITICAL: You MUST search for and quote what the spec actually says about each topic, not just state what it doesn't say.`
