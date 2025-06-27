package specs

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v57/github"
)

// LoadSpec loads MCP specification from local directory or GitHub repo
func LoadSpec(source SpecSource) ([]string, error) {
	switch source.Type {
	case "local_dir":
		return loadSpecFromLocal(source.Path)
	case "github_repo":
		return loadSpecFromMCPRepo(source.Path)
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
func loadSpecFromMCPRepo(repoPath string) ([]string, error) {
	// Create GitHub client
	var client *github.Client
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		client = github.NewClient(nil).WithAuthToken(token)
	} else {
		client = github.NewClient(nil)
	}

	// Get directory tree recursively
	tree, _, err := client.Git.GetTree(context.Background(), MCPRepoOwner, MCPRepoName, MCPRepoBranch, true)
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
		if strings.HasPrefix(*entry.Path, repoPath) && (strings.HasSuffix(*entry.Path, ".md") || strings.HasSuffix(*entry.Path, ".mdx")) {
			// Get file content
			fileContent, _, _, err := client.Repositories.GetContents(context.Background(), MCPRepoOwner, MCPRepoName, *entry.Path, &github.RepositoryContentGetOptions{
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
	var chunks []string
	
	// Split by double newlines to get paragraphs/sections
	sections := strings.Split(content, "\n\n")
	for _, section := range sections {
		trimmed := strings.TrimSpace(section)
		if len(trimmed) > 0 {
			chunks = append(chunks, trimmed)
		}
	}
	
	return chunks
}