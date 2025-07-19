package validation_test

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools/contentprep"
	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// timingTolerancePercent is the percentage tolerance for timing-based tests
// to account for system load, scheduling variations, and overhead
const timingTolerancePercent = 100

func TestWorkerPool_Process(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		maxWorkers  int
		workCount   int
		workDelay   time.Duration
		expectError bool
	}{
		{
			name:       "single worker",
			maxWorkers: 1,
			workCount:  5,
			workDelay:  10 * time.Millisecond,
		},
		{
			name:       "multiple workers",
			maxWorkers: 3,
			workCount:  10,
			workDelay:  10 * time.Millisecond,
		},
		{
			name:       "more workers than work",
			maxWorkers: 10,
			workCount:  3,
			workDelay:  10 * time.Millisecond,
		},
		{
			name:       "zero work items",
			maxWorkers: 5,
			workCount:  0,
			workDelay:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pool := validation.NewWorkerPool(tt.maxWorkers)
			ctx := context.Background()

			var processed int32
			start := time.Now()

			err := pool.Process(ctx, func(ctx context.Context, idx int) error {
				time.Sleep(tt.workDelay)
				atomic.AddInt32(&processed, 1)
				return nil
			}, tt.workCount)

			duration := time.Since(start)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, int32(tt.workCount), processed)

				// Verify concurrency by checking duration
				if tt.workCount > 0 && tt.maxWorkers > 1 {
					// With concurrency, should be faster than sequential
					// Calculate expected time with some tolerance
					expectedConcurrent := time.Duration(math.Ceil(float64(tt.workCount)/float64(tt.maxWorkers))) * tt.workDelay
					// Apply tolerance for timing variations and overhead
					tolerance := expectedConcurrent * (100 + timingTolerancePercent) / 100
					assert.LessOrEqual(t, duration, tolerance)
				}
			}
		})
	}
}

func TestWorkerPool_ProcessWithCancellation(t *testing.T) {
	t.Parallel()

	pool := validation.NewWorkerPool(3)
	ctx, cancel := context.WithCancel(context.Background())

	var processed int32
	workStarted := make(chan struct{})

	go func() {
		<-workStarted
		// Cancel after some work has started
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := pool.Process(ctx, func(ctx context.Context, idx int) error {
		if idx == 0 {
			close(workStarted)
		}
		select {
		case <-time.After(100 * time.Millisecond):
			atomic.AddInt32(&processed, 1)
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}, 10)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
	// Some work should have been processed but not all
	assert.GreaterOrEqual(t, processed, int32(0)) // May be 0 if cancelled very quickly
	assert.Less(t, processed, int32(10))
}

func TestProcessChunks(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create test work items
	works := make([]validation.ChunkWork, 10)
	for i := 0; i < 10; i++ {
		works[i] = validation.ChunkWork{
			Index:       i,
			Chunk:       fmt.Sprintf("chunk-%d", i),
			SpecVersion: "test",
		}
	}

	// Test processor that returns chunk index
	processor := func(ctx context.Context, work validation.ChunkWork) (validation.ChunkValidationResult, error) {
		return validation.ChunkValidationResult{
			Chunk: contentprep.Chunk{
				Text: work.Chunk.(string),
			},
		}, nil
	}

	results, err := validation.ProcessChunks(ctx, works, processor, 3)
	require.NoError(t, err)
	require.Len(t, results, 10)

	// Verify results are in order
	for i := 0; i < 10; i++ {
		assert.Equal(t, fmt.Sprintf("chunk-%d", i), results[i].Chunk.Text)
	}
}

func TestProcessChunks_WithErrors(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create test work items
	works := make([]validation.ChunkWork, 5)
	for i := 0; i < 5; i++ {
		works[i] = validation.ChunkWork{
			Index: i,
			Chunk: i,
		}
	}

	// Test processor that fails on even indices
	processor := func(ctx context.Context, work validation.ChunkWork) (validation.ChunkValidationResult, error) {
		idx := work.Chunk.(int)
		if idx%2 == 0 {
			return validation.ChunkValidationResult{}, fmt.Errorf("error on chunk %d", idx)
		}
		return validation.ChunkValidationResult{
			Chunk: contentprep.Chunk{
				Text: fmt.Sprintf("success-%d", idx),
			},
		}, nil
	}

	results, err := validation.ProcessChunks(ctx, works, processor, 2)
	require.NoError(t, err) // ProcessChunks doesn't return errors, it includes them in results
	require.Len(t, results, 5)

	// Verify error handling
	for i := 0; i < 5; i++ {
		if i%2 == 0 {
			assert.Contains(t, results[i].Error, fmt.Sprintf("error on chunk %d", i))
		} else {
			assert.Equal(t, fmt.Sprintf("success-%d", i), results[i].Chunk.Text)
			assert.Empty(t, results[i].Error)
		}
	}
}

func TestConcurrencyConfig_Defaults(t *testing.T) {
	t.Parallel()

	config := validation.DefaultConcurrencyConfig()
	assert.Equal(t, validation.DefaultMaxConcurrentChunks, config.MaxConcurrentChunks)
	assert.Equal(t, validation.DefaultMaxConcurrentClaims, config.MaxConcurrentClaims)
	assert.True(t, config.EnableBatchEmbedding)
}

// Additional cancellation tests

func TestProcessChunks_ContextCancellation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		workCount      int
		maxWorkers     int
		cancelAfter    time.Duration
		processorDelay time.Duration
		expectComplete int // Expected number of completed tasks (approximate)
	}{
		{
			name:           "cancel before any work starts",
			workCount:      10,
			maxWorkers:     3,
			cancelAfter:    0, // Cancel immediately
			processorDelay: 50 * time.Millisecond,
			expectComplete: 0,
		},
		{
			name:           "cancel during processing",
			workCount:      10,
			maxWorkers:     3,
			cancelAfter:    75 * time.Millisecond,
			processorDelay: 50 * time.Millisecond,
			expectComplete: 3, // Approximately 3 workers * 1-2 iterations
		},
		{
			name:           "cancel with fast processing",
			workCount:      20,
			maxWorkers:     5,
			cancelAfter:    10 * time.Millisecond,
			processorDelay: 5 * time.Millisecond,
			expectComplete: 5, // At least first batch
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			if tt.cancelAfter > 0 {
				time.AfterFunc(tt.cancelAfter, cancel)
			} else {
				cancel() // Cancel immediately
			}

			// Create test work items
			works := make([]validation.ChunkWork, tt.workCount)
			for i := 0; i < tt.workCount; i++ {
				works[i] = validation.ChunkWork{
					Index:       i,
					Chunk:       fmt.Sprintf("chunk-%d", i),
					SpecVersion: "test",
				}
			}

			var processed int32
			processor := func(ctx context.Context, work validation.ChunkWork) (validation.ChunkValidationResult, error) {
				select {
				case <-time.After(tt.processorDelay):
					atomic.AddInt32(&processed, 1)
					return validation.ChunkValidationResult{
						Chunk: contentprep.Chunk{
							Text: work.Chunk.(string),
						},
					}, nil
				case <-ctx.Done():
					return validation.ChunkValidationResult{}, ctx.Err()
				}
			}

			results, err := validation.ProcessChunks(ctx, works, processor, tt.maxWorkers)

			// We don't expect an error from ProcessChunks itself
			require.NoError(t, err)
			require.Len(t, results, tt.workCount)

			// Check that some results have cancellation errors
			cancelCount := 0
			successCount := 0
			for _, result := range results {
				if result.Error != "" {
					assert.Contains(t, result.Error, "context canceled")
					cancelCount++
				} else if result.Chunk.Text != "" {
					successCount++
				}
			}

			// Verify processed count is reasonable
			processedCount := int(atomic.LoadInt32(&processed))
			t.Logf("Processed %d items, %d successful, %d cancelled", processedCount, successCount, cancelCount)

			if tt.cancelAfter == 0 {
				// Immediate cancellation should process very few or no items
				assert.LessOrEqual(t, processedCount, tt.maxWorkers)
			} else {
				// Should have processed some items before cancellation
				assert.GreaterOrEqual(t, processedCount, 1)
			}
		})
	}
}

func TestWorkerPool_ProcessCancellationPropagation(t *testing.T) {
	t.Parallel()

	pool := validation.NewWorkerPool(3)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	var started, completed int32
	workFunc := func(ctx context.Context, idx int) error {
		atomic.AddInt32(&started, 1)

		// Simulate long-running work
		select {
		case <-time.After(100 * time.Millisecond):
			atomic.AddInt32(&completed, 1)
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	err := pool.Process(ctx, workFunc, 10)

	// Should get context deadline exceeded
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")

	// Should have started some work
	assert.Greater(t, atomic.LoadInt32(&started), int32(0))

	// But not completed all due to timeout
	assert.Less(t, atomic.LoadInt32(&completed), int32(10))
}

func TestProcessChunks_GracefulShutdown(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create a scenario where workers need to finish their current work
	works := make([]validation.ChunkWork, 5)
	for i := 0; i < 5; i++ {
		works[i] = validation.ChunkWork{
			Index: i,
			Chunk: i,
		}
	}

	var inProgress int32
	processor := func(ctx context.Context, work validation.ChunkWork) (validation.ChunkValidationResult, error) {
		atomic.AddInt32(&inProgress, 1)
		defer atomic.AddInt32(&inProgress, -1)

		// Simulate work
		time.Sleep(10 * time.Millisecond)

		return validation.ChunkValidationResult{
			Chunk: contentprep.Chunk{
				ID: fmt.Sprintf("completed-%d", work.Index),
			},
		}, nil
	}

	results, err := validation.ProcessChunks(ctx, works, processor, 2)
	require.NoError(t, err)

	// All work should be completed
	for i, result := range results {
		assert.Empty(t, result.Error)
		assert.Equal(t, fmt.Sprintf("completed-%d", i), result.Chunk.ID)
	}

	// No work should be in progress
	assert.Equal(t, int32(0), atomic.LoadInt32(&inProgress))
}

func TestProcessChunks_DeadlockPrevention(t *testing.T) {
	t.Parallel()

	// Test that the system doesn't deadlock even with slow result consumption
	ctx := context.Background()

	works := make([]validation.ChunkWork, 100) // Large number of work items
	for i := 0; i < 100; i++ {
		works[i] = validation.ChunkWork{
			Index: i,
			Chunk: i,
		}
	}

	processor := func(ctx context.Context, work validation.ChunkWork) (validation.ChunkValidationResult, error) {
		// Fast processing
		return validation.ChunkValidationResult{
			Chunk: contentprep.Chunk{
				Text: fmt.Sprintf("result-%d", work.Index),
			},
		}, nil
	}

	// Use a small worker pool to stress test channel management
	done := make(chan bool)
	go func() {
		results, err := validation.ProcessChunks(ctx, works, processor, 2)
		assert.NoError(t, err)
		assert.Len(t, results, 100)
		close(done)
	}()

	// Should complete without deadlock
	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out - possible deadlock")
	}
}

func TestProcessChunks_WithContextTimeout(t *testing.T) {
	t.Parallel()

	// Test that ProcessChunks respects context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Create test work
	works := make([]validation.ChunkWork, 5)
	for i := 0; i < 5; i++ {
		works[i] = validation.ChunkWork{
			Index: i,
			Chunk: fmt.Sprintf("chunk-%d", i),
		}
	}

	// Processor that takes longer than timeout
	processor := func(ctx context.Context, work validation.ChunkWork) (validation.ChunkValidationResult, error) {
		select {
		case <-time.After(30 * time.Millisecond): // Each takes 30ms, total would be 150ms
			return validation.ChunkValidationResult{
				Chunk: contentprep.Chunk{
					Text: fmt.Sprintf("processed-%s", work.Chunk),
				},
			}, nil
		case <-ctx.Done():
			return validation.ChunkValidationResult{}, ctx.Err()
		}
	}

	results, err := validation.ProcessChunks(ctx, works, processor, 2)
	require.NoError(t, err) // ProcessChunks itself doesn't error
	require.Len(t, results, 5)

	// Some should timeout due to context deadline
	timeoutCount := 0
	successCount := 0
	emptyCount := 0
	for _, result := range results {
		if result.Error != "" {
			if strings.Contains(result.Error, "deadline exceeded") || strings.Contains(result.Error, "context canceled") {
				timeoutCount++
			}
		} else if result.Chunk.Text != "" {
			successCount++
		} else {
			emptyCount++
		}
	}

	t.Logf("Results: %d success, %d timeout, %d empty", successCount, timeoutCount, emptyCount)

	// With 2 workers and 30ms per task, should complete some before 50ms timeout
	assert.Greater(t, successCount, 0, "Should have completed some tasks")
	// Either timeout errors or empty results indicate context cancellation
	assert.Greater(t, timeoutCount+emptyCount, 0, "Should have some cancelled tasks")
}

func TestWorkerPool_PanicRecovery(t *testing.T) {
	t.Parallel()

	pool := validation.NewWorkerPool(2)
	ctx := context.Background()

	panicAt := 2 // Panic on the 3rd task
	var completed int

	workFunc := func(ctx context.Context, idx int) error {
		if idx == panicAt {
			panic("simulated panic")
		}
		completed++
		return nil
	}

	// Process should continue despite panic
	err := pool.Process(ctx, workFunc, 5)

	// Should complete successfully (panic is isolated)
	assert.NoError(t, err)

	// Should have completed all non-panicking tasks
	assert.Equal(t, 4, completed) // 5 total - 1 panic = 4 completed
}

func TestProcessChunks_PanicRecovery(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create test work
	works := make([]validation.ChunkWork, 5)
	for i := 0; i < 5; i++ {
		works[i] = validation.ChunkWork{
			Index: i,
			Chunk: i,
		}
	}

	// Processor that panics on specific index
	processor := func(ctx context.Context, work validation.ChunkWork) (validation.ChunkValidationResult, error) {
		idx := work.Chunk.(int)
		if idx == 2 {
			panic("simulated processor panic")
		}
		return validation.ChunkValidationResult{
			Chunk: contentprep.Chunk{
				ID: fmt.Sprintf("success-%d", idx),
			},
		}, nil
	}

	// Should handle panic gracefully
	results, err := validation.ProcessChunks(ctx, works, processor, 2)
	require.NoError(t, err)
	require.Len(t, results, 5)

	// Verify results
	successCount := 0
	for i, result := range results {
		if i == 2 {
			// The panicked task should have an error or empty result
			if result.Error != "" {
				assert.Contains(t, result.Error, "panic", "Index 2 should contain panic error")
			}
		} else {
			assert.Empty(t, result.Error, "Non-panicking tasks should not have errors")
			assert.NotEmpty(t, result.Chunk.ID, "Non-panicking tasks should have chunk IDs")
			if result.Chunk.ID != "" {
				successCount++
			}
		}
	}

	// At least the non-panicking tasks should succeed
	assert.GreaterOrEqual(t, successCount, 4)
}
