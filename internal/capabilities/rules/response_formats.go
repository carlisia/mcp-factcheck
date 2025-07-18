package rules

// ClaimCheckResponseFormat defines the JSON response format for comprehensive claim checking.
// This format includes detailed validation results, missing best practices, and
// language strength issues.
const ClaimCheckResponseFormat = `Return a JSON object with these fields. If any list item from the input is missing in your output, your answer is incorrect:

{
  "claims": [
    {
      "claim": "full expanded claim text",
      "is_accurate": true/false,
      "correction": "suggested rewording that matches the spec's language strength",
      "explanation": "why this is accurate or inaccurate based on spec - quote exact spec text that justifies the decision and explain modal differences (e.g., SHOULD vs MUST vs enforces)"
    }
  ],
  "missing_best_practices": [
    "List of SHOULD requirements from spec not addressed in content"
  ],
  "advisory_language_issues": [
    "Issues where content uses wrong modal verb strength (e.g., claims 'enforces' when spec says 'SHOULD')"
  ],
  "overall_is_accurate": true/false,
  "summary": "Brief summary of findings including accuracy, completeness, modal verb usage, and language strength issues"
}`

// QuickClaimResponseFormat defines the response format for quick single-claim validation.
// This format is simpler and focused on providing a clear verdict with evidence.
const QuickClaimResponseFormat = `RESPONSE FORMAT:
1. Start with either:
   - "✓ ACCURATE" if the claim is fully supported by the spec
   - "✗ INACCURATE" if the claim is not supported or contradicts the spec
2. Quote the relevant spec text that supports or contradicts the claim
3. Provide a brief explanation (1-2 sentences) based on the spec evidence
4. If inaccurate, state what the spec actually says (if anything) about the topic

Be strict and evidence-based. When the spec doesn't explicitly support a claim, mark it as INACCURATE.`
