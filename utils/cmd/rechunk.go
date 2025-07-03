package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/carlisia/mcp-factcheck/utils/specs"
	"github.com/spf13/cobra"
)

var rechunkCmd = &cobra.Command{
	Use:   "rechunk",
	Short: "Re-chunk existing spec files with different strategies",
	Long:  "Re-chunk existing spec JSON files using fine-grained chunking for better search accuracy",
	RunE:  runRechunk,
}

var (
	rechunkVersion  string
	rechunkStrategy string
	rechunkInput    string
	rechunkOutput   string
)

func init() {
	rechunkCmd.Flags().StringVar(&rechunkVersion, "version", "", "MCP spec version to rechunk (required)")
	rechunkCmd.Flags().StringVar(&rechunkStrategy, "strategy", "fine", "Chunking strategy: paragraph, sentence, bullet, or fine")
	rechunkCmd.Flags().StringVar(&rechunkInput, "input", "", "Input spec file (default: ./data/specs/{version}-spec.json)")
	rechunkCmd.Flags().StringVar(&rechunkOutput, "output", "", "Output spec file (default: ./data/specs/{version}-spec-{strategy}.json)")

	rechunkCmd.MarkFlagRequired("version")
}

func runRechunk(cmd *cobra.Command, args []string) error {
	log.Printf("Re-chunking MCP specification version: %s with strategy: %s", rechunkVersion, rechunkStrategy)

	// Validate strategy
	strategy, exists := specs.DefaultStrategies[rechunkStrategy]
	if !exists {
		return fmt.Errorf("invalid strategy: %s. Valid strategies: paragraph, sentence, bullet, fine", rechunkStrategy)
	}

	// Set default paths
	if rechunkInput == "" {
		rechunkInput = fmt.Sprintf("./data/specs/%s-spec.json", rechunkVersion)
	}
	if rechunkOutput == "" {
		rechunkOutput = fmt.Sprintf("./data/specs/%s-spec-%s.json", rechunkVersion, rechunkStrategy)
	}

	// Load existing spec file
	file, err := os.Open(rechunkInput)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer file.Close()

	var data struct {
		Version string   `json:"version"`
		Chunks  []string `json:"chunks"`
		Count   int      `json:"count"`
	}

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return fmt.Errorf("failed to decode JSON: %w", err)
	}

	log.Printf("Loaded %d original chunks from %s", len(data.Chunks), rechunkInput)

	// Combine all chunks back into full content
	fullContent := ""
	for _, chunk := range data.Chunks {
		fullContent += chunk + "\n\n"
	}

	// Re-chunk with new strategy
	chunkResults := specs.ParseMarkdownSectionsV2(fullContent, strategy)
	newChunks := specs.ConvertToStrings(chunkResults)

	log.Printf("Re-chunked into %d chunks using %s strategy", len(newChunks), rechunkStrategy)

	// Log chunk statistics
	var totalLen, minLen, maxLen int
	for i, chunk := range newChunks {
		chunkLen := len(chunk)
		totalLen += chunkLen
		if i == 0 || chunkLen < minLen {
			minLen = chunkLen
		}
		if chunkLen > maxLen {
			maxLen = chunkLen
		}
	}
	avgLen := totalLen / len(newChunks)
	log.Printf("Chunk statistics: min=%d, max=%d, avg=%d chars", minLen, maxLen, avgLen)

	// Save re-chunked data
	outputData := map[string]any{
		"version":  rechunkVersion,
		"strategy": rechunkStrategy,
		"chunks":   newChunks,
		"count":    len(newChunks),
		"stats": map[string]int{
			"min_length": minLen,
			"max_length": maxLen,
			"avg_length": avgLen,
		},
	}

	// Create output directory if needed
	outputDir := filepath.Dir(rechunkOutput)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write to file
	outFile, err := os.Create(rechunkOutput)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	encoder := json.NewEncoder(outFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(outputData); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	log.Printf("Saved re-chunked spec to: %s", rechunkOutput)
	return nil
}
