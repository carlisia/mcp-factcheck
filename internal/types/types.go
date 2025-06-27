package types

import (
	"fmt"
	"strings"
)

type VerifyRequest struct {
	Content     string `json:"content"`
	Type        string `json:"type"`        // "file" or "blurb"
	SpecVersion string `json:"spec_version"` // MCP spec version (optional, defaults to latest)
}

// Validate checks if the request is valid
func (v *VerifyRequest) Validate() error {
	if strings.TrimSpace(v.Content) == "" {
		return fmt.Errorf("content cannot be empty")
	}
	if v.Type != "file" && v.Type != "blurb" {
		return fmt.Errorf("type must be 'file' or 'blurb', got: %s", v.Type)
	}
	return nil
}

type Feedback struct {
	Section     string `json:"section"`
	Explanation string `json:"explanation"`
}

type VerifyResponse struct {
	Feedback []Feedback `json:"feedback"`
}
