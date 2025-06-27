package main

import (
	"fmt"
	"log"

	"github.com/carlisia/mcp-factcheck/utils/embedding"
	"github.com/spf13/cobra"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Test embedding generation with sample data",
	Long:  "Test the embedding generation and vector database functionality with a small sample",
	RunE:  runTest,
}


func runTest(cmd *cobra.Command, args []string) error {
	log.Println("Testing embedding generation...")

	// Create test chunks
	testChunks := []string{
		"The Model Context Protocol (MCP) is a protocol for integrating AI assistants with external systems.",
		"MCP servers expose resources and tools that clients can discover and use.",
		"Resources in MCP represent data that can be read by clients, such as files or database records.",
		"Tools in MCP represent actions that can be performed by clients, such as executing code or making API calls.",
	}

	// Create batch embedding generator
	generator, err := embedding.NewBatchGenerator()
	if err != nil {
		return fmt.Errorf("failed to create embedding generator: %w", err)
	}

	// Generate embeddings for test chunks
	specEmbedding, err := generator.GenerateSpecEmbeddings("test", testChunks)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	log.Printf("Generated embeddings for %d chunks", specEmbedding.Count)

	// Store in embedding database
	embeddingStore := embedding.NewEmbeddingStore("./data/embeddings")
	if err := embeddingStore.Store(specEmbedding); err != nil {
		return fmt.Errorf("failed to store embeddings: %w", err)
	}

	log.Println("Stored test embeddings in database")

	// Test query embedding generation
	queryGenerator, err := embedding.NewGenerator()
	if err != nil {
		return fmt.Errorf("failed to create query generator: %w", err)
	}
	queryEmbedding, err := queryGenerator.GenerateEmbedding("What are MCP tools?")
	if err != nil {
		return fmt.Errorf("failed to generate query embedding: %w", err)
	}

	log.Printf("Generated query embedding with %d dimensions", len(queryEmbedding))

	log.Println("Test completed successfully!")
	return nil
}