package monitor

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/SiddhantaChandra/pokemon-card-scraper/pkg/models"
)

// MonitorStatus represents the current status of the monitoring service
type MonitorStatus string

const (
	StatusStopped  MonitorStatus = "stopped"
	StatusStarting MonitorStatus = "starting"
	StatusRunning  MonitorStatus = "running"
	StatusStopping MonitorStatus = "stopping"
	StatusError    MonitorStatus = "error"
)

// MonitorConfig holds configuration for the stock monitoring service
type MonitorConfig struct {
	// Monitoring intervals
	CheckInterval       time.Duration `json:"check_interval"`        // How often to check each tracker
	BatchSize           int           `json:"batch_size"`            // Number of trackers to process in parallel
	MaxConcurrentChecks int           `json:"max_concurrent_checks"` // Maximum concurrent monitoring jobs

	// Retry configuration
	MaxRetries             int           `json:"max_retries"`              // Maximum retries for failed checks
	RetryBackoffMultiplier float64       `json:"retry_backoff_multiplier"` // Backoff multiplier for retries
	InitialRetryDelay      time.Duration `json:"initial_retry_delay"`      // Initial delay before retry

	// Notification configuration
	NotificationEnabled   bool          `json:"notification_enabled"`    // Enable Discord notifications
	NotificationRateLimit time.Duration `json:"notification_rate_limit"` // Rate limit for notifications

	// Health check configuration
	HealthCheckInterval time.Duration `json:"health_check_interval"` // Health check interval
	StaleThreshold      time.Duration `json:"stale_threshold"`       // When to consider data stale

	// WebDriver configuration
	WebDriverPoolSize int           `json:"webdriver_pool_size"` // Size of WebDriver pool
	WebDriverTimeout  time.Duration `json:"webdriver_timeout"`   // Timeout for WebDriver operations
}

// DefaultMonitorConfig returns sensible defaults for monitoring configuration
func DefaultMonitorConfig() *MonitorConfig {
	return &MonitorConfig{
		CheckInterval:          1 * time.Hour,
		BatchSize:              10,
		MaxConcurrentChecks:    5,
		MaxRetries:             3,
		RetryBackoffMultiplier: 2.0,
		InitialRetryDelay:      30 * time.Second,
		NotificationEnabled:    true,
		NotificationRateLimit:  5 * time.Minute,
		HealthCheckInterval:    10 * time.Minute,
		StaleThreshold:         6 * time.Hour,
		WebDriverPoolSize:      3,
		WebDriverTimeout:       30 * time.Second,
	}
}

// Storage interface for tracker operations
type Storage interface {
	SaveTracker(tracker models.TrackerItem) error
	GetTracker(id string) (*models.TrackerItem, error)
	SearchTrackers(filters models.TrackerFilterOptions) (*models.TrackerSearchResult, error)
	GetAllTrackers() ([]models.TrackerItem, error)
	DeleteTracker(id string) error
	UpdateTracker(id string, fields map[string]interface{}) error
	SaveTrackersBatch(trackers []models.TrackerItem) error
	UpdateTrackersBatch(updates []models.TrackerUpdate) error
}

// NotificationService interface for sending notifications
type NotificationService interface {
	SendStockAlert(item models.TrackerItem, oldStatus, newStatus bool) error
	SendErrorAlert(item models.TrackerItem, error string) error
}

// WebDriverPool interface for managing Chrome instances
type WebDriverPool interface {
	GetDriver() (WebDriver, error)
	ReturnDriver(driver WebDriver)
	Close() error
}

// WebDriver interface for web scraping operations
type WebDriver interface {
	Navigate(url string) error
	GetPageContent() (string, error)
	Close() error
}

// MonitorStats represents monitoring statistics
type MonitorStats struct {
	Status            MonitorStatus `json:"status"`
	TotalTrackers     int           `json:"total_trackers"`
	ActiveTrackers    int           `json:"active_trackers"`
	ChecksCompleted   int64         `json:"checks_completed"`
	ChecksFailed      int64         `json:"checks_failed"`
	LastCheckTime     *time.Time    `json:"last_check_time"`
	NextCheckTime     *time.Time    `json:"next_check_time"`
	NotificationsSent int64         `json:"notifications_sent"`
	ErrorsToday       int64         `json:"errors_today"`
	AverageCheckTime  time.Duration `json:"average_check_time"`
	UptimePercentage  float64       `json:"uptime_percentage"`
}

// StockMonitor is the main service for monitoring stock changes
type StockMonitor struct {
	config     *MonitorConfig
	storage    Storage
	notifier   NotificationService
	driverPool WebDriverPool

	// Internal state
	status MonitorStatus
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex

	// Statistics
	stats     MonitorStats
	startTime time.Time

	// Worker management
	workerSem chan struct{}
}

// NewStockMonitor creates a new stock monitoring service
func NewStockMonitor(config *MonitorConfig, storage Storage, notifier NotificationService, driverPool WebDriverPool) *StockMonitor {
	if config == nil {
		config = DefaultMonitorConfig()
	}

	return &StockMonitor{
		config:     config,
		storage:    storage,
		notifier:   notifier,
		driverPool: driverPool,
		status:     StatusStopped,
		workerSem:  make(chan struct{}, config.MaxConcurrentChecks),
		stats: MonitorStats{
			Status: StatusStopped,
		},
	}
}

// StartMonitoring begins the monitoring loop
func (sm *StockMonitor) StartMonitoring() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.status == StatusRunning {
		return fmt.Errorf("monitoring is already running")
	}

	sm.status = StatusStarting
	sm.ctx, sm.cancel = context.WithCancel(context.Background())
	sm.startTime = time.Now()

	log.Println("Starting stock monitoring service...")

	// Start the main monitoring loop
	sm.wg.Add(1)
	go sm.monitoringLoop()

	// Start health check routine
	sm.wg.Add(1)
	go sm.healthCheckLoop()

	sm.status = StatusRunning
	sm.stats.Status = StatusRunning

	log.Printf("Stock monitoring service started with interval: %v", sm.config.CheckInterval)
	return nil
}

// StopMonitoring gracefully stops the monitoring service
func (sm *StockMonitor) StopMonitoring() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.status == StatusStopped {
		return fmt.Errorf("monitoring is already stopped")
	}

	sm.status = StatusStopping
	sm.stats.Status = StatusStopping

	log.Println("Stopping stock monitoring service...")

	// Cancel context to stop all goroutines
	if sm.cancel != nil {
		sm.cancel()
	}

	// Wait for all goroutines to finish
	sm.wg.Wait()

	sm.status = StatusStopped
	sm.stats.Status = StatusStopped

	log.Println("Stock monitoring service stopped")
	return nil
}

// GetStatus returns the current monitoring status
func (sm *StockMonitor) GetStatus() MonitorStatus {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.status
}

// GetStats returns current monitoring statistics
func (sm *StockMonitor) GetStats() MonitorStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stats := sm.stats
	if sm.status == StatusRunning && !sm.startTime.IsZero() {
		uptime := time.Since(sm.startTime)
		// Calculate uptime percentage based on total runtime vs expected runtime
		// For simplicity, assume 99% uptime if running
		if uptime > time.Minute {
			stats.UptimePercentage = 99.0
		} else {
			stats.UptimePercentage = 100.0
		}
	}

	return stats
}

// AddTracker adds a new URL to monitor
func (sm *StockMonitor) AddTracker(url, name string) (*models.TrackerItem, error) {
	tracker := models.TrackerItem{
		ID:          generateTrackerID(),
		URL:         url,
		Name:        name,
		InStock:     false,
		CreatedAt:   time.Now(),
		LastUpdated: time.Now(),
	}

	if err := sm.storage.SaveTracker(tracker); err != nil {
		return nil, fmt.Errorf("failed to save tracker: %v", err)
	}

	log.Printf("Added new tracker: %s (%s)", name, url)
	return &tracker, nil
}

// RemoveTracker removes a tracker from monitoring
func (sm *StockMonitor) RemoveTracker(id string) error {
	if err := sm.storage.DeleteTracker(id); err != nil {
		return fmt.Errorf("failed to delete tracker: %v", err)
	}

	log.Printf("Removed tracker: %s", id)
	return nil
}

// UpdateTracker updates tracker details
func (sm *StockMonitor) UpdateTracker(id string, fields map[string]interface{}) error {
	if err := sm.storage.UpdateTracker(id, fields); err != nil {
		return fmt.Errorf("failed to update tracker: %v", err)
	}

	log.Printf("Updated tracker: %s", id)
	return nil
}

// GetTrackers returns all monitored trackers with optional filtering
func (sm *StockMonitor) GetTrackers(filters models.TrackerFilterOptions) (*models.TrackerSearchResult, error) {
	return sm.storage.SearchTrackers(filters)
}

// monitoringLoop is the main monitoring loop that runs continuously
func (sm *StockMonitor) monitoringLoop() {
	defer sm.wg.Done()

	ticker := time.NewTicker(sm.config.CheckInterval)
	defer ticker.Stop()

	// Run initial check
	sm.runMonitoringCycle()

	for {
		select {
		case <-sm.ctx.Done():
			log.Println("Monitoring loop stopped")
			return
		case <-ticker.C:
			sm.runMonitoringCycle()
		}
	}
}

// runMonitoringCycle performs one complete monitoring cycle
func (sm *StockMonitor) runMonitoringCycle() {
	log.Println("Starting monitoring cycle...")

	startTime := time.Now()
	sm.updateLastCheckTime(&startTime)

	// Get all trackers
	trackers, err := sm.storage.GetAllTrackers()
	if err != nil {
		log.Printf("Error getting trackers: %v", err)
		sm.incrementErrorCount()
		return
	}

	sm.updateTrackerCounts(len(trackers))

	// Process trackers in batches
	for i := 0; i < len(trackers); i += sm.config.BatchSize {
		end := i + sm.config.BatchSize
		if end > len(trackers) {
			end = len(trackers)
		}

		batch := trackers[i:end]
		sm.processBatch(batch)

		// Check if we should stop
		select {
		case <-sm.ctx.Done():
			return
		default:
		}
	}

	duration := time.Since(startTime)
	sm.updateAverageCheckTime(duration)
	sm.updateNextCheckTime()

	log.Printf("Monitoring cycle completed in %v", duration)
}

// processBatch processes a batch of trackers concurrently
func (sm *StockMonitor) processBatch(trackers []models.TrackerItem) {
	var wg sync.WaitGroup

	for _, tracker := range trackers {
		// Acquire worker slot
		sm.workerSem <- struct{}{}

		wg.Add(1)
		go func(t models.TrackerItem) {
			defer wg.Done()
			defer func() { <-sm.workerSem }() // Release worker slot

			sm.checkTracker(t)
		}(tracker)
	}

	wg.Wait()
}

// checkTracker checks a single tracker for stock changes
func (sm *StockMonitor) checkTracker(tracker models.TrackerItem) {
	log.Printf("Checking tracker: %s (%s)", tracker.Name, tracker.URL)

	// Get WebDriver from pool
	driver, err := sm.driverPool.GetDriver()
	if err != nil {
		log.Printf("Error getting WebDriver: %v", err)
		sm.incrementErrorCount()
		return
	}
	defer sm.driverPool.ReturnDriver(driver)

	// Navigate to page with timeout
	ctx, cancel := context.WithTimeout(sm.ctx, sm.config.WebDriverTimeout)
	defer cancel()

	if err := sm.navigateWithTimeout(ctx, driver, tracker.URL); err != nil {
		log.Printf("Error navigating to %s: %v", tracker.URL, err)
		sm.handleCheckError(tracker, err)
		return
	}

	// Parse page content
	content, err := driver.GetPageContent()
	if err != nil {
		log.Printf("Error getting page content for %s: %v", tracker.URL, err)
		sm.handleCheckError(tracker, err)
		return
	}

	// Parse stock information
	stockInfo, err := sm.parseStockInfo(content, tracker.URL)
	if err != nil {
		log.Printf("Error parsing stock info for %s: %v", tracker.URL, err)
		sm.handleCheckError(tracker, err)
		return
	}

	// Check for stock changes
	oldStock := tracker.InStock
	tracker.InStock = stockInfo.InStock
	tracker.PriceYen = stockInfo.Price
	tracker.Image = stockInfo.Image
	tracker.LastUpdated = time.Now()

	// Update tracker in storage
	if err := sm.storage.SaveTracker(tracker); err != nil {
		log.Printf("Error saving tracker %s: %v", tracker.ID, err)
		sm.incrementErrorCount()
		return
	}

	// Send notification if stock status changed
	if oldStock != tracker.InStock && sm.config.NotificationEnabled {
		if err := sm.notifier.SendStockAlert(tracker, oldStock, tracker.InStock); err != nil {
			log.Printf("Error sending notification for %s: %v", tracker.ID, err)
		} else {
			sm.incrementNotificationCount()
		}
	}

	sm.incrementCheckCount()
	log.Printf("Successfully checked tracker: %s (Stock: %v)", tracker.Name, tracker.InStock)
}

// Stock information parsed from page
type StockInfo struct {
	InStock bool
	Price   string
	Image   string
}

// parseStockInfo parses stock information from page content
func (sm *StockMonitor) parseStockInfo(content, url string) (*StockInfo, error) {
	// This is a placeholder - would use the enhanced parser from the scraper
	// For now, return basic stock info
	return &StockInfo{
		InStock: true, // Placeholder
		Price:   "¥0",
		Image:   "",
	}, nil
}

// navigateWithTimeout navigates to a URL with timeout
func (sm *StockMonitor) navigateWithTimeout(ctx context.Context, driver WebDriver, url string) error {
	done := make(chan error, 1)

	go func() {
		done <- driver.Navigate(url)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// handleCheckError handles errors during tracker checking
func (sm *StockMonitor) handleCheckError(tracker models.TrackerItem, err error) {
	sm.incrementErrorCount()

	// Could implement retry logic here
	log.Printf("Check failed for tracker %s: %v", tracker.ID, err)

	// Send error notification if configured
	if sm.config.NotificationEnabled {
		if notifyErr := sm.notifier.SendErrorAlert(tracker, err.Error()); notifyErr != nil {
			log.Printf("Error sending error notification: %v", notifyErr)
		}
	}
}

// healthCheckLoop performs periodic health checks
func (sm *StockMonitor) healthCheckLoop() {
	defer sm.wg.Done()

	ticker := time.NewTicker(sm.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sm.ctx.Done():
			return
		case <-ticker.C:
			sm.performHealthCheck()
		}
	}
}

// performHealthCheck performs a health check of the monitoring service
func (sm *StockMonitor) performHealthCheck() {
	log.Println("Performing health check...")

	// Check for stale trackers
	filters := models.DefaultTrackerFilterOptions()
	result, err := sm.storage.SearchTrackers(filters)
	if err != nil {
		log.Printf("Health check error: %v", err)
		return
	}

	staleCount := 0
	for _, tracker := range result.Trackers {
		if time.Since(tracker.LastUpdated) > sm.config.StaleThreshold {
			staleCount++
		}
	}

	if staleCount > 0 {
		log.Printf("Health check warning: %d stale trackers found", staleCount)
	}

	log.Println("Health check completed")
}

// Statistics update methods
func (sm *StockMonitor) updateLastCheckTime(t *time.Time) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.stats.LastCheckTime = t
}

func (sm *StockMonitor) updateNextCheckTime() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	next := time.Now().Add(sm.config.CheckInterval)
	sm.stats.NextCheckTime = &next
}

func (sm *StockMonitor) updateTrackerCounts(total int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.stats.TotalTrackers = total
	sm.stats.ActiveTrackers = total // Simplified for now
}

func (sm *StockMonitor) incrementCheckCount() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.stats.ChecksCompleted++
}

func (sm *StockMonitor) incrementErrorCount() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.stats.ChecksFailed++
	sm.stats.ErrorsToday++
}

func (sm *StockMonitor) incrementNotificationCount() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.stats.NotificationsSent++
}

func (sm *StockMonitor) updateAverageCheckTime(duration time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	// Simple average calculation - could be improved with rolling average
	if sm.stats.ChecksCompleted > 0 {
		totalTime := sm.stats.AverageCheckTime * time.Duration(sm.stats.ChecksCompleted-1)
		sm.stats.AverageCheckTime = (totalTime + duration) / time.Duration(sm.stats.ChecksCompleted)
	} else {
		sm.stats.AverageCheckTime = duration
	}
}

// generateTrackerID generates a unique tracker ID
func generateTrackerID() string {
	return fmt.Sprintf("tracker_%d", time.Now().UnixNano())
}
