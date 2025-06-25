package api

import (
	"context"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/SiddhantaChandra/pokemon-card-scraper/internal/storage"
	"github.com/SiddhantaChandra/pokemon-card-scraper/internal/tracker"
	"github.com/SiddhantaChandra/pokemon-card-scraper/pkg/models"
	"github.com/chromedp/chromedp"
	"github.com/gin-gonic/gin"
)

// SimpleTrackerResponse represents a simplified tracker response for API compatibility
type SimpleTrackerResponse struct {
	ID          string    `json:"id"`
	URL         string    `json:"url"`
	Name        string    `json:"name"`
	InStock     bool      `json:"in_stock"`
	Price       float64   `json:"price"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	LastChecked time.Time `json:"last_checked"`
	ImageURL    string    `json:"image_url"`
}

// convertToSimpleResponse converts a TrackerEntry to SimpleTrackerResponse
func convertToSimpleResponse(tracker models.TrackerEntry) SimpleTrackerResponse {
	return SimpleTrackerResponse{
		ID:          tracker.ID,
		URL:         tracker.URL,
		Name:        tracker.Name,
		InStock:     tracker.InStock,
		Price:       tracker.Price,
		Status:      string(tracker.Status),
		CreatedAt:   tracker.CreatedAt,
		UpdatedAt:   tracker.UpdatedAt,
		LastChecked: tracker.LastChecked,
		ImageURL:    tracker.ImageURL,
	}
}

// convertFromSimpleResponse converts a SimpleTrackerResponse to TrackerEntry
func convertFromSimpleResponse(simple SimpleTrackerResponse) models.TrackerEntry {
	return models.TrackerEntry{
		ID:          simple.ID,
		URL:         simple.URL,
		Name:        simple.Name,
		InStock:     simple.InStock,
		Price:       simple.Price,
		Status:      models.TrackerStatus(simple.Status),
		CreatedAt:   simple.CreatedAt,
		UpdatedAt:   simple.UpdatedAt,
		LastChecked: simple.LastChecked,
		ImageURL:    simple.ImageURL,
	}
}

// SimpleTrackerStats represents simplified tracker statistics
type SimpleTrackerStats struct {
	TotalTrackers   int `json:"total_trackers"`
	ActiveTrackers  int `json:"active_trackers"`
	InStockTrackers int `json:"in_stock_trackers"`
}

// SimpleWorkerStatus represents simplified worker status
type SimpleWorkerStatus struct {
	IsRunning       bool      `json:"is_running"`
	LastScanTime    time.Time `json:"last_scan_time"`
	NextScanTime    time.Time `json:"next_scan_time"`
	ScanInterval    int       `json:"scan_interval_seconds"`
	ItemsProcessed  int       `json:"items_processed"`
	ErrorsEncounted int       `json:"errors_encountered"`
}

// SimplifiedTrackerHandlers provides basic tracker endpoints
type SimplifiedTrackerHandlers struct {
	storage         storage.TrackerStorage
	isWorkerRunning bool
	lastScanTime    time.Time
	workerStopChan  chan bool
}

// NewSimplifiedTrackerHandlers creates new simplified tracker handlers
func NewSimplifiedTrackerHandlers(st interface{}) *SimplifiedTrackerHandlers {
	// Try to cast to TrackerStorage interface
	trackerStorage, ok := st.(storage.TrackerStorage)
	if !ok {
		log.Printf("Warning: Provided storage does not implement TrackerStorage interface")
		// For now, create a minimal implementation - we'll fix this properly later
		return &SimplifiedTrackerHandlers{
			storage:         nil,
			isWorkerRunning: false,
			workerStopChan:  make(chan bool),
		}
	}

	sth := &SimplifiedTrackerHandlers{
		storage:         trackerStorage,
		isWorkerRunning: false,
		workerStopChan:  make(chan bool),
	}

	// Start the background worker
	go sth.startBackgroundWorker()

	return sth
}

// startBackgroundWorker runs periodic checks every 30 seconds
func (sth *SimplifiedTrackerHandlers) startBackgroundWorker() {
	sth.isWorkerRunning = true

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if sth.storage == nil {
				continue // Skip if storage is not available
			}
			trackers, err := sth.storage.GetActiveTrackers()
			if err != nil {
				continue
			}
			if len(trackers) > 0 {
				sth.performBackgroundCheck()
			}
		case <-sth.workerStopChan:
			sth.isWorkerRunning = false
			return
		}
	}
}

// performBackgroundCheck checks all trackers in background
func (sth *SimplifiedTrackerHandlers) performBackgroundCheck() {
	if sth.storage == nil {
		return
	}

	sth.lastScanTime = time.Now()
	successCount := 0

	trackers, err := sth.storage.GetActiveTrackers()
	if err != nil {
		return
	}

	for _, tracker := range trackers {
		simple := convertToSimpleResponse(tracker)

		if sth.checkTrackerURL(&simple) {
			successCount++
			// Update the tracker in storage with new data using UpdateTrackerStatus
			if err := sth.storage.UpdateTrackerStatus(tracker.ID, simple.InStock, simple.Price, simple.ImageURL); err != nil {
				// Silent fail - don't spam logs
			}
		}
	}
}

// AddTracker handles POST /api/tracker/add
func (sth *SimplifiedTrackerHandlers) AddTracker(c *gin.Context) {
	if sth.storage == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Tracker storage not available",
			"message": "Persistent tracker storage is not properly configured",
		})
		return
	}

	var req struct {
		URL  string `json:"url" binding:"required"`
		Name string `json:"name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
		return
	}

	// Auto-generate name if not provided
	name := req.Name
	if name == "" {
		name = generateNameFromURL(req.URL)
	}

	// Create new tracker entry
	tracker := models.TrackerEntry{
		URL:    req.URL,
		Name:   name,
		Status: models.TrackerStatusActive,
	}

	// Save to storage
	if err := sth.storage.SaveTracker(tracker); err != nil {
		if err == storage.ErrTrackerURLExists {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "URL already tracked",
				"message": "This URL is already being tracked",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to save tracker",
			"message": err.Error(),
		})
		return
	}

	response := convertToSimpleResponse(tracker)
	c.JSON(http.StatusCreated, gin.H{
		"message": "Tracker added successfully",
		"tracker": response,
	})
}

// GetTrackers handles GET /api/tracker/list
func (sth *SimplifiedTrackerHandlers) GetTrackers(c *gin.Context) {
	if sth.storage == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Tracker storage not available",
			"message": "Persistent tracker storage is not properly configured",
		})
		return
	}

	trackers, err := sth.storage.GetAllTrackers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get trackers",
			"message": err.Error(),
		})
		return
	}

	// Convert to simple responses
	simpleTrackers := make([]SimpleTrackerResponse, len(trackers))
	for i, tracker := range trackers {
		simpleTrackers[i] = convertToSimpleResponse(tracker)
	}

	c.JSON(http.StatusOK, gin.H{
		"trackers":    simpleTrackers,
		"total":       len(simpleTrackers),
		"page":        1,
		"page_size":   50,
		"total_pages": 1,
	})
}

// GetTracker handles GET /api/tracker/:id
func (sth *SimplifiedTrackerHandlers) GetTracker(c *gin.Context) {
	if sth.storage == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Tracker storage not available",
			"message": "Persistent tracker storage is not properly configured",
		})
		return
	}

	trackerID := c.Param("id")

	tracker, err := sth.storage.GetTracker(trackerID)
	if err != nil {
		if err == storage.ErrTrackerNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Tracker not found",
				"message": "Tracker with the specified ID does not exist",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get tracker",
			"message": err.Error(),
		})
		return
	}

	response := convertToSimpleResponse(*tracker)
	c.JSON(http.StatusOK, response)
}

// DeleteTracker handles DELETE /api/tracker/:id
func (sth *SimplifiedTrackerHandlers) DeleteTracker(c *gin.Context) {
	if sth.storage == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Tracker storage not available",
			"message": "Persistent tracker storage is not properly configured",
		})
		return
	}

	trackerID := c.Param("id")

	if err := sth.storage.DeleteTracker(trackerID); err != nil {
		if err == storage.ErrTrackerNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Tracker not found",
				"message": "Tracker with the specified ID does not exist",
			})
			return
		}
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
func (sth *SimplifiedTrackerHandlers) CheckNow(c *gin.Context) {
	if sth.storage == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Tracker storage not available",
			"message": "Persistent tracker storage is not properly configured",
		})
		return
	}

	trackers, err := sth.storage.GetActiveTrackers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get trackers",
			"message": err.Error(),
		})
		return
	}

	checkedCount := len(trackers)
	successfulChecks := 0
	failedChecks := 0
	startTime := time.Now()

	log.Printf("Starting manual check of %d trackers", checkedCount)

	// Actually check each tracker URL
	for _, tracker := range trackers {
		log.Printf("Checking tracker %s: %s", tracker.ID, tracker.URL)

		simple := convertToSimpleResponse(tracker)
		// Perform real scraping check
		success := sth.checkTrackerURL(&simple)
		if success {
			successfulChecks++
			log.Printf("Successfully checked tracker %s - In stock: %t, Price: %.2f", tracker.ID, simple.InStock, simple.Price)

			// Update the tracker in storage
			updatedTracker := convertFromSimpleResponse(simple)
			if err := sth.storage.UpdateTracker(updatedTracker); err != nil {
				log.Printf("Error updating tracker %s: %v", tracker.ID, err)
			}
		} else {
			failedChecks++
			log.Printf("Failed to check tracker %s", tracker.ID)
		}
	}

	totalTime := time.Since(startTime).Seconds()
	log.Printf("Manual check completed in %.2f seconds - Success: %d, Failed: %d", totalTime, successfulChecks, failedChecks)

	c.JSON(http.StatusOK, gin.H{
		"message":           "Manual check completed",
		"total_checked":     checkedCount,
		"successful_checks": successfulChecks,
		"failed_checks":     failedChecks,
		"total_time":        totalTime,
	})
}

// checkTrackerURL performs actual web scraping to check a tracker URL using ChromeDP
func (sth *SimplifiedTrackerHandlers) checkTrackerURL(tracker *SimpleTrackerResponse) bool {
	defer func() {
		if r := recover(); r != nil {
			// Silent recovery - don't spam logs
		}
	}()

	// Create context with Docker-friendly Chrome options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-features", "VizDisplayCompositor"),
		chromedp.Flag("disable-logging", true),
		chromedp.Flag("silent", true),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.Flag("log-level", "3"), // Only show fatal errors
		chromedp.Flag("enable-logging", false),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("disable-background-networking", true),
		chromedp.UserAgent("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(func(string, ...interface{}) {}))
	defer cancel()

	// Set longer timeout for Japanese sites
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var foundPrice float64
	var isInStock bool
	var imageURL string

	// Variables to store extracted data
	var priceText string
	var stockText string
	var imageSrc string

	// Run ChromeDP tasks - based on real HTML structure from torecacamp-pokemon.com
	err := chromedp.Run(ctx,
		// Navigate to the URL
		chromedp.Navigate(tracker.URL),

		// Wait for page to be ready
		chromedp.WaitReady("body"),

		// Extract price text - look for "販売価格¥" pattern
		chromedp.Evaluate(`
			(() => {
				// Method 1: Look for specific "販売価格¥" pattern
				const priceMatch1 = document.body.innerText.match(/販売価格¥([0-9,]+)/);
				if (priceMatch1) return priceMatch1[0];
				
				// Method 2: Look for any ¥ symbol followed by numbers
				const priceMatch2 = document.body.innerText.match(/¥([0-9,]+)/);
				if (priceMatch2) return priceMatch2[0];
				
				// Method 3: Look for price in specific elements
				const priceElements = document.querySelectorAll('.price, .product-price, [data-price]');
				for (const el of priceElements) {
					if (el.textContent && el.textContent.includes('¥')) {
						return el.textContent.trim();
					}
				}
				
				// Method 4: Look for any text containing yen symbol and numbers
				const textContent = document.body.innerText;
				const allPriceMatches = textContent.match(/[¥￥][0-9,]+/g);
				if (allPriceMatches && allPriceMatches.length > 0) {
					return allPriceMatches[0];
				}
				
				return '';
			})()
		`, &priceText),

		// Extract stock status with targeted approach
		chromedp.Evaluate(`
			(() => {
				// Method 1: Look for explicit out of stock indicators
				const outOfStockTexts = ['在庫なし', '在庫切れ', '売り切れ'];
				for (const text of outOfStockTexts) {
					if (document.body.innerText.includes(text)) {
						return 'OUT_OF_STOCK';
					}
				}
				
				// Method 2: Look for stock quantity
				const stockMatch = document.body.innerText.match(/在庫数?\s*(\d+)\s*個/);
				if (stockMatch) {
					return 'IN_STOCK_' + stockMatch[1];
				}
				
				// Method 3: Look for stock availability text
				if (document.body.innerText.includes('在庫数')) {
					return 'HAS_STOCK_TEXT';
				}
				
				// Method 4: Check for add to cart functionality
				if (document.body.innerText.includes('カートに追加')) {
					return 'CAN_ADD_TO_CART';
				}
				
				return 'NO_STOCK_INFO';
			})()
		`, &stockText),

		// Extract product image - try multiple selectors
		chromedp.Evaluate(`
			(() => {
				// Look for product images in common locations
				const imgSelectors = [
					'img[data-zoom]',
					'img[data-srcset]', 
					'.product-gallery__image img',
					'.product-image img',
					'img[alt*="ポケモン"]',
					'img[src*="product"]',
					'img[src*="cdn/shop"]'
				];
				
				for (const selector of imgSelectors) {
					const img = document.querySelector(selector);
					if (img) {
						return img.getAttribute('data-zoom') || 
							   img.getAttribute('data-srcset') || 
							   img.src || '';
					}
				}
				return '';
			})()
		`, &imageSrc),
	)

	if err != nil {
		return false
	}

	// Process stock text
	if stockText == "OUT_OF_STOCK" {
		isInStock = false
	} else if strings.HasPrefix(stockText, "IN_STOCK_") {
		// Extract quantity from "IN_STOCK_7" format
		stockNumStr := strings.TrimPrefix(stockText, "IN_STOCK_")
		if stockNum, err := strconv.Atoi(stockNumStr); err == nil && stockNum > 0 {
			isInStock = true
		} else {
			isInStock = false
		}
	} else if stockText == "HAS_STOCK_TEXT" {
		isInStock = true
	} else if stockText == "CAN_ADD_TO_CART" {
		isInStock = true
	} else {
		isInStock = false
	}

	// Process image URL
	if imageSrc != "" {
		// Clean up data-srcset format if needed
		if strings.Contains(imageSrc, " ") {
			parts := strings.Split(imageSrc, " ")
			imageURL = parts[0]
		} else {
			imageURL = imageSrc
		}

		// Convert relative URLs to absolute
		if strings.HasPrefix(imageURL, "//") {
			imageURL = "https:" + imageURL
		} else if strings.HasPrefix(imageURL, "/") {
			imageURL = "https://torecacamp-pokemon.com" + imageURL
		}
	}

	// Process price - extract from various Japanese price formats
	if priceText != "" {
		// Try multiple regex patterns for different price formats
		patterns := []string{
			`販売価格¥([0-9,]+)`, // "販売価格¥3,280"
			`¥([0-9,]+)`,     // "¥3,280"
			`￥([0-9,]+)`,     // "￥3,280" (full-width yen)
			`([0-9,]+)円`,     // "3,280円"
		}

		var extractedPrice string
		for _, pattern := range patterns {
			re := regexp.MustCompile(pattern)
			matches := re.FindStringSubmatch(priceText)
			if len(matches) > 1 {
				extractedPrice = matches[1]
				break
			}
		}

		if extractedPrice != "" {
			cleanedPrice := strings.ReplaceAll(extractedPrice, ",", "")
			if price, err := strconv.ParseFloat(cleanedPrice, 64); err == nil {
				foundPrice = price
			}
		}
	}

	// Update tracker with results
	tracker.InStock = isInStock
	if foundPrice > 0 {
		tracker.Price = foundPrice
	}
	if imageURL != "" {
		tracker.ImageURL = imageURL
	}

	return true
}

// GetTrackerStatus handles GET /api/tracker/status
func (sth *SimplifiedTrackerHandlers) GetTrackerStatus(c *gin.Context) {
	var trackers []models.TrackerEntry

	if sth.storage != nil {
		var err error
		trackers, err = sth.storage.GetActiveTrackers()
		if err != nil {
			log.Printf("Error getting trackers for status: %v", err)
			trackers = []models.TrackerEntry{} // fallback to empty slice
		}
	} else {
		trackers = []models.TrackerEntry{} // fallback to empty slice if no storage
	}

	status := SimpleWorkerStatus{
		IsRunning:       sth.isWorkerRunning,
		LastScanTime:    sth.lastScanTime,
		NextScanTime:    sth.lastScanTime.Add(30 * time.Second),
		ScanInterval:    30,
		ItemsProcessed:  len(trackers),
		ErrorsEncounted: 0,
	}

	c.JSON(http.StatusOK, status)
}

// GetTrackerOptions handles GET /api/tracker/options
func (sth *SimplifiedTrackerHandlers) GetTrackerOptions(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"statuses":     []string{"active", "paused", "deleted"},
		"sort_options": []string{"name", "created_at", "last_checked", "price"},
		"supported_sites": []string{
			"torecacamp-pokemon.com",
			"pokemon-card.com",
			"Generic sites",
		},
	})
}

// TestNotification handles POST /api/tracker/test-notification
func (sth *SimplifiedTrackerHandlers) TestNotification(c *gin.Context) {
	// Check if storage is available
	if sth.storage == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Tracker storage not available",
			"message": "Persistent tracker storage is not properly configured",
		})
		return
	}

	// Get all active trackers
	trackers, err := sth.storage.GetActiveTrackers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get trackers",
			"message": err.Error(),
		})
		return
	}

	// Filter for in-stock trackers
	var inStockTrackers []models.TrackerEntry
	for _, tracker := range trackers {
		if tracker.InStock {
			inStockTrackers = append(inStockTrackers, tracker)
		}
	}

	// Check if Discord webhook is configured
	webhookURL := os.Getenv("DISCORD_WEBHOOK_URL")
	if webhookURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Discord webhook not configured",
			"message": "DISCORD_WEBHOOK_URL environment variable is not set",
		})
		return
	}

	// Create Discord notifier
	discordConfig := &tracker.DiscordConfig{
		WebhookURL: webhookURL,
		Username:   "Pokemon Card Tracker",
		Timeout:    15 * time.Second,
	}

	notifier := tracker.NewDiscordNotifier(discordConfig)

	// Send notifications for all in-stock cards
	sentCount := 0
	failedCount := 0

	if len(inStockTrackers) == 0 {
		// Send a test notification if no cards are in stock
		testTracker := models.TrackerEntry{
			ID:       "test-notification",
			Name:     "Test - No cards currently in stock",
			URL:      "https://torecacamp-pokemon.com/test",
			InStock:  true,
			Price:    0.0,
			ImageURL: "",
		}

		if err := notifier.SendStockAlert(testTracker, true); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to send test notification",
				"message": err.Error(),
			})
			return
		}
		sentCount = 1
	} else {
		// Send notifications for each in-stock card
		for _, tracker := range inStockTrackers {
			if err := notifier.SendStockAlert(tracker, true); err != nil {
				log.Printf("Failed to send notification for tracker %s: %v", tracker.ID, err)
				failedCount++
			} else {
				sentCount++
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":              "Discord test notifications completed",
		"in_stock_cards":       len(inStockTrackers),
		"notifications_sent":   sentCount,
		"notifications_failed": failedCount,
		"webhook":              "***configured***", // Don't expose the actual URL
	})
}

// Helper functions
func generateSimpleID() string {
	return time.Now().Format("20060102150405") + "000"
}

func generateNameFromURL(url string) string {
	// Simple name generation from URL
	if len(url) > 50 {
		return "Pokemon Card - " + url[len(url)-10:]
	}
	return "Pokemon Card - " + url[len(url)-min(20, len(url)):]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
