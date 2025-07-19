package validation

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools"
	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools/contentprep"
)

// Mock functions for benchmarking
// mockEmbedFunc simulates individual embedding latency (~50ms per chunk)
func mockEmbedFunc(ctx context.Context, text string) ([]float64, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Simulate embedding generation time
	time.Sleep(50 * time.Millisecond)
	return make([]float64, 1536), nil
}

// mockBatchEmbedFunc simulates batch embedding latency (30ms per chunk, faster than individual embedding)
func mockBatchEmbedFunc(ctx context.Context, texts []string) ([][]float64, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Simulate batch embedding time - more efficient than individual
	time.Sleep(30 * time.Millisecond * time.Duration(len(texts)))

	embeddings := make([][]float64, len(texts))
	for i := range texts {
		embeddings[i] = make([]float64, 1536)
	}
	return embeddings, nil
}

// mockSearchFunc simulates search latency (~20ms per search)
func mockSearchFunc(version string, embedding []float64, topK int) ([]tools.SearchResult, error) {
	// Note: In real usage, this would receive context from the validation functions
	// For benchmarking, we simulate the search without context checks

	// Simulate search time
	time.Sleep(20 * time.Millisecond)
	results := make([]tools.SearchResult, topK)
	for i := 0; i < topK; i++ {
		results[i] = tools.SearchResult{
			Content:    fmt.Sprintf("Mock result %d", i),
			Similarity: float64(topK-i) / float64(topK),
		}
	}
	return results, nil
}

// mockLLMFunc simulates LLM call latency (~100ms per call)
func mockLLMFunc(ctx context.Context, prompt string, temperature float64, maxTokens int) (string, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	// Simulate LLM call time
	time.Sleep(100 * time.Millisecond)
	return `{
		"is_accurate": true,
		"confidence": 0.95,
		"claims": ["Test claim"],
		"inaccuracies": [],
		"corrections": [],
		"missing_best_practices": [],
		"suggestions": [],
		"explanation": "Test validation"
	}`, nil
}

func createTestChunks(count int) []contentprep.Chunk {
	chunks := make([]contentprep.Chunk, count)
	for i := 0; i < count; i++ {
		chunks[i] = contentprep.Chunk{
			ID:       fmt.Sprintf("chunk-%d", i),
			Text:     fmt.Sprintf("This is test chunk %d with some content about MCP.", i),
			Position: i,
			Type:     "paragraph",
		}
	}
	return chunks
}

func BenchmarkChunkValidation(b *testing.B) {
	// Test with various chunk sizes including larger ones
	chunkCounts := []int{5, 10, 20, 50, 100}

	for _, count := range chunkCounts {
		// Sequential processing benchmark
		b.Run(fmt.Sprintf("Sequential_IndividualEmbedding_%d_chunks", count), func(b *testing.B) {
			chunks := createTestChunks(count)
			// Add realistic timeout
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := validateChunksBatch(ctx, chunks, "test", mockEmbedFunc, mockSearchFunc, mockLLMFunc)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		// Concurrent processing with individual embeddings
		b.Run(fmt.Sprintf("Concurrent_IndividualEmbedding_%d_chunks", count), func(b *testing.B) {
			chunks := createTestChunks(count)
			// Add realistic timeout
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			ctx = context.WithValue(ctx, ContextKeyConcurrencyConfig, ConcurrencyConfig{
				MaxConcurrentChunks:  5,
				EnableBatchEmbedding: false,            // Test individual embedding concurrency
				ProcessingTimeout:    30 * time.Second, // Generous timeout for realistic LLM and search latencies
			})

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := validateChunksBatchConcurrent(ctx, chunks, "test", mockEmbedFunc, mockSearchFunc, mockLLMFunc)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		// Concurrent processing with batch embeddings
		b.Run(fmt.Sprintf("Concurrent_BatchEmbedding_%d_chunks", count), func(b *testing.B) {
			chunks := createTestChunks(count)
			// Add realistic timeout
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			// Add batch embedding function to context
			ctx = context.WithValue(ctx, tools.ContextKeyBatchEmbedFunc, tools.BatchEmbeddingFunc(mockBatchEmbedFunc))
			ctx = context.WithValue(ctx, ContextKeyConcurrencyConfig, ConcurrencyConfig{
				MaxConcurrentChunks:  5,
				EnableBatchEmbedding: true,             // Enable batch embedding
				ProcessingTimeout:    30 * time.Second, // Generous timeout for realistic LLM and search latencies
			})

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := validateChunksBatchConcurrent(ctx, chunks, "test", mockEmbedFunc, mockSearchFunc, mockLLMFunc)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Run benchmarks with:
// go test -bench=BenchmarkChunkValidation -benchtime=1x ./internal/capabilities/tools/validation
// For specific scenarios:
// go test -bench=BenchmarkChunkValidation/Sequential -benchtime=1x ./internal/capabilities/tools/validation
// go test -bench=BenchmarkChunkValidation/Concurrent_Individual -benchtime=1x ./internal/capabilities/tools/validation
// go test -bench=BenchmarkChunkValidation/Concurrent_Batch -benchtime=1x ./internal/capabilities/tools/validation
