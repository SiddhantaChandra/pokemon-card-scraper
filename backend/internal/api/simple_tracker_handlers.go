package api

import (
	"context"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/SiddhantaChandra/pokemon-card-scraper/internal/storage"
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
	log.Println("Starting simplified tracker background worker (30 second intervals)")
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
				log.Printf("Error getting active trackers: %v", err)
				continue
			}
			if len(trackers) > 0 {
				log.Printf("Background worker: checking %d trackers", len(trackers))
				sth.performBackgroundCheck()
			}
		case <-sth.workerStopChan:
			log.Println("Background worker stopped")
			sth.isWorkerRunning = false
			return
		}
	}
}

// performBackgroundCheck checks all trackers in background
func (sth *SimplifiedTrackerHandlers) performBackgroundCheck() {
	if sth.storage == nil {
		log.Println("Storage not available for background check")
		return
	}

	sth.lastScanTime = time.Now()
	successCount := 0

	trackers, err := sth.storage.GetActiveTrackers()
	if err != nil {
		log.Printf("Error getting active trackers: %v", err)
		return
	}

	for _, tracker := range trackers {
		simple := convertToSimpleResponse(tracker)
		if sth.checkTrackerURL(&simple) {
			successCount++
			// Update the tracker in storage with new data
			updatedTracker := convertFromSimpleResponse(simple)
			if err := sth.storage.UpdateTracker(updatedTracker); err != nil {
				log.Printf("Error updating tracker %s: %v", tracker.ID, err)
			}
		}
	}

	log.Printf("Background check completed: %d/%d trackers checked successfully", successCount, len(trackers))
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
	log.Printf("DEBUG: checkTrackerURL called with URL: %s", tracker.URL)

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic while checking URL %s: %v", tracker.URL, r)
		}
	}()

	log.Printf("Starting ChromeDP check for URL: %s", tracker.URL)

	// Create context with Docker-friendly Chrome options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-features", "VizDisplayCompositor"),
		chromedp.UserAgent("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
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
	log.Printf("Starting navigation to: %s", tracker.URL)

	err := chromedp.Run(ctx,
		// Navigate to the URL
		chromedp.Navigate(tracker.URL),

		// Wait for page to be ready
		chromedp.WaitReady("body"),

		// Extract price text - look for "販売価格¥" pattern
		chromedp.Evaluate(`
			(() => {
				// Look for price pattern in page text
				const priceMatch = document.body.innerText.match(/販売価格¥([0-9,]+)/);
				return priceMatch ? priceMatch[0] : '';
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
		log.Printf("ChromeDP error for %s: %v", tracker.URL, err)
		return false
	}

	// Process stock text
	if stockText == "OUT_OF_STOCK" {
		isInStock = false
		log.Printf("Stock: OUT OF STOCK")
	} else if strings.HasPrefix(stockText, "IN_STOCK_") {
		// Extract quantity from "IN_STOCK_7" format
		stockNumStr := strings.TrimPrefix(stockText, "IN_STOCK_")
		if stockNum, err := strconv.Atoi(stockNumStr); err == nil && stockNum > 0 {
			isInStock = true
			log.Printf("Stock: IN STOCK (%d items available)", stockNum)
		} else {
			isInStock = false
			log.Printf("Stock: OUT OF STOCK (0 items)")
		}
	} else if stockText == "HAS_STOCK_TEXT" {
		isInStock = true
		log.Printf("Stock: IN STOCK (found stock text)")
	} else if stockText == "CAN_ADD_TO_CART" {
		isInStock = true
		log.Printf("Stock: IN STOCK (add to cart available)")
	} else {
		isInStock = false
		log.Printf("Stock: UNKNOWN/OUT OF STOCK (no stock indicators found)")
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

	// Process price - extract from "販売価格¥3,280" format
	if priceText != "" {
		// Use regex to extract price from Japanese format
		re := regexp.MustCompile(`販売価格¥([0-9,]+)`)
		matches := re.FindStringSubmatch(priceText)

		if len(matches) > 1 {
			cleanedPrice := strings.ReplaceAll(matches[1], ",", "")
			if price, err := strconv.ParseFloat(cleanedPrice, 64); err == nil {
				foundPrice = price
				log.Printf("Parsed price: ¥%.2f", price)
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

	log.Printf("Final result for %s: InStock=%t, Price=%.2f, Image=%s", tracker.ID, tracker.InStock, tracker.Price, imageURL)
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
