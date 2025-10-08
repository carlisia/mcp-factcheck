// Package tools provides common utilities for MCP tool implementations.
// This file contains generic validation functions and helpers that can be
// used by all tools. Tool-specific request builders should be kept in each
// tool's package.
package tools

import (
	"fmt"
	"strings"

	"github.com/carlisia/mcp-factcheck/internal/capabilities"
	"github.com/carlisia/mcp-factcheck/internal/security"
)

// ValidationErrors represents multiple validation errors
type ValidationErrors struct {
	Errors []error
}

// Error implements the error interface
func (ve ValidationErrors) Error() string {
	if len(ve.Errors) == 0 {
		return "no validation errors"
	}
	messages := make([]string, len(ve.Errors))
	for i, err := range ve.Errors {
		messages[i] = err.Error()
	}
	return fmt.Sprintf("validation errors: %s", strings.Join(messages, "; "))
}

// Common validation functions

// ValidateSpecVersion validates and normalizes a spec version
func ValidateSpecVersion(version string) (string, error) {
	if version == "" {
		version = capabilities.Latest
	}
	if !capabilities.IsValidSpecVersion(version) {
		return version, fmt.Errorf("invalid spec version: %s, must be one of: %v", version, capabilities.ValidSpecVersions)
	}
	return version, nil
}

// ValidateContentLength validates content length, trims whitespace, and checks for injection attacks
func ValidateContentLength(content string, fieldName string, maxLength int) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return content, fmt.Errorf("%s cannot be empty", fieldName)
	}
	if maxLength > 0 && len(content) > maxLength {
		return content, fmt.Errorf("%s length exceeds maximum of %d characters", fieldName, maxLength)
	}

	detector := security.NewInjectionDetector()
	result := detector.Detect(content)
	if result.IsInjection {
		return "", fmt.Errorf("%s contains invalid patterns: %s", fieldName, result.Reason)
	}

	return content, nil
}

// ValidateTopK validates and bounds the topK parameter
func ValidateTopK(topK int, minValue, maxValue int) (int, error) {
	if topK < minValue {
		return minValue, fmt.Errorf("topK must be at least %d, using minimum value", minValue)
	}
	if topK > maxValue {
		return maxValue, fmt.Errorf("topK cannot exceed %d, using maximum value", maxValue)
	}
	return topK, nil
}
