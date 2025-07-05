package prompts

// claimExtractionRules defines how to extract claims from content.
// This constant provides detailed rules for parsing and extracting individual
// claims from compound sentences and lists in MCP-related content.
const claimExtractionRules = `- Your output will be considered incorrect if you miss, combine, or summarize any claim or list item, or if you output a claim for only the first & last item in a list but not every item.
- Identify every claim about MCP, even if phrased as a fragment, list item, or implicit subject.
- When encountering a clause with a subject & verb followed by a list (e.g., "enforces voice recognition, database integration, and blockchain validation"), create a separate claim for every item in the list by combining the subject & verb with each item.
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

// accuracyCheckingRules defines how to check claims against the MCP specification.
// It provides strict criteria for determining whether claims are supported by
// the specification, including important distinctions between protocol-level
// and implementation-level requirements.
const accuracyCheckingRules = `- A claim is inaccurate if:
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

// specificationGuidanceNote explains the modal verb distinctions in MCP specifications.
// It provides comprehensive guidance on interpreting requirement levels (MUST, SHOULD, MAY)
// and helps ensure accurate understanding of specification language strength.
const specificationGuidanceNote = `Specification Guidance Note:
MCP specifications may use different levels of requirement language:
- "MUST", "REQUIRED" = hard requirements
- "SHOULD", "RECOMMENDED" = best practices, strong recommendations
- "MUST NOT", "SHOULD NOT" = explicit prohibitions or discouraged practices
- "MAY" = optional features
- Sections labeled "Best Practice" = important guidance

You must treat all these as relevant guidance when comparing content to the specification, even if not marked as a requirement.

Example: If the spec says "Clients SHOULD implement request timeouts" but the content doesn't mention timeouts, flag this as a missing best practice.

Key distinctions:
1. "MCP enforces/provides" → The protocol itself implements this
2. "Implementations MUST" → Required for any compliant implementation
3. "Implementations SHOULD" → Best practice but not mandatory
4. "Implementations MAY" → Optional feature

Important patterns to recognize:
- Sections titled "Best Practices" contain important guidance that should be followed
- Headers containing modal verbs (e.g., "Servers SHOULD implement", "Implementations MUST support") apply that directive to all items listed under them
- Items in a list under such headers inherit the modal verb from the header, even if not explicitly repeated

When validating content:
- Check if mandatory requirements (MUST) are properly conveyed
- Identify missing best practices (SHOULD recommendations)
- Look for "Best Practices" sections that haven't been addressed
- Check for list items under headers with modal verbs (SHOULD, MUST, MAY, etc.)
- Note optional features (MAY) presented as requirements
- Flag any modal verb confusion that changes the meaning

Interpretation Guidance for Language Strength Mismatches:
When comparing content claims to the spec, recognize strength mismatches:
- If content claims "MCP enforces X" but spec only says "Implementations SHOULD X", this is an overstatement
- If content claims "MCP guarantees Y" but spec says "Servers MAY Y", this misrepresents optionality as requirement
- If content uses definitive language ("always", "never", "enforces", "guarantees") but spec uses recommendation language ("SHOULD", "MAY", "RECOMMENDED"), flag this as a language-strength mismatch

For each strength mismatch:
1. Explain the difference between the claim and the spec
2. Suggest accurate rewording that matches the spec's language strength
3. Example correction: "MCP validates all message schemas" → "MCP recommends validating message schemas" (when spec says "SHOULD")

This ensures content accurately represents the protocol's actual requirements vs recommendations vs options.`