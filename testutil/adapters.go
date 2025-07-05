package testutil

import (
	"context"
	"strings"

	"github.com/carlisia/mcp-factcheck/internal/embedding/core"
)

// VectorDBAdapter provides common adapter functionality for tests that need to
// adapt a testutil.VectorDB to production interfaces. This thin adapter enables
// reuse of test mocks across different handler interfaces.
type VectorDBAdapter struct {
	VectorDB
}

// Search delegates to the underlying mock VectorDB
func (a *VectorDBAdapter) Search(version string, queryEmbedding []float64, topK int) ([]core.SearchResult, error) {
	return a.VectorDB.Search(version, queryEmbedding, topK)
}

// EmbeddingGeneratorAdapter provides common adapter functionality for tests that need
// to adapt a testutil.EmbeddingGenerator to production interfaces.
type EmbeddingGeneratorAdapter struct {
	EmbeddingGenerator
}

// GenerateEmbedding delegates to the underlying mock
func (a *EmbeddingGeneratorAdapter) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	return a.EmbeddingGenerator.GenerateEmbedding(ctx, text)
}

// FactCheckingEmbeddingGeneratorAdapter extends EmbeddingGeneratorAdapter with
// fact-checking capabilities for handlers that require the FactCheckAgainstSpec method.
// This adapter provides minimal simulation logic for testing purposes.
type FactCheckingEmbeddingGeneratorAdapter struct {
	EmbeddingGeneratorAdapter
}

// FactCheckAgainstSpec provides minimal simulation of fact-checking behavior for testing.
// It uses simple heuristics based on the claim and spec sections to determine accuracy,
// simulating just enough logic to verify handler code paths without duplicating
// production fact-checking logic.
func (a *FactCheckingEmbeddingGeneratorAdapter) FactCheckAgainstSpec(ctx context.Context, claim string, specSections []string, compoundEvidence map[string]string) (*core.FactCheckResult, error) {
	// Try to generate embedding first - some tests expect this behavior
	_, err := a.GenerateEmbedding(ctx, claim)
	if err != nil {
		// Return uncertain result if embedding fails
		return &core.FactCheckResult{
			IsAccurate: false,
			Claims: []core.Claim{{
				Claim:       claim,
				IsAccurate:  false,
				Explanation: "Unable to verify claim",
			}},
		}, nil
	}

	// Simple heuristics for testing different code paths
	claimLower := strings.ToLower(claim)
	for _, section := range specSections {
		sectionLower := strings.ToLower(section)

		// Test accurate claims
		if strings.Contains(claimLower, "json-rpc") && strings.Contains(sectionLower, "json-rpc") {
			return &core.FactCheckResult{
				IsAccurate: true,
				Claims: []core.Claim{{
					Claim:      claim,
					IsAccurate: true,
				}},
			}, nil
		}

		// Test inaccurate claims
		if strings.Contains(claimLower, "xml") && strings.Contains(sectionLower, "json") && !strings.Contains(sectionLower, "xml") {
			return &core.FactCheckResult{
				IsAccurate: false,
				Claims: []core.Claim{{
					Claim:      claim,
					IsAccurate: false,
				}},
			}, nil
		}

		// Test unicode support
		if strings.Contains(claimLower, "unicode") && strings.Contains(sectionLower, "unicode") {
			return &core.FactCheckResult{
				IsAccurate: true,
				Claims: []core.Claim{{
					Claim:      claim,
					IsAccurate: true,
				}},
			}, nil
		}
	}

	// Default to uncertain/inaccurate
	return &core.FactCheckResult{
		IsAccurate: false,
		Claims: []core.Claim{{
			Claim:       claim,
			IsAccurate:  false,
			Explanation: "? Uncertain",
		}},
	}, nil
}

// NewVectorDBAdapter creates a VectorDBAdapter wrapping the given mock
func NewVectorDBAdapter(mock VectorDB) *VectorDBAdapter {
	return &VectorDBAdapter{VectorDB: mock}
}

// NewEmbeddingGeneratorAdapter creates an EmbeddingGeneratorAdapter wrapping the given mock
func NewEmbeddingGeneratorAdapter(mock EmbeddingGenerator) *EmbeddingGeneratorAdapter {
	return &EmbeddingGeneratorAdapter{EmbeddingGenerator: mock}
}

// NewFactCheckingEmbeddingGeneratorAdapter creates a FactCheckingEmbeddingGeneratorAdapter
// wrapping the given mock with fact-checking simulation capabilities
func NewFactCheckingEmbeddingGeneratorAdapter(mock EmbeddingGenerator) *FactCheckingEmbeddingGeneratorAdapter {
	return &FactCheckingEmbeddingGeneratorAdapter{
		EmbeddingGeneratorAdapter: EmbeddingGeneratorAdapter{EmbeddingGenerator: mock},
	}
}
