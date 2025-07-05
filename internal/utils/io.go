// Package utils provides common utility functions used throughout the internal packages.
// It includes file I/O operations, schema marshaling, and other shared functionality.
package utils

import (
	"fmt"
	"os"
)

func ReadFile(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return content, nil
}
