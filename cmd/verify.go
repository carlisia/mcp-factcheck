package cmd

import (
	"fmt"
	"os"

	"github.com/carlisia/mcp-factcheck/internal/client"
	"github.com/carlisia/mcp-factcheck/internal/utils"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Fact-check content against the MCP spec",
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, err := cmd.Flags().GetString("file")
		if err != nil {
			return fmt.Errorf("getting file flag: %w", err)
		}
		blurb, err := cmd.Flags().GetString("blurb")
		if err != nil {
			return fmt.Errorf("getting blurb flag: %w", err)
		}
		server, err := cmd.Flags().GetString("server")
		if err != nil {
			return fmt.Errorf("getting server flag: %w", err)
		}

		content := ""
		if filePath != "" {
			// Check if file exists
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				return fmt.Errorf("file does not exist: %s", filePath)
			}
			b, err := utils.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("reading file %s: %w", filePath, err)
			}
			content = string(b)
		} else if blurb != "" {
			content = blurb
		} else {
			return fmt.Errorf("must provide --file or --blurb")
		}

		result, err := client.VerifyContent(server, content)
		if err != nil {
			return err
		}

		fmt.Println("🧠 Fact-check Results:")
		for _, f := range result.Feedback {
			fmt.Printf("✏️  Section: %s\n\n", f.Section)
			fmt.Printf("💡 Explanation: %s\n", f.Explanation)
			fmt.Println("---------")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(verifyCmd)
	verifyCmd.Flags().String("file", "", "Path to the content file")
	verifyCmd.Flags().String("blurb", "", "Paste a content snippet")
	verifyCmd.Flags().String("server", "http://localhost:8080", "MCP fact-check server URL")
}
