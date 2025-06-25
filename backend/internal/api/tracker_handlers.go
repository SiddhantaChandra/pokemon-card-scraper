package api

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/SiddhantaChandra/pokemon-card-scraper/internal/storage"
	"github.com/SiddhantaChandra/pokemon-card-scraper/pkg/models"
	"github.com/gin-gonic/gin"
)

// TrackerHandlers contains all tracker-related HTTP handlers
type TrackerHandlers struct {
	storage storage.TrackerStorage
}

// NewTrackerHandlers creates a new tracker handlers instance
func NewTrackerHandlers(storage storage.TrackerStorage) *TrackerHandlers {
	return &TrackerHandlers{
		storage: storage,
	}
}

// AddTracker handles POST /api/tracker/add
func (th *TrackerHandlers) AddTracker(c *gin.Context) {
	var req models.AddTrackerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
		return
	}

	// Validate URL
	if err := th.validateURL(req.URL); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid URL",
			"message": err.Error(),
		})
		return
	}

	// Generate name if not provided
	if req.Name == "" {
		req.Name = th.generateTrackerName(req.URL)
	}

	// Create tracker entry
	tracker := models.TrackerEntry{
		URL:     req.URL,
		Name:    req.Name,
		UserID:  req.UserID,
		Status:  models.TrackerStatusActive,
		InStock: false, // Will be updated on first check
		Price:   0,     // Will be updated on first check
	}

	// Save tracker
	if err := th.storage.SaveTracker(tracker); err != nil {
		if err == storage.ErrTrackerURLExists {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "URL already being tracked",
				"message": "This URL is already in your tracker list",
			})
			return
		}

		log.Printf("Failed to save tracker: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to add tracker",
			"message": "Could not save tracker to database",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Tracker added successfully",
		"tracker": tracker,
	})
}

// GetTrackers handles GET /api/tracker/list
func (th *TrackerHandlers) GetTrackers(c *gin.Context) {
	// Parse filter options from query parameters
	filters := th.parseTrackerFilterOptions(c.Request.URL.Query())

	// Search trackers
	result, err := th.storage.SearchTrackers(filters)
	if err != nil {
		log.Printf("Failed to get trackers: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve trackers",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetTracker handles GET /api/tracker/:id
func (th *TrackerHandlers) GetTracker(c *gin.Context) {
	trackerID := c.Param("id")
	if trackerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Missing tracker ID",
			"message": "Tracker ID is required",
		})
		return
	}

	// Get tracker from storage
	tracker, err := th.storage.GetTracker(trackerID)
	if err != nil {
		if err == storage.ErrTrackerNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Tracker not found",
				"message": fmt.Sprintf("Tracker with ID %s not found", trackerID),
			})
			return
		}

		log.Printf("Failed to get tracker: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve tracker",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, tracker)
}

// UpdateTracker handles PUT /api/tracker/:id
func (th *TrackerHandlers) UpdateTracker(c *gin.Context) {
	trackerID := c.Param("id")
	if trackerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Missing tracker ID",
			"message": "Tracker ID is required",
		})
		return
	}

	var req models.UpdateTrackerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
		return
	}

	// Get existing tracker
	tracker, err := th.storage.GetTracker(trackerID)
	if err != nil {
		if err == storage.ErrTrackerNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Tracker not found",
				"message": fmt.Sprintf("Tracker with ID %s not found", trackerID),
			})
			return
		}

		log.Printf("Failed to get tracker for update: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve tracker",
			"message": err.Error(),
		})
		return
	}

	// Update fields
	if req.Name != nil {
		tracker.Name = *req.Name
	}
	if req.Status != nil {
		tracker.Status = *req.Status
	}

	// Save updated tracker
	if err := th.storage.UpdateTracker(*tracker); err != nil {
		log.Printf("Failed to update tracker: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update tracker",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Tracker updated successfully",
		"tracker": tracker,
	})
}

// DeleteTracker handles DELETE /api/tracker/:id
func (th *TrackerHandlers) DeleteTracker(c *gin.Context) {
	trackerID := c.Param("id")
	if trackerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Missing tracker ID",
			"message": "Tracker ID is required",
		})
		return
	}

	// Delete tracker
	if err := th.storage.DeleteTracker(trackerID); err != nil {
		if err == storage.ErrTrackerNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Tracker not found",
				"message": fmt.Sprintf("Tracker with ID %s not found", trackerID),
			})
			return
		}

		log.Printf("Failed to delete tracker: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete tracker",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Tracker deleted successfully",
	})
}

// CheckNow handles POST /api/tracker/check-now
func (th *TrackerHandlers) CheckNow(c *gin.Context) {
	// Get all active trackers
	trackers, err := th.storage.GetActiveTrackers()
	if err != nil {
		log.Printf("Failed to get active trackers: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get trackers",
			"message": err.Error(),
		})
		return
	}

	if len(trackers) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "No active trackers to check",
			"result": models.BatchCheckResult{
				TotalChecked:     0,
				SuccessfulChecks: 0,
				FailedChecks:     0,
				Results:          []models.TrackerCheckResult{},
				TotalTime:        0,
				AverageTime:      0,
			},
		})
		return
	}

	// TODO: Implement actual checking logic here
	// For now, return a placeholder response
	result := models.BatchCheckResult{
		TotalChecked:     len(trackers),
		SuccessfulChecks: 0,
		FailedChecks:     len(trackers),
		Results:          make([]models.TrackerCheckResult, len(trackers)),
		TotalTime:        0,
		AverageTime:      0,
	}

	for i, tracker := range trackers {
		result.Results[i] = models.TrackerCheckResult{
			TrackerID: tracker.ID,
			Success:   false,
			Error:     "Checking functionality not yet implemented",
			CheckTime: 0,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Manual check initiated",
		"result":  result,
	})
}

// GetTrackerStatus handles GET /api/tracker/status
func (th *TrackerHandlers) GetTrackerStatus(c *gin.Context) {
	// Get tracker statistics
	stats, err := th.storage.GetTrackerStats()
	if err != nil {
		log.Printf("Failed to get tracker stats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get tracker statistics",
			"message": err.Error(),
		})
		return
	}

	// TODO: Get actual worker status from tracker worker
	// For now, return placeholder status
	workerStatus := models.TrackerWorkerStatus{
		IsRunning:        false, // Will be updated when worker is implemented
		LastScanTime:     time.Time{},
		NextScanTime:     time.Time{},
		ScanInterval:     3600, // 1 hour
		ItemsProcessed:   0,
		ErrorsEncounted:  0,
		AverageCheckTime: 0,
	}

	c.JSON(http.StatusOK, gin.H{
		"stats":  stats,
		"worker": workerStatus,
	})
}

// GetTrackerOptions handles GET /api/tracker/options
func (th *TrackerHandlers) GetTrackerOptions(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"statuses": []string{
			string(models.TrackerStatusActive),
			string(models.TrackerStatusPaused),
		},
		"sort_options": []string{
			"name",
			"created_at",
			"last_checked",
			"price",
		},
		"sort_orders": []string{
			"asc",
			"desc",
		},
	})
}

// Helper functions

func (th *TrackerHandlers) validateURL(urlStr string) error {
	if urlStr == "" {
		return fmt.Errorf("URL is required")
	}

	// Parse URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %v", err)
	}

	// Check scheme
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL must use http or https protocol")
	}

	// Check host
	if parsedURL.Host == "" {
		return fmt.Errorf("URL must have a valid host")
	}

	// Optional: Add domain whitelist validation here
	// For now, allow any valid URL

	return nil
}

func (th *TrackerHandlers) generateTrackerName(urlStr string) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "Unknown Item"
	}

	// Extract a meaningful name from the URL
	host := parsedURL.Host
	path := parsedURL.Path

	// Remove common prefixes
	host = strings.TrimPrefix(host, "www.")

	// Try to extract item name from path
	pathParts := strings.Split(path, "/")
	for i := len(pathParts) - 1; i >= 0; i-- {
		part := strings.TrimSpace(pathParts[i])
		if part != "" && part != "products" && part != "item" {
			// Clean up the part
			part = strings.ReplaceAll(part, "-", " ")
			part = strings.ReplaceAll(part, "_", " ")
			return fmt.Sprintf("%s - %s", strings.Title(part), host)
		}
	}

	return fmt.Sprintf("Item from %s", host)
}

func (th *TrackerHandlers) parseTrackerFilterOptions(values url.Values) models.TrackerFilterOptions {
	filters := models.DefaultTrackerFilterOptions()

	// Status filter
	if status := values.Get("status"); status != "" {
		filters.Status = models.TrackerStatus(status)
	}

	// In stock filter
	if inStock := values.Get("in_stock"); inStock != "" {
		if inStock == "true" {
			inStockBool := true
			filters.InStock = &inStockBool
		} else if inStock == "false" {
			inStockBool := false
			filters.InStock = &inStockBool
		}
	}

	// User ID filter
	if userID := values.Get("user_id"); userID != "" {
		filters.UserID = userID
	}

	// Sort options
	if sortBy := values.Get("sort_by"); sortBy != "" {
		filters.SortBy = sortBy
	}

	if sortOrder := values.Get("sort_order"); sortOrder != "" {
		filters.SortOrder = sortOrder
	}

	// Pagination
	if pageStr := values.Get("page"); pageStr != "" {
		if page := parseInt(pageStr, 1); page > 0 {
			filters.Page = page
		}
	}

	if pageSizeStr := values.Get("page_size"); pageSizeStr != "" {
		if pageSize := parseInt(pageSizeStr, 50); pageSize > 0 && pageSize <= 100 {
			filters.PageSize = pageSize
		}
	}

	return filters
}

func parseInt(str string, defaultVal int) int {
	if str == "" {
		return defaultVal
	}

	// Simple integer parsing
	result := 0
	for _, r := range str {
		if r >= '0' && r <= '9' {
			result = result*10 + int(r-'0')
		} else {
			return defaultVal
		}
	}

	if result == 0 {
		return defaultVal
	}
	return result
}
