package prompts

// AccuracyCheckingRulesShort provides a condensed version of accuracy checking rules
// for use in prompt templates. It summarizes the key principles for validating
// claims against MCP specifications.
const AccuracyCheckingRulesShort = `Check each claim against the MCP specification:
- Claims must be explicitly supported by the spec
- Distinguish between "MCP provides" vs "implementations SHOULD"
- Mark unsupported claims as inaccurate
- Be strict—no assumptions beyond the provided spec`

// StylePreservationGuidelines defines the rules for maintaining the original
// content style, tone, and formatting when making corrections or updates.
// These guidelines ensure that validated content remains consistent with
// the author's original intent.
const StylePreservationGuidelines = `- Preserve the original tone, voice, and style of the content when making corrections or suggestions
- Match the level of formality, technicality, and overall style unless otherwise directed
- Maintain the author's intended audience and communication approach
- Keep formatting conventions consistent with the original content`

// SpecificationGuidanceNote provides comprehensive guidance on interpreting
// MCP specification requirement levels and modal verbs (MUST, SHOULD, MAY).
// It helps distinguish between protocol requirements, implementation requirements,
// best practices, and optional features. This constant is used in prompts to
// ensure accurate interpretation of specification language.
const SpecificationGuidanceNote = `Specification Guidance Note:
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