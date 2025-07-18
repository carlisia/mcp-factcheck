package rules

// ParaphrasingRules defines acceptable paraphrasing guidelines for claim validation.
// It helps validators understand when different wording represents the same concept.
const ParaphrasingRules = `PARAPHRASING IS ACCEPTABLE:
- If content says "output sanitization" and spec says "sanitize tool outputs", these mean the same thing
- If content says "access controls" and spec says "access control", these are equivalent
- Natural language variations that preserve meaning should be considered ACCURATE
- Only flag as inaccurate if the meaning is substantively different`

// ConceptualEquivalence explains semantic understanding rules for validating claims.
// It provides examples of how different words can express the same concept.
const ConceptualEquivalence = `CONCEPTUAL EQUIVALENCE RULES:
- Different words can express the same concept - focus on MEANING not exact wording
- Singular/plural variations are equivalent (e.g., "control" vs "controls")
- Verb/noun forms are equivalent when describing the same action (e.g., "validate" vs "validation")
- Process descriptions are equivalent to their outcomes (e.g., "exchange X" = "send and receive X")
- Active/passive voice variations are equivalent (e.g., "server exposes" = "exposed by server")

Examples of conceptual equivalence:
- Content: "performs authentication" ↔ Spec: "authenticate users" ✓ SAME CONCEPT
- Content: "data synchronization" ↔ Spec: "synchronize data between systems" ✓ SAME CONCEPT
- Content: "error handling mechanisms" ↔ Spec: "handle errors appropriately" ✓ SAME CONCEPT
- Content: "secure communication" ↔ Spec: "communications must be secured" ✓ SAME CONCEPT`

// SemanticUnderstanding provides guidance on semantic analysis of claims.
// It helps validators look beyond exact wording to understand conceptual matches.
const SemanticUnderstanding = `SEMANTIC UNDERSTANDING:
- When content describes a process or flow, check if the spec describes that same process even with different words
- When content mentions a concept, check if the spec addresses that concept in ANY form
- Compound statements should be evaluated by checking if EACH component exists somewhere in the spec
- The components don't need to be in the same sentence or paragraph in the spec
- Look for the CONCEPT being described, not just exact wording

Example semantic matches:
- Content: "bi-directional communication" ↔ Spec shows: messages going both directions ✓ ACCURATE
- Content: "structured data format" ↔ Spec shows: JSON schema definitions ✓ ACCURATE  
- Content: "asynchronous processing" ↔ Spec shows: non-blocking operations ✓ ACCURATE
- Content: "extensible architecture" ↔ Spec shows: plugin/extension mechanisms ✓ ACCURATE`
