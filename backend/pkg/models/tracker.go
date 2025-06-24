package models

import (
	"time"
)

// TrackerItem represents a monitored URL for stock tracking
type TrackerItem struct {
	ID          string    `json:"id" db:"id"`
	URL         string    `json:"url" db:"url"`
	Name        string    `json:"name" db:"name"`
	InStock     bool      `json:"in_stock" db:"in_stock"`
	Image       string    `json:"image" db:"image"`
	PriceYen    string    `json:"price_yen" db:"price_yen"`
	LastUpdated time.Time `json:"last_updated" db:"last_updated"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UserID      string    `json:"user_id" db:"user_id"`
}

// TrackerSearchResult represents the result of a tracker search operation
type TrackerSearchResult struct {
	Trackers   []TrackerItem `json:"trackers"`
	Total      int           `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	TotalPages int           `json:"total_pages"`
}

// TrackerFilterOptions represents search and filter parameters for trackers
type TrackerFilterOptions struct {
	Query       string `json:"query" form:"q"`
	InStockOnly bool   `json:"in_stock_only" form:"in_stock_only"`
	UserID      string `json:"user_id" form:"user_id"`
	SortBy      string `json:"sort_by" form:"sort_by"`       // name, created_at, last_updated
	SortOrder   string `json:"sort_order" form:"sort_order"` // asc, desc
	Page        int    `json:"page" form:"page"`
	PageSize    int    `json:"page_size" form:"page_size"`
}

// DefaultTrackerFilterOptions returns filter options with sensible defaults
func DefaultTrackerFilterOptions() TrackerFilterOptions {
	return TrackerFilterOptions{
		InStockOnly: false,
		SortBy:      "created_at",
		SortOrder:   "desc",
		Page:        1,
		PageSize:    50,
	}
}

// TrackerUpdate represents an update to a tracker item
type TrackerUpdate struct {
	ID     string
	Fields map[string]interface{}
}

// IsActive returns true if the tracker was recently updated (within last 24 hours)
func (t *TrackerItem) IsActive() bool {
	return time.Since(t.LastUpdated) <= 24*time.Hour
}

// IsStale returns true if the tracker hasn't been updated for more than 6 hours
func (t *TrackerItem) IsStale() bool {
	return time.Since(t.LastUpdated) > 6*time.Hour
}
