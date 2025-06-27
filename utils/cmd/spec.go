package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	specs "github.com/carlisia/mcp-factcheck/internal/specs"
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
	
	specCmd.MarkFlagRequired("version")
}

func runSpec(cmd *cobra.Command, args []string) error {
	// Validate version
	if !specs.IsValidSpecVersion(specVersion) {
		return fmt.Errorf("invalid spec version: %s. Valid versions: %v", specVersion, specs.ValidSpecVersions)
	}

	log.Printf("Extracting MCP specification version: %s", specVersion)

	// Extract spec content from GitHub
	specPath := utilspecs.BuildSpecPath(specVersion)
	specSource := utilspecs.SpecSource{
		Type: "github_repo",
		Path: specPath,
	}

	chunks, err := utilspecs.LoadSpec(specSource)
	if err != nil {
		return fmt.Errorf("failed to load spec: %w", err)
	}

	log.Printf("Successfully loaded %d chunks from GitHub", len(chunks))

	// Set default output path if not specified
	if specOutputPath == "" {
		specOutputPath = fmt.Sprintf("./data/specs/%s-spec.json", specVersion)
	}

	// Save raw chunks to JSON file
	if err := saveSpecToFile(chunks, specOutputPath); err != nil {
		return fmt.Errorf("failed to save to file: %w", err)
	}
	log.Printf("Saved spec chunks to: %s", specOutputPath)

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
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(specData)
}