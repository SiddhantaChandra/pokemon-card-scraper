package scraper

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/SiddhantaChandra/pokemon-card-scraper/pkg/models"
	"github.com/gocolly/colly/v2"
)

// ScraperInterface defines common methods for all scraper types
type ScraperInterface interface {
	IsRunning() bool
	IsPaused() bool
	GetStatus() ScrapingStatus
	Stop()
	Pause() bool
	Resume() bool
	ScrapeAllPages(params SearchParams, progressCallback func(ScrapeProgress)) error
	ScrapePage(pageURL string) ([]models.Card, error)
}

// Scraper represents the main scraping engine
type Scraper struct {
	config      *CollectorConfig
	parser      *ProductParser
	baseURL     string
	searchURL   string
	onCardFound func(models.Card)
	onProgress  func(ScrapeProgress)

	// Status management
	mu        sync.RWMutex
	running   bool
	paused    bool
	status    ScrapingStatus
	stopChan  chan struct{}
	pauseChan chan struct{}
}

// ScrapingStatus represents the current status of scraping operations
type ScrapingStatus struct {
	StartTime              time.Time      `json:"start_time"`
	LastUpdated            time.Time      `json:"last_updated"`
	CurrentPage            int            `json:"current_page"`
	TotalPages             int            `json:"total_pages"`
	ItemsScraped           int            `json:"items_scraped"`
	CardsPerMinute         float64        `json:"cards_per_minute"`
	EstimatedTimeRemaining *time.Duration `json:"estimated_time_remaining"`
	IsPaused               bool           `json:"is_paused"`
	PausedAt               *time.Time     `json:"paused_at,omitempty"`
}

// ScrapeProgress represents progress information during scraping
type ScrapeProgress struct {
	StartTime      time.Time `json:"start_time"`
	CurrentPage    int       `json:"current_page"`
	TotalPages     int       `json:"total_pages"`
	ItemsProcessed int       `json:"items_processed"`
}

// ScraperConfig holds configuration for the scraper
type ScraperConfig struct {
	BaseURL         string
	SearchURL       string
	CollectorConfig *CollectorConfig
}

// DefaultScraperConfig returns sensible defaults
func DefaultScraperConfig() *ScraperConfig {
	return &ScraperConfig{
		BaseURL:         "https://torecacamp-pokemon.com",
		SearchURL:       "https://torecacamp-pokemon.com/search?type=product&options%5Bprefix%5D=last&options%5Bunavailable_products%5D=last&q=.",
		CollectorConfig: DefaultCollectorConfig(),
	}
}

// NewScraper creates a new scraper instance
func NewScraper(config *ScraperConfig) *Scraper {
	if config == nil {
		config = DefaultScraperConfig()
	}

	return &Scraper{
		config:    config.CollectorConfig,
		parser:    NewProductParser(config.BaseURL),
		baseURL:   config.BaseURL,
		searchURL: config.SearchURL,
		stopChan:  make(chan struct{}),
		pauseChan: make(chan struct{}),
		status: ScrapingStatus{
			StartTime:   time.Now(),
			LastUpdated: time.Now(),
		},
	}
}

// SetCardFoundCallback sets the callback function for when cards are found
func (s *Scraper) SetCardFoundCallback(callback func(models.Card)) {
	s.onCardFound = callback
}

// SetProgressCallback sets the callback function for progress updates
func (s *Scraper) SetProgressCallback(callback func(ScrapeProgress)) {
	s.onProgress = callback
}

// ScrapeAll scrapes all products from the website using URL-based pagination
func (s *Scraper) ScrapeAll(progressCallback func(ScrapeProgress)) error {
	log.Println("Starting full website scrape with URL-based pagination...")

	// Use URL-based scraping
	params := SearchParams{
		InStockOnly: true,
		Page:        1,
	}

	return s.ScrapeAllPages(params, progressCallback)
}

// ScrapeAllPages scrapes all pages using the URL-based approach
func (s *Scraper) ScrapeAllPages(params SearchParams, progressCallback func(ScrapeProgress)) error {
	// Set running state
	s.setRunning(true)
	defer s.setRunning(false)

	baseURL := GetBaseURL()

	// First, discover total pages by scraping page 1
	params.Page = 1
	firstPageURL := BuildSearchURL(params)

	log.Printf("Discovering total pages from: %s", firstPageURL)

	var totalPages int
	var totalItems int
	var firstPageCards []models.Card

	// Create search collector
	searchCollector := CreateSearchCollector(s.config)

	searchCollector.OnHTML("html", func(e *colly.HTMLElement) {
		pageInfo, err := ParseProductPage(e.DOM, baseURL)
		if err != nil {
			log.Printf("Error parsing first page: %v", err)
			return
		}

		totalPages = pageInfo.TotalPages
		totalItems = pageInfo.TotalItems
		firstPageCards = pageInfo.Cards

		log.Printf("Discovered: %d total pages, %d total items", totalPages, totalItems)
	})

	// Visit first page to get pagination info
	if err := searchCollector.Visit(firstPageURL); err != nil {
		return fmt.Errorf("failed to visit first page: %w", err)
	}

	searchCollector.Wait()

	if totalPages == 0 {
		return fmt.Errorf("could not determine total pages")
	}

	// Apply maxPages limit if specified
	if params.MaxPages > 0 && params.MaxPages < totalPages {
		totalPages = params.MaxPages
		log.Printf("Limiting scraping to %d pages (user requested)", totalPages)
	}

	// Update status with total pages discovered
	s.updateStatus(0, totalPages, 0)

	// Initialize progress
	progress := ScrapeProgress{
		StartTime:   time.Now(),
		TotalPages:  totalPages,
		CurrentPage: 1,
	}

	// Save first page cards
	totalScraped := 0
	for _, card := range firstPageCards {
		// Check for stop signal while processing first page cards
		select {
		case <-s.stopChan:
			log.Printf("Stop signal received while processing first page")
			return nil
		default:
		}

		// Call the card found callback if set
		if s.onCardFound != nil {
			s.onCardFound(card)
		}
		totalScraped++

		// Update status after each card for real-time updates
		s.updateStatus(1, totalPages, totalScraped)
	}

	progress.ItemsProcessed = totalScraped
	if progressCallback != nil {
		progressCallback(progress)
	}

	log.Printf("Page 1: Found %d cards, Total: %d", len(firstPageCards), totalScraped)

	// Now scrape remaining pages
	for page := 2; page <= totalPages; page++ {
		// Check for stop signal before processing each page
		select {
		case <-s.stopChan:
			log.Printf("Stop signal received, stopping scraper at page %d", page)
			return nil
		default:
		}

		// Check for pause signal
		if s.IsPaused() {
			log.Printf("Scraper paused at page %d", page)
			for s.IsPaused() {
				time.Sleep(500 * time.Millisecond)

				// Check for stop signal while paused
				select {
				case <-s.stopChan:
					log.Printf("Stop signal received while paused, stopping scraper at page %d", page)
					return nil
				default:
				}
			}
			log.Printf("Scraper resumed at page %d", page)
		}

		params.Page = page
		pageURL := BuildSearchURL(params)

		log.Printf("Scraping page %d/%d: %s", page, totalPages, pageURL)

		var pageCards []models.Card

		// Create new collector for each page to avoid conflicts
		pageCollector := CreateSearchCollector(s.config)

		pageCollector.OnHTML("html", func(e *colly.HTMLElement) {
			pageInfo, err := ParseProductPage(e.DOM, baseURL)
			if err != nil {
				log.Printf("Error parsing page %d: %v", page, err)
				return
			}

			pageCards = pageInfo.Cards

			// Process cards and update status in real-time
			for _, card := range pageCards {
				if s.onCardFound != nil {
					s.onCardFound(card)
				}
				totalScraped++

				// Update status after each card for real-time updates
				s.updateStatus(page, totalPages, totalScraped)
			}

			// Update progress after page completion
			progress.CurrentPage = page
			progress.ItemsProcessed = totalScraped
			if progressCallback != nil {
				progressCallback(progress)
			}

			log.Printf("Page %d: Found %d cards, Total: %d",
				page, len(pageCards), totalScraped)
		})

		// Visit the page
		if err := pageCollector.Visit(pageURL); err != nil {
			log.Printf("Failed to visit page %d: %v", page, err)
			continue // Continue with next page
		}

		pageCollector.Wait()

		// If no cards found, we might have reached the end early
		if len(pageCards) == 0 {
			log.Printf("No cards found on page %d, stopping early", page)
			break
		}
	}

	log.Printf("Scraping completed. Total cards: %d (Expected: %d)", totalScraped, totalItems)
	return nil
}

// GetPaginationInfo gets pagination info without scraping all pages
func (s *Scraper) GetPaginationInfo(params SearchParams) (totalPages, totalItems int, err error) {
	params.Page = 1
	pageURL := BuildSearchURL(params)

	searchCollector := CreateSearchCollector(s.config)

	searchCollector.OnHTML("html", func(e *colly.HTMLElement) {
		pageInfo, parseErr := ParseProductPage(e.DOM, GetBaseURL())
		if parseErr != nil {
			err = parseErr
			return
		}

		totalPages = pageInfo.TotalPages
		totalItems = pageInfo.TotalItems
	})

	if visitErr := searchCollector.Visit(pageURL); visitErr != nil {
		return 0, 0, visitErr
	}

	searchCollector.Wait()
	return totalPages, totalItems, err
}

// ScrapePage scrapes a single page by URL
func (s *Scraper) ScrapePage(pageURL string) ([]models.Card, error) {
	var cards []models.Card
	var err error

	searchCollector := CreateSearchCollector(s.config)

	searchCollector.OnHTML("html", func(e *colly.HTMLElement) {
		pageInfo, parseErr := ParseProductPage(e.DOM, GetBaseURL())
		if parseErr != nil {
			err = parseErr
			return
		}
		cards = pageInfo.Cards
	})

	if err := searchCollector.Visit(pageURL); err != nil {
		return nil, err
	}

	searchCollector.Wait()
	return cards, err
}

// ScrapeProduct scrapes a single product by URL
func (s *Scraper) ScrapeProduct(productURL string) (*models.Card, error) {
	log.Printf("Scraping single product: %s", productURL)

	productCollector := CreateProductCollector(s.config)

	var scrapedCard *models.Card

	// Setup product detail parsing
	s.parser.ParseProductDetails(productCollector, func(card models.Card) {
		scrapedCard = &card
		log.Printf("Scraped product details: %s", card.Name)
	})

	err := productCollector.Visit(productURL)
	if err != nil {
		return nil, fmt.Errorf("failed to scrape product %s: %v", productURL, err)
	}

	productCollector.Wait()

	if scrapedCard == nil {
		return nil, fmt.Errorf("no product data found for URL: %s", productURL)
	}

	return scrapedCard, nil
}

// SearchProducts searches for products with a specific query
func (s *Scraper) SearchProducts(query string, maxResults int) ([]models.Card, error) {
	log.Printf("Searching for products with query: %s", query)

	searchCollector := CreateSearchCollector(s.config)

	var cards []models.Card
	cardCount := 0

	// Setup search result parsing with limit
	s.parser.ParseSearchResults(searchCollector, func(card models.Card) {
		if maxResults > 0 && cardCount >= maxResults {
			return
		}

		cards = append(cards, card)
		cardCount++

		log.Printf("Found card: %s", card.Name)
	})

	// Build search URL with query
	searchURL := fmt.Sprintf("%s&q=%s", s.searchURL, query)

	err := searchCollector.Visit(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to search for products: %v", err)
	}

	searchCollector.Wait()

	log.Printf("Search completed. Found %d cards", len(cards))
	return cards, nil
}

// TestConnection tests if the website is accessible
func (s *Scraper) TestConnection() error {
	log.Println("Testing connection to website...")

	testCollector := NewCollector(s.config)

	var connectionOK bool

	testCollector.OnResponse(func(r *colly.Response) {
		if r.StatusCode == 200 {
			connectionOK = true
			log.Printf("Connection test successful: %d", r.StatusCode)
		} else {
			log.Printf("Connection test returned status: %d", r.StatusCode)
		}
	})

	testCollector.OnError(func(r *colly.Response, err error) {
		log.Printf("Connection test failed: %v", err)
	})

	err := testCollector.Visit(s.baseURL)
	if err != nil {
		return fmt.Errorf("connection test failed: %v", err)
	}

	testCollector.Wait()

	if !connectionOK {
		return fmt.Errorf("website returned non-200 status code")
	}

	log.Println("Connection test passed")
	return nil
}

// GetEstimatedTotalProducts tries to estimate the total number of products
func (s *Scraper) GetEstimatedTotalProducts() (int, error) {
	log.Println("Estimating total number of products...")

	searchCollector := CreateSearchCollector(s.config)

	totalProducts := 0

	// Try to find pagination info to estimate total
	searchCollector.OnHTML("body", func(e *colly.HTMLElement) {
		// This would need to be implemented based on the actual website structure
		// For now, return a conservative estimate
		totalProducts = 1000 // Placeholder
	})

	err := searchCollector.Visit(s.searchURL)
	if err != nil {
		return 0, fmt.Errorf("failed to estimate product count: %v", err)
	}

	searchCollector.Wait()

	log.Printf("Estimated total products: %d", totalProducts)
	return totalProducts, nil
}

// ValidateConfiguration checks if the scraper configuration is valid
func (s *Scraper) ValidateConfiguration() error {
	if s.baseURL == "" {
		return fmt.Errorf("base URL is required")
	}

	if s.searchURL == "" {
		return fmt.Errorf("search URL is required")
	}

	if s.config == nil {
		return fmt.Errorf("collector configuration is required")
	}

	if s.parser == nil {
		return fmt.Errorf("parser is required")
	}

	// Test that we can create a collector
	testCollector := NewCollector(s.config)
	if err := ValidateCollector(testCollector); err != nil {
		return fmt.Errorf("collector validation failed: %v", err)
	}

	return nil
}

// GetSupportedDomains returns the list of domains this scraper supports
func (s *Scraper) GetSupportedDomains() []string {
	return []string{"torecacamp-pokemon.com"}
}

// IsRunning returns whether the scraper is currently running
func (s *Scraper) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetStatus returns the current scraping status
func (s *Scraper) GetStatus() ScrapingStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

// Stop stops the current scraping operation
func (s *Scraper) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		close(s.stopChan)
		s.running = false
		s.paused = false
		s.status.IsPaused = false
		s.status.PausedAt = nil
		s.status.LastUpdated = time.Now()
	}
}

// Pause pauses the current scraping operation
func (s *Scraper) Pause() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running && !s.paused {
		s.paused = true
		now := time.Now()
		s.status.IsPaused = true
		s.status.PausedAt = &now
		s.status.LastUpdated = now
		// Send pause signal
		select {
		case s.pauseChan <- struct{}{}:
		default:
		}
		return true
	}
	return false
}

// Resume resumes the paused scraping operation
func (s *Scraper) Resume() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running && s.paused {
		s.paused = false
		s.status.IsPaused = false
		s.status.PausedAt = nil
		s.status.LastUpdated = time.Now()
		return true
	}
	return false
}

// IsPaused returns whether the scraper is currently paused
func (s *Scraper) IsPaused() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.paused
}

// updateStatus updates the scraping status (internal method)
func (s *Scraper) updateStatus(currentPage, totalPages, itemsScraped int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.status.CurrentPage = currentPage
	s.status.TotalPages = totalPages
	s.status.ItemsScraped = itemsScraped
	s.status.LastUpdated = time.Now()

	// Calculate cards per minute
	if s.status.ItemsScraped > 0 {
		elapsed := time.Since(s.status.StartTime)
		minutes := elapsed.Minutes()
		if minutes > 0 {
			s.status.CardsPerMinute = float64(s.status.ItemsScraped) / minutes
		}
	}

	// Calculate estimated time remaining
	if s.status.CurrentPage > 0 && s.status.TotalPages > 0 {
		elapsed := time.Since(s.status.StartTime)
		avgTimePerPage := elapsed / time.Duration(s.status.CurrentPage)
		remainingPages := s.status.TotalPages - s.status.CurrentPage

		if remainingPages > 0 {
			estimatedRemaining := avgTimePerPage * time.Duration(remainingPages)
			s.status.EstimatedTimeRemaining = &estimatedRemaining
		} else {
			zero := time.Duration(0)
			s.status.EstimatedTimeRemaining = &zero
		}
	}
}

// setRunning sets the running status (internal method)
func (s *Scraper) setRunning(running bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.running = running
	if running {
		s.status.StartTime = time.Now()
		s.stopChan = make(chan struct{})
		s.pauseChan = make(chan struct{}, 1) // Buffered to prevent blocking
		s.paused = false
		s.status.IsPaused = false
		s.status.PausedAt = nil
	} else {
		s.paused = false
		s.status.IsPaused = false
		s.status.PausedAt = nil
	}
	s.status.LastUpdated = time.Now()
}
