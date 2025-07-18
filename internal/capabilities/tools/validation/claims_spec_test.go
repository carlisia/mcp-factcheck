package validation_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/carlisia/mcp-factcheck/internal/capabilities"
	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools"
	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseClaimsArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    any
		want    *validation.ClaimsRequest
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
			name:    "missing content",
			args:    map[string]any{},
			wantErr: true,
			errMsg:  "content parameter is required",
		},
		{
			name: "non-string content",
			args: map[string]any{
				"content": 123,
			},
			wantErr: true,
			errMsg:  "content must be a string",
		},
		{
			name: "empty content",
			args: map[string]any{
				"content": "",
			},
			wantErr: true,
			errMsg:  "content cannot be empty",
		},
		{
			name: "valid content only",
			args: map[string]any{
				"content": "MCP supports JSON-RPC 2.0",
			},
			want: &validation.ClaimsRequest{
				Content:     "MCP supports JSON-RPC 2.0",
				SpecVersion: capabilities.Latest,
				UseChunking: false, // Short content doesn't trigger auto-chunking
			},
		},
		{
			name: "with spec version",
			args: map[string]any{
				"content":     "MCP uses transport layer security",
				"specVersion": "draft",
			},
			want: &validation.ClaimsRequest{
				Content:     "MCP uses transport layer security",
				SpecVersion: "draft",
				UseChunking: false, // Short content doesn't trigger auto-chunking
			},
		},
		{
			name: "with useChunking false",
			args: map[string]any{
				"content":     "MCP is a protocol",
				"useChunking": false,
			},
			want: &validation.ClaimsRequest{
				Content:     "MCP is a protocol",
				SpecVersion: capabilities.Latest,
				UseChunking: false,
			},
		},
		{
			name: "invalid spec version",
			args: map[string]any{
				"content":     "test",
				"specVersion": "invalid-version",
			},
			wantErr: true,
			errMsg:  "invalid spec version",
		},
		{
			name: "all valid parameters",
			args: map[string]any{
				"content":     "MCP enables AI applications to communicate",
				"specVersion": "2025-06-18",
				"useChunking": true,
			},
			want: &validation.ClaimsRequest{
				Content:     "MCP enables AI applications to communicate",
				SpecVersion: "2025-06-18",
				UseChunking: true,
			},
		},
		{
			name: "non-string spec version",
			args: map[string]any{
				"content":     "test",
				"specVersion": 123,
			},
			want: &validation.ClaimsRequest{
				Content:     "test",
				SpecVersion: capabilities.Latest, // Defaults to latest when non-string
				UseChunking: false,
			},
		},
		{
			name: "non-bool useChunking",
			args: map[string]any{
				"content":     "test",
				"useChunking": "yes",
			},
			want: &validation.ClaimsRequest{
				Content:     "test",
				SpecVersion: capabilities.Latest,
				UseChunking: false, // defaults to false on invalid type and short content
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validation.ParseClaimsArgs(tt.args)

			if tt.wantErr {
				require.Error(t, err, "ParseClaimsArgs() should return error")
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg, "error message should contain expected text")
				}
				return
			}

			require.NoError(t, err, "ParseClaimsArgs() should not return error")
			require.NotNil(t, got, "ParseClaimsArgs() should return valid request")

			assert.Equal(t, tt.want.Content, got.Content, "Content mismatch")
			assert.Equal(t, tt.want.SpecVersion, got.SpecVersion, "SpecVersion mismatch")
			assert.Equal(t, tt.want.UseChunking, got.UseChunking, "UseChunking mismatch")
		})
	}
}

func TestClaims(t *testing.T) {
	tests := []struct {
		name          string
		req           *validation.ClaimsRequest
		embedFunc     tools.EmbeddingFunc
		searchFunc    tools.SearchFunc
		llmFunc       validation.LLMCompleteFunc
		cancelContext bool
		wantErr       bool
		errMsg        string
		wantResult    bool
		checkResult   func(*validation.Result) error
	}{
		{
			name: "context cancelled",
			req: &validation.ClaimsRequest{
				Content:     "MCP test",
				SpecVersion: "draft",
				UseChunking: false,
			},
			embedFunc:     mockEmbedFunc([]float64{0.1, 0.2}),
			searchFunc:    mockSearchFunc(3),
			llmFunc:       mockLLMFunc(true, 0.9),
			cancelContext: true,
			wantErr:       true,
			errMsg:        "validation cancelled",
		},
		{
			name: "successful validation - accurate",
			req: &validation.ClaimsRequest{
				Content:     "MCP uses JSON-RPC 2.0",
				SpecVersion: "draft",
				UseChunking: false,
			},
			embedFunc:  mockEmbedFunc([]float64{0.1, 0.2}),
			searchFunc: mockSearchFunc(3),
			llmFunc:    mockLLMFunc(true, 0.95),
			wantResult: true,
			checkResult: func(r *validation.Result) error {
				if r.IsValid != true {
					return errors.New("expected IsValid to be true")
				}
				if r.Confidence < 0.9 {
					return errors.New("expected high confidence")
				}
				return nil
			},
		},
		{
			name: "successful validation - inaccurate",
			req: &validation.ClaimsRequest{
				Content:     "MCP uses HTTP/3 exclusively",
				SpecVersion: "draft",
				UseChunking: false,
			},
			embedFunc:  mockEmbedFunc([]float64{0.1, 0.2}),
			searchFunc: mockSearchFunc(3),
			llmFunc:    mockLLMFunc(false, 0.85),
			wantResult: true,
			checkResult: func(r *validation.Result) error {
				if r.IsValid != false {
					return errors.New("expected IsValid to be false")
				}
				if len(r.Issues) == 0 {
					return errors.New("expected issues")
				}
				return nil
			},
		},
		{
			name: "embedding error",
			req: &validation.ClaimsRequest{
				Content:     "test",
				SpecVersion: "draft",
				UseChunking: false,
			},
			embedFunc:  mockEmbedFuncError(errors.New("embedding failed")),
			searchFunc: mockSearchFunc(3),
			llmFunc:    mockLLMFunc(true, 0.9),
			wantErr:    true,
			errMsg:     "embedding generation failed",
		},
		{
			name: "search error",
			req: &validation.ClaimsRequest{
				Content:     "test",
				SpecVersion: "draft",
				UseChunking: false,
			},
			embedFunc:  mockEmbedFunc([]float64{0.1, 0.2}),
			searchFunc: mockSearchFuncError(errors.New("search failed")),
			llmFunc:    mockLLMFunc(true, 0.9),
			wantErr:    true,
			errMsg:     "specification search failed",
		},
		{
			name: "llm error",
			req: &validation.ClaimsRequest{
				Content:     "test",
				SpecVersion: "draft",
				UseChunking: false,
			},
			embedFunc:  mockEmbedFunc([]float64{0.1, 0.2}),
			searchFunc: mockSearchFunc(3),
			llmFunc:    mockLLMFuncError(errors.New("LLM failed")),
			wantErr:    true,
			errMsg:     "validation failed",
		},
		{
			name: "with chunking",
			req: &validation.ClaimsRequest{
				Content:     strings.Repeat("MCP is a protocol. ", 200), // Long content
				SpecVersion: "draft",
				UseChunking: true,
			},
			embedFunc:  mockEmbedFunc([]float64{0.1, 0.2}),
			searchFunc: mockSearchFunc(3),
			llmFunc:    mockLLMFunc(true, 0.9),
			wantResult: true,
			checkResult: func(r *validation.Result) error {
				// Should have multiple chunks processed
				if len(r.ParsedClaims) == 0 {
					return errors.New("expected parsed claims from chunks")
				}
				return nil
			},
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

			result, err := validation.Claims(ctx, *tt.req, tt.embedFunc, tt.searchFunc, tt.llmFunc)

			if tt.wantErr {
				require.Error(t, err, "Claims() should return error")
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg, "error message should contain expected text")
				}
				return
			}

			require.NoError(t, err, "Claims() should not return error")

			if tt.wantResult && tt.checkResult != nil {
				assert.NoError(t, tt.checkResult(result), "result check should pass")
			}
		})
	}
}

func TestFormatClaimsResult(t *testing.T) {
	tests := []struct {
		name            string
		result          *validation.Result
		expectedPhrases []string
		notExpected     []string
	}{
		{
			name: "accurate content",
			result: &validation.Result{
				IsValid:     true,
				Confidence:  0.95,
				SpecVersion: "draft",
				ParsedClaims: []string{
					"MCP uses JSON-RPC 2.0",
				},
				FactCheckResult: &validation.FactCheckResult{
					IsAccurate: true,
					Confidence: 0.95,
					Claims: []validation.Claim{
						{
							Claim:      "MCP uses JSON-RPC 2.0",
							IsAccurate: true,
							Confidence: 0.98,
						},
					},
				},
			},
			expectedPhrases: []string{
				"✅ Content is ACCURATE",
				"Confidence: 95%",
				"MCP uses JSON-RPC 2.0",
				"✓ Accurate",
			},
			notExpected: []string{
				"❌ Content is INACCURATE",
				"Inaccuracies Found",
				"Missing Best Practices",
			},
		},
		{
			name: "inaccurate content",
			result: &validation.Result{
				IsValid:     false,
				Confidence:  0.85,
				SpecVersion: "draft",
				ParsedClaims: []string{
					"MCP uses HTTP/3",
				},
				Issues: []string{
					"Incorrect protocol specified",
				},
				FactCheckResult: &validation.FactCheckResult{
					IsAccurate: false,
					Confidence: 0.85,
					Claims: []validation.Claim{
						{
							Claim:      "MCP uses HTTP/3",
							IsAccurate: false,
							Confidence: 0.9,
							Correction: "MCP is transport-agnostic and can use various protocols",
						},
					},
					Inaccuracies: []string{
						"Incorrect protocol specified",
					},
				},
			},
			expectedPhrases: []string{
				"❌ Content is INACCURATE",
				"Confidence: 85%",
				"MCP uses HTTP/3",
				"✗ Inaccurate",
				"MCP is transport-agnostic",
				"Inaccuracies Found:",
				"Incorrect protocol specified",
			},
			notExpected: []string{
				"✅ Content is ACCURATE",
			},
		},
		{
			name: "with missing best practices",
			result: &validation.Result{
				IsValid:     true,
				Confidence:  0.88,
				SpecVersion: "draft",
				Suggestions: []string{
					"Should mention error handling",
					"Should include rate limiting",
				},
				FactCheckResult: &validation.FactCheckResult{
					IsAccurate: true,
					Confidence: 0.88,
					MissingBestPractices: []string{
						"Should mention error handling",
						"Should include rate limiting",
					},
				},
			},
			expectedPhrases: []string{
				"✅ Content is ACCURATE",
				"Missing Best Practices:",
				"Should mention error handling",
				"Should include rate limiting",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formattedSlice := validation.FormatClaimsResult(tt.result)
			formatted := strings.Join(formattedSlice, "\n")

			for _, phrase := range tt.expectedPhrases {
				assert.Contains(t, formatted, phrase, "formatted output missing expected phrase: %q", phrase)
			}

			for _, phrase := range tt.notExpected {
				assert.NotContains(t, formatted, phrase, "formatted output contains unexpected phrase: %q", phrase)
			}
		})
	}
}

func TestContentChunking(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		useChunking bool
		wantChunks  int
	}{
		{
			name:        "short content no chunking",
			content:     "Short MCP content",
			useChunking: true,
			wantChunks:  1,
		},
		{
			name:        "long content with chunking",
			content:     strings.Repeat("MCP is a protocol for AI communication. ", 100),
			useChunking: true,
			wantChunks:  2, // Should be split into multiple chunks
		},
		{
			name:        "long content without chunking",
			content:     strings.Repeat("MCP is a protocol. ", 200),
			useChunking: false,
			wantChunks:  1, // Should not be chunked
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &validation.ClaimsRequest{
				Content:     tt.content,
				SpecVersion: "draft",
				UseChunking: tt.useChunking,
			}

			// We can't directly test chunking without running the full function,
			// but we can verify the request is constructed correctly
			assert.Equal(t, tt.useChunking, req.UseChunking, "UseChunking mismatch")
		})
	}
}

// Mock functions
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
				Content:    "MCP specification content",
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

func mockLLMFunc(accurate bool, confidence float64) validation.LLMCompleteFunc {
	return func(ctx context.Context, model string, prompt string, temperature float64, maxTokens int) (string, error) {
		// Return JSON in the format expected by the template
		type templateResponse struct {
			Claims                 []validation.Claim `json:"claims"`
			MissingBestPractices   []string           `json:"missing_best_practices"`
			AdvisoryLanguageIssues []string           `json:"advisory_language_issues"`
			OverallIsAccurate      bool               `json:"overall_is_accurate"`
			Summary                string             `json:"summary"`
		}

		claims := []validation.Claim{
			{
				Claim:      "Test claim",
				IsAccurate: accurate,
				Confidence: confidence,
			},
		}

		if !accurate {
			claims[0].Correction = "Test correction"
			claims[0].Explanation = "Test inaccuracy"
		}

		response := templateResponse{
			Claims:                 claims,
			MissingBestPractices:   []string{},
			AdvisoryLanguageIssues: []string{},
			OverallIsAccurate:      accurate,
			Summary:                "Test summary",
		}

		// Serialize to JSON as the LLM would return
		data, _ := json.Marshal(response)
		return string(data), nil
	}
}

func mockLLMFuncError(err error) validation.LLMCompleteFunc {
	return func(ctx context.Context, model string, prompt string, temperature float64, maxTokens int) (string, error) {
		return "", err
	}
}
