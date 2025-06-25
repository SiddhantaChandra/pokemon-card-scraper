package tracker

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/SiddhantaChandra/pokemon-card-scraper/internal/storage"
	"github.com/SiddhantaChandra/pokemon-card-scraper/pkg/models"
)

// TrackerWorker manages background scanning of tracked URLs
type TrackerWorker struct {
	storage      storage.TrackerStorage
	scraper      TrackerScraper
	batchScraper *BatchScraper
	notifier     NotificationService
	config       *WorkerConfig

	// Worker state
	isRunning bool
	mu        sync.RWMutex
	stopChan  chan struct{}
	ticker    *time.Ticker

	// Statistics
	stats   WorkerStats
	statsMu sync.RWMutex
}

// WorkerConfig holds configuration for the tracker worker
type WorkerConfig struct {
	ScanInterval        time.Duration `json:"scan_interval"`
	MaxWorkers          int           `json:"max_workers"`
	TimeoutPerURL       time.Duration `json:"timeout_per_url"`
	RetryAttempts       int           `json:"retry_attempts"`
	RetryDelay          time.Duration `json:"retry_delay"`
	EnableNotifications bool          `json:"enable_notifications"`
}

// DefaultWorkerConfig returns default worker configuration
func DefaultWorkerConfig() *WorkerConfig {
	return &WorkerConfig{
		ScanInterval:        1 * time.Hour, // Scan every hour
		MaxWorkers:          5,
		TimeoutPerURL:       30 * time.Second,
		RetryAttempts:       3,
		RetryDelay:          5 * time.Second,
		EnableNotifications: true,
	}
}

// WorkerStats tracks worker performance and statistics
type WorkerStats struct {
	TotalScans        int           `json:"total_scans"`
	SuccessfulScans   int           `json:"successful_scans"`
	FailedScans       int           `json:"failed_scans"`
	LastScanTime      time.Time     `json:"last_scan_time"`
	NextScanTime      time.Time     `json:"next_scan_time"`
	AverageCheckTime  time.Duration `json:"average_check_time"`
	TotalCheckTime    time.Duration `json:"total_check_time"`
	StockChanges      int           `json:"stock_changes"`
	PriceChanges      int           `json:"price_changes"`
	ErrorsEncountered int           `json:"errors_encountered"`
}

// NotificationService interface for sending notifications
type NotificationService interface {
	SendStockAlert(tracker models.TrackerEntry, statusChanged bool) error
	SendErrorAlert(message string, err error) error
}

// NewTrackerWorker creates a new tracker worker instance
func NewTrackerWorker(storage storage.TrackerStorage, scraper TrackerScraper, notifier NotificationService, config *WorkerConfig) *TrackerWorker {
	if config == nil {
		config = DefaultWorkerConfig()
	}

	batchScraper := NewBatchScraper(scraper, config.MaxWorkers, config.TimeoutPerURL)

	return &TrackerWorker{
		storage:      storage,
		scraper:      scraper,
		batchScraper: batchScraper,
		notifier:     notifier,
		config:       config,
		stopChan:     make(chan struct{}),
		stats:        WorkerStats{},
	}
}

// Start begins the background scanning process
func (tw *TrackerWorker) Start() error {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.isRunning {
		return fmt.Errorf("worker is already running")
	}

	log.Printf("Starting tracker worker with scan interval: %v", tw.config.ScanInterval)

	tw.isRunning = true
	tw.ticker = time.NewTicker(tw.config.ScanInterval)

	// Update next scan time
	tw.statsMu.Lock()
	tw.stats.NextScanTime = time.Now().Add(tw.config.ScanInterval)
	tw.statsMu.Unlock()

	// Start the main worker goroutine
	go tw.workerLoop()

	return nil
}

// Stop stops the background scanning process
func (tw *TrackerWorker) Stop() error {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if !tw.isRunning {
		return fmt.Errorf("worker is not running")
	}

	log.Println("Stopping tracker worker...")

	tw.isRunning = false

	if tw.ticker != nil {
		tw.ticker.Stop()
	}

	// Send stop signal
	close(tw.stopChan)

	// Reset stop channel for potential restart
	tw.stopChan = make(chan struct{})

	log.Println("Tracker worker stopped")
	return nil
}

// IsRunning returns whether the worker is currently running
func (tw *TrackerWorker) IsRunning() bool {
	tw.mu.RLock()
	defer tw.mu.RUnlock()
	return tw.isRunning
}

// GetStats returns current worker statistics
func (tw *TrackerWorker) GetStats() WorkerStats {
	tw.statsMu.RLock()
	defer tw.statsMu.RUnlock()
	return tw.stats
}

// CheckNow performs an immediate check of all active trackers
func (tw *TrackerWorker) CheckNow() (*models.BatchCheckResult, error) {
	log.Println("Manual tracker check initiated")

	// Get all active trackers
	trackers, err := tw.storage.GetActiveTrackers()
	if err != nil {
		return nil, fmt.Errorf("failed to get active trackers: %v", err)
	}

	if len(trackers) == 0 {
		return &models.BatchCheckResult{
			TotalChecked:     0,
			SuccessfulChecks: 0,
			FailedChecks:     0,
			Results:          []models.TrackerCheckResult{},
			TotalTime:        0,
			AverageTime:      0,
		}, nil
	}

	return tw.checkTrackers(trackers)
}

// workerLoop is the main worker loop that runs in a goroutine
func (tw *TrackerWorker) workerLoop() {
	for {
		select {
		case <-tw.stopChan:
			log.Println("Worker loop stopped")
			return
		case <-tw.ticker.C:
			tw.performScheduledScan()
		}
	}
}

// performScheduledScan performs a scheduled scan of all active trackers
func (tw *TrackerWorker) performScheduledScan() {
	log.Println("Starting scheduled tracker scan")

	start := time.Now()

	// Update stats
	tw.statsMu.Lock()
	tw.stats.LastScanTime = start
	tw.stats.NextScanTime = start.Add(tw.config.ScanInterval)
	tw.stats.TotalScans++
	tw.statsMu.Unlock()

	// Get active trackers
	trackers, err := tw.storage.GetActiveTrackers()
	if err != nil {
		log.Printf("Failed to get active trackers: %v", err)
		tw.updateErrorStats()
		return
	}

	if len(trackers) == 0 {
		log.Println("No active trackers to scan")
		return
	}

	log.Printf("Scanning %d active trackers", len(trackers))

	// Check trackers
	result, err := tw.checkTrackers(trackers)
	if err != nil {
		log.Printf("Failed to check trackers: %v", err)
		tw.updateErrorStats()
		return
	}

	// Update statistics
	tw.updateScanStats(result, time.Since(start))

	log.Printf("Scheduled scan completed: %d successful, %d failed",
		result.SuccessfulChecks, result.FailedChecks)
}

// checkTrackers checks a list of trackers and returns results
func (tw *TrackerWorker) checkTrackers(trackers []models.TrackerEntry) (*models.BatchCheckResult, error) {
	start := time.Now()

	// Extract URLs
	urls := make([]string, len(trackers))
	trackerMap := make(map[string]models.TrackerEntry)

	for i, tracker := range trackers {
		urls[i] = tracker.URL
		trackerMap[tracker.URL] = tracker
	}

	// Scrape URLs using batch scraper
	scraperResults := tw.batchScraper.ScrapeURLs(urls)

	// Process results
	results := make([]models.TrackerCheckResult, 0, len(scraperResults))
	successCount := 0
	failCount := 0
	totalCheckTime := time.Duration(0)

	for _, scraperResult := range scraperResults {
		tracker := trackerMap[scraperResult.URL]

		checkResult := models.TrackerCheckResult{
			TrackerID: tracker.ID,
			Success:   scraperResult.Error == "",
			InStock:   scraperResult.InStock,
			Price:     scraperResult.Price,
			ImageURL:  scraperResult.ImageURL,
			CheckTime: scraperResult.Duration.Seconds(),
		}

		if scraperResult.Error != "" {
			checkResult.Error = scraperResult.Error
			failCount++

			// Update error stats
			tw.statsMu.Lock()
			tw.stats.ErrorsEncountered++
			tw.statsMu.Unlock()
		} else {
			successCount++

			// Check for changes and update database
			if err := tw.processTrackerUpdate(tracker, scraperResult); err != nil {
				log.Printf("Failed to process update for tracker %s: %v", tracker.ID, err)
				checkResult.Error = err.Error()
				checkResult.Success = false
				failCount++
				successCount--
			}
		}

		results = append(results, checkResult)
		totalCheckTime += scraperResult.Duration
	}

	totalTime := time.Since(start)
	averageTime := float64(0)
	if len(results) > 0 {
		averageTime = totalTime.Seconds() / float64(len(results))
	}

	return &models.BatchCheckResult{
		TotalChecked:     len(results),
		SuccessfulChecks: successCount,
		FailedChecks:     failCount,
		Results:          results,
		TotalTime:        totalTime.Seconds(),
		AverageTime:      averageTime,
	}, nil
}

// processTrackerUpdate updates a tracker with new information and sends notifications if needed
func (tw *TrackerWorker) processTrackerUpdate(tracker models.TrackerEntry, result ScraperResult) error {
	// Check for stock status change
	stockChanged := tracker.InStock != result.InStock

	// Check for price change (with small tolerance for floating point comparison)
	priceChanged := false
	if tracker.Price > 0 && result.Price > 0 {
		priceDiff := result.Price - tracker.Price
		if priceDiff < 0 {
			priceDiff = -priceDiff
		}
		// Consider price changed if difference is more than 1 yen/dollar
		priceChanged = priceDiff > 1.0
	} else if tracker.Price != result.Price {
		priceChanged = true
	}

	// Update tracker in database
	err := tw.storage.UpdateTrackerStatus(tracker.ID, result.InStock, result.Price, result.ImageURL)
	if err != nil {
		return fmt.Errorf("failed to update tracker status: %v", err)
	}

	// Update statistics
	if stockChanged {
		tw.statsMu.Lock()
		tw.stats.StockChanges++
		tw.statsMu.Unlock()
	}

	if priceChanged {
		tw.statsMu.Lock()
		tw.stats.PriceChanges++
		tw.statsMu.Unlock()
	}

	// Send notifications if enabled and there were changes
	if tw.config.EnableNotifications && tw.notifier != nil && (stockChanged || priceChanged) {
		// Update tracker data for notification
		updatedTracker := tracker
		updatedTracker.InStock = result.InStock
		updatedTracker.Price = result.Price
		updatedTracker.ImageURL = result.ImageURL

		if err := tw.notifier.SendStockAlert(updatedTracker, stockChanged); err != nil {
			log.Printf("Failed to send notification for tracker %s: %v", tracker.ID, err)
			// Don't return error as this shouldn't fail the entire update
		}
	}

	return nil
}

// updateScanStats updates worker statistics after a scan
func (tw *TrackerWorker) updateScanStats(result *models.BatchCheckResult, scanDuration time.Duration) {
	tw.statsMu.Lock()
	defer tw.statsMu.Unlock()

	if result.SuccessfulChecks > 0 {
		tw.stats.SuccessfulScans++
	}
	if result.FailedChecks > 0 {
		tw.stats.FailedScans++
	}

	// Update average check time
	totalPreviousTime := tw.stats.AverageCheckTime * time.Duration(tw.stats.TotalScans-1)
	tw.stats.TotalCheckTime = totalPreviousTime + scanDuration
	tw.stats.AverageCheckTime = tw.stats.TotalCheckTime / time.Duration(tw.stats.TotalScans)
}

// updateErrorStats updates error statistics
func (tw *TrackerWorker) updateErrorStats() {
	tw.statsMu.Lock()
	defer tw.statsMu.Unlock()

	tw.stats.FailedScans++
	tw.stats.ErrorsEncountered++
}

// GetWorkerStatus returns the current worker status for the API
func (tw *TrackerWorker) GetWorkerStatus() models.TrackerWorkerStatus {
	tw.mu.RLock()
	isRunning := tw.isRunning
	tw.mu.RUnlock()

	stats := tw.GetStats()

	return models.TrackerWorkerStatus{
		IsRunning:        isRunning,
		LastScanTime:     stats.LastScanTime,
		NextScanTime:     stats.NextScanTime,
		ScanInterval:     int(tw.config.ScanInterval.Seconds()),
		ItemsProcessed:   stats.SuccessfulScans,
		ErrorsEncounted:  stats.ErrorsEncountered,
		AverageCheckTime: stats.AverageCheckTime.Seconds(),
	}
}

// UpdateConfig updates the worker configuration (requires restart if running)
func (tw *TrackerWorker) UpdateConfig(newConfig *WorkerConfig) error {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	wasRunning := tw.isRunning

	if wasRunning {
		// Stop the worker temporarily
		tw.isRunning = false
		if tw.ticker != nil {
			tw.ticker.Stop()
		}
	}

	// Update configuration
	tw.config = newConfig
	tw.batchScraper = NewBatchScraper(tw.scraper, newConfig.MaxWorkers, newConfig.TimeoutPerURL)

	if wasRunning {
		// Restart with new configuration
		tw.isRunning = true
		tw.ticker = time.NewTicker(tw.config.ScanInterval)

		// Update next scan time
		tw.statsMu.Lock()
		tw.stats.NextScanTime = time.Now().Add(tw.config.ScanInterval)
		tw.statsMu.Unlock()
	}

	return nil
}
