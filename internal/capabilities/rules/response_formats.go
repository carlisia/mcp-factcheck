package rules

// CommonResponseRequirements defines shared requirements for all validation response formats
const CommonResponseRequirements = `EVIDENCE REQUIREMENTS:
- For each concept/claim: provide either a quote from the spec or state its absence
- Quotes must use double quotes: "exact text from spec"
- Absence must be stated as: "The specification doesn't mention [concept]"
- Evidence MUST come before explanations

MODAL VERB EXPLANATIONS:
- When spec says "implementations SHOULD/MAY/MIGHT X": ALWAYS explain "This is a recommendation for implementations, not a function of MCP itself"
- When spec says "servers/clients SHOULD X": ALWAYS explain "This is a recommendation for the parties, not an enforcement action by MCP"
- Distinguish protocol functions (what MCP does) from implementation recommendations (what parties should do)
- Be explicit: "MCP does not enforce X" when spec only contains SHOULD/MAY/MIGHT for X

ACCURACY DETERMINATION:
- ACCURATE: Only if the claim is fully supported by the specification
- INACCURATE: If the claim is only partially supported or not supported at all
- For negative claims (e.g., "MCP doesn't X"): These are ACCURATE if spec doesn't mention X or only mentions X as SHOULD/MAY/MIGHT
- For compound negative claims with "or" (e.g., "MCP never X or Y"): ACCURATE if spec doesn't mention BOTH X and Y

EXPLANATION REQUIREMENTS:
- Explain what each piece of evidence means in relation to the claim
- Connect all evidence to support your verdict
- For absent features in negative claims: Confirm "MCP doesn't do this since it's not in the specification"`

// ClaimCheckResponseFormat defines the JSON response format for comprehensive claim checking.
// This format includes detailed validation results, missing best practices, and
// language strength issues.
const ClaimCheckResponseFormat = `Follow the COMMON RESPONSE REQUIREMENTS above for evidence, modal verbs, accuracy determination, and explanations.

Return a JSON object with these fields. If any list item from the input is missing in your output, your answer is incorrect:

{
  "claims": [
    {
      "claim": "full expanded claim text",
      "is_accurate": true/false (based on ACCURACY DETERMINATION rules),
      "correction": "suggested rewording that matches the spec's language strength",
      "explanation": "Apply EVIDENCE REQUIREMENTS: quote spec text, then explain per EXPLANATION REQUIREMENTS"
    }
  ],
  "missing_best_practices": [
    "List of SHOULD requirements from spec not addressed in content"
  ],
  "advisory_language_issues": [
    "Issues where content uses wrong modal verb strength (apply MODAL VERB EXPLANATIONS)"
  ],
  "overall_is_accurate": true/false,
  "summary": "Brief summary of findings including accuracy, completeness, modal verb usage, and language strength issues"
}`

// QuickClaimResponseFormat defines the response format for quick single-claim validation.
// This format is simpler and focused on providing a clear verdict with evidence.
const QuickClaimResponseFormat = `Follow the COMMON RESPONSE REQUIREMENTS above for evidence, modal verbs, accuracy determination, and explanations.

RESPONSE FORMAT (YOU MUST FOLLOW EXACTLY):
1. First line MUST be EXACTLY one of these verdicts:
   - "✓ ACCURATE: [original claim]" if fully supported by spec
   - "✗ INACCURATE: [original claim]" if not fully supported

2. Then provide evidence for EACH key concept/assertion in the claim:
   **[Concept]**: [one of the following]
   - "exact quote from spec in double quotes" if spec mentions it
   - The specification doesn't mention [concept] (if absent)

3. After ALL evidence, provide explanation that:
   - Connects the evidence to your verdict
   - For SHOULD/MAY/MIGHT: explicitly states "This is a recommendation for implementations, not a function of MCP itself"
   - For negative claims: confirms the accuracy based on absence or SHOULD/MAY/MIGHT status

4. End with:
   **Confidence**: 0.XX (as decimal)

CRITICAL REMINDERS:
- Negative claims like "MCP does not X" are ACCURATE if spec doesn't mention X or only has SHOULD/MAY/MIGHT for X
- Compound negative claims with "or" (e.g., "MCP never X or Y") are ACCURATE if BOTH parts are not done by MCP
- If spec doesn't mention something, that means MCP doesn't do it (silence = doesn't do it)
- Always distinguish "MCP does X" (protocol function) from "implementations SHOULD do X" (recommendation)
- Evidence MUST come before explanation`
