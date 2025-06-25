package factcheck

import (
	"os"
	"path/filepath"
	"strings"
)

func LoadSpecMarkdown(specDir string) ([]string, error) {
	var chunks []string

	err := filepath.Walk(specDir, func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".md") {
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, section := range strings.Split(string(b), "\n\n") {
				chunks = append(chunks, strings.TrimSpace(section))
			}
		}
		return nil
	})

	return chunks, err
}
