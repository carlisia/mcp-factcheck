package list_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools/list"
)

const (
	// Test constants for spec versions
	specVersionDraft     = "draft"
	specVersionLatest    = "2025-06-18" // latest version
	specVersionMarch2025 = "2025-03-26"
	specVersionNov2024   = "2024-11-05"

	// Tool names used in formatted output
	toolNameClaims    = "check-mcp-claims"
	toolNameQuickFact = "check-mcp-quick-claim"
	toolNameSearch    = "search-spec"
)

// List versions tool doesn't need to parse args since it has no parameters

func TestListVersions(t *testing.T) {
	tests := []struct {
		name          string
		listFunc      list.ListFunc
		cancelContext bool
		wantErr       bool
		errMsg        string
		wantVersions  []string
		checkVersions bool
	}{
		{
			name:          "context cancelled",
			listFunc:      mockListFunc([]string{specVersionDraft, specVersionLatest}),
			cancelContext: true,
			wantErr:       true,
			errMsg:        "list operation cancelled",
		},
		{
			name:          "successful list",
			listFunc:      mockListFunc([]string{specVersionDraft, specVersionLatest, specVersionNov2024}),
			wantVersions:  []string{specVersionDraft, specVersionLatest, specVersionNov2024},
			checkVersions: true,
		},
		{
			name:     "list error",
			listFunc: mockListFuncError(errors.New("database error")),
			wantErr:  true,
			errMsg:   "database error",
		},
		{
			name:          "empty versions list",
			listFunc:      mockListFunc([]string{}),
			wantVersions:  []string{},
			checkVersions: true,
		},
		{
			name:          "single version",
			listFunc:      mockListFunc([]string{specVersionDraft}),
			wantVersions:  []string{specVersionDraft},
			checkVersions: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			versions, err := list.ListVersions(ctx, tt.listFunc)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListVersions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error message should contain %q, got %q", tt.errMsg, err.Error())
				}
				return
			}
			if tt.checkVersions {
				if !slices.Equal(versions, tt.wantVersions) {
					t.Errorf("got versions %v, want %v", versions, tt.wantVersions)
				}
			}
		})
	}
}

func TestFormatResults(t *testing.T) {
	tests := []struct {
		name            string
		versions        []string
		expectedPhrases []string
	}{
		{
			name:     "multiple versions",
			versions: []string{specVersionDraft, specVersionLatest, specVersionNov2024},
			expectedPhrases: []string{
				"Available MCP specification versions:",
				fmt.Sprintf("- %s", specVersionDraft),
				fmt.Sprintf("- %s", specVersionLatest),
				fmt.Sprintf("- %s", specVersionNov2024),
				fmt.Sprintf("You can use these versions with other tools like %s", toolNameClaims),
			},
		},
		{
			name:     "single version",
			versions: []string{specVersionDraft},
			expectedPhrases: []string{
				"Available MCP specification versions:",
				fmt.Sprintf("- %s", specVersionDraft),
				fmt.Sprintf("You can use these versions with other tools like %s", toolNameClaims),
			},
		},
		{
			name:     "empty versions",
			versions: []string{},
			expectedPhrases: []string{
				"Available MCP specification versions:",
				fmt.Sprintf("You can use these versions with other tools like %s", toolNameClaims),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := list.FormatResults(tt.versions)

			for _, phrase := range tt.expectedPhrases {
				if !strings.Contains(formatted, phrase) {
					t.Errorf("formatted output missing expected phrase: %q", phrase)
				}
			}

			// The formatter uses double newline between sections
			// So the output has sections separated by \n\n
			sections := strings.Split(formatted, "\n\n")
			if len(sections) < 2 {
				t.Errorf("expected at least 2 sections, got %d", len(sections))
			}
		})
	}
}

func TestFormatResultsConsistency(t *testing.T) {
	// Test that formatting is consistent
	versions := []string{specVersionDraft, specVersionLatest, specVersionNov2024}

	// Format multiple times
	result1 := list.FormatResults(versions)
	result2 := list.FormatResults(versions)

	if result1 != result2 {
		t.Error("FormatResults should produce consistent output")
	}

	// Test that order is preserved
	for i, version := range versions {
		expectedLine := "- " + version
		if !strings.Contains(result1, expectedLine) {
			t.Errorf("version %q not found in output", version)
		}
		// Check that versions appear in order
		if i > 0 {
			prevVersion := "- " + versions[i-1]
			prevIdx := strings.Index(result1, prevVersion)
			currIdx := strings.Index(result1, expectedLine)
			if prevIdx >= currIdx {
				t.Errorf("version %q should appear after %q", version, versions[i-1])
			}
		}
	}
}

// Mock functions for testing
func mockListFunc(versions []string) list.ListFunc {
	return func() ([]string, error) {
		return versions, nil
	}
}

func mockListFuncError(err error) list.ListFunc {
	return func() ([]string, error) {
		return nil, err
	}
}
