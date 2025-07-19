package validation

import (
	"context"
	"errors"
	"fmt"

	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools"
	"github.com/carlisia/mcp-factcheck/internal/capabilities/tools/contentprep"
	"github.com/carlisia/mcp-factcheck/pkg/logger"
	"go.uber.org/zap"
)

// Standardized errors for consistent error handling
var (
	ErrEmbedFailure      = errors.New("embedding generation failed")
	ErrSearchFailure     = errors.New("search specification failed")
	ErrValidationFail    = errors.New("validation failed")
	ErrValidationTimeout = errors.New("validation timeout")
	ErrNoEmbedding       = errors.New("no embedding available for chunk")
)

// validateChunksBatchConcurrent validates multiple chunks concurrently
// It handles embedding generation, concurrent validation, and error aggregation
func validateChunksBatchConcurrent(ctx context.Context, chunks []contentprep.Chunk, specVersion string,
	embedFunc tools.EmbeddingFunc, searchFunc tools.SearchFunc, llmFunc LLMCompleteFunc) ([]ChunkValidationResult, error) {

	// 1. Fetch configuration
	config := fetchConcurrencyConfig(ctx)

	// 2. Generate embeddings (batch or concurrent)
	embeddings, embeddingErrors := generateChunkEmbeddings(ctx, chunks, embedFunc, config)

	// 3. Create work items from embeddings
	works := createChunkWork(chunks, embeddings, specVersion)

	// 4. Concurrently process chunks
	processor := func(ctx context.Context, work ChunkWork) (ChunkValidationResult, error) {
		return processChunk(ctx, work, embeddingErrors, embedFunc, searchFunc, llmFunc, config)
	}

	results, err := ProcessChunks(ctx, works, processor, config.MaxConcurrentChunks)
	if err != nil {
		return nil, fmt.Errorf("concurrent chunk processing failed: %w", err)
	}

	return results, nil
}

// fetchConcurrencyConfig retrieves concurrency configuration from context or returns defaults
func fetchConcurrencyConfig(ctx context.Context) ConcurrencyConfig {
	config := DefaultConcurrencyConfig()
	if cfg, ok := ctx.Value(ContextKeyConcurrencyConfig).(ConcurrencyConfig); ok {
		config = cfg
	}
	return config
}

// generateChunkEmbeddings generates embeddings for all chunks using batch or concurrent processing
func generateChunkEmbeddings(ctx context.Context, chunks []contentprep.Chunk,
	embedFunc tools.EmbeddingFunc, config ConcurrencyConfig) ([][]float64, []error) {

	// Extract text from all chunks
	chunkTexts := make([]string, len(chunks))
	for i, chunk := range chunks {
		chunkTexts[i] = chunk.Text
	}

	// Initialize result slices
	embeddings := make([][]float64, len(chunks))
	embeddingErrors := make([]error, len(chunks))

	// Try to get batch embed function from context
	batchEmbedFunc, hasBatch := ctx.Value(tools.ContextKeyBatchEmbedFunc).(tools.BatchEmbeddingFunc)
	if hasBatch && config.EnableBatchEmbedding && len(chunks) > 1 {
		// Use batch embedding for better performance
		batchEmbeddings, err := batchEmbedFunc(ctx, chunkTexts)
		if err != nil {
			// If batch fails, fall back to concurrent individual embedding
			logger.Get().Debug("Batch embedding failed, falling back to concurrent individual embedding",
				zap.Error(err))
			hasBatch = false
		} else {
			embeddings = batchEmbeddings
		}
	}

	// If batch not available or failed, use concurrent embedding
	if !hasBatch || len(chunks) == 1 {
		// Use concurrent embedding with rate limiting
		// Note: Each goroutine writes to its own index, so no mutex needed
		pool := NewWorkerPool(config.MaxConcurrentChunks)

		err := pool.Process(ctx, func(ctx context.Context, i int) error {
			embedding, err := embedFunc(ctx, chunkTexts[i])
			if err != nil {
				// Each goroutine writes to its own index - thread safe
				embeddingErrors[i] = fmt.Errorf("%w: chunk %d: %v", ErrEmbedFailure, i, err)
				return nil // Don't fail the entire batch
			}
			// Each goroutine writes to its own index - thread safe
			embeddings[i] = embedding
			return nil
		}, len(chunks))

		if err != nil && err != context.Canceled {
			// Log partial embedding completion - partial results are still valid for processing
			logger.Get().Info("Embedding completed with some errors; proceeding with partial results",
				zap.Error(err),
				zap.Int("total_chunks", len(chunks)),
				zap.String("note", "Partial embeddings are valid for downstream processing"),
			)
		}
	}

	// Log summary if there were any embedding errors
	var errorCount int
	for _, err := range embeddingErrors {
		if err != nil {
			errorCount++
		}
	}
	if errorCount > 0 {
		logger.Get().Warn("Embedding generation completed with errors",
			zap.Int("error_count", errorCount),
			zap.Int("total_chunks", len(chunks)),
		)
	}

	return embeddings, embeddingErrors
}

// createChunkWork creates work items for concurrent processing
func createChunkWork(chunks []contentprep.Chunk, embeddings [][]float64, specVersion string) []ChunkWork {
	works := make([]ChunkWork, len(chunks))
	for i, chunk := range chunks {
		works[i] = ChunkWork{
			Index:       i,
			Chunk:       chunk,
			Embedding:   embeddings[i],
			SpecVersion: specVersion,
		}
	}
	return works
}

// processChunk processes a single chunk with validation
func processChunk(ctx context.Context, work ChunkWork, embeddingErrors []error,
	embedFunc tools.EmbeddingFunc, searchFunc tools.SearchFunc, llmFunc LLMCompleteFunc,
	config ConcurrencyConfig) (ChunkValidationResult, error) {

	// Apply timeout if configured
	if config.ProcessingTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, config.ProcessingTimeout)
		defer cancel()
	}

	// Safe type assertion with error handling
	chunk, ok := work.Chunk.(contentprep.Chunk)
	if !ok {
		logger.Get().Error("Chunk type assertion failed",
			zap.Int("index", work.Index),
			zap.String("actual_type", fmt.Sprintf("%T", work.Chunk)),
		)
		return ChunkValidationResult{
			Error: fmt.Sprintf("chunk type assertion failed: expected contentprep.Chunk, got %T", work.Chunk),
		}, nil
	}

	// Check if this chunk had an embedding error
	if embeddingErrors[work.Index] != nil {
		return ChunkValidationResult{
			Chunk: chunk,
			Error: embeddingErrors[work.Index].Error(),
		}, nil
	}

	// Skip chunks without embeddings
	if len(work.Embedding) == 0 {
		// Log empty embeddings for diagnostics
		logger.Get().Warn("Empty embedding detected",
			zap.Int("chunk_index", work.Index),
			zap.String("chunk_id", chunk.ID),
		)
		return ChunkValidationResult{
			Chunk: chunk,
			Error: ErrNoEmbedding.Error(),
		}, nil
	}

	// Search for relevant spec sections
	searchResults, err := performSearch(ctx, chunk, work, embedFunc, searchFunc)
	if err != nil {
		return ChunkValidationResult{
			Chunk: chunk,
			Error: fmt.Errorf("%w: %v", ErrSearchFailure, err).Error(),
		}, nil
	}

	// Perform fact-checking on this chunk
	factCheckResult, err := performClaimCheck(ctx, llmFunc, chunk.Text, searchResults)
	if err != nil {
		// Handle context errors explicitly
		// Use switch for cleaner context error handling
		switch ctx.Err() {
		case context.DeadlineExceeded:
			return ChunkValidationResult{
				Chunk: chunk,
				Error: fmt.Errorf("%w after %v: %v", ErrValidationTimeout, config.ProcessingTimeout, err).Error(),
			}, nil
		case context.Canceled:
			return ChunkValidationResult{
				Chunk: chunk,
				Error: "validation canceled",
			}, nil
		}
		return ChunkValidationResult{
			Chunk: chunk,
			Error: fmt.Errorf("%w: %v", ErrValidationFail, err).Error(),
		}, nil
	}

	// Build validation result for this chunk
	result := &Result{
		IsValid:          factCheckResult.IsAccurate,
		Confidence:       factCheckResult.Confidence,
		ParsedClaims:     factCheckResult.ParsedClaims,
		Issues:           factCheckResult.Inaccuracies,
		Suggestions:      factCheckResult.Suggestions,
		CorrectedVersion: factCheckResult.CorrectedVersion,
		SpecVersion:      work.SpecVersion,
		FactCheckResult:  factCheckResult,
	}

	return ChunkValidationResult{
		Chunk:        chunk,
		Validation:   result,
		SearchResult: searchResults,
	}, nil
}

// performSearch executes the appropriate search strategy based on the chunk content
func performSearch(ctx context.Context, chunk contentprep.Chunk, work ChunkWork,
	embedFunc tools.EmbeddingFunc, searchFunc tools.SearchFunc) ([]tools.SearchResult, error) {

	// Negative claims require special handling to search for the absence of information
	// Regular claims can use the pre-computed embedding for efficiency
	if isNegativeClaim(chunk.Text) {
		// Use enhanced search for negative claims that looks for absence of information
		return searchForNegativeClaim(ctx, chunk.Text, work.SpecVersion, embedFunc, searchFunc)
	}

	// Use the pre-computed embedding for regular search
	return searchFunc(work.SpecVersion, work.Embedding, chunkSearchTopK)
}
