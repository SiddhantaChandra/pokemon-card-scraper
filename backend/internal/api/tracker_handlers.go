package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/SiddhantaChandra/pokemon-card-scraper/internal/monitor"
	"github.com/SiddhantaChandra/pokemon-card-scraper/pkg/models"
	"github.com/gin-gonic/gin"
)

// TrackerHandlers handles tracker-related HTTP requests
type TrackerHandlers struct {
	monitor *monitor.StockMonitor
}

// NewTrackerHandlers creates new tracker handlers
func NewTrackerHandlers(stockMonitor *monitor.StockMonitor) *TrackerHandlers {
	return &TrackerHandlers{
		monitor: stockMonitor,
	}
}

// AddTracker handles POST /api/tracker - Add new tracker
func (th *TrackerHandlers) AddTracker(c *gin.Context) {
	var request AddTrackerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate required fields
	if request.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "URL is required",
		})
		return
	}

	// Add tracker via monitor service
	tracker, err := th.monitor.AddTracker(request.URL, request.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to add tracker",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Tracker added successfully",
		"tracker": tracker,
	})
}

// GetTrackers handles GET /api/tracker - List all trackers with filtering
func (th *TrackerHandlers) GetTrackers(c *gin.Context) {
	// Parse query parameters
	filters := models.TrackerFilterOptions{
		Query:       c.Query("q"),
		InStockOnly: c.Query("in_stock_only") == "true",
		UserID:      c.Query("user_id"),
		SortBy:      c.DefaultQuery("sort_by", "created_at"),
		SortOrder:   c.DefaultQuery("sort_order", "desc"),
	}

	// Parse pagination
	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			filters.Page = p
		} else {
			filters.Page = 1
		}
	} else {
		filters.Page = 1
	}

	if pageSize := c.Query("page_size"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil && ps > 0 && ps <= 100 {
			filters.PageSize = ps
		} else {
			filters.PageSize = 50
		}
	} else {
		filters.PageSize = 50
	}

	// Get trackers via monitor service
	result, err := th.monitor.GetTrackers(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get trackers",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetTracker handles GET /api/tracker/:id - Get specific tracker
func (th *TrackerHandlers) GetTracker(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Tracker ID is required",
		})
		return
	}

	// Get single tracker (would need to add this method to monitor service)
	filters := models.TrackerFilterOptions{
		Page:     1,
		PageSize: 1,
	}

	result, err := th.monitor.GetTrackers(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get tracker",
			"details": err.Error(),
		})
		return
	}

	// Find tracker by ID
	var tracker *models.TrackerItem
	for _, t := range result.Trackers {
		if t.ID == id {
			tracker = &t
			break
		}
	}

	if tracker == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Tracker not found",
		})
		return
	}

	c.JSON(http.StatusOK, tracker)
}

// UpdateTracker handles PUT /api/tracker/:id - Update tracker
func (th *TrackerHandlers) UpdateTracker(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Tracker ID is required",
		})
		return
	}

	var request UpdateTrackerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Convert request to update fields map
	fields := make(map[string]interface{})
	if request.Name != nil {
		fields["name"] = *request.Name
	}
	if request.InStock != nil {
		fields["in_stock"] = *request.InStock
	}
	if request.Image != nil {
		fields["image"] = *request.Image
	}
	if request.PriceYen != nil {
		fields["price_yen"] = *request.PriceYen
	}
	if request.UserID != nil {
		fields["user_id"] = *request.UserID
	}

	// Update tracker via monitor service
	if err := th.monitor.UpdateTracker(id, fields); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update tracker",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Tracker updated successfully",
	})
}

// DeleteTracker handles DELETE /api/tracker/:id - Remove tracker
func (th *TrackerHandlers) DeleteTracker(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Tracker ID is required",
		})
		return
	}

	// Remove tracker via monitor service
	if err := th.monitor.RemoveTracker(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete tracker",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Tracker deleted successfully",
	})
}

// BulkAddTrackers handles POST /api/tracker/bulk - Bulk add trackers
func (th *TrackerHandlers) BulkAddTrackers(c *gin.Context) {
	var request BulkAddTrackersRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	if len(request.Trackers) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "At least one tracker is required",
		})
		return
	}

	if len(request.Trackers) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Maximum 100 trackers allowed per bulk operation",
		})
		return
	}

	// Add trackers one by one (could be optimized with batch operations)
	var addedTrackers []models.TrackerItem
	var errors []string

	for _, trackerReq := range request.Trackers {
		if trackerReq.URL == "" {
			errors = append(errors, "URL is required for all trackers")
			continue
		}

		tracker, err := th.monitor.AddTracker(trackerReq.URL, trackerReq.Name)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to add tracker %s: %v", trackerReq.URL, err))
			continue
		}

		addedTrackers = append(addedTrackers, *tracker)
	}

	response := gin.H{
		"message":        fmt.Sprintf("Bulk operation completed. Added %d trackers", len(addedTrackers)),
		"added_trackers": addedTrackers,
		"total_added":    len(addedTrackers),
		"total_failed":   len(errors),
	}

	if len(errors) > 0 {
		response["errors"] = errors
	}

	// Return 207 Multi-Status if there were partial failures
	if len(errors) > 0 && len(addedTrackers) > 0 {
		c.JSON(http.StatusMultiStatus, response)
	} else if len(errors) > 0 {
		c.JSON(http.StatusBadRequest, response)
	} else {
		c.JSON(http.StatusCreated, response)
	}
}

// Monitor Control Handlers

// StartMonitoring handles POST /api/monitor/start - Start monitoring
func (th *TrackerHandlers) StartMonitoring(c *gin.Context) {
	if err := th.monitor.StartMonitoring(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to start monitoring",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Monitoring started successfully",
		"status":  th.monitor.GetStatus(),
	})
}

// StopMonitoring handles POST /api/monitor/stop - Stop monitoring
func (th *TrackerHandlers) StopMonitoring(c *gin.Context) {
	if err := th.monitor.StopMonitoring(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to stop monitoring",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Monitoring stopped successfully",
		"status":  th.monitor.GetStatus(),
	})
}

// GetMonitorStatus handles GET /api/monitor/status - Get monitoring status
func (th *TrackerHandlers) GetMonitorStatus(c *gin.Context) {
	status := th.monitor.GetStatus()
	stats := th.monitor.GetStats()

	c.JSON(http.StatusOK, gin.H{
		"status": status,
		"stats":  stats,
	})
}

// GetMonitorStats handles GET /api/monitor/stats - Get monitoring statistics
func (th *TrackerHandlers) GetMonitorStats(c *gin.Context) {
	stats := th.monitor.GetStats()

	c.JSON(http.StatusOK, stats)
}

// Request/Response structures

// AddTrackerRequest represents a request to add a new tracker
type AddTrackerRequest struct {
	URL    string `json:"url" binding:"required"`
	Name   string `json:"name"`
	UserID string `json:"user_id"`
}

// UpdateTrackerRequest represents a request to update a tracker
type UpdateTrackerRequest struct {
	Name     *string `json:"name"`
	InStock  *bool   `json:"in_stock"`
	Image    *string `json:"image"`
	PriceYen *string `json:"price_yen"`
	UserID   *string `json:"user_id"`
}

// BulkAddTrackersRequest represents a request to bulk add trackers
type BulkAddTrackersRequest struct {
	Trackers []AddTrackerRequest `json:"trackers" binding:"required"`
}

// RegisterTrackerRoutes registers all tracker-related routes
func RegisterTrackerRoutes(router *gin.Engine, stockMonitor *monitor.StockMonitor) {
	handlers := NewTrackerHandlers(stockMonitor)

	api := router.Group("/api")
	{
		// Tracker CRUD operations
		tracker := api.Group("/tracker")
		{
			tracker.POST("", handlers.AddTracker)
			tracker.GET("", handlers.GetTrackers)
			tracker.GET("/:id", handlers.GetTracker)
			tracker.PUT("/:id", handlers.UpdateTracker)
			tracker.DELETE("/:id", handlers.DeleteTracker)
			tracker.POST("/bulk", handlers.BulkAddTrackers)
		}

		// Monitor control operations
		monitor := api.Group("/monitor")
		{
			monitor.POST("/start", handlers.StartMonitoring)
			monitor.POST("/stop", handlers.StopMonitoring)
			monitor.GET("/status", handlers.GetMonitorStatus)
			monitor.GET("/stats", handlers.GetMonitorStats)
		}
	}
}
