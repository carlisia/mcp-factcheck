package specs

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v57/github"
)

// File extensions for spec files
const (
	markdownExt = ".md"
	mdxExt      = ".mdx"
)

// LoadSpec loads MCP specification from local directory or GitHub repo
func LoadSpec(ctx context.Context, source SpecSource) ([]string, error) {
	switch source.Type {
	case "local_dir":
		return loadSpecFromLocal(source.Path)
	case "github_repo":
		return loadSpecFromMCPRepo(ctx, source.Path)
	default:
		return nil, fmt.Errorf("unsupported spec source type: %s", source.Type)
	}
}

// loadSpecFromLocal loads markdown files from a local directory
func loadSpecFromLocal(specDir string) ([]string, error) {
	// This is a simplified implementation - the full version would walk directories
	return nil, fmt.Errorf("local loading not implemented")
}

// loadSpecFromMCPRepo loads markdown files from the MCP repository using GitHub API
func loadSpecFromMCPRepo(ctx context.Context, repoPath string) ([]string, error) {
	// Debug: check all environment variables
	fmt.Println("Checking for GitHub authentication...")

	// Try multiple common environment variable names
	tokenVars := []string{"GITHUB_TOKEN", "GH_TOKEN", "GITHUB_ACCESS_TOKEN"}
	var token string
	for _, varName := range tokenVars {
		if val := os.Getenv(varName); val != "" {
			token = val
			fmt.Printf("Found token in %s\n", varName)
			break
		}
	}

	// Create GitHub client
	var client *github.Client
	if token != "" {
		// Debug: log that we're using a token
		tokenPreview := token
		if len(token) > 4 {
			tokenPreview = token[:4] + "..."
		}
		fmt.Printf("Using GitHub token: %s\n", tokenPreview)
		client = github.NewClient(nil).WithAuthToken(token)
	} else {
		fmt.Println("No GitHub token found in any of:", tokenVars)
		fmt.Println("Using unauthenticated client (subject to lower rate limits)")
		client = github.NewClient(nil)
	}

	// Get directory tree recursively
	tree, _, err := client.Git.GetTree(ctx, MCPRepoOwner, MCPRepoName, MCPRepoBranch, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub tree: %w", err)
	}

	var allChunks []string

	// Find all markdown files in the specified directory
	for _, entry := range tree.Entries {
		if entry.Path == nil || entry.Type == nil {
			continue
		}

		// Check if file is in the target directory and is a markdown file
		if strings.HasPrefix(*entry.Path, repoPath) && (strings.HasSuffix(*entry.Path, markdownExt) || strings.HasSuffix(*entry.Path, mdxExt)) {
			// Get file content
			fileContent, _, _, err := client.Repositories.GetContents(ctx, MCPRepoOwner, MCPRepoName, *entry.Path, &github.RepositoryContentGetOptions{
				Ref: MCPRepoBranch,
			})
			if err != nil {
				continue // Skip files we can't read
			}

			if fileContent != nil {
				content, err := fileContent.GetContent()
				if err != nil {
					continue // Skip files we can't decode
				}

				chunks := parseMarkdownSections(content)
				allChunks = append(allChunks, chunks...)
			}
		}
	}

	if len(allChunks) == 0 {
		return nil, fmt.Errorf("no markdown files found in repository path: %s", repoPath)
	}

	return allChunks, nil
}

// parseMarkdownSections splits markdown content into logical sections
func parseMarkdownSections(content string) []string {
	// Use fine-grained chunking strategy for better search accuracy
	strategy := DefaultStrategies["fine"]
	chunkResults := ParseMarkdownSectionsV2(content, strategy)
	return ConvertToStrings(chunkResults)
}
