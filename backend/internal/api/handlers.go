package api

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/SiddhantaChandra/pokemon-card-scraper/internal/scraper"
	"github.com/SiddhantaChandra/pokemon-card-scraper/internal/search"
	"github.com/SiddhantaChandra/pokemon-card-scraper/internal/storage"
	"github.com/gin-gonic/gin"
)

// Handlers contains all HTTP handlers
type Handlers struct {
	storage      storage.Storage
	scraper      *scraper.Scraper
	searchEngine *search.SearchEngine
	queryBuilder *search.QueryBuilder
}

// NewHandlers creates a new handlers instance
func NewHandlers(storage storage.Storage, scraper *scraper.Scraper, searchEngine *search.SearchEngine) *Handlers {
	return &Handlers{
		storage:      storage,
		scraper:      scraper,
		searchEngine: searchEngine,
		queryBuilder: search.NewQueryBuilder(),
	}
}

// HealthCheck handles health check requests
func (h *Handlers) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	})
}

// GetAllCards handles GET /api/cards
func (h *Handlers) GetAllCards(c *gin.Context) {
	// Parse query parameters
	filterOpts, err := h.queryBuilder.BuildFilterOptions(c.Request.URL.Query())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid query parameters",
			"message": err.Error(),
		})
		return
	}

	// Get cards from storage
	result, err := h.storage.SearchCards(filterOpts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve cards",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// SearchCards handles GET /api/cards/search
func (h *Handlers) SearchCards(c *gin.Context) {
	// Parse search options
	searchOpts, err := h.queryBuilder.BuildSearchOptions(c.Request.URL.Query())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid search parameters",
			"message": err.Error(),
		})
		return
	}

	// Validate search options
	if err := h.queryBuilder.ValidateSearchOptions(searchOpts); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid search parameters",
			"message": err.Error(),
		})
		return
	}

	// Perform search
	result, err := h.searchEngine.SearchCards(searchOpts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Search failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetCard handles GET /api/cards/:id
func (h *Handlers) GetCard(c *gin.Context) {
	cardID := c.Param("id")
	if cardID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Missing card ID",
			"message": "Card ID is required",
		})
		return
	}

	// Get card from storage
	card, err := h.storage.GetCard(cardID)
	if err != nil {
		if err == storage.ErrCardNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Card not found",
				"message": fmt.Sprintf("Card with ID %s not found", cardID),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve card",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, card)
}

// GetSuggestions handles GET /api/cards/suggestions
func (h *Handlers) GetSuggestions(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Missing query parameter",
			"message": "Query parameter 'q' is required",
		})
		return
	}

	// Parse limit
	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 50 {
			limit = parsedLimit
		}
	}

	// Get suggestions
	suggestions, err := h.searchEngine.GetSuggestions(query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get suggestions",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"suggestions": suggestions,
		"query":       query,
		"limit":       limit,
	})
}

// StartScrape handles POST /api/scrape/start
func (h *Handlers) StartScrape(c *gin.Context) {
	// Parse request body for scrape options
	var req StartScrapeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Use default parameters if no body provided
		req = StartScrapeRequest{
			InStockOnly: true,
			MaxPages:    0, // 0 means scrape all pages
		}
	}

	// Check if scraper is already running
	if h.scraper.IsRunning() {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Scraper already running",
			"message": "Another scrape job is currently in progress",
		})
		return
	}

	// Start scraping in a goroutine
	go func() {
		params := scraper.SearchParams{
			InStockOnly: req.InStockOnly,
			MaxPages:    req.MaxPages,
		}

		// Always use ScrapeAllPages for proper status tracking
		h.scraper.ScrapeAllPages(params, nil)
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message":       "Scrape job started",
		"in_stock_only": req.InStockOnly,
		"max_pages":     req.MaxPages,
		"started_at":    time.Now().UTC(),
	})
}

// GetScrapeStatus handles GET /api/scrape/status
func (h *Handlers) GetScrapeStatus(c *gin.Context) {
	status := h.scraper.GetStatus()

	progressPercent := 0.0
	if status.TotalPages > 0 {
		progressPercent = float64(status.CurrentPage) / float64(status.TotalPages) * 100
	}

	c.JSON(http.StatusOK, gin.H{
		"running":                  h.scraper.IsRunning(),
		"paused":                   h.scraper.IsPaused(),
		"current_page":             status.CurrentPage,
		"total_pages":              status.TotalPages,
		"items_scraped":            status.ItemsScraped,
		"cards_per_minute":         status.CardsPerMinute,
		"estimated_time_remaining": status.EstimatedTimeRemaining,
		"last_updated":             status.LastUpdated,
		"start_time":               status.StartTime,
		"paused_at":                status.PausedAt,
		"progress_percent":         progressPercent,
	})
}

// StopScrape handles POST /api/scrape/stop
func (h *Handlers) StopScrape(c *gin.Context) {
	if !h.scraper.IsRunning() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "No scrape job running",
			"message": "There is no active scrape job to stop",
		})
		return
	}

	h.scraper.Stop()

	c.JSON(http.StatusOK, gin.H{
		"message":    "Scrape job stopped",
		"stopped_at": time.Now().UTC(),
	})
}

// PauseScrape handles POST /api/scrape/pause
func (h *Handlers) PauseScrape(c *gin.Context) {
	if !h.scraper.IsRunning() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "No scrape job running",
			"message": "There is no active scrape job to pause",
		})
		return
	}

	if h.scraper.IsPaused() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Scrape job already paused",
			"message": "The scrape job is already paused",
		})
		return
	}

	if h.scraper.Pause() {
		c.JSON(http.StatusOK, gin.H{
			"message":   "Scrape job paused",
			"paused_at": time.Now().UTC(),
		})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to pause scrape job",
			"message": "Could not pause the scrape job",
		})
	}
}

// ResumeScrape handles POST /api/scrape/resume
func (h *Handlers) ResumeScrape(c *gin.Context) {
	if !h.scraper.IsRunning() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "No scrape job running",
			"message": "There is no scrape job to resume",
		})
		return
	}

	if !h.scraper.IsPaused() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Scrape job not paused",
			"message": "The scrape job is not currently paused",
		})
		return
	}

	if h.scraper.Resume() {
		c.JSON(http.StatusOK, gin.H{
			"message":    "Scrape job resumed",
			"resumed_at": time.Now().UTC(),
		})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to resume scrape job",
			"message": "Could not resume the scrape job",
		})
	}
}

// GetStats handles GET /api/stats
func (h *Handlers) GetStats(c *gin.Context) {
	// Get search statistics
	searchStats, err := h.searchEngine.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get statistics",
			"message": err.Error(),
		})
		return
	}

	// Get scraper status
	scraperStatus := h.scraper.GetStatus()

	c.JSON(http.StatusOK, gin.H{
		"database": searchStats,
		"scraper": gin.H{
			"running":       h.scraper.IsRunning(),
			"last_run":      scraperStatus.LastUpdated,
			"items_scraped": scraperStatus.ItemsScraped,
		},
		"generated_at": time.Now().UTC(),
	})
}

// GetSortOptions handles GET /api/sort-options
func (h *Handlers) GetSortOptions(c *gin.Context) {
	options := h.queryBuilder.GetSortOptions()

	c.JSON(http.StatusOK, gin.H{
		"sort_options": options,
	})
}

// NotFound handles 404 errors
func (h *Handlers) NotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{
		"error":   "Not found",
		"message": fmt.Sprintf("Endpoint %s %s not found", c.Request.Method, c.Request.URL.Path),
		"path":    c.Request.URL.Path,
		"method":  c.Request.Method,
	})
}

// Request/Response structs

// StartScrapeRequest represents a request to start scraping
type StartScrapeRequest struct {
	InStockOnly bool `json:"in_stock_only"`
	MaxPages    int  `json:"max_pages"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Helper functions for common response patterns

// RespondWithError sends an error response with the specified status code
func RespondWithError(c *gin.Context, statusCode int, error string, message string) {
	c.JSON(statusCode, ErrorResponse{
		Error:   error,
		Message: message,
		Code:    statusCode,
	})
}

// RespondWithSuccess sends a success response with optional data
func RespondWithSuccess(c *gin.Context, message string, data interface{}) {
	response := SuccessResponse{
		Message: message,
	}

	if data != nil {
		response.Data = data
	}

	c.JSON(http.StatusOK, response)
}

// ImageProxy handles image proxy requests
func (h *Handlers) ImageProxy(c *gin.Context) {
	// Parse image URL
	imageURL, err := url.Parse(c.Query("url"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid image URL",
			"message": err.Error(),
		})
		return
	}

	// Fetch image data
	resp, err := http.Get(imageURL.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch image",
			"message": err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	// Check if the image is accessible
	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{
			"error":   "Image not accessible",
			"message": fmt.Sprintf("Image status code: %d", resp.StatusCode),
		})
		return
	}

	// Copy image data to response
	c.Writer.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	c.Writer.Header().Set("Content-Length", resp.Header.Get("Content-Length"))
	c.Writer.Header().Set("Cache-Control", "public, max-age=31536000")
	c.Writer.Header().Set("Expires", time.Now().Add(30*24*time.Hour).Format(http.TimeFormat))
	io.Copy(c.Writer, resp.Body)
}

// ResetDatabase handles DELETE /api/database/reset
func (h *Handlers) ResetDatabase(c *gin.Context) {
	// Check if scraper is currently running
	if h.scraper.IsRunning() {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Cannot reset database while scraping",
			"message": "Please stop the current scrape job before resetting the database",
		})
		return
	}

	// Get current stats before deletion
	var statsBeforeDeletion interface{}
	if stats, err := h.searchEngine.GetStats(); err == nil {
		statsBeforeDeletion = stats
	}

	// Clear all data
	if clearable, ok := h.storage.(interface{ ClearAllData() error }); ok {
		err := clearable.ClearAllData()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to reset database",
				"message": err.Error(),
			})
			return
		}
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Database reset not supported",
			"message": "The current storage implementation does not support database reset",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":            "Database successfully reset",
		"reset_at":           time.Now().UTC(),
		"stats_before_reset": statsBeforeDeletion,
		"next_steps":         "You can now start a new scraping job to populate the database",
	})
}

// RestartScrape handles POST /api/scrape/restart
func (h *Handlers) RestartScrape(c *gin.Context) {
	// Parse request body for scrape options
	var req StartScrapeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Use default parameters if no body provided
		req = StartScrapeRequest{
			InStockOnly: true,
			MaxPages:    0, // 0 means scrape all pages
		}
	}

	// Stop current scraping if running
	wasRunning := h.scraper.IsRunning()
	if wasRunning {
		h.scraper.Stop()
		// Wait a moment for the scraper to fully stop
		time.Sleep(2 * time.Second)
	}

	// Start scraping in a goroutine
	go func() {
		params := scraper.SearchParams{
			InStockOnly: req.InStockOnly,
		}

		if req.MaxPages > 0 {
			// Scrape limited pages
			for page := 1; page <= req.MaxPages; page++ {
				params.Page = page
				url := scraper.BuildSearchURL(params)
				_, err := h.scraper.ScrapePage(url)
				if err != nil {
					break
				}
			}
		} else {
			// Scrape all pages
			h.scraper.ScrapeAllPages(params, nil)
		}
	}()

	// Wait a moment to ensure scraping has started
	time.Sleep(1 * time.Second)

	status := h.scraper.GetStatus()
	c.JSON(http.StatusOK, gin.H{
		"message":       "Scraping restarted successfully",
		"was_running":   wasRunning,
		"started_at":    time.Now().UTC(),
		"running":       h.scraper.IsRunning(),
		"current_page":  status.CurrentPage,
		"items_scraped": status.ItemsScraped,
		"options": gin.H{
			"in_stock_only": req.InStockOnly,
			"max_pages":     req.MaxPages,
		},
	})
}
