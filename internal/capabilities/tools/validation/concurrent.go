package validation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/carlisia/mcp-factcheck/pkg/logger"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// WorkerPool manages concurrent execution with bounded parallelism
type WorkerPool struct {
	maxWorkers int
	semaphore  chan struct{}
}

// NewWorkerPool creates a new worker pool with the specified max workers
func NewWorkerPool(maxWorkers int) *WorkerPool {
	if maxWorkers <= 0 {
		maxWorkers = 1
	}
	return &WorkerPool{
		maxWorkers: maxWorkers,
		semaphore:  make(chan struct{}, maxWorkers),
	}
}

// acquire attempts to acquire a semaphore slot
func (wp *WorkerPool) acquire(ctx context.Context) error {
	select {
	case wp.semaphore <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// release releases a semaphore slot
func (wp *WorkerPool) release() {
	<-wp.semaphore
}

// Process executes work items concurrently with bounded parallelism
func (wp *WorkerPool) Process(ctx context.Context, work func(context.Context, int) error, count int) error {
	if count <= 0 {
		return nil
	}

	g, ctx := errgroup.WithContext(ctx)

	for i := 0; i < count; i++ {
		idx := i
		g.Go(func() error {
			// Add panic recovery
			defer func() {
				if r := recover(); r != nil {
					// Log panic for debugging
					logger.Get().Error("WorkerPool.Process recovered panic",
						zap.Any("panic", r),
						zap.Stack("stack"),
					)
				}
			}()

			// Acquire semaphore
			if err := wp.acquire(ctx); err != nil {
				return err
			}
			defer wp.release()

			// Execute work
			return work(ctx, idx)
		})
	}

	return g.Wait()
}

// ChunkWork represents a unit of work for chunk validation
type ChunkWork struct {
	Index       int
	Chunk       interface{}
	Embedding   []float64
	SpecVersion string
}

// ChunkResult represents the result of chunk validation
type ChunkResult struct {
	Index  int
	Result ChunkValidationResult
	Error  error
}

// ProcessChunks processes chunks concurrently and returns ordered results
func ProcessChunks(ctx context.Context, works []ChunkWork, processor func(context.Context, ChunkWork) (ChunkValidationResult, error), maxWorkers int) ([]ChunkValidationResult, error) {
	if len(works) == 0 {
		return []ChunkValidationResult{}, nil
	}

	// Create channels for work distribution and result collection
	workChan := make(chan ChunkWork, len(works))
	resultChan := make(chan ChunkResult, len(works))

	// Create worker pool
	var wg sync.WaitGroup
	pool := NewWorkerPool(maxWorkers)

	// Start workers with explicit IDs
	for workerID := 0; workerID < pool.maxWorkers && workerID < len(works); workerID++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					// Log panic with worker ID for better debugging
					logger.Get().Error("ProcessChunks worker recovered panic",
						zap.Int("worker_id", id),
						zap.Any("panic", r),
						zap.Stack("stack"),
					)
					select {
					case resultChan <- ChunkResult{
						Index: -1, // Signal panic
						Error: fmt.Errorf("worker %d panic: %v", id, r),
					}:
					case <-ctx.Done():
					}
				}
			}()

			logger.Get().Debug("Worker started",
				zap.Int("worker_id", id),
				zap.Int("total_works", len(works)),
			)

			for work := range workChan {
				// Immediate context check
				if ctx.Err() != nil {
					if ctx.Err() == context.Canceled {
						logger.Get().Info("Worker exiting due to context cancellation",
							zap.Int("worker_id", id),
						)
					}
					return
				}

				// Apply task-level timeout if configured
				taskCtx := ctx
				var cancel context.CancelFunc
				if config, ok := ctx.Value(ContextKeyConcurrencyConfig).(ConcurrencyConfig); ok && config.ProcessingTimeout > 0 {
					taskCtx, cancel = context.WithTimeout(ctx, config.ProcessingTimeout)
				}

				result, err := processor(taskCtx, work)

				if cancel != nil {
					cancel()
				}

				// Check for timeout explicitly
				if taskCtx.Err() == context.DeadlineExceeded {
					logger.Get().Warn("Task timed out",
						zap.Int("worker_id", id),
						zap.Int("work_index", work.Index),
						zap.Error(err),
					)
				}

				select {
				case resultChan <- ChunkResult{
					Index:  work.Index,
					Result: result,
					Error:  err,
				}:
				case <-ctx.Done():
					logger.Get().Info("Worker exiting due to context cancellation during result send",
						zap.Int("worker_id", id),
						zap.Int("work_index", work.Index),
					)
					return
				}
			}

			logger.Get().Debug("Worker stopped", zap.Int("worker_id", id))
		}(workerID)
	}

	// Send work to workers
	go func() {
		defer close(workChan)
		for _, work := range works {
			select {
			case workChan <- work:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results in order
	results := make([]ChunkValidationResult, len(works))

	// Collect all results directly into the slice
	for result := range resultChan {
		// Handle panic results
		if result.Index == -1 {
			// Worker panic already logged, skip this result
			logger.Get().Warn("Skipping result from panicked worker",
				zap.Error(result.Error),
			)
			continue // Result intentionally skipped due to worker panic
		}

		if result.Error != nil {
			results[result.Index] = ChunkValidationResult{
				Error: result.Error.Error(),
			}
		} else {
			results[result.Index] = result.Result
		}
	}

	return results, nil
}

// Default concurrency settings
const (
	DefaultMaxConcurrentChunks = 5
	DefaultMaxConcurrentClaims = 3
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

// Context keys for configuration
const (
	ContextKeyConcurrencyConfig contextKey = "concurrency_config"
	ContextKeyUseConcurrent     contextKey = "use_concurrent"
)

// ConcurrencyConfig holds configuration for concurrent operations
type ConcurrencyConfig struct {
	MaxConcurrentChunks  int
	MaxConcurrentClaims  int
	EnableBatchEmbedding bool
	ProcessingTimeout    time.Duration // Timeout for individual processing tasks
}

// DefaultConcurrencyConfig returns default concurrency configuration
func DefaultConcurrencyConfig() ConcurrencyConfig {
	return ConcurrencyConfig{
		MaxConcurrentChunks:  DefaultMaxConcurrentChunks,
		MaxConcurrentClaims:  DefaultMaxConcurrentClaims,
		EnableBatchEmbedding: true,
		ProcessingTimeout:    5 * time.Minute, // Default 5 minute timeout per task
	}
}
