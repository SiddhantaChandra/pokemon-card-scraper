package scraper

import (
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
)

// CollectorPool manages a pool of colly collectors for reuse
type CollectorPool struct {
	collectors chan *colly.Collector
	maxSize    int
	config     *CollectorConfig
	mu         sync.RWMutex
	created    int
	inUse      int
}

// NewCollectorPool creates a new collector pool
func NewCollectorPool(maxSize int, config *CollectorConfig) *CollectorPool {
	if config == nil {
		config = DefaultCollectorConfig()
	}

	pool := &CollectorPool{
		collectors: make(chan *colly.Collector, maxSize),
		maxSize:    maxSize,
		config:     config,
	}

	// Pre-populate pool with initial collectors
	initialSize := maxSize / 2
	if initialSize < 1 {
		initialSize = 1
	}

	for i := 0; i < initialSize; i++ {
		collector := pool.createCollector()
		pool.collectors <- collector
		pool.created++
	}

	return pool
}

// Get retrieves a collector from the pool
func (cp *CollectorPool) Get() *colly.Collector {
	cp.mu.Lock()
	cp.inUse++
	cp.mu.Unlock()

	select {
	case collector := <-cp.collectors:
		// Reset the collector for reuse
		cp.resetCollector(collector)
		return collector
	default:
		// Pool is empty, create a new one if under limit
		cp.mu.Lock()
		if cp.created < cp.maxSize {
			cp.created++
			cp.mu.Unlock()
			return cp.createCollector()
		}
		cp.mu.Unlock()

		// Pool is at max size, wait for one to become available
		collector := <-cp.collectors
		cp.resetCollector(collector)
		return collector
	}
}

// Put returns a collector to the pool
func (cp *CollectorPool) Put(collector *colly.Collector) {
	if collector == nil {
		return
	}

	cp.mu.Lock()
	cp.inUse--
	cp.mu.Unlock()

	// Clean up the collector before returning to pool
	cp.cleanupCollector(collector)

	select {
	case cp.collectors <- collector:
		// Successfully returned to pool
	default:
		// Pool is full, discard the collector
		// This shouldn't happen often if pool is sized correctly
	}
}

// createCollector creates a new collector with the pool's configuration
func (cp *CollectorPool) createCollector() *colly.Collector {
	collector := NewCollector(cp.config)

	// Set up common configurations that won't change between uses
	collector.AllowedDomains = []string{"torecacamp-pokemon.com"}

	// Set up rate limiting for the collector
	collector.Limit(&colly.LimitRule{
		DomainGlob:  "*torecacamp-pokemon.com*",
		Parallelism: 1, // Individual collectors should not be parallel
		Delay:       cp.config.DelayMin,
	})

	return collector
}

// resetCollector resets a collector for reuse
func (cp *CollectorPool) resetCollector(collector *colly.Collector) {
	// Wait for any pending requests to complete first
	collector.Wait()

	// Don't clear handlers - just let the new usage set up fresh ones
	// This avoids nil pointer issues with colly's internal state
}

// cleanupCollector cleans up a collector before returning it to the pool
func (cp *CollectorPool) cleanupCollector(collector *colly.Collector) {
	// Wait for any pending requests to complete
	collector.Wait()

	// Don't try to clear handlers here as it can cause issues
	// Let the next usage override them
}

// createFreshCollector creates a brand new collector for reset purposes
func (cp *CollectorPool) createFreshCollector() *colly.Collector {
	return NewCollector(cp.config)
}

// Close closes the collector pool and cleans up resources
func (cp *CollectorPool) Close() {
	close(cp.collectors)

	// Drain remaining collectors
	for collector := range cp.collectors {
		cp.cleanupCollector(collector)
	}
}

// Stats returns statistics about the collector pool
func (cp *CollectorPool) Stats() CollectorPoolStats {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	return CollectorPoolStats{
		MaxSize:   cp.maxSize,
		Created:   cp.created,
		InUse:     cp.inUse,
		Available: len(cp.collectors),
		Timestamp: time.Now(),
	}
}

// CollectorPoolStats holds statistics about the collector pool
type CollectorPoolStats struct {
	MaxSize   int       `json:"max_size"`
	Created   int       `json:"created"`
	InUse     int       `json:"in_use"`
	Available int       `json:"available"`
	Timestamp time.Time `json:"timestamp"`
}

// IsHealthy returns true if the pool is in a healthy state
func (cp *CollectorPool) IsHealthy() bool {
	stats := cp.Stats()

	// Pool is healthy if:
	// - We have available collectors or can create more
	// - We're not at maximum capacity with all collectors in use
	return stats.Available > 0 || stats.Created < stats.MaxSize
}

// WarmUp pre-populates the pool with collectors up to the specified size
func (cp *CollectorPool) WarmUp(targetSize int) {
	if targetSize > cp.maxSize {
		targetSize = cp.maxSize
	}

	cp.mu.Lock()
	defer cp.mu.Unlock()

	needed := targetSize - cp.created
	if needed <= 0 {
		return
	}

	for i := 0; i < needed; i++ {
		collector := cp.createCollector()
		select {
		case cp.collectors <- collector:
			cp.created++
		default:
			// Pool channel is full, stop creating
			break
		}
	}
}
