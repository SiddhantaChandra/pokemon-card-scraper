package scraper

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/SiddhantaChandra/pokemon-card-scraper/internal/storage"
	"github.com/SiddhantaChandra/pokemon-card-scraper/pkg/models"
	"github.com/gocolly/colly/v2"
)

// ParallelScraper handles concurrent page scraping with worker pools
type ParallelScraper struct {
	config         *ParallelScraperConfig
	collectorPool  *CollectorPool
	batchProcessor *storage.BatchProcessor

	// Worker management
	pageWorkers int
	pageQueue   chan PageJob
	resultQueue chan PageResult
	errorQueue  chan error

	// Synchronization
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc

	// Callbacks
	onCardFound func(models.Card)
	onProgress  func(ScrapeProgress)

	// Status tracking
	mu      sync.RWMutex
	running bool
	status  ParallelScrapingStatus
}

// ParallelScraperConfig holds configuration for parallel scraping
type ParallelScraperConfig struct {
	PageWorkers       int
	CollectorPoolSize int
	BatchSize         int
	FlushInterval     time.Duration
	MaxRetries        int
	RetryDelay        time.Duration

	// Inherited from original scraper
	BaseURL         string
	SearchURL       string
	CollectorConfig *CollectorConfig
}

// PageJob represents a single page scraping task
type PageJob struct {
	PageURL   string
	PageNum   int
	Retries   int
	Timestamp time.Time
}

// PageResult represents the result of scraping a page
type PageResult struct {
	PageNum   int
	Cards     []models.Card
	Error     error
	Duration  time.Duration
	Timestamp time.Time
}

// ParallelScrapingStatus tracks the status of parallel scraping
type ParallelScrapingStatus struct {
	StartTime       time.Time `json:"start_time"`
	LastUpdated     time.Time `json:"last_updated"`
	PagesProcessed  int       `json:"pages_processed"`
	PagesInProgress int       `json:"pages_in_progress"`
	TotalPages      int       `json:"total_pages"`
	CardsFound      int       `json:"cards_found"`
	ActiveWorkers   int       `json:"active_workers"`
	ErrorCount      int       `json:"error_count"`
	AvgPageDuration float64   `json:"avg_page_duration_ms"`
}

// DefaultParallelScraperConfig returns sensible defaults for parallel scraping
func DefaultParallelScraperConfig() *ParallelScraperConfig {
	return &ParallelScraperConfig{
		PageWorkers:       10,
		CollectorPoolSize: 20,
		BatchSize:         100,
		FlushInterval:     5 * time.Second,
		MaxRetries:        3,
		RetryDelay:        time.Second,
		BaseURL:           "https://torecacamp-pokemon.com",
		SearchURL:         "https://torecacamp-pokemon.com/search?type=product&options%5Bprefix%5D=last&options%5Bunavailable_products%5D=last&q=.",
		CollectorConfig:   DefaultCollectorConfig(),
	}
}

// NewParallelScraper creates a new parallel scraper instance
func NewParallelScraper(config *ParallelScraperConfig, batchProcessor *storage.BatchProcessor) *ParallelScraper {
	if config == nil {
		config = DefaultParallelScraperConfig()
	}

	// Optimize collector config for parallel processing
	collectorConfig := config.CollectorConfig
	collectorConfig.ConcurrentRequests = config.PageWorkers
	collectorConfig.DelayMin = 500 * time.Millisecond
	collectorConfig.DelayMax = 1 * time.Second

	ctx, cancel := context.WithCancel(context.Background())

	return &ParallelScraper{
		config:         config,
		collectorPool:  NewCollectorPool(config.CollectorPoolSize, collectorConfig),
		batchProcessor: batchProcessor,
		pageWorkers:    config.PageWorkers,
		pageQueue:      make(chan PageJob, config.PageWorkers*2),
		resultQueue:    make(chan PageResult, config.PageWorkers*2),
		errorQueue:     make(chan error, config.PageWorkers),
		ctx:            ctx,
		cancel:         cancel,
		status: ParallelScrapingStatus{
			StartTime:   time.Now(),
			LastUpdated: time.Now(),
		},
	}
}

// SetCardFoundCallback sets the callback function for when cards are found
func (ps *ParallelScraper) SetCardFoundCallback(callback func(models.Card)) {
	ps.onCardFound = callback
}

// SetProgressCallback sets the callback function for progress updates
func (ps *ParallelScraper) SetProgressCallback(callback func(ScrapeProgress)) {
	ps.onProgress = callback
}

// ScrapeAllPagesParallel scrapes all pages using parallel processing
func (ps *ParallelScraper) ScrapeAllPagesParallel(params SearchParams, progressCallback func(ScrapeProgress)) error {
	log.Println("Starting parallel scraping with worker pool...")

	ps.mu.Lock()
	ps.running = true
	ps.status.StartTime = time.Now()
	ps.mu.Unlock()

	defer func() {
		ps.mu.Lock()
		ps.running = false
		ps.mu.Unlock()
	}()

	// Get total pages first
	totalPages, totalItems, err := ps.getPaginationInfo(params)
	if err != nil {
		return fmt.Errorf("failed to get pagination info: %w", err)
	}

	log.Printf("Discovered %d total pages with %d items", totalPages, totalItems)

	ps.mu.Lock()
	ps.status.TotalPages = totalPages
	ps.mu.Unlock()

	// Start worker goroutines
	for i := 0; i < ps.pageWorkers; i++ {
		ps.wg.Add(1)
		go ps.pageWorker(i)
	}

	// Start result collector
	ps.wg.Add(1)
	go ps.resultCollector(progressCallback)

	// Queue all pages for processing
	for page := 1; page <= totalPages; page++ {
		params.Page = page
		pageURL := BuildSearchURL(params)

		select {
		case ps.pageQueue <- PageJob{
			PageURL:   pageURL,
			PageNum:   page,
			Retries:   0,
			Timestamp: time.Now(),
		}:
		case <-ps.ctx.Done():
			log.Println("Scraping cancelled, stopping page queueing")
			break
		}
	}

	// Close the page queue to signal workers to finish
	close(ps.pageQueue)

	// Wait for all workers to complete
	ps.wg.Wait()

	// Flush any remaining cards in batch processor
	if ps.batchProcessor != nil {
		ps.batchProcessor.Flush()
	}

	ps.mu.RLock()
	finalStatus := ps.status
	ps.mu.RUnlock()

	log.Printf("Parallel scraping completed. Pages: %d, Cards: %d, Errors: %d",
		finalStatus.PagesProcessed, finalStatus.CardsFound, finalStatus.ErrorCount)

	return nil
}

// pageWorker processes pages from the queue
func (ps *ParallelScraper) pageWorker(workerID int) {
	defer ps.wg.Done()

	log.Printf("Page worker %d started", workerID)
	defer log.Printf("Page worker %d stopped", workerID)

	for {
		select {
		case job, ok := <-ps.pageQueue:
			if !ok {
				// Channel closed, worker should exit
				return
			}

			ps.mu.Lock()
			ps.status.ActiveWorkers++
			ps.status.PagesInProgress++
			ps.mu.Unlock()

			result := ps.processPage(job, workerID)

			ps.mu.Lock()
			ps.status.ActiveWorkers--
			ps.status.PagesInProgress--
			ps.mu.Unlock()

			select {
			case ps.resultQueue <- result:
			case <-ps.ctx.Done():
				return
			}

		case <-ps.ctx.Done():
			return
		}
	}
}

// processPage processes a single page and returns the result
func (ps *ParallelScraper) processPage(job PageJob, workerID int) PageResult {
	startTime := time.Now()

	result := PageResult{
		PageNum:   job.PageNum,
		Timestamp: startTime,
	}

	// Get collector from pool
	collector := ps.collectorPool.Get()
	defer ps.collectorPool.Put(collector)

	var pageCards []models.Card
	var parseError error

	// Set up parsing for this page
	collector.OnHTML("html", func(e *colly.HTMLElement) {
		pageInfo, err := ParseProductPage(e.DOM, ps.config.BaseURL)
		if err != nil {
			parseError = fmt.Errorf("failed to parse page %d: %w", job.PageNum, err)
			return
		}
		pageCards = pageInfo.Cards
	})

	// Visit the page
	if err := collector.Visit(job.PageURL); err != nil {
		result.Error = fmt.Errorf("failed to visit page %d: %w", job.PageNum, err)
	} else {
		collector.Wait()

		if parseError != nil {
			result.Error = parseError
		} else {
			result.Cards = pageCards
			log.Printf("Worker %d processed page %d: found %d cards",
				workerID, job.PageNum, len(pageCards))
		}
	}

	result.Duration = time.Since(startTime)
	return result
}

// resultCollector processes results from workers
func (ps *ParallelScraper) resultCollector(progressCallback func(ScrapeProgress)) {
	defer ps.wg.Done()

	log.Println("Result collector started")
	defer log.Println("Result collector stopped")

	var totalCards int
	var totalDuration time.Duration
	var processedPages int

	// Use a ticker for periodic progress updates
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case result, ok := <-ps.resultQueue:
			if !ok {
				// Channel closed, collector should exit
				return
			}

			processedPages++
			totalDuration += result.Duration

			if result.Error != nil {
				log.Printf("Error processing page %d: %v", result.PageNum, result.Error)
				ps.mu.Lock()
				ps.status.ErrorCount++
				ps.mu.Unlock()

				// Retry logic could be added here
				continue
			}

			// Process cards from this page
			for _, card := range result.Cards {
				totalCards++

				// Add to batch processor if available
				if ps.batchProcessor != nil {
					ps.batchProcessor.AddCard(card)
				}

				// Call individual card callback if set
				if ps.onCardFound != nil {
					ps.onCardFound(card)
				}
			}

			// Update status
			ps.mu.Lock()
			ps.status.PagesProcessed = processedPages
			ps.status.CardsFound = totalCards
			ps.status.LastUpdated = time.Now()
			if processedPages > 0 {
				ps.status.AvgPageDuration = float64(totalDuration.Nanoseconds()) / float64(processedPages) / 1e6
			}
			ps.mu.Unlock()

		case <-ticker.C:
			// Periodic progress update
			if progressCallback != nil {
				ps.mu.RLock()
				progress := ScrapeProgress{
					CurrentPage:    ps.status.PagesProcessed,
					TotalPages:     ps.status.TotalPages,
					ItemsProcessed: ps.status.CardsFound,
					StartTime:      ps.status.StartTime,
				}
				ps.mu.RUnlock()

				progressCallback(progress)
			}

		case <-ps.ctx.Done():
			return
		}
	}
}

// getPaginationInfo gets total pages and items (reuse existing logic)
func (ps *ParallelScraper) getPaginationInfo(params SearchParams) (totalPages, totalItems int, err error) {
	params.Page = 1
	pageURL := BuildSearchURL(params)

	collector := ps.collectorPool.Get()
	defer ps.collectorPool.Put(collector)

	collector.OnHTML("html", func(e *colly.HTMLElement) {
		pageInfo, parseErr := ParseProductPage(e.DOM, ps.config.BaseURL)
		if parseErr != nil {
			err = parseErr
			return
		}

		totalPages = pageInfo.TotalPages
		totalItems = pageInfo.TotalItems
	})

	if visitErr := collector.Visit(pageURL); visitErr != nil {
		return 0, 0, visitErr
	}

	collector.Wait()
	return totalPages, totalItems, err
}

// Stop gracefully stops the parallel scraper
func (ps *ParallelScraper) Stop() {
	ps.mu.Lock()
	if !ps.running {
		ps.mu.Unlock()
		return
	}
	ps.mu.Unlock()

	log.Println("Stopping parallel scraper...")
	ps.cancel()
}

// GetStatus returns the current scraping status (compatible with interface)
func (ps *ParallelScraper) GetStatus() ScrapingStatus {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	// Convert parallel status to regular status for interface compatibility
	return ScrapingStatus{
		StartTime:              ps.status.StartTime,
		LastUpdated:            ps.status.LastUpdated,
		CurrentPage:            ps.status.PagesProcessed,
		TotalPages:             ps.status.TotalPages,
		ItemsScraped:           ps.status.CardsFound,
		CardsPerMinute:         0,     // Calculate if needed
		EstimatedTimeRemaining: nil,   // Calculate if needed
		IsPaused:               false, // Parallel scraper doesn't support pause yet
		PausedAt:               nil,
	}
}

// GetParallelStatus returns the native parallel scraping status
func (ps *ParallelScraper) GetParallelStatus() ParallelScrapingStatus {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.status
}

// IsRunning returns whether the scraper is currently running
func (ps *ParallelScraper) IsRunning() bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.running
}

// IsPaused returns whether the scraper is currently paused (always false for parallel scraper)
func (ps *ParallelScraper) IsPaused() bool {
	// Parallel scraper doesn't support pause/resume yet
	return false
}

// Pause pauses the scraper (not implemented for parallel scraper)
func (ps *ParallelScraper) Pause() bool {
	// Parallel scraper doesn't support pause/resume yet
	return false
}

// Resume resumes the scraper (not implemented for parallel scraper)
func (ps *ParallelScraper) Resume() bool {
	// Parallel scraper doesn't support pause/resume yet
	return false
}

// ScrapeAllPages implements the interface method (wrapper for ScrapeAllPagesParallel)
func (ps *ParallelScraper) ScrapeAllPages(params SearchParams, progressCallback func(ScrapeProgress)) error {
	return ps.ScrapeAllPagesParallel(params, progressCallback)
}

// ScrapePage scrapes a single page (simplified version for interface compatibility)
func (ps *ParallelScraper) ScrapePage(pageURL string) ([]models.Card, error) {
	collector := ps.collectorPool.Get()
	defer ps.collectorPool.Put(collector)

	var pageCards []models.Card
	var parseError error

	collector.OnHTML("html", func(e *colly.HTMLElement) {
		pageInfo, err := ParseProductPage(e.DOM, ps.config.BaseURL)
		if err != nil {
			parseError = fmt.Errorf("failed to parse page: %w", err)
			return
		}
		pageCards = pageInfo.Cards
	})

	if err := collector.Visit(pageURL); err != nil {
		return nil, fmt.Errorf("failed to visit page: %w", err)
	}

	collector.Wait()

	if parseError != nil {
		return nil, parseError
	}

	return pageCards, nil
}
