package metadata

import (
	"context"
	"testing"
)

func TestGetBranchOrTag_Draft(t *testing.T) {
	// Test the special case for "draft" version
	ref, refType, err := GetBranchOrTag(context.Background(), "any-owner", "any-repo", "draft")

	if err != nil {
		t.Errorf("GetBranchOrTag() for draft returned error: %v", err)
	}

	if ref != "main" {
		t.Errorf("Expected ref 'main' for draft, got '%s'", ref)
	}

	if refType != "branch" {
		t.Errorf("Expected refType 'branch' for draft, got '%s'", refType)
	}
}

// Note: Testing GetBranchOrTag and GetLatestCommitHash for non-draft versions
// would require mocking the GitHub API client. Since these functions directly
// create and use a GitHub client internally, they would need to be refactored
// to accept a GitHub client interface for proper unit testing.
//
// For now, we can only test the "draft" special case which doesn't make API calls.
