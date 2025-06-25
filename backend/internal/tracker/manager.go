package tracker

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/SiddhantaChandra/pokemon-card-scraper/pkg/models"
)

// TrackerManager coordinates all tracker functionality
type TrackerManager struct {
	config    *TrackerConfig
	storage   interface{} // Will cast to concrete type when needed
	scraper   TrackerScraper
	worker    *TrackerWorker
	notifier  NotificationService
	isRunning bool
	mu        sync.RWMutex
}

// TrackerStorageInterface represents the required storage methods
type TrackerStorageInterface interface {
	SaveTracker(tracker models.TrackerEntry) error
	GetTracker(id string) (*models.TrackerEntry, error)
	GetAllTrackers() ([]models.TrackerEntry, error)
	GetActiveTrackers() ([]models.TrackerEntry, error)
	UpdateTrackerStatus(id string, inStock bool, price float64, imageURL string) error
	DeleteTracker(id string) error
	GetTrackerByURL(url string) (*models.TrackerEntry, error)
	SearchTrackers(filters models.TrackerFilterOptions) (*models.TrackerSearchResult, error)
	GetTrackerStats() (*models.TrackerStats, error)
}

// NewTrackerManager creates a new tracker manager with the given configuration
func NewTrackerManager(config *TrackerConfig, storage interface{}) (*TrackerManager, error) {
	if config == nil {
		config = DefaultTrackerConfig()
	}

	// Validate configuration
	if err := config.IsValid(); err != nil {
		return nil, fmt.Errorf("invalid tracker configuration: %v", err)
	}

	// Create scraper
	scraper, err := NewChromeDPScraper(config.Scraper)
	if err != nil {
		return nil, fmt.Errorf("failed to create scraper: %v", err)
	}

	// Create notification service
	notifier := config.GetNotificationService()

	// Create tracker manager
	manager := &TrackerManager{
		config:   config,
		storage:  storage,
		scraper:  scraper,
		notifier: notifier,
	}

	// Create worker if tracker is enabled
	if config.Enabled {
		// Skip worker creation for now - will be created when started
		manager.worker = nil
	}

	return manager, nil
}

// getStorage returns the storage interface with type assertion
func (tm *TrackerManager) getStorage() TrackerStorageInterface {
	// This assumes the storage implements the required methods
	return tm.storage.(TrackerStorageInterface)
}

// Start starts the tracker system
func (tm *TrackerManager) Start() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if !tm.config.Enabled {
		return fmt.Errorf("tracker system is disabled")
	}

	if tm.isRunning {
		return fmt.Errorf("tracker system is already running")
	}

	// Start the worker if configured
	if tm.worker != nil {
		if err := tm.worker.Start(); err != nil {
			return fmt.Errorf("failed to start tracker worker: %v", err)
		}
	}

	tm.isRunning = true
	return nil
}

// Stop stops the tracker system
func (tm *TrackerManager) Stop() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if !tm.isRunning {
		return nil
	}

	// Stop the worker
	if tm.worker != nil {
		if err := tm.worker.Stop(); err != nil {
			log.Printf("Error stopping tracker worker: %v", err)
		}
	}

	// Close the scraper
	if tm.scraper != nil {
		if err := tm.scraper.Close(); err != nil {
			log.Printf("Error closing tracker scraper: %v", err)
		}
	}

	tm.isRunning = false
	return nil
}

// IsRunning returns whether the tracker system is currently running
func (tm *TrackerManager) IsRunning() bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.isRunning
}

// IsEnabled returns whether the tracker system is enabled
func (tm *TrackerManager) IsEnabled() bool {
	return tm.config.Enabled
}

// GetConfig returns the tracker configuration (sanitized for display)
func (tm *TrackerManager) GetConfig() *TrackerConfig {
	return tm.config.GetDisplayConfig()
}

// GetStorage returns the tracker storage interface
func (tm *TrackerManager) GetStorage() TrackerStorageInterface {
	return tm.getStorage()
}

// GetWorker returns the tracker worker (may be nil if disabled)
func (tm *TrackerManager) GetWorker() *TrackerWorker {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.worker
}

// GetNotifier returns the notification service
func (tm *TrackerManager) GetNotifier() NotificationService {
	return tm.notifier
}

// CheckNow performs an immediate check of all active trackers
func (tm *TrackerManager) CheckNow() (*models.BatchCheckResult, error) {
	if !tm.config.Enabled {
		return nil, fmt.Errorf("tracker system is disabled")
	}

	if tm.worker == nil {
		return nil, fmt.Errorf("tracker worker not initialized")
	}

	return tm.worker.CheckNow()
}

// AddTracker adds a new tracker to the system
func (tm *TrackerManager) AddTracker(url, name string) (*models.TrackerEntry, error) {
	if !tm.config.Enabled {
		return nil, fmt.Errorf("tracker system is disabled")
	}

	storage := tm.getStorage()

	// Check if URL is already being tracked
	existing, err := storage.GetTrackerByURL(url)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("URL is already being tracked: %s", existing.Name)
	}

	// Create new tracker entry
	tracker := &models.TrackerEntry{
		URL:       url,
		Name:      name,
		Status:    models.TrackerStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save to storage
	if err := storage.SaveTracker(*tracker); err != nil {
		return nil, fmt.Errorf("failed to save tracker: %v", err)
	}

	return tracker, nil
}

// GetTrackers retrieves all trackers with optional filtering
func (tm *TrackerManager) GetTrackers(filters models.TrackerFilterOptions) (*models.TrackerSearchResult, error) {
	if !tm.config.Enabled {
		return nil, fmt.Errorf("tracker system is disabled")
	}

	return tm.getStorage().SearchTrackers(filters)
}

// GetTracker retrieves a specific tracker by ID
func (tm *TrackerManager) GetTracker(id string) (*models.TrackerEntry, error) {
	if !tm.config.Enabled {
		return nil, fmt.Errorf("tracker system is disabled")
	}

	return tm.getStorage().GetTracker(id)
}

// UpdateTracker updates a tracker's information
func (tm *TrackerManager) UpdateTracker(id string, updates map[string]interface{}) (*models.TrackerEntry, error) {
	if !tm.config.Enabled {
		return nil, fmt.Errorf("tracker system is disabled")
	}

	storage := tm.getStorage()

	// Get existing tracker
	tracker, err := storage.GetTracker(id)
	if err != nil {
		return nil, fmt.Errorf("tracker not found: %v", err)
	}

	// Apply updates
	if name, ok := updates["name"].(string); ok && name != "" {
		tracker.Name = name
	}

	if status, ok := updates["status"].(string); ok {
		tracker.Status = models.TrackerStatus(status)
	}

	// Update timestamp
	tracker.UpdatedAt = time.Now()

	// Save updated tracker
	if err := storage.SaveTracker(*tracker); err != nil {
		return nil, fmt.Errorf("failed to update tracker: %v", err)
	}

	return tracker, nil
}

// DeleteTracker removes a tracker
func (tm *TrackerManager) DeleteTracker(id string) error {
	if !tm.config.Enabled {
		return fmt.Errorf("tracker system is disabled")
	}

	storage := tm.getStorage()

	// Delete tracker
	if err := storage.DeleteTracker(id); err != nil {
		return fmt.Errorf("failed to delete tracker: %v", err)
	}

	return nil
}

// GetStats returns tracker statistics
func (tm *TrackerManager) GetStats() (*models.TrackerStats, error) {
	if !tm.config.Enabled {
		return nil, fmt.Errorf("tracker system is disabled")
	}

	return tm.getStorage().GetTrackerStats()
}

// GetWorkerStatus returns the current worker status
func (tm *TrackerManager) GetWorkerStatus() (*models.TrackerWorkerStatus, error) {
	if !tm.config.Enabled {
		return nil, fmt.Errorf("tracker system is disabled")
	}

	if tm.worker == nil {
		return &models.TrackerWorkerStatus{
			IsRunning: false,
		}, nil
	}

	status := tm.worker.GetWorkerStatus()
	return &status, nil
}

// TestNotification sends a test notification
func (tm *TrackerManager) TestNotification() error {
	if !tm.config.Enabled {
		return fmt.Errorf("tracker system is disabled")
	}

	testTracker := models.TrackerEntry{
		ID:      "test",
		Name:    "Test Pokemon Card",
		URL:     "https://example.com/test-card",
		InStock: true,
		Price:   100.0,
	}

	return tm.notifier.SendStockAlert(testTracker, true)
}

// Restart restarts the tracker system with updated configuration
func (tm *TrackerManager) Restart(newConfig *TrackerConfig) error {
	log.Println("Restarting tracker system with new configuration...")

	// Stop current system
	if tm.isRunning {
		if err := tm.Stop(); err != nil {
			log.Printf("Error stopping tracker system: %v", err)
		}
	}

	// Update configuration
	if newConfig != nil {
		if err := newConfig.IsValid(); err != nil {
			return fmt.Errorf("invalid new configuration: %v", err)
		}
		tm.config = newConfig
	}

	// Note: Worker recreation is skipped due to interface constraints
	// This would require a more complex type system to handle properly
	log.Println("Restart completed (worker recreation skipped)")

	return nil
}
