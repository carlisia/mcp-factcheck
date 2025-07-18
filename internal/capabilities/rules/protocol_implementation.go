package rules

// ProtocolVsImplementation explains the distinction between protocol-level claims
// and implementation-level requirements in MCP specifications.
const ProtocolVsImplementation = `PROTOCOL vs IMPLEMENTATION:
- "MCP provides a way to..." → Check if the protocol specification shows this capability
- "MCP allows servers to..." → Check if servers CAN do this according to the spec
- "MCP enables clients to..." → Check if clients CAN do this according to the spec
- These are DIFFERENT from "MCP enforces" or "MCP requires" which imply mandatory behavior`

// ImplementationRecommendations clarifies how to handle implementation recommendations
// when they appear in user content, distinguishing formal from informal language.
const ImplementationRecommendations = `CRITICAL: Implementation recommendations ARE part of the spec:
- If spec says "Implementations SHOULD X", then "implementations should X" is ACCURATE
- If spec says "Servers SHOULD Y", then "servers should Y" is ACCURATE  
- Do NOT flag lowercase "should" in content if the recommendation itself is accurate`

// KeyDistinction clarifies the critical difference between what MCP as a protocol does
// versus what implementations, servers, or clients are required or recommended to do.
const KeyDistinction = `Key distinction: When content claims "MCP enforces X" or "MCP does X", check if the spec says:
- "Implementations SHOULD X" → This means it's a recommendation for implementations, NOT something MCP enforces
- "Servers MUST X" or "Clients MUST X" → These are requirements for implementations, NOT features of the protocol itself
- Only if the spec says "MCP does X" or "The protocol enforces X" can you say MCP itself does something`

// CaseSensitivityNote explains how to handle RFC 2119 keywords when they appear
// in different cases in user content versus specifications.
const CaseSensitivityNote = `IMPORTANT: Case sensitivity matters for RFC 2119 keywords:
- "Implementations SHOULD" (uppercase) is the formal requirement level
- "Implementations should" (lowercase) is acceptable in user content as it conveys the same meaning
- Do NOT flag lowercase "should" as incorrect if the requirement level is accurate`
