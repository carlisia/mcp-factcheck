package metadata

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v57/github"
)

// GetLatestCommitHash gets the latest commit hash for a branch or tag
func GetLatestCommitHash(owner, repo, ref string) (string, error) {
	// Create GitHub client
	var client *github.Client
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		client = github.NewClient(nil).WithAuthToken(token)
	} else {
		client = github.NewClient(nil)
	}

	// Get the reference (branch or tag)
	gitRef, _, err := client.Git.GetRef(context.Background(), owner, repo, "refs/heads/"+ref)
	if err != nil {
		// Try as a tag if branch lookup fails
		gitRef, _, err = client.Git.GetRef(context.Background(), owner, repo, "refs/tags/"+ref)
		if err != nil {
			return "", fmt.Errorf("failed to get ref %s: %w", ref, err)
		}
	}

	// Extract the commit SHA
	if gitRef.Object != nil && gitRef.Object.SHA != nil {
		return *gitRef.Object.SHA, nil
	}

	return "", fmt.Errorf("no commit SHA found for ref %s", ref)
}

// GetBranchOrTag determines if a ref is a branch or tag and returns the appropriate value
func GetBranchOrTag(owner, repo, version string) (string, string, error) {
	if version == "draft" {
		// Draft always uses main branch
		return "main", "branch", nil
	}

	// For other versions, check if it's a tag
	var client *github.Client
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		client = github.NewClient(nil).WithAuthToken(token)
	} else {
		client = github.NewClient(nil)
	}

	// Try to get it as a tag first
	tagRef := "v" + version // Assuming tags are prefixed with 'v'
	_, _, err := client.Git.GetRef(context.Background(), owner, repo, "refs/tags/"+tagRef)
	if err == nil {
		return tagRef, "tag", nil
	}

	// If not a tag, might be a branch
	_, _, err = client.Git.GetRef(context.Background(), owner, repo, "refs/heads/"+version)
	if err == nil {
		return version, "branch", nil
	}

	// Default to assuming it's a tag without 'v' prefix
	return version, "tag", nil
}