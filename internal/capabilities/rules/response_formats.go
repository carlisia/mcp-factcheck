package rules

// CommonResponseRequirements defines shared requirements for all validation response formats
const CommonResponseRequirements = `EVIDENCE REQUIREMENTS:
- For each concept/claim: provide either a quote from the spec or state its absence
- Quotes must use double quotes: "exact text from spec"
- Absence must be stated as: "The specification doesn't mention [concept]"
- Evidence MUST come before explanations

MODAL VERB EXPLANATIONS:
- For SHOULD/MAY/MIGHT: Clearly state these are "recommendations for implementations, not functions of MCP itself"
- Distinguish between what MCP does vs what implementations should do
- Be explicit about enforcement vs recommendations

ACCURACY DETERMINATION:
- ACCURATE: Only if the claim is fully supported by the specification
- INACCURATE: If the claim is only partially supported or not supported at all
- For negative claims (e.g., "MCP doesn't X"): These are accurate if spec doesn't mention X or only mentions X as SHOULD/MAY/MIGHT

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
1. Start with EXACTLY one of these (per ACCURACY DETERMINATION rules):
   - "✓ ACCURATE" (with checkmark and capitals) if fully supported
   - "✗ INACCURATE" (with X and capitals) if only partially or not supported

2. For EACH key concept in the claim (per EVIDENCE REQUIREMENTS):
   - **[Concept from claim]**: 
     * Quote the spec OR state absence as specified in EVIDENCE REQUIREMENTS

3. After listing all evidence, provide explanation (per EXPLANATION REQUIREMENTS):
   - Apply all rules from MODAL VERB EXPLANATIONS when relevant
   - Apply all rules from EXPLANATION REQUIREMENTS

4. Keep it concise but complete

REMEMBER: All COMMON RESPONSE REQUIREMENTS apply to your response`
