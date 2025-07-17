// Package tools provides common utilities for MCP tool implementations.
// This file contains the generic ResultFormatter that can be used by any tool
// to build formatted output. Tool-specific formatting logic should be kept
// in each tool's package.
package tools

import (
	"fmt"
	"strings"
)

// ResultFormatter provides a fluent interface for building formatted results.
// It supports both string and structured content output and is designed to be
// used by all MCP tools for consistent formatting.
type ResultFormatter struct {
	sections []string
}

// NewResultFormatter creates a new result formatter
func NewResultFormatter() *ResultFormatter {
	return &ResultFormatter{
		sections: make([]string, 0),
	}
}

// WithConfidence adds a confidence section
func (rf *ResultFormatter) WithConfidence(confidence float64) *ResultFormatter {
	rf.sections = append(rf.sections, fmt.Sprintf("**Confidence**: %.2f", confidence))
	return rf
}

// WithParsedClaims adds a parsed claims section
func (rf *ResultFormatter) WithParsedClaims(claims []string) *ResultFormatter {
	if len(claims) == 0 {
		return rf
	}

	var section strings.Builder
	section.WriteString("## Parsed Claims\n")
	for _, claim := range claims {
		section.WriteString(fmt.Sprintf("- %s\n", claim))
	}

	rf.sections = append(rf.sections, strings.TrimSpace(section.String()))
	return rf
}

// WithIssues adds an issues section
func (rf *ResultFormatter) WithIssues(issues []string) *ResultFormatter {
	if len(issues) == 0 {
		return rf
	}

	var section strings.Builder
	section.WriteString("## Issues Found\n")
	for _, issue := range issues {
		section.WriteString(fmt.Sprintf("- %s\n", issue))
	}

	rf.sections = append(rf.sections, strings.TrimSpace(section.String()))
	return rf
}

// WithSuggestions adds a suggestions section
func (rf *ResultFormatter) WithSuggestions(suggestions []string) *ResultFormatter {
	if len(suggestions) == 0 {
		return rf
	}

	var section strings.Builder
	section.WriteString("## Suggestions\n")
	for _, suggestion := range suggestions {
		section.WriteString(fmt.Sprintf("- %s\n", suggestion))
	}

	rf.sections = append(rf.sections, strings.TrimSpace(section.String()))
	return rf
}

// WithCorrectedVersion adds a corrected version section
func (rf *ResultFormatter) WithCorrectedVersion(corrected string) *ResultFormatter {
	if corrected == "" {
		return rf
	}

	section := fmt.Sprintf("## Corrected Version\n```\n%s\n```", corrected)
	rf.sections = append(rf.sections, section)
	return rf
}

// WithCustomSection adds a custom section with a title and content
func (rf *ResultFormatter) WithCustomSection(title, content string) *ResultFormatter {
	if content == "" {
		return rf
	}

	rf.sections = append(rf.sections, fmt.Sprintf("## %s\n%s", title, content))
	return rf
}

// WithText adds plain text without a section header
func (rf *ResultFormatter) WithText(text string) *ResultFormatter {
	if text != "" {
		rf.sections = append(rf.sections, text)
	}
	return rf
}

// BuildSection returns all sections as a single formatted string
func (rf *ResultFormatter) BuildSection() string {
	return strings.Join(rf.sections, "\n\n")
}

// BuildSections returns individual sections as separate strings
func (rf *ResultFormatter) BuildSections() []string {
	return rf.sections
}
