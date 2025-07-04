package testutil

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// VersionLister defines the interface for listing versions
type VersionLister interface {
	ListVersions() ([]string, error)
}

// VersionListerTestCase defines a test case for VersionLister handlers
type VersionListerTestCase struct {
	Name           string
	SetupMock      func() VersionLister
	Args           any
	WantErr        bool
	ValidateError  func(t *testing.T, err error)
	ValidateResult func(t *testing.T, result []mcp.Content)
}

// VersionListerHandlerFunc defines the signature for handlers that use VersionLister
type VersionListerHandlerFunc func(ctx context.Context, vl VersionLister, args any) ([]mcp.Content, error)

// RunVersionListerTestCases runs test cases for VersionLister handlers
func RunVersionListerTestCases(t *testing.T, handler VersionListerHandlerFunc, cases []VersionListerTestCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Helper()

			ctx := context.Background()

			// Setup mock
			mock := tc.SetupMock()

			// Execute handler
			result, err := handler(ctx, mock, tc.Args)

			// Check error expectation
			if (err != nil) != tc.WantErr {
				t.Errorf("error = %v, wantErr = %v", err, tc.WantErr)
				return
			}

			// Validate error if present
			if err != nil && tc.ValidateError != nil {
				tc.ValidateError(t, err)
			}

			// Validate result if no error
			if err == nil && tc.ValidateResult != nil {
				tc.ValidateResult(t, result)
			}
		})
	}
}

// MockVersionLister is a test implementation of VersionLister
type MockVersionLister struct {
	ListVersionsFunc func() ([]string, error)
}

func (m *MockVersionLister) ListVersions() ([]string, error) {
	if m.ListVersionsFunc != nil {
		return m.ListVersionsFunc()
	}
	return []string{"draft"}, nil
}

// Compile-time interface conformance check
var _ VersionLister = (*MockVersionLister)(nil)

// NewMockVersionLister creates a MockVersionLister with specified versions
func NewMockVersionLister(versions []string) *MockVersionLister {
	return &MockVersionLister{
		ListVersionsFunc: func() ([]string, error) {
			return versions, nil
		},
	}
}

// NewMockVersionListerWithError creates a MockVersionLister that returns an error
func NewMockVersionListerWithError(err error) *MockVersionLister {
	return &MockVersionLister{
		ListVersionsFunc: func() ([]string, error) {
			return nil, err
		},
	}
}
