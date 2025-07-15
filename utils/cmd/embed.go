package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/carlisia/mcp-factcheck/internal/embedding"
	"github.com/carlisia/mcp-factcheck/internal/integrations/llm"
	"github.com/carlisia/mcp-factcheck/utils/metadata"
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
)

func init() {
	embedCmd.Flags().StringVar(&embedVersion, "version", "", "MCP spec version to generate embeddings for (required)")

	if err := embedCmd.MarkFlagRequired("version"); err != nil {
		// This is a programming error, panic is appropriate
		panic(fmt.Sprintf("Failed to mark 'version' flag as required: %v", err))
	}
}

func runEmbed(cmd *cobra.Command, args []string) error {

	log.Printf("Generating embeddings for MCP specification version: %s", embedVersion)

	// Load chunks from local JSON file
	// Check if this is a fine-grained version
	var specFile string
	if strings.HasSuffix(embedVersion, "-fine") {
		// For fine-grained versions, use the fine-chunked file
		baseVersion := strings.TrimSuffix(embedVersion, "-fine")
		specFile = fmt.Sprintf("%s%s-spec-fine%s", dataSpecsDir, baseVersion, jsonExt)
	} else {
		// Regular spec file
		specFile = fmt.Sprintf("%s%s%s", dataSpecsDir, embedVersion, specFileSuffix)
	}
	chunks, err := loadChunksFromJSON(specFile)
	if err != nil {
		return fmt.Errorf("failed to load chunks from %s: %w", specFile, err)
	}

	log.Printf("Successfully loaded %d chunks from %s", len(chunks), specFile)

	// Generate embeddings
	log.Println("Generating embeddings...")

	// Create LLM client
	client, err := llm.New()
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}

	// Generate embeddings
	specEmbedding, err := embedding.ProcessSpec(context.Background(), embedVersion, chunks, client.CreateEmbedding)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Store embeddings
	// Always use the embedded storage directory
	embedDataDir := "./internal/storage/embeddings"
	log.Printf("Storing embeddings in: %s", embedDataDir)
	storer := embedding.NewEmbeddingStorage(embedDataDir)
	if err := storer.WriteEmbeddings(specEmbedding); err != nil {
		return fmt.Errorf("failed to store embeddings: %w", err)
	}

	log.Printf("Generated and stored embeddings for %d chunks", specEmbedding.Count)

	// Update metadata
	log.Printf("Updating metadata...")
	meta, err := metadata.LoadMetadata()
	if err != nil {
		log.Printf("Warning: Failed to load metadata: %v", err)
	} else {
		// Determine strategy from version name
		strategy := "regular"
		baseVersion := embedVersion
		if strings.HasSuffix(embedVersion, "-fine") {
			strategy = "fine"
			baseVersion = strings.TrimSuffix(embedVersion, "-fine")
		}

		// Update metadata with actual chunk count
		if err := meta.UpdateEmbeddingGeneration(baseVersion, strategy, specEmbedding.Count); err != nil {
			log.Printf("Warning: Failed to update metadata: %v", err)
		} else {
			log.Printf("Updated metadata for %s strategy with %d chunks", strategy, specEmbedding.Count)
		}
	}

	log.Printf("Embedding generation complete for version %s", embedVersion)
	return nil
}

func loadChunksFromJSON(filePath string) ([]string, error) {
	return embedding.LoadChunksFromJSON(filePath)
}
