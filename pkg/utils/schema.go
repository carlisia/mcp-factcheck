// Package utils provides common utility functions for the MCP fact-check system.
package utils

import (
	"encoding/json"
	"fmt"
)

// MustMarshalSchema marshals a schema or panics if it fails.
// This is only used for static schemas that should never fail to marshal.
// The panic is appropriate here because schema marshaling failures indicate
// a programming error that should be caught during development.
func MustMarshalSchema(schema any, toolName string) []byte {
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal schema for %s: %v", toolName, err))
	}
	return schemaBytes
}