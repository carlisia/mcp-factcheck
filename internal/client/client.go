package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/carlisia/mcp-factcheck/internal/types"
)

func VerifyContent(serverURL, content string) (*types.VerifyResponse, error) {
	// Validate inputs
	if strings.TrimSpace(serverURL) == "" {
		return nil, fmt.Errorf("server URL cannot be empty")
	}
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("content cannot be empty")
	}

	// Validate URL format
	parsedURL, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %w", err)
	}
	if parsedURL.Scheme == "" {
		return nil, fmt.Errorf("server URL must include scheme (http/https)")
	}

	// Determine type based on content characteristics
	contentType := "blurb"
	if len(content) > 1000 || strings.Contains(content, "\n\n") {
		contentType = "file"
	}

	reqBody := types.VerifyRequest{
		Content: content,
		Type:    contentType,
	}

	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(reqBody); err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create request with context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", serverURL+"/verify", buf)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, resp.Status)
	}

	var result types.VerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &result, nil
}
