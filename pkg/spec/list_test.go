package spec

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/carlisia/mcp-factcheck/testutil"
	"github.com/mark3labs/mcp-go/mcp"
)

// Compile-time interface conformance check
var _ VersionLister = (*mockVersionLister)(nil)

// mockVersionLister is a test implementation of VersionLister interface
type mockVersionLister struct {
	listFunc func() ([]string, error)
	versions []string
	err      error
}

func (m *mockVersionLister) ListVersions() ([]string, error) {
	// Optional trace logging for debugging - uncomment when needed
	// fmt.Printf("[mockListVersions] called\n")

	if m.listFunc != nil {
		return m.listFunc()
	}
	return m.versions, m.err
}


func TestGetListSpecVersionsTool(t *testing.T) {
	tool := GetListSpecVersionsTool()

	// Test tool properties
	if tool.Name != ListSpecVersionsToolName {
		t.Errorf("Expected tool name '%s', got '%s'", ListSpecVersionsToolName, tool.Name)
	}

	if tool.Description == "" {
		t.Error("Tool description should not be empty")
	}

	// Verify description contains key phrases that help LLMs understand when to use this tool
	expectedPhrases := []string{"MCP spec", "versions", "available"}
	description := strings.ToLower(tool.Description)
	for _, phrase := range expectedPhrases {
		if !strings.Contains(description, strings.ToLower(phrase)) {
			t.Errorf("Tool description should mention '%s' for better LLM understanding", phrase)
		}
	}
}

func TestHandleListSpecVersions(t *testing.T) {
	// Adapter function to work with local VersionLister interface
	handler := func(ctx context.Context, vl testutil.VersionLister, args any) ([]mcp.Content, error) {
		// Convert testutil.VersionLister to local VersionLister
		// Since both have the same method signature, we can use a simple adapter
		localVL := versionListerAdapter{vl}
		return HandleListSpecVersions(ctx, localVL, args)
	}

	testCases := []testutil.VersionListerTestCase{
		{
			Name: "standard MCP versions",
			SetupMock: func() testutil.VersionLister {
				return testutil.NewMockVersionLister([]string{"draft", "2025-06-18", "2025-03-26", "2024-11-05"})
			},
			WantErr: false,
			ValidateResult: func(t *testing.T, result []mcp.Content) {
				testutil.AssertContentCount(t, result, 5) // header + 4 versions
				testutil.AssertAllTextContent(t, result)
				testutil.AssertTextContains(t, result, "Available MCP specification versions:")
				testutil.AssertTextContainsAny(t, result, "draft", "2025-06-18", "2025-03-26", "2024-11-05")
			},
		},
		{
			Name: "empty version list",
			SetupMock: func() testutil.VersionLister {
				return testutil.NewMockVersionLister([]string{})
			},
			WantErr: false,
			ValidateResult: func(t *testing.T, result []mcp.Content) {
				testutil.AssertContentCount(t, result, 1) // just header
				testutil.AssertTextContains(t, result, "Available MCP specification versions:")
			},
		},
		{
			Name: "database error",
			SetupMock: func() testutil.VersionLister {
				return testutil.NewMockVersionListerWithError(errors.New("connection timeout"))
			},
			WantErr: true,
			ValidateError: func(t *testing.T, err error) {
				testutil.AssertErrorContains(t, err, "failed to list spec versions")
				testutil.AssertErrorContains(t, err, "connection timeout")
			},
		},
		{
			Name: "context cancellation",
			SetupMock: func() testutil.VersionLister {
				return testutil.NewMockVersionListerWithError(context.Canceled)
			},
			WantErr: true,
			ValidateError: func(t *testing.T, err error) {
				if !errors.Is(err, context.Canceled) {
					t.Errorf("Expected context.Canceled, got %v", err)
				}
			},
		},
		{
			Name: "special characters in versions",
			SetupMock: func() testutil.VersionLister {
				return testutil.NewMockVersionLister([]string{
					"v1.0.0-alpha.1",
					"2025-06-18+build.456",
					"draft/experimental",
					"你好-version",
				})
			},
			WantErr: false,
			ValidateResult: func(t *testing.T, result []mcp.Content) {
				testutil.AssertContentCount(t, result, 5) // header + 4 versions
				testutil.AssertTextContains(t, result, "v1.0.0-alpha.1")
				testutil.AssertTextContains(t, result, "2025-06-18+build.456")
				testutil.AssertTextContains(t, result, "draft/experimental")
				testutil.AssertTextContains(t, result, "你好-version")
			},
		},
		{
			Name: "nil versions list treated as empty",
			SetupMock: func() testutil.VersionLister {
				return &testutil.MockVersionLister{
					ListVersionsFunc: func() ([]string, error) {
						return nil, nil // nil slice
					},
				}
			},
			WantErr: false,
			ValidateResult: func(t *testing.T, result []mcp.Content) {
				testutil.AssertContentCount(t, result, 1) // just header
			},
		},
		{
			Name: "very long version names",
			SetupMock: func() testutil.VersionLister {
				longVersion := "v1.0.0-this-is-a-very-long-version-name-that-might-cause-display-issues-in-some-interfaces-but-should-still-work-correctly"
				return testutil.NewMockVersionLister([]string{longVersion, "short"})
			},
			WantErr: false,
			ValidateResult: func(t *testing.T, result []mcp.Content) {
				testutil.AssertContentCount(t, result, 3) // header + 2 versions
				testutil.AssertTextContains(t, result, "this-is-a-very-long-version-name")
				testutil.AssertTextContains(t, result, "short")
			},
		},
		{
			Name: "duplicate versions handled gracefully",
			SetupMock: func() testutil.VersionLister {
				return testutil.NewMockVersionLister([]string{"draft", "draft", "2025-06-18", "draft"})
			},
			WantErr: false,
			ValidateResult: func(t *testing.T, result []mcp.Content) {
				testutil.AssertContentCount(t, result, 5) // header + 4 versions (including duplicates)
				testutil.AssertAllTextContent(t, result)
				// Skip header when counting version occurrences
				testutil.AssertTextOccurrenceCount(t, result[1:], "- draft\n", 3)
			},
		},
		{
			Name: "versions returned in original order",
			SetupMock: func() testutil.VersionLister {
				return testutil.NewMockVersionLister([]string{"2024-11-05", "draft", "2025-06-18", "2025-03-26"})
			},
			WantErr: false,
			ValidateResult: func(t *testing.T, result []mcp.Content) {
				testutil.AssertContentCount(t, result, 5) // header + 4 versions
				// Verify order is preserved
				if len(result) >= 5 {
					testutil.AssertTextContains(t, []mcp.Content{result[1]}, "2024-11-05")
					testutil.AssertTextContains(t, []mcp.Content{result[2]}, "draft")
					testutil.AssertTextContains(t, []mcp.Content{result[3]}, "2025-06-18")
					testutil.AssertTextContains(t, []mcp.Content{result[4]}, "2025-03-26")
				}
			},
		},
	}

	testutil.RunVersionListerTestCases(t, handler, testCases)
}

// versionListerAdapter adapts testutil.VersionLister to local VersionLister interface
type versionListerAdapter struct {
	testutil.VersionLister
}
