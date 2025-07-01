package prompts

// Shared template components for reuse across different prompts

// ClaimExtractionRules defines how to extract claims from content
const ClaimExtractionRules = `- Your output will be considered incorrect if you miss, combine, or summarize any claim or list item, or if you output a claim for only the first & last item in a list but not every item.
- Identify every claim about MCP, even if phrased as a fragment, list item, or implicit subject.
- When encountering a clause with a subject & verb followed by a list (e.g., "enforces ACLs, rate limits, & provenance"), create a separate claim for every item in the list by combining the subject & verb with each item.
    - For example:  
      - Input: "Implements blockchain validation; supports distributed file storage, voice recognition, & NoSQL database integration."
      - Output claims:
        - "MCP implements blockchain validation"
        - "MCP supports distributed file storage"
        - "MCP supports voice recognition"
        - "MCP supports NoSQL database integration"
- For every list (e.g., "supports distributed file storage, voice recognition, & NoSQL database integration"), output a separate claim for each item in the list.
- Do not skip any item—not the first, middle, or last. Skipping a list item is a critical error.
- The output claims must match the count of list items exactly.
- If any item in a list appears in the user content, your output must contain a separate, fully expanded claim for that item, even if the list contains two items, three items, or more.
- If you output claims for the first & last items of a list but not all intermediate items, this is a critical error. Every list item must become its own claim.
- If a claim lacks a subject, assume "MCP" as the subject.
- If a claim lacks a verb but is part of a list or compound sentence, inherit the verb from the prior clause.
    - Example: "supports distributed file storage, voice recognition, & NoSQL database integration" becomes three claims, each beginning with "MCP supports".
- Preserve the original order of claims as found in the text.
- Do not omit any item from lists or compound sentences.
- Treat each bullet or numbered item as a separate claim.
- For each claim, output the fully expanded version with subject & verb included.
- The number of output claims must exactly match the number of items in the input lists, plus any other standalone claims.
- If your output does not contain a separate claim for each item in every user content list, your answer is incorrect, even if the other claims are correct.`

// AccuracyCheckingRules defines how to check claims against the MCP specification
const AccuracyCheckingRules = `- A claim is inaccurate if:
    - The spec does NOT explicitly state that MCP provides, enforces, or implements the claimed feature or behavior.
    - The specification assigns responsibility or permission (e.g., "implementations SHOULD", "servers MAY", or "clients MAY/MUST") but does not state that "MCP" itself provides, enforces, or implements a feature. In this case, mark as inaccurate any claim that assigns that feature directly to MCP.
    - The spec only recommends (e.g., SHOULD, MAY) or suggests an implementation, but does not require or provide it as part of MCP itself.
    - The spec does NOT mention it.
    - It contradicts the spec.
    - It misrepresents MCP's purpose/capabilities.
- Be strict: If the spec doesn't explicitly support a claim, mark it as inaccurate.
- Use only the supplied spec sections—no prior knowledge or assumptions.
- Important distinction: If the specification says "Implementations SHOULD validate message content," but does not say "MCP validates message content," then the claim "MCP validates message content" is inaccurate. The only accurate claim is one that matches the language & subject of the specification exactly.`

// StylePreservationGuidelines defines how to maintain content style
const StylePreservationGuidelines = `- Preserve the original tone, voice, and style of the content when making corrections or suggestions
- Match the level of formality, technicality, and overall style unless otherwise directed
- Maintain the author's intended audience and communication approach
- Keep formatting conventions consistent with the original content`

// ClaimExtractionRulesShort provides a condensed version for reference
const ClaimExtractionRulesShort = `Extract every claim about MCP from the content:
- Split compound sentences and lists into individual claims
- Include implicit subjects (assume "MCP" if missing)
- Preserve all list items—skipping any is a critical error
- Output fully expanded claims with subject and verb`

// AccuracyCheckingRulesShort provides a condensed version for reference
const AccuracyCheckingRulesShort = `Check each claim against the MCP specification:
- Claims must be explicitly supported by the spec
- Distinguish between "MCP provides" vs "implementations SHOULD"
- Mark unsupported claims as inaccurate
- Be strict—no assumptions beyond the provided spec`