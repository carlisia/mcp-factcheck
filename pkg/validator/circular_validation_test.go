package validator

import (
	"strings"
	"testing"
)

// TestCircularValidation reproduces the issue where content gets validated repeatedly
func TestCircularValidation(t *testing.T) {
	// The exact content that causes circular validation
	testContent := `The MCP server's design emphasizes its focused role in exposing structured capabilities, responding to client requests, and operating with clear boundaries that enable composability, security, and scalable integration into real-world systems.

- Shares **capability metadata** at session start (version, transports, optional extensions such as ` + "`progress`" + ` or ` + "`cancellation`" + `).
- Exposes **Resources** and **Tools** with clear JSON Schema definitions.
- Executes Tool calls deterministically; streams back progress and completions.
- Never forwards raw model traffic; enforces ACLs, rate limits, and provenance.`

	// Test data structure to track validation cycles
	type ValidationCycle struct {
		Iteration       int
		InputContent    string
		OutputContent   string
		Issues          []string
		IsValid         bool
		Confidence      float64
		ChangesDetected bool
	}

	cycles := []ValidationCycle{}
	maxIterations := 5

	// Simulate multiple validation rounds
	currentContent := testContent
	for i := 0; i < maxIterations; i++ {
		cycle := ValidationCycle{
			Iteration:    i + 1,
			InputContent: currentContent,
		}

		// Mock validation result (this would be replaced with actual validation)
		result := mockValidateContent(currentContent)

		cycle.OutputContent = result.CorrectedVersion
		cycle.Issues = result.Issues
		cycle.IsValid = result.IsValid
		cycle.Confidence = result.Confidence
		cycle.ChangesDetected = currentContent != result.CorrectedVersion

		cycles = append(cycles, cycle)

		// Check for circular pattern
		if i > 0 {
			// Check if we've seen this content before
			for j := 0; j < i; j++ {
				if cycles[j].OutputContent == cycle.OutputContent {
					t.Logf("Circular validation detected at iteration %d, matching iteration %d", i+1, j+1)
					break
				}
			}
		}

		// Use the corrected version for next iteration
		if result.CorrectedVersion != "" && result.CorrectedVersion != currentContent {
			currentContent = result.CorrectedVersion
		} else {
			// No changes, should stop here
			t.Logf("No changes detected at iteration %d", i+1)
			break
		}
	}

	// Analyze the cycles
	t.Log("\n=== Validation Cycle Analysis ===")
	for _, cycle := range cycles {
		t.Logf("\nIteration %d:", cycle.Iteration)
		t.Logf("  Changes Detected: %v", cycle.ChangesDetected)
		t.Logf("  Is Valid: %v", cycle.IsValid)
		t.Logf("  Confidence: %.2f", cycle.Confidence)
		if len(cycle.Issues) > 0 {
			t.Logf("  Issues: %v", cycle.Issues)
		}

		// Show content diff if changed
		if cycle.ChangesDetected {
			t.Logf("  Content Diff:")
			showContentDiff(t, cycle.InputContent, cycle.OutputContent)
		}
	}

	// Check for problematic patterns
	if len(cycles) >= maxIterations {
		t.Error("Validation did not stabilize within expected iterations")
	}

	// Check for flip-flopping
	if len(cycles) >= 3 {
		// Check if content alternates between states
		content1 := normalizeContent(cycles[0].OutputContent)
		content2 := normalizeContent(cycles[1].OutputContent)
		content3 := normalizeContent(cycles[2].OutputContent)

		if content1 == content3 && content1 != content2 {
			t.Error("Content is flip-flopping between two states")
		}
	}
}

// mockValidateContent simulates the validation behavior that causes loops
func mockValidateContent(content string) ValidationResult {
	// This mock simulates the specific issue with "enforces ACLs, rate limits, and provenance"
	result := ValidationResult{
		IsValid:     true,
		Confidence:  0.85,
		SpecVersion: "draft",
	}

	// Simulate the problematic validation logic
	if strings.Contains(content, "enforces ACLs, rate limits, and provenance") {
		// Sometimes the validator thinks this should be "implements" or "supports"
		// This simulates inconsistent behavior
		if hashString(content)%2 == 0 {
			result.IsValid = false
			result.Issues = []string{"'enforces' might be too strong - spec says 'SHOULD implement'"}
			result.CorrectedVersion = strings.Replace(content,
				"enforces ACLs, rate limits, and provenance",
				"implements ACLs, rate limits, and provenance",
				1)
		}
	} else if strings.Contains(content, "implements ACLs, rate limits, and provenance") {
		// And sometimes it thinks it should be "enforces"
		if hashString(content)%2 == 1 {
			result.IsValid = false
			result.Issues = []string{"Content suggests enforcement which aligns with security requirements"}
			result.CorrectedVersion = strings.Replace(content,
				"implements ACLs, rate limits, and provenance",
				"enforces ACLs, rate limits, and provenance",
				1)
		}
	}

	// If no corrections, content is considered valid
	if result.CorrectedVersion == "" {
		result.IsValid = true
		result.CorrectedVersion = content
	}

	return result
}

// showContentDiff displays the differences between two content strings
func showContentDiff(t *testing.T, original, modified string) {
	origLines := strings.Split(original, "\n")
	modLines := strings.Split(modified, "\n")

	for i := 0; i < len(origLines) || i < len(modLines); i++ {
		if i >= len(origLines) {
			t.Logf("    + %s", modLines[i])
		} else if i >= len(modLines) {
			t.Logf("    - %s", origLines[i])
		} else if origLines[i] != modLines[i] {
			t.Logf("    - %s", origLines[i])
			t.Logf("    + %s", modLines[i])
		}
	}
}

// hashString creates a simple hash for deterministic mock behavior
func hashString(s string) int {
	hash := 0
	for _, ch := range s {
		hash = hash*31 + int(ch)
	}
	return hash
}

// TestStabilityChecker tests the stability checker functionality
func TestStabilityChecker(t *testing.T) {
	checker := NewContentStabilityChecker()

	tests := []struct {
		name         string
		original     string
		validated    string
		expectStable bool
		expectLoop   bool
	}{
		{
			name:         "Identical content",
			original:     "MCP enforces rate limits",
			validated:    "MCP enforces rate limits",
			expectStable: true,
			expectLoop:   false,
		},
		{
			name:         "Whitespace differences",
			original:     "MCP  enforces   rate limits",
			validated:    "MCP enforces rate limits",
			expectStable: true,
			expectLoop:   false,
		},
		{
			name:         "Bullet style differences",
			original:     "• Item one\n* Item two",
			validated:    "- Item one\n- Item two",
			expectStable: true,
			expectLoop:   false,
		},
		{
			name:         "Actual content change",
			original:     "MCP enforces rate limits",
			validated:    "MCP implements rate limits",
			expectStable: false,
			expectLoop:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.CheckStability(tt.original, tt.validated)

			if result.IsStable != tt.expectStable {
				t.Errorf("Expected IsStable=%v, got %v", tt.expectStable, result.IsStable)
			}

			if result.IsInLoop != tt.expectLoop {
				t.Errorf("Expected IsInLoop=%v, got %v", tt.expectLoop, result.IsInLoop)
			}
		})
	}
}

// TestLoopDetection tests that the stability checker can detect validation loops
func TestLoopDetection(t *testing.T) {
	checker := NewContentStabilityChecker()

	// Simulate a validation loop: A -> B -> C -> B -> C -> B
	contents := []string{
		"Original content",
		"First modification",
		"Second modification",
		"First modification", // Loop starts here
		"Second modification",
		"First modification",
	}

	for i, content := range contents {
		if i == 0 {
			continue
		}

		result := checker.CheckStability(contents[i-1], content)

		if i < 3 {
			// Before the loop starts
			if result.IsInLoop {
				t.Errorf("Iteration %d: Expected no loop, but loop detected", i)
			}
		} else if i >= 3 {
			// Loop should be detected
			if !result.IsInLoop {
				t.Errorf("Iteration %d: Expected loop detection, but none found", i)
			}

			// Check loop length
			expectedLoopLength := 2 // alternating between two states
			if i >= 4 && result.LoopLength != expectedLoopLength {
				t.Errorf("Iteration %d: Expected loop length %d, got %d",
					i, expectedLoopLength, result.LoopLength)
			}
		}
	}
}
