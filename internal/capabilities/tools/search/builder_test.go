package search

import (
	"testing"

	"github.com/carlisia/mcp-factcheck/internal/capabilities"
)

func TestSearchRequestBuilder(t *testing.T) {
	tests := []struct {
		name      string
		buildFunc func() (*SearchRequest, error)
		want      *SearchRequest
		wantErr   bool
	}{
		{
			name: "default values",
			buildFunc: func() (*SearchRequest, error) {
				return newSearchRequestBuilder().
					WithQuery("test").
					Build()
			},
			want: &SearchRequest{
				Query:       "test",
				SpecVersion: capabilities.Latest,
				TopK:        defaultTopK,
			},
		},
		{
			name: "empty query",
			buildFunc: func() (*SearchRequest, error) {
				return newSearchRequestBuilder().
					WithQuery("").
					Build()
			},
			wantErr: true,
		},
		{
			name: "valid spec version",
			buildFunc: func() (*SearchRequest, error) {
				return newSearchRequestBuilder().
					WithQuery("test").
					WithSpecVersion("draft").
					Build()
			},
			want: &SearchRequest{
				Query:       "test",
				SpecVersion: "draft",
				TopK:        defaultTopK,
			},
		},
		{
			name: "invalid spec version",
			buildFunc: func() (*SearchRequest, error) {
				return newSearchRequestBuilder().
					WithQuery("test").
					WithSpecVersion("invalid").
					Build()
			},
			wantErr: true,
		},
		{
			name: "valid topK",
			buildFunc: func() (*SearchRequest, error) {
				return newSearchRequestBuilder().
					WithQuery("test").
					WithTopK(10).
					Build()
			},
			want: &SearchRequest{
				Query:       "test",
				SpecVersion: capabilities.Latest,
				TopK:        10,
			},
		},
		{
			name: "topK below minimum",
			buildFunc: func() (*SearchRequest, error) {
				return newSearchRequestBuilder().
					WithQuery("test").
					WithTopK(0).
					Build()
			},
			wantErr: true,
		},
		{
			name: "topK above maximum",
			buildFunc: func() (*SearchRequest, error) {
				return newSearchRequestBuilder().
					WithQuery("test").
					WithTopK(25).
					Build()
			},
			wantErr: true,
		},
		{
			name: "all parameters",
			buildFunc: func() (*SearchRequest, error) {
				return newSearchRequestBuilder().
					WithQuery("authentication").
					WithSpecVersion("2025-06-18").
					WithTopK(7).
					Build()
			},
			want: &SearchRequest{
				Query:       "authentication",
				SpecVersion: "2025-06-18",
				TopK:        7,
			},
		},
		{
			name: "multiple errors",
			buildFunc: func() (*SearchRequest, error) {
				return newSearchRequestBuilder().
					WithQuery("").
					WithSpecVersion("invalid").
					WithTopK(100).
					Build()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.buildFunc()
			if (err != nil) != tt.wantErr {
				t.Errorf("Build() error = %v, wantErr %v", err, tt.wantErr)
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

func TestValidateSearchRequest(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		version     string
		topK        any
		wantQuery   string
		wantVersion string
		wantTopK    int
		wantErr     bool
	}{
		{
			name:        "nil topK uses default",
			query:       "test",
			version:     "",
			topK:        nil,
			wantQuery:   "test",
			wantVersion: capabilities.Latest,
			wantTopK:    defaultTopK,
		},
		{
			name:        "string topK ignored",
			query:       "test",
			version:     "",
			topK:        "invalid",
			wantQuery:   "test",
			wantVersion: capabilities.Latest,
			wantTopK:    defaultTopK,
		},
		{
			name:        "float64 topK converted",
			query:       "test",
			version:     "draft",
			topK:        float64(8),
			wantQuery:   "test",
			wantVersion: "draft",
			wantTopK:    8,
		},
		{
			name:        "int topK used directly",
			query:       "test",
			version:     "2025-06-18",
			topK:        12,
			wantQuery:   "test",
			wantVersion: "2025-06-18",
			wantTopK:    12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateSearchRequest(tt.query, tt.version, tt.topK)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSearchRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Query != tt.wantQuery {
					t.Errorf("Query = %q, want %q", got.Query, tt.wantQuery)
				}
				if got.SpecVersion != tt.wantVersion {
					t.Errorf("SpecVersion = %q, want %q", got.SpecVersion, tt.wantVersion)
				}
				if got.TopK != tt.wantTopK {
					t.Errorf("TopK = %d, want %d", got.TopK, tt.wantTopK)
				}
			}
		})
	}
}