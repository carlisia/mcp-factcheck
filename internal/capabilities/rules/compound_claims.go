package rules

// CompoundClaimInstructions provides instructions for handling compound claims
// that contain multiple concepts joined by "and" or similar conjunctions.
const CompoundClaimInstructions = `IMPORTANT: When extracting claims that contain "and" or multiple concepts:
- If a claim says "X and Y", evaluate BOTH X and Y separately
- Look for evidence for each part of the compound claim
- The compound claim is accurate only if ALL parts are accurate`

// CompoundEvidenceInstructions provides guidance for using pre-analyzed compound evidence
// when evaluating complex claims with multiple components.
const CompoundEvidenceInstructions = `Use this pre-analysis to inform your evaluation. If evidence was found for all subclaims, the compound claim should be marked as accurate.`

// CompoundClaimEvaluation explains the detailed process for evaluating compound claims.
// It provides examples of how to break down and validate multi-part claims.
const CompoundClaimEvaluation = `COMPOUND CLAIM EVALUATION:
- For claims with "and" or multiple concepts, evaluate each part separately
- Example: "A and B" is accurate if:
  - The spec mentions concept A anywhere (even with different words)
  - The spec mentions concept B anywhere (even with different words)
- Don't require both concepts to appear together in the spec
- Each concept just needs to be supported somewhere in the provided spec sections

Example compound evaluations:
- Content: "authentication and authorization"
  - Check: Does spec mention authentication? ✓
  - Check: Does spec mention authorization? ✓
  - Result: ACCURATE (both concepts found)
  
- Content: "monitoring and logging capabilities"
  - Check: Does spec mention monitoring? ✓
  - Check: Does spec mention logging? ✓
  - Result: ACCURATE (even if mentioned in different sections)`
