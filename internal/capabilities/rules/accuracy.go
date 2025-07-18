package rules

// AccuracyChecking defines comprehensive rules for checking claims against the MCP specification.
// It builds on CommonValidationRules and adds specific guidance for full claim validation.
var AccuracyChecking = CommonValidationRules + `

ADDITIONAL CHECKS FOR COMPREHENSIVE VALIDATION:
- Missing best practices (SHOULD requirements not mentioned in content)
- Ignored advisory language (MAY options not acknowledged)
- Incomplete requirements (partial MUST/MUST NOT coverage)
- Check if content uses appropriate modal verbs matching the spec's language strength`

// AccuracyCheckingShort provides a condensed version of accuracy checking rules
// for use in prompt templates. It summarizes the key principles for validating
// claims against MCP specifications.
const AccuracyCheckingShort = `Check each claim against the MCP specification:
- Claims must be explicitly supported by the spec
- Distinguish between "MCP provides" vs "implementations SHOULD"
- Mark unsupported claims as inaccurate
- Be strict—no assumptions beyond the provided spec`

// QuickClaimValidation provides validation rules specifically for quick fact-checking
// of single claims. It uses the common rules with additional focus on single claim validation.
var QuickClaimValidation = CommonValidationRules + `

QUICK CLAIM SPECIFIC GUIDANCE:
- Focus on validating the single claim provided
- Provide clear, concise explanations
- Always quote relevant spec text or note its absence
- Be definitive in your verdict - either ACCURATE or INACCURATE

VERDICT DETERMINATION:
- For pure negative claims: "MCP never/doesn't X" → ACCURATE if spec doesn't mention X
- For compound negative with "or": "MCP never X or Y" → ACCURATE if spec doesn't mention BOTH X and Y
- For mixed claims with "and": "MCP never X and enforces Y" → ACCURATE only if BOTH parts are accurate
- If spec mentions something as "SHOULD/MAY/MIGHT", always quote it and explain it's a recommendation, not enforcement
- When spec mentions rate limiting: Quote "Both parties SHOULD implement rate limiting" if found

EVIDENCE FORMAT EXAMPLES:
For any claim about X:
- If spec mentions X: **X**: "quote from spec about X"
- If spec doesn't mention X: **X**: The specification doesn't mention X

REMEMBER: Always provide the quote or absence statement BEFORE explaining what it means`
