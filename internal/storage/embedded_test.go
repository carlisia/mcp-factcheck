package storage_test

import (
	"testing"

	"github.com/carlisia/mcp-factcheck/internal/storage"
)

func TestEmbeddedEmbeddings(t *testing.T) {
	testCases := []struct {
		name    string
		version string
		wantErr bool
	}{
		// Regular embeddings
		{"draft regular", "draft", false},
		{"2025-06-18 regular", "2025-06-18", false},
		{"2025-03-26 regular", "2025-03-26", false},
		{"2024-11-05 regular", "2024-11-05", false},
		
		// Fine-grained embeddings
		{"draft fine", "draft-fine", false},
		{"2025-06-18 fine", "2025-06-18-fine", false},
		{"2025-03-26 fine", "2025-03-26-fine", false},
		{"2024-11-05 fine", "2024-11-05-fine", false},
		
		// Non-existent version
		{"non-existent", "non-existent", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := storage.LoadEmbeddedEmbeddings(tc.version)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error for version %s, but got none", tc.version)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for version %s: %v", tc.version, err)
				}
				if len(data) == 0 {
					t.Errorf("empty data for version %s", tc.version)
				}
			}
		})
	}
}

func TestGetAvailableVersions(t *testing.T) {
	versions := storage.GetAvailableVersions()
	
	// Should have 8 versions (4 regular + 4 fine)
	if len(versions) != 8 {
		t.Errorf("expected 8 versions, got %d", len(versions))
	}
	
	// Check that both regular and fine versions exist
	expectedVersions := map[string]bool{
		"draft":           false,
		"draft-fine":      false,
		"2025-06-18":      false,
		"2025-06-18-fine": false,
		"2025-03-26":      false,
		"2025-03-26-fine": false,
		"2024-11-05":      false,
		"2024-11-05-fine": false,
	}
	
	for _, v := range versions {
		if _, exists := expectedVersions[v]; exists {
			expectedVersions[v] = true
		} else {
			t.Errorf("unexpected version: %s", v)
		}
	}
	
	// Verify all expected versions were found
	for v, found := range expectedVersions {
		if !found {
			t.Errorf("missing expected version: %s", v)
		}
	}
}