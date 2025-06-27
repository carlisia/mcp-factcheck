package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/carlisia/mcp-factcheck/utils/embedding"
	"github.com/spf13/cobra"
)

var embedCmd = &cobra.Command{
	Use:   "embed",
	Short: "Generate embeddings from local spec files",
	Long:  "Generate embeddings from existing spec JSON files in data/specs/",
	RunE:  runEmbed,
}

var (
	embedVersion string
	embedDataDir string
)

func init() {
	embedCmd.Flags().StringVar(&embedVersion, "version", "", "MCP spec version to generate embeddings for (required)")
	embedCmd.Flags().StringVar(&embedDataDir, "data-dir", "./data/embeddings", "Directory to store vector database")
	
	embedCmd.MarkFlagRequired("version")
}

func runEmbed(cmd *cobra.Command, args []string) error {

	log.Printf("Generating embeddings for MCP specification version: %s", embedVersion)

	// Load chunks from local JSON file
	specFile := fmt.Sprintf("./data/specs/%s-spec.json", embedVersion)
	chunks, err := loadChunksFromJSON(specFile)
	if err != nil {
		return fmt.Errorf("failed to load chunks from %s: %w", specFile, err)
	}

	log.Printf("Successfully loaded %d chunks from %s", len(chunks), specFile)

	// Generate embeddings
	log.Println("Generating embeddings...")
	
	// Create batch embedding generator
	generator, err := embedding.NewBatchGenerator()
	if err != nil {
		return fmt.Errorf("failed to create embedding generator: %w", err)
	}

	// Generate embeddings for all chunks
	specEmbedding, err := generator.GenerateSpecEmbeddings(embedVersion, chunks)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	log.Printf("Generated embeddings for %d chunks", specEmbedding.Count)

	// Store in embedding database
	embeddingStore := embedding.NewEmbeddingStore(embedDataDir)
	if err := embeddingStore.Store(specEmbedding); err != nil {
		return fmt.Errorf("failed to store embeddings: %w", err)
	}
	log.Printf("Stored embeddings in database: %s", embedDataDir)

	log.Printf("Embedding generation complete for version %s", embedVersion)
	return nil
}

func loadChunksFromJSON(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var data struct {
		Chunks []string `json:"chunks"`
		Count  int      `json:"count"`
	}

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	if len(data.Chunks) == 0 {
		return nil, fmt.Errorf("no chunks found in file")
	}

	return data.Chunks, nil
}