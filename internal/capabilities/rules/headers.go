package rules

// Headers for different sections of prompts

// ClaimCheckHeader provides the main header for claim extraction and checking
const ClaimCheckHeader = `MCP Claim Extraction & Claim-Checking Prompt`

// ExtractionRulesHeader introduces the extraction rules section
const ExtractionRulesHeader = `Extraction Rules`

// FactCheckingRulesHeader introduces the fact-checking rules section
const FactCheckingRulesHeader = `Fact-Checking Rules

After extracting all claims, check each one against the provided MCP specification sections:`

// CompoundEvidenceHeader introduces compound claim pre-analysis
const CompoundEvidenceHeader = `Compound Claim Pre-Analysis:
The following compound claims have been identified and analyzed for evidence:`

// UserContentHeader introduces the user content section
const UserContentHeader = `USER CONTENT TO CHECK:`

// SpecificationSectionsHeader introduces the spec sections
const SpecificationSectionsHeader = `OFFICIAL MCP SPECIFICATION SECTIONS:`

// ResponseFormatHeader introduces the response format section
const ResponseFormatHeader = `Response Format`

// QuickClaimHeader introduces a quick claim check
const QuickClaimHeader = `CLAIM TO CHECK:`

// QuickClaimSpecHeader introduces spec sections for quick claims
const QuickClaimSpecHeader = `RELEVANT MCP SPECIFICATION SECTIONS:`

// QuickClaimValidationHeader introduces validation rules
const QuickClaimValidationHeader = `VALIDATION RULES:`
