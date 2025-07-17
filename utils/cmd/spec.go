package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/carlisia/mcp-factcheck/internal/capabilities"
	"github.com/carlisia/mcp-factcheck/utils/metadata"
	utilspecs "github.com/carlisia/mcp-factcheck/utils/specs"
	"github.com/spf13/cobra"
)

var specCmd = &cobra.Command{
	Use:   "spec",
	Short: "Extract MCP specification from GitHub",
	Long:  "Extract MCP specification content from GitHub and save as JSON files",
	RunE:  runSpec,
}

var (
	specVersion    string
	specOutputPath string
)

func init() {
	specCmd.Flags().StringVar(&specVersion, "version", "", "MCP spec version to extract (required)")
	specCmd.Flags().StringVar(&specOutputPath, "output", "", "Output path for spec JSON file (default: ./data/specs/{version}-spec.json)")

	if err := specCmd.MarkFlagRequired("version"); err != nil {
		// This is a programming error, panic is appropriate
		panic(fmt.Sprintf("Failed to mark 'version' flag as required: %v", err))
	}
}

func runSpec(cmd *cobra.Command, args []string) error {
	// Validate version
	if !capabilities.IsValidSpecVersion(specVersion) {
		return fmt.Errorf("invalid spec version: %s. Valid versions: %v", specVersion, capabilities.ValidSpecVersions)
	}

	log.Printf("Extracting MCP specification version: %s", specVersion)

	// Extract spec content from GitHub
	specPath := utilspecs.BuildSpecPath(specVersion)
	specSource := utilspecs.SpecSource{
		Type: "github_repo",
		Path: specPath,
	}

	chunks, err := utilspecs.LoadSpec(context.Background(), specSource)
	if err != nil {
		return fmt.Errorf("failed to load spec: %w", err)
	}

	log.Printf("Successfully loaded %d chunks from GitHub", len(chunks))

	// Set default output path if not specified
	if specOutputPath == "" {
		specOutputPath = fmt.Sprintf("%s%s%s", dataSpecsDir, specVersion, specFileSuffix)
	}

	// Save raw chunks to JSON file
	if err := saveSpecToFile(chunks, specOutputPath); err != nil {
		return fmt.Errorf("failed to save to file: %w", err)
	}
	log.Printf("Saved spec chunks to: %s", specOutputPath)

	// Update metadata
	log.Printf("Updating metadata...")
	meta, err := metadata.LoadMetadata()
	if err != nil {
		log.Printf("Warning: Failed to load metadata: %v", err)
	} else {
		// Get commit hash and branch/tag info
		ref, _, err := metadata.GetBranchOrTag(context.Background(), utilspecs.MCPRepoOwner, utilspecs.MCPRepoName, specVersion)
		if err != nil {
			log.Printf("Warning: Failed to determine ref type: %v", err)
			ref = utilspecs.MCPRepoBranch
		}

		commitHash, err := metadata.GetLatestCommitHash(utilspecs.MCPRepoOwner, utilspecs.MCPRepoName, ref)
		if err != nil {
			log.Printf("Warning: Failed to get commit hash: %v", err)
			commitHash = "unknown"
		}

		// Update metadata
		repo := fmt.Sprintf("%s/%s", utilspecs.MCPRepoOwner, utilspecs.MCPRepoName)
		if err := meta.UpdateSpecExtraction(specVersion, commitHash, repo, ref, len(chunks)); err != nil {
			log.Printf("Warning: Failed to update metadata: %v", err)
		} else {
			log.Printf("Updated metadata with commit %s", commitHash)
		}
	}

	log.Printf("Extraction complete for version %s", specVersion)
	return nil
}

func saveSpecToFile(chunks []string, path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Create extraction data structure
	specData := map[string]any{
		"version": specVersion,
		"chunks":  chunks,
		"count":   len(chunks),
	}

	// Write to JSON file
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Warning: Failed to close file %s: %v", path, err)
		}
	}()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(specData)
}
