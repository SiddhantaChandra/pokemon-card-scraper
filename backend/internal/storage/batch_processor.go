package storage

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/SiddhantaChandra/pokemon-card-scraper/pkg/models"
)

// BatchProcessor handles batched database operations for improved performance
type BatchProcessor struct {
	storage       Storage
	cardBuffer    []models.Card
	batchSize     int
	flushInterval time.Duration

	// Synchronization
	mu     sync.Mutex
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc

	// Channels for communication
	cardChannel  chan models.Card
	flushChannel chan struct{}

	// Statistics
	stats   BatchProcessorStats
	statsMu sync.RWMutex

	// Configuration
	config *BatchProcessorConfig
}

// BatchProcessorConfig holds configuration for batch processing
type BatchProcessorConfig struct {
	BatchSize     int
	FlushInterval time.Duration
	MaxBufferSize int
	AutoFlush     bool
	MaxRetries    int
	RetryDelay    time.Duration
	EnableMetrics bool
}

// BatchProcessorStats tracks batch processing statistics
type BatchProcessorStats struct {
	TotalCards        int64     `json:"total_cards"`
	TotalBatches      int64     `json:"total_batches"`
	SuccessfulBatches int64     `json:"successful_batches"`
	FailedBatches     int64     `json:"failed_batches"`
	AverageBatchSize  float64   `json:"average_batch_size"`
	LastFlushTime     time.Time `json:"last_flush_time"`
	BufferSize        int       `json:"buffer_size"`
	ProcessingTime    float64   `json:"processing_time_ms"`
}

// DefaultBatchProcessorConfig returns sensible defaults for batch processing
func DefaultBatchProcessorConfig() *BatchProcessorConfig {
	return &BatchProcessorConfig{
		BatchSize:     100,
		FlushInterval: 5 * time.Second,
		MaxBufferSize: 1000,
		AutoFlush:     true,
		MaxRetries:    3,
		RetryDelay:    time.Second,
		EnableMetrics: true,
	}
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(storage Storage, config *BatchProcessorConfig) *BatchProcessor {
	if config == nil {
		config = DefaultBatchProcessorConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	bp := &BatchProcessor{
		storage:       storage,
		cardBuffer:    make([]models.Card, 0, config.BatchSize),
		batchSize:     config.BatchSize,
		flushInterval: config.FlushInterval,
		ctx:           ctx,
		cancel:        cancel,
		cardChannel:   make(chan models.Card, config.MaxBufferSize),
		flushChannel:  make(chan struct{}, 1),
		config:        config,
		stats: BatchProcessorStats{
			LastFlushTime: time.Now(),
		},
	}

	// Start background processing if auto flush is enabled
	if config.AutoFlush {
		bp.wg.Add(1)
		go bp.processLoop()
	}

	return bp
}

// AddCard adds a card to the batch for processing
func (bp *BatchProcessor) AddCard(card models.Card) error {
	select {
	case bp.cardChannel <- card:
		return nil
	case <-bp.ctx.Done():
		return bp.ctx.Err()
	default:
		// Channel is full, try to flush and retry
		bp.triggerFlush()

		select {
		case bp.cardChannel <- card:
			return nil
		case <-bp.ctx.Done():
			return bp.ctx.Err()
		default:
			return ErrBufferFull
		}
	}
}

// AddCardsBatch adds multiple cards to the batch
func (bp *BatchProcessor) AddCardsBatch(cards []models.Card) error {
	for _, card := range cards {
		if err := bp.AddCard(card); err != nil {
			return err
		}
	}
	return nil
}

// Flush manually flushes any pending cards in the buffer
func (bp *BatchProcessor) Flush() error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if len(bp.cardBuffer) == 0 {
		return nil
	}

	return bp.flushBuffer()
}

// triggerFlush sends a flush signal to the processing loop
func (bp *BatchProcessor) triggerFlush() {
	select {
	case bp.flushChannel <- struct{}{}:
	default:
		// Flush already triggered
	}
}

// processLoop runs the main processing loop
func (bp *BatchProcessor) processLoop() {
	defer bp.wg.Done()

	ticker := time.NewTicker(bp.flushInterval)
	defer ticker.Stop()

	log.Println("Batch processor started")
	defer log.Println("Batch processor stopped")

	for {
		select {
		case card, ok := <-bp.cardChannel:
			if !ok {
				// Channel closed, flush remaining cards and exit
				bp.Flush()
				return
			}

			bp.mu.Lock()
			bp.cardBuffer = append(bp.cardBuffer, card)
			shouldFlush := len(bp.cardBuffer) >= bp.batchSize
			bp.mu.Unlock()

			if shouldFlush {
				bp.flushBuffer()
			}

		case <-bp.flushChannel:
			bp.flushBuffer()

		case <-ticker.C:
			// Periodic flush
			bp.flushBuffer()

		case <-bp.ctx.Done():
			// Context cancelled, flush remaining cards and exit
			bp.Flush()
			return
		}
	}
}

// flushBuffer flushes the current buffer to storage
func (bp *BatchProcessor) flushBuffer() error {
	bp.mu.Lock()
	if len(bp.cardBuffer) == 0 {
		bp.mu.Unlock()
		return nil
	}

	// Copy buffer to avoid holding lock during database operation
	batch := make([]models.Card, len(bp.cardBuffer))
	copy(batch, bp.cardBuffer)
	bp.cardBuffer = bp.cardBuffer[:0] // Clear buffer
	bp.mu.Unlock()

	// Perform database operation
	startTime := time.Now()
	err := bp.saveBatchWithRetry(batch)
	duration := time.Since(startTime)

	// Update statistics
	bp.updateStats(len(batch), err == nil, duration)

	if err != nil {
		log.Printf("Failed to save batch of %d cards: %v", len(batch), err)
		return err
	}

	log.Printf("Successfully saved batch of %d cards in %v", len(batch), duration)
	return nil
}

// saveBatchWithRetry saves a batch with retry logic
func (bp *BatchProcessor) saveBatchWithRetry(batch []models.Card) error {
	var lastErr error

	for attempt := 0; attempt < bp.config.MaxRetries; attempt++ {
		if err := bp.saveBatch(batch); err != nil {
			lastErr = err
			log.Printf("Batch save attempt %d failed: %v", attempt+1, err)

			// Wait before retrying
			if attempt < bp.config.MaxRetries-1 {
				time.Sleep(bp.config.RetryDelay * time.Duration(attempt+1))
			}
			continue
		}

		return nil // Success
	}

	return lastErr
}

// saveBatch saves a batch of cards to storage
func (bp *BatchProcessor) saveBatch(batch []models.Card) error {
	// Check if the storage supports batch operations
	if batchStorage, ok := bp.storage.(BatchStorage); ok {
		return batchStorage.SaveCardsBatch(batch)
	}

	// Fallback to individual saves within a transaction-like operation
	for _, card := range batch {
		if err := bp.storage.SaveCard(card); err != nil {
			return err
		}
	}

	return nil
}

// updateStats updates processing statistics
func (bp *BatchProcessor) updateStats(batchSize int, success bool, duration time.Duration) {
	if !bp.config.EnableMetrics {
		return
	}

	bp.statsMu.Lock()
	defer bp.statsMu.Unlock()

	bp.stats.TotalCards += int64(batchSize)
	bp.stats.TotalBatches++

	if success {
		bp.stats.SuccessfulBatches++
	} else {
		bp.stats.FailedBatches++
	}

	bp.stats.LastFlushTime = time.Now()
	bp.stats.ProcessingTime = float64(duration.Nanoseconds()) / 1e6

	// Calculate average batch size
	if bp.stats.TotalBatches > 0 {
		bp.stats.AverageBatchSize = float64(bp.stats.TotalCards) / float64(bp.stats.TotalBatches)
	}

	// Update current buffer size
	bp.mu.Lock()
	bp.stats.BufferSize = len(bp.cardBuffer)
	bp.mu.Unlock()
}

// GetStats returns current batch processing statistics
func (bp *BatchProcessor) GetStats() BatchProcessorStats {
	bp.statsMu.RLock()
	defer bp.statsMu.RUnlock()

	// Update buffer size
	bp.mu.Lock()
	stats := bp.stats
	stats.BufferSize = len(bp.cardBuffer)
	bp.mu.Unlock()

	return stats
}

// Close gracefully shuts down the batch processor
func (bp *BatchProcessor) Close() error {
	log.Println("Shutting down batch processor...")

	// Cancel context to stop processing loop
	bp.cancel()

	// Close card channel to signal shutdown
	close(bp.cardChannel)

	// Wait for processing loop to finish
	bp.wg.Wait()

	// Flush any remaining cards
	if err := bp.Flush(); err != nil {
		log.Printf("Error during final flush: %v", err)
		return err
	}

	log.Println("Batch processor shut down successfully")
	return nil
}

// IsHealthy returns true if the batch processor is operating normally
func (bp *BatchProcessor) IsHealthy() bool {
	bp.statsMu.RLock()
	defer bp.statsMu.RUnlock()

	// Check if too many batches are failing
	if bp.stats.TotalBatches > 10 {
		failureRate := float64(bp.stats.FailedBatches) / float64(bp.stats.TotalBatches)
		if failureRate > 0.1 { // More than 10% failure rate
			return false
		}
	}

	// Check if buffer is too full
	bp.mu.Lock()
	bufferUtilization := float64(len(bp.cardBuffer)) / float64(bp.config.MaxBufferSize)
	bp.mu.Unlock()

	if bufferUtilization > 0.9 { // More than 90% full
		return false
	}

	return true
}

// BatchStorage interface for storages that support batch operations
type BatchStorage interface {
	SaveCardsBatch(cards []models.Card) error
	UpdateCardsBatch(updates []CardUpdate) error
}

// Custom errors
var (
	ErrBufferFull = fmt.Errorf("batch processor buffer is full")
)
