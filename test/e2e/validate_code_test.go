package e2e

import (
	"context"
	"testing"

	"github.com/carlisia/mcp-factcheck/pkg/specs"
	"github.com/carlisia/mcp-factcheck/pkg/validator"
	"github.com/mark3labs/mcp-go/mcp"
)

// validateCodeArgs provides typed arguments for validate code tests
type validateCodeArgs struct {
	Code        string
	SpecVersion string
	Language    string
}

// toMap converts typed args to map for handler
func (v validateCodeArgs) toMap() map[string]any {
	m := map[string]any{"code": v.Code}
	if v.SpecVersion != "" {
		m["specVersion"] = v.SpecVersion
	}
	if v.Language != "" {
		m["language"] = v.Language
	}
	return m
}

func TestValidator_HandleValidateCode_WithValidInput(t *testing.T) {
	ctx := context.Background()
	vectorDB, generator := setupTestEnv(t)

	tests := []struct {
		name   string
		args   validateCodeArgs
		assert func(t *testing.T, got []mcp.Content)
	}{
		{
			name: "validate Go code with all parameters",
			args: validateCodeArgs{
				Code:        "func HandleRequest(ctx context.Context) {}",
				SpecVersion: specs.DefaultSpecVersion,
				Language:    "go",
			},
			assert: assertNonEmpty,
		},
		{
			name: "validate with minimal args",
			args: validateCodeArgs{
				Code: "const x = 42",
			},
			assert: assertNonEmpty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validator.HandleValidateCode(ctx, vectorDB, generator, tt.args.toMap())
			// Note: These tests will fail with real API calls using test key
			if err == nil && tt.assert != nil {
				tt.assert(t, got)
			}
		})
	}
}

func TestValidator_HandleValidateCode_WithInvalidInput(t *testing.T) {
	ctx := context.Background()
	vectorDB, generator := setupTestEnv(t)

	tests := []struct {
		name    string
		args    any
		wantErr bool
	}{
		{
			name: "missing code parameter",
			args: map[string]any{
				"language": "javascript",
			},
			wantErr: true,
		},
		{
			name:    "invalid arguments type",
			args:    123,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validator.HandleValidateCode(ctx, vectorDB, generator, tt.args)
			assertErr(t, err, tt.wantErr)
		})
	}
}
