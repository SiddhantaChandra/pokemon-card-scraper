package models

import (
	"fmt"
	"time"
)

// TrackerStatus represents the current status of a tracker entry
type TrackerStatus string

const (
	TrackerStatusActive  TrackerStatus = "active"
	TrackerStatusPaused  TrackerStatus = "paused"
	TrackerStatusDeleted TrackerStatus = "deleted"
)

// TrackerEntry represents a tracked Pokemon card URL
type TrackerEntry struct {
	ID          string        `json:"id" db:"id"`
	URL         string        `json:"url" db:"url"`
	Name        string        `json:"name" db:"name"`
	InStock     bool          `json:"in_stock" db:"in_stock"`
	Price       float64       `json:"price" db:"price"`
	ImageURL    string        `json:"image_url" db:"image_url"`
	Status      TrackerStatus `json:"status" db:"status"`
	LastChecked time.Time     `json:"last_checked" db:"last_checked"`
	CreatedAt   time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at" db:"updated_at"`
	UserID      string        `json:"user_id" db:"user_id"` // For future multi-user support
}

// TrackerSearchResult represents the result of a tracker search operation
type TrackerSearchResult struct {
	Trackers   []TrackerEntry `json:"trackers"`
	Total      int            `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

// TrackerFilterOptions represents search and filter parameters for trackers
type TrackerFilterOptions struct {
	Status    TrackerStatus `json:"status" form:"status"`
	InStock   *bool         `json:"in_stock" form:"in_stock"`
	UserID    string        `json:"user_id" form:"user_id"`
	SortBy    string        `json:"sort_by" form:"sort_by"`       // name, created_at, last_checked, price
	SortOrder string        `json:"sort_order" form:"sort_order"` // asc, desc
	Page      int           `json:"page" form:"page"`
	PageSize  int           `json:"page_size" form:"page_size"`
}

// DefaultTrackerFilterOptions returns filter options with sensible defaults
func DefaultTrackerFilterOptions() TrackerFilterOptions {
	return TrackerFilterOptions{
		Status:    TrackerStatusActive,
		SortBy:    "created_at",
		SortOrder: "desc",
		Page:      1,
		PageSize:  50,
	}
}

// IsActive returns true if the tracker is active
func (t *TrackerEntry) IsActive() bool {
	return t.Status == TrackerStatusActive
}

// FormattedPrice returns the price formatted as a string with currency
func (t *TrackerEntry) FormattedPrice() string {
	if t.Price > 0 {
		return fmt.Sprintf("¥%.2f", t.Price)
	}
	return "N/A"
}

// TimeSinceLastCheck returns a human-readable string of time since last check
func (t *TrackerEntry) TimeSinceLastCheck() string {
	if t.LastChecked.IsZero() {
		return "Never"
	}

	duration := time.Since(t.LastChecked)

	if duration < time.Minute {
		return "Just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		return fmt.Sprintf("%d hours ago", hours)
	} else {
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%d days ago", days)
	}
}

// Validate checks if the tracker entry has required fields
func (t *TrackerEntry) Validate() error {
	if t.URL == "" {
		return fmt.Errorf("URL is required")
	}

	if t.Name == "" {
		return fmt.Errorf("name is required")
	}

	if t.Status == "" {
		t.Status = TrackerStatusActive
	}

	return nil
}

// TrackerStats represents statistics about tracked items
type TrackerStats struct {
	TotalTrackers    int `json:"total_trackers"`
	ActiveTrackers   int `json:"active_trackers"`
	InStockTrackers  int `json:"in_stock_trackers"`
	OutStockTrackers int `json:"out_stock_trackers"`
	PausedTrackers   int `json:"paused_trackers"`
}

// TrackerWorkerStatus represents the status of the tracker worker
type TrackerWorkerStatus struct {
	IsRunning        bool      `json:"is_running"`
	LastScanTime     time.Time `json:"last_scan_time"`
	NextScanTime     time.Time `json:"next_scan_time"`
	ScanInterval     int       `json:"scan_interval_seconds"`
	ItemsProcessed   int       `json:"items_processed"`
	ErrorsEncounted  int       `json:"errors_encountered"`
	AverageCheckTime float64   `json:"average_check_time_seconds"`
}

// AddTrackerRequest represents a request to add a new tracker
type AddTrackerRequest struct {
	URL    string `json:"url" binding:"required"`
	Name   string `json:"name"`
	UserID string `json:"user_id"`
}

// UpdateTrackerRequest represents a request to update tracker fields
type UpdateTrackerRequest struct {
	Name   *string        `json:"name"`
	Status *TrackerStatus `json:"status"`
	UserID string         `json:"user_id"`
}

// TrackerCheckResult represents the result of checking a single tracker
type TrackerCheckResult struct {
	TrackerID string  `json:"tracker_id"`
	Success   bool    `json:"success"`
	InStock   bool    `json:"in_stock"`
	Price     float64 `json:"price"`
	ImageURL  string  `json:"image_url"`
	Error     string  `json:"error,omitempty"`
	CheckTime float64 `json:"check_time_seconds"`
}

// BatchCheckResult represents the result of checking multiple trackers
type BatchCheckResult struct {
	TotalChecked     int                  `json:"total_checked"`
	SuccessfulChecks int                  `json:"successful_checks"`
	FailedChecks     int                  `json:"failed_checks"`
	Results          []TrackerCheckResult `json:"results"`
	TotalTime        float64              `json:"total_time_seconds"`
	AverageTime      float64              `json:"average_time_seconds"`
}
