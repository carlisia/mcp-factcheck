package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// File path constants shared across commands
const (
	dataSpecsDir   = "./data/specs/"
	specFileSuffix = "-spec.json"
	jsonExt        = ".json"
)

var rootCmd = &cobra.Command{
	Use:   "specloader",
	Short: "Utility tool for managing MCP fact-check specifications",
	Long:  "A utility tool for extracting, embedding, and managing MCP specification versions for the fact-check server.",
}

func init() {
	rootCmd.AddCommand(specCmd)
	rootCmd.AddCommand(embedCmd)
	rootCmd.AddCommand(rechunkCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
