package migrate_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/carlisia/mcp-factcheck/internal/capabilities/prompts/migrate"
)

func TestParseMigrateArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]string
		want    *migrate.MigrateRequest
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil args",
			args:    nil,
			wantErr: true,
			errMsg:  "required argument missing: current_version",
		},
		{
			name:    "empty args",
			args:    map[string]string{},
			wantErr: true,
			errMsg:  "required argument missing: current_version",
		},
		{
			name: "missing current_version",
			args: map[string]string{
				"target_version": expectedVersionLatest,
			},
			wantErr: true,
			errMsg:  "required argument missing: current_version",
		},
		{
			name: "empty current_version",
			args: map[string]string{
				"current_version": "",
				"target_version":  expectedVersionLatest,
			},
			wantErr: true,
			errMsg:  "required argument missing: current_version",
		},
		{
			name: "missing target_version",
			args: map[string]string{
				"current_version": expectedVersionDraft,
			},
			wantErr: true,
			errMsg:  "required argument missing: target_version",
		},
		{
			name: "empty target_version",
			args: map[string]string{
				"current_version": expectedVersionDraft,
				"target_version":  "",
			},
			wantErr: true,
			errMsg:  "required argument missing: target_version",
		},
		{
			name: "valid minimal args",
			args: map[string]string{
				"current_version": expectedVersionDraft,
				"target_version":  expectedVersionLatest,
			},
			want: &migrate.MigrateRequest{
				CurrentVersion: expectedVersionDraft,
				TargetVersion:  expectedVersionLatest,
				UpdateScope:    expectedScopeComprehensive, // default
			},
		},
		{
			name: "valid with update_scope",
			args: map[string]string{
				"current_version": expectedVersionNov,
				"target_version":  expectedVersionLatest,
				"update_scope":    expectedScopeCritical,
			},
			want: &migrate.MigrateRequest{
				CurrentVersion: expectedVersionNov,
				TargetVersion:  expectedVersionLatest,
				UpdateScope:    expectedScopeCritical,
			},
		},
		{
			name: "invalid current_version",
			args: map[string]string{
				"current_version": "invalid-version",
				"target_version":  expectedVersionLatest,
			},
			wantErr: true,
			errMsg:  "invalid current_version: invalid-version",
		},
		{
			name: "invalid target_version",
			args: map[string]string{
				"current_version": expectedVersionDraft,
				"target_version":  "2025-01-01",
			},
			wantErr: true,
			errMsg:  "invalid target_version: 2025-01-01",
		},
		{
			name: "invalid update_scope",
			args: map[string]string{
				"current_version": expectedVersionDraft,
				"target_version":  expectedVersionLatest,
				"update_scope":    "minimal",
			},
			wantErr: true,
			errMsg:  "invalid update_scope: minimal",
		},
		{
			name: "all scopes valid - comprehensive",
			args: map[string]string{
				"current_version": expectedVersionDraft,
				"target_version":  expectedVersionLatest,
				"update_scope":    expectedScopeComprehensive,
			},
			want: &migrate.MigrateRequest{
				CurrentVersion: expectedVersionDraft,
				TargetVersion:  expectedVersionLatest,
				UpdateScope:    expectedScopeComprehensive,
			},
		},
		{
			name: "all scopes valid - enhancement_focused",
			args: map[string]string{
				"current_version": expectedVersionMarch,
				"target_version":  expectedVersionLatest,
				"update_scope":    expectedScopeEnhancement,
			},
			want: &migrate.MigrateRequest{
				CurrentVersion: expectedVersionMarch,
				TargetVersion:  expectedVersionLatest,
				UpdateScope:    expectedScopeEnhancement,
			},
		},
		{
			name: "empty update_scope defaults to comprehensive",
			args: map[string]string{
				"current_version": expectedVersionDraft,
				"target_version":  expectedVersionLatest,
				"update_scope":    "",
			},
			want: &migrate.MigrateRequest{
				CurrentVersion: expectedVersionDraft,
				TargetVersion:  expectedVersionLatest,
				UpdateScope:    expectedScopeComprehensive,
			},
		},
		{
			name: "multiple errors",
			args: map[string]string{
				"current_version": "invalid1",
				"target_version":  "invalid2",
				"update_scope":    "invalid3",
			},
			wantErr: true,
			errMsg:  "validation errors:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := migrate.ParseMigrateArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMigrateArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error message should contain %q, got %q", tt.errMsg, err.Error())
				}
				return
			}
			if !tt.wantErr {
				if got.CurrentVersion != tt.want.CurrentVersion {
					t.Errorf("CurrentVersion = %q, want %q", got.CurrentVersion, tt.want.CurrentVersion)
				}
				if got.TargetVersion != tt.want.TargetVersion {
					t.Errorf("TargetVersion = %q, want %q", got.TargetVersion, tt.want.TargetVersion)
				}
				if got.UpdateScope != tt.want.UpdateScope {
					t.Errorf("UpdateScope = %q, want %q", got.UpdateScope, tt.want.UpdateScope)
				}
			}
		})
	}
}

func TestRender(t *testing.T) {
	tests := []struct {
		name           string
		req            *migrate.MigrateRequest
		wantErr        bool
		checkContent   []string
		notWantContent []string
	}{
		{
			name: "basic migration prompt",
			req: &migrate.MigrateRequest{
				CurrentVersion: expectedVersionDraft,
				TargetVersion:  expectedVersionLatest,
				UpdateScope:    expectedScopeComprehensive,
			},
			checkContent: []string{
				expectedVersionDraft,
				expectedVersionLatest,
				expectedScopeComprehensive,
			},
		},
		{
			name: "critical only scope",
			req: &migrate.MigrateRequest{
				CurrentVersion: expectedVersionNov,
				TargetVersion:  expectedVersionLatest,
				UpdateScope:    expectedScopeCritical,
			},
			checkContent: []string{
				expectedVersionNov,
				expectedVersionLatest,
				expectedScopeCritical,
			},
		},
		{
			name: "enhancement focused scope",
			req: &migrate.MigrateRequest{
				CurrentVersion: expectedVersionMarch,
				TargetVersion:  expectedVersionLatest,
				UpdateScope:    expectedScopeEnhancement,
			},
			checkContent: []string{
				expectedVersionMarch,
				expectedVersionLatest,
				expectedScopeEnhancement,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := migrate.Render(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Check result structure
			if len(result.Messages) != 1 {
				t.Errorf("expected 1 message, got %d", len(result.Messages))
				return
			}

			msg := result.Messages[0]
			if msg.Role != "user" {
				t.Errorf("expected role 'user', got %q", msg.Role)
			}

			// Get content as string - mcp.NewTextContent returns a type with GetText() method
			content := ""
			if textContent, ok := msg.Content.(interface{ GetText() string }); ok {
				content = textContent.GetText()
			} else {
				// Try direct string conversion as fallback
				content = fmt.Sprintf("%v", msg.Content)
			}

			// Debug: print first 200 chars of content if test fails
			if t.Failed() && content != "" {
				t.Logf("Content preview: %.200s...", content)
			}

			// Check expected content
			for _, phrase := range tt.checkContent {
				if !strings.Contains(content, phrase) {
					t.Errorf("content missing expected phrase: %q", phrase)
				}
			}

			// Check unwanted content
			for _, phrase := range tt.notWantContent {
				if strings.Contains(content, phrase) {
					t.Errorf("content contains unwanted phrase: %q", phrase)
				}
			}
		})
	}
}

func TestValidVersionCombinations(t *testing.T) {
	// Test all valid version combinations
	validVersions := []string{
		expectedVersionLatest,
		expectedVersionMarch,
		expectedVersionNov,
		expectedVersionDraft,
	}

	for _, current := range validVersions {
		for _, target := range validVersions {
			t.Run(fmt.Sprintf("%s_to_%s", current, target), func(t *testing.T) {
				args := map[string]string{
					"current_version": current,
					"target_version":  target,
				}
				req, err := migrate.ParseMigrateArgs(args)
				if err != nil {
					t.Errorf("valid version combination %s->%s failed: %v", current, target, err)
					return
				}
				if req.CurrentVersion != current {
					t.Errorf("CurrentVersion = %q, want %q", req.CurrentVersion, current)
				}
				if req.TargetVersion != target {
					t.Errorf("TargetVersion = %q, want %q", req.TargetVersion, target)
				}
			})
		}
	}
}

func TestUpdateScopeDefaults(t *testing.T) {
	tests := []struct {
		name        string
		scopeValue  string
		wantScope   string
		shouldError bool
	}{
		{
			name:       "empty string defaults to comprehensive",
			scopeValue: "",
			wantScope:  expectedScopeComprehensive,
		},
		{
			name:       "comprehensive explicit",
			scopeValue: expectedScopeComprehensive,
			wantScope:  expectedScopeComprehensive,
		},
		{
			name:       "critical_only",
			scopeValue: expectedScopeCritical,
			wantScope:  expectedScopeCritical,
		},
		{
			name:       "enhancement_focused",
			scopeValue: expectedScopeEnhancement,
			wantScope:  expectedScopeEnhancement,
		},
		{
			name:        "invalid scope",
			scopeValue:  "fast_mode",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]string{
				"current_version": expectedVersionDraft,
				"target_version":  expectedVersionLatest,
				"update_scope":    tt.scopeValue,
			}
			req, err := migrate.ParseMigrateArgs(args)
			if tt.shouldError {
				if err == nil {
					t.Error("expected error for invalid scope")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if req.UpdateScope != tt.wantScope {
				t.Errorf("UpdateScope = %q, want %q", req.UpdateScope, tt.wantScope)
			}
		})
	}
}

func TestRenderContentStructure(t *testing.T) {
	req := &migrate.MigrateRequest{
		CurrentVersion: expectedVersionDraft,
		TargetVersion:  expectedVersionLatest,
		UpdateScope:    expectedScopeComprehensive,
	}

	result, err := migrate.Render(req)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	// Extract content
	content := ""
	if textContent, ok := result.Messages[0].Content.(interface{ GetText() string }); ok {
		content = textContent.GetText()
	} else {
		// Try direct string conversion as fallback
		content = fmt.Sprintf("%v", result.Messages[0].Content)
	}

	// Check that template includes expected sections
	expectedSections := []string{
		"current spec version", // Should mention current version context
		"target spec version",  // Should mention target version context
		"update scope",         // Should mention the scope
	}

	for _, section := range expectedSections {
		if !strings.Contains(strings.ToLower(content), section) {
			t.Errorf("content missing expected section about: %q", section)
		}
	}
}
