package search_test

import (
	"github.com/carlisia/mcp-factcheck/internal/capabilities"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools"
	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools/search"
)

func TestParseSearchArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    any
		want    *search.SearchRequest
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil args",
			args:    nil,
			wantErr: true,
			errMsg:  "arguments must be a map",
		},
		{
			name:    "non-map args",
			args:    "invalid",
			wantErr: true,
			errMsg:  "arguments must be a map",
		},
		{
			name:    "missing query",
			args:    map[string]any{},
			wantErr: true,
			errMsg:  "query must be a string",
		},
		{
			name: "non-string query",
			args: map[string]any{
				"query": 123,
			},
			wantErr: true,
			errMsg:  "query must be a string",
		},
		{
			name: "valid query only",
			args: map[string]any{
				"query": "authentication",
			},
			want: &search.SearchRequest{
				Query:       "authentication",
				SpecVersion: capabilities.Latest,
				TopK:        5, // default value
			},
		},
		{
			name: "valid with spec version",
			args: map[string]any{
				"query":       "rate limits",
				"specVersion": "draft",
			},
			want: &search.SearchRequest{
				Query:       "rate limits",
				SpecVersion: "draft",
				TopK:        5,
			},
		},
		{
			name: "valid with topK as float64",
			args: map[string]any{
				"query": "resources",
				"topK":  float64(10),
			},
			want: &search.SearchRequest{
				Query:       "resources",
				SpecVersion: capabilities.Latest,
				TopK:        10,
			},
		},
		{
			name: "valid with topK as int",
			args: map[string]any{
				"query": "tools",
				"topK":  15,
			},
			want: &search.SearchRequest{
				Query:       "tools",
				SpecVersion: capabilities.Latest,
				TopK:        15,
			},
		},
		{
			name: "empty query",
			args: map[string]any{
				"query": "",
			},
			wantErr: true,
			errMsg:  "query cannot be empty",
		},
		{
			name: "query too long",
			args: map[string]any{
				"query": string(make([]byte, 501)), // Over 500 char limit
			},
			wantErr: true,
			errMsg:  "query length exceeds maximum",
		},
		{
			name: "invalid spec version",
			args: map[string]any{
				"query":       "test",
				"specVersion": "invalid-version",
			},
			wantErr: true,
			errMsg:  "invalid spec version",
		},
		{
			name: "topK below minimum",
			args: map[string]any{
				"query": "test",
				"topK":  float64(0),
			},
			wantErr: true,
			errMsg:  "topK must be at least",
		},
		{
			name: "topK above maximum",
			args: map[string]any{
				"query": "test",
				"topK":  float64(25),
			},
			wantErr: true,
			errMsg:  "topK cannot exceed",
		},
		{
			name: "all valid parameters",
			args: map[string]any{
				"query":       "MCP transport",
				"specVersion": "2025-06-18",
				"topK":        float64(7),
			},
			want: &search.SearchRequest{
				Query:       "MCP transport",
				SpecVersion: "2025-06-18",
				TopK:        7,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := search.ParseSearchArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSearchArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error message should contain %q, got %q", tt.errMsg, err.Error())
				}
				return
			}
			if !tt.wantErr {
				if got.Query != tt.want.Query {
					t.Errorf("Query = %q, want %q", got.Query, tt.want.Query)
				}
				if got.SpecVersion != tt.want.SpecVersion {
					t.Errorf("SpecVersion = %q, want %q", got.SpecVersion, tt.want.SpecVersion)
				}
				if got.TopK != tt.want.TopK {
					t.Errorf("TopK = %d, want %d", got.TopK, tt.want.TopK)
				}
			}
		})
	}
}

func TestSearch(t *testing.T) {
	tests := []struct {
		name          string
		req           *search.SearchRequest
		embedFunc     tools.EmbeddingFunc
		searchFunc    tools.SearchFunc
		cancelContext bool
		wantErr       bool
		errMsg        string
		wantResults   int
	}{
		{
			name: "context cancelled",
			req: &search.SearchRequest{
				Query:       "test",
				SpecVersion: "draft",
				TopK:        5,
			},
			embedFunc:     mockEmbedFunc([]float64{0.1, 0.2, 0.3}),
			searchFunc:    mockSearchFunc(5),
			cancelContext: true,
			wantErr:       true,
			errMsg:        "search cancelled before execution",
		},
		{
			name: "successful search",
			req: &search.SearchRequest{
				Query:       "authentication",
				SpecVersion: "draft",
				TopK:        3,
			},
			embedFunc:   mockEmbedFunc([]float64{0.1, 0.2, 0.3}),
			searchFunc:  mockSearchFunc(3),
			wantResults: 3,
		},
		{
			name: "embedding error",
			req: &search.SearchRequest{
				Query:       "test",
				SpecVersion: "draft",
				TopK:        5,
			},
			embedFunc:  mockEmbedFuncError(errors.New("embedding failed")),
			searchFunc: mockSearchFunc(5),
			wantErr:    true,
			errMsg:     "embedding failed",
		},
		{
			name: "search error",
			req: &search.SearchRequest{
				Query:       "test",
				SpecVersion: "draft",
				TopK:        5,
			},
			embedFunc:  mockEmbedFunc([]float64{0.1, 0.2}),
			searchFunc: mockSearchFuncError(errors.New("search failed")),
			wantErr:    true,
			errMsg:     "search failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			results, err := search.Search(ctx, tt.req, tt.embedFunc, tt.searchFunc)
			if (err != nil) != tt.wantErr {
				t.Errorf("Search() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error message should contain %q, got %q", tt.errMsg, err.Error())
				}
				return
			}
			if !tt.wantErr && len(results) != tt.wantResults {
				t.Errorf("got %d results, want %d", len(results), tt.wantResults)
			}
		})
	}
}

func TestFormatResults(t *testing.T) {
	results := []tools.SearchResult{
		{
			Content:    "MCP uses JSON-RPC 2.0 for communication",
			Similarity: 0.9234,
			Rank:       1,
		},
		{
			Content:    "Authentication is handled at the transport layer",
			Similarity: 0.8567,
			Rank:       2,
		},
	}

	formatted := search.FormatResults("authentication", "draft", results)

	// Check for expected content
	expectedPhrases := []string{
		"Search results for 'authentication' in MCP draft:",
		"Rank 1 (similarity: 0.9234)",
		"MCP uses JSON-RPC 2.0",
		"Rank 2 (similarity: 0.8567)",
		"transport layer",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(formatted, phrase) {
			t.Errorf("formatted output missing expected phrase: %q", phrase)
		}
	}
}

// Mock functions for testing
func mockEmbedFunc(embedding []float64) tools.EmbeddingFunc {
	return func(ctx context.Context, content string) ([]float64, error) {
		return embedding, nil
	}
}

func mockEmbedFuncError(err error) tools.EmbeddingFunc {
	return func(ctx context.Context, content string) ([]float64, error) {
		return nil, err
	}
}

func mockSearchFunc(numResults int) tools.SearchFunc {
	return func(version string, queryEmbedding []float64, topK int) ([]tools.SearchResult, error) {
		results := make([]tools.SearchResult, numResults)
		for i := 0; i < numResults; i++ {
			results[i] = tools.SearchResult{
				Content:    "Test content",
				Similarity: 0.9 - float64(i)*0.1,
				Rank:       i + 1,
			}
		}
		return results, nil
	}
}

func mockSearchFuncError(err error) tools.SearchFunc {
	return func(version string, queryEmbedding []float64, topK int) ([]tools.SearchResult, error) {
		return nil, err
	}
}