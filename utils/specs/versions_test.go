package specs

import (
	"testing"
)

func TestBuildSpecPath(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "draft version",
			version:  "draft",
			expected: "docs/specification/draft",
		},
		{
			name:     "tagged version 2025-06-18",
			version:  "2025-06-18",
			expected: "docs/specification/2025-06-18",
		},
		{
			name:     "tagged version 2024-11-05",
			version:  "2024-11-05",
			expected: "docs/specification/2024-11-05",
		},
		{
			name:     "empty version",
			version:  "",
			expected: "docs/specification/",
		},
		{
			name:     "custom version",
			version:  "custom-v1",
			expected: "docs/specification/custom-v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildSpecPath(tt.version)
			AssertEqual(t, result, tt.expected, "BuildSpecPath(\""+tt.version+"\")")
		})
	}
}
