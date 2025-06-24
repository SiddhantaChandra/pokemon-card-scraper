package monitor

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// HTTPWebDriverPool implements WebDriverPool interface using HTTP clients
type HTTPWebDriverPool struct {
	client    *http.Client
	poolSize  int
	timeout   time.Duration
	userAgent string

	// Pool management
	workers chan *HTTPWebDriver
	mu      sync.RWMutex

	// Health check
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// HTTPWebDriver implements WebDriver interface using HTTP client
type HTTPWebDriver struct {
	client    *http.Client
	userAgent string
	lastUsed  time.Time
	mu        sync.Mutex
}

// WebDriverPoolConfig holds configuration for the WebDriver pool
type WebDriverPoolConfig struct {
	PoolSize   int           `json:"pool_size"`   // Number of workers in pool
	Timeout    time.Duration `json:"timeout"`     // Request timeout
	UserAgent  string        `json:"user_agent"`  // User agent string
	MaxRetries int           `json:"max_retries"` // Maximum retries for failed requests
}

// DefaultWebDriverPoolConfig returns sensible defaults
func DefaultWebDriverPoolConfig() *WebDriverPoolConfig {
	return &WebDriverPoolConfig{
		PoolSize:   3,
		Timeout:    30 * time.Second,
		UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		MaxRetries: 3,
	}
}

// NewHTTPWebDriverPool creates a new HTTP-based WebDriver pool
func NewHTTPWebDriverPool(config *WebDriverPoolConfig) (*HTTPWebDriverPool, error) {
	if config == nil {
		config = DefaultWebDriverPoolConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: false,
		},
	}

	pool := &HTTPWebDriverPool{
		client:    client,
		poolSize:  config.PoolSize,
		timeout:   config.Timeout,
		userAgent: config.UserAgent,
		workers:   make(chan *HTTPWebDriver, config.PoolSize),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Initialize workers
	if err := pool.initializeWorkers(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize workers: %v", err)
	}

	log.Printf("HTTPWebDriverPool initialized with %d workers", config.PoolSize)
	return pool, nil
}

// initializeWorkers creates the initial pool of HTTP workers
func (pool *HTTPWebDriverPool) initializeWorkers() error {
	for i := 0; i < pool.poolSize; i++ {
		worker := &HTTPWebDriver{
			client:    pool.client,
			userAgent: pool.userAgent,
			lastUsed:  time.Now(),
		}

		pool.workers <- worker
	}

	return nil
}

// GetDriver gets a worker from the pool
func (pool *HTTPWebDriverPool) GetDriver() (WebDriver, error) {
	select {
	case worker := <-pool.workers:
		worker.lastUsed = time.Now()
		return worker, nil

	case <-time.After(pool.timeout):
		return nil, fmt.Errorf("timeout waiting for available worker")

	case <-pool.ctx.Done():
		return nil, fmt.Errorf("pool is shutting down")
	}
}

// ReturnDriver returns a worker to the pool
func (pool *HTTPWebDriverPool) ReturnDriver(driver WebDriver) {
	httpDriver, ok := driver.(*HTTPWebDriver)
	if !ok {
		log.Printf("Warning: attempted to return non-HTTP driver to pool")
		return
	}

	// Return to pool
	select {
	case pool.workers <- httpDriver:
		// Successfully returned
	default:
		log.Printf("Warning: worker pool is full")
	}
}

// Close closes the pool and cleans up resources
func (pool *HTTPWebDriverPool) Close() error {
	log.Println("Shutting down HTTPWebDriverPool...")

	// Cancel context
	pool.cancel()

	// Wait for any running operations
	pool.wg.Wait()

	// Drain the channel
	close(pool.workers)
	for range pool.workers {
		// Drain remaining workers
	}

	log.Println("HTTPWebDriverPool shutdown complete")
	return nil
}

// HTTPWebDriver methods implementing WebDriver interface

// Navigate navigates to the specified URL (loads page content)
func (hwd *HTTPWebDriver) Navigate(url string) error {
	hwd.mu.Lock()
	defer hwd.mu.Unlock()

	hwd.lastUsed = time.Now()

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("User-Agent", hwd.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	// Make request
	resp, err := hwd.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return nil
}

// GetPageContent returns the current page's HTML content
func (hwd *HTTPWebDriver) GetPageContent() (string, error) {
	// For HTTP-based implementation, this would need the URL
	// This is a limitation of the WebDriver interface
	return "", fmt.Errorf("use NavigateAndGetContent for HTTP-based scraping")
}

// NavigateAndGetContent combines navigation and content retrieval for HTTP-based scraping
func (hwd *HTTPWebDriver) NavigateAndGetContent(url string) (string, error) {
	hwd.mu.Lock()
	defer hwd.mu.Unlock()

	hwd.lastUsed = time.Now()

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers to mimic a real browser
	req.Header.Set("User-Agent", hwd.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	// Make request
	resp, err := hwd.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %v", err)
	}

	// Return HTML content
	html, err := doc.Html()
	if err != nil {
		return "", fmt.Errorf("failed to get HTML: %v", err)
	}

	return html, nil
}

// Close closes the HTTP driver (no-op for HTTP client)
func (hwd *HTTPWebDriver) Close() error {
	// HTTP client doesn't need explicit closing
	return nil
}

// GetLastUsed returns when this driver was last used
func (hwd *HTTPWebDriver) GetLastUsed() time.Time {
	hwd.mu.Lock()
	defer hwd.mu.Unlock()
	return hwd.lastUsed
}

// IsHealthy checks if the HTTP client is healthy
func (hwd *HTTPWebDriver) IsHealthy() bool {
	// Simple health check - consider healthy if used recently
	return time.Since(hwd.GetLastUsed()) < 1*time.Hour
}

// StockDetector provides stock detection logic using existing parser components
type StockDetector struct {
	pool WebDriverPool
}

// NewStockDetector creates a new stock detector
func NewStockDetector(pool WebDriverPool) *StockDetector {
	return &StockDetector{
		pool: pool,
	}
}

// DetectStock detects stock information for a given URL
func (sd *StockDetector) DetectStock(url string) (*StockInfo, error) {
	// Get driver from pool
	driver, err := sd.pool.GetDriver()
	if err != nil {
		return nil, fmt.Errorf("failed to get driver: %v", err)
	}
	defer sd.pool.ReturnDriver(driver)

	// Navigate and get content
	var content string
	if httpDriver, ok := driver.(*HTTPWebDriver); ok {
		content, err = httpDriver.NavigateAndGetContent(url)
	} else {
		// Fallback for other driver types
		if err := driver.Navigate(url); err != nil {
			return nil, fmt.Errorf("failed to navigate: %v", err)
		}
		content, err = driver.GetPageContent()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get page content: %v", err)
	}

	// Parse stock information
	return sd.parseStockInfo(content, url)
}

// parseStockInfo parses stock information from HTML content
func (sd *StockDetector) parseStockInfo(content, url string) (*StockInfo, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %v", err)
	}

	stockInfo := &StockInfo{
		InStock: false,
		Price:   "",
		Image:   "",
	}

	// Basic stock detection logic - this should be enhanced with site-specific selectors
	// Look for common stock indicators
	stockText := doc.Find("body").Text()
	stockTextLower := strings.ToLower(stockText)

	// Check for in-stock indicators
	inStockIndicators := []string{
		"in stock",
		"available",
		"add to cart",
		"buy now",
		"購入",
		"在庫あり",
	}

	for _, indicator := range inStockIndicators {
		if strings.Contains(stockTextLower, indicator) {
			stockInfo.InStock = true
			break
		}
	}

	// Check for out-of-stock indicators
	outOfStockIndicators := []string{
		"out of stock",
		"sold out",
		"unavailable",
		"品切れ",
		"在庫切れ",
		"完売",
	}

	for _, indicator := range outOfStockIndicators {
		if strings.Contains(stockTextLower, indicator) {
			stockInfo.InStock = false
			break
		}
	}

	// Try to extract price (basic implementation)
	priceElement := doc.Find(".price, .amount, .cost, [class*='price'], [class*='amount']").First()
	if priceElement.Length() > 0 {
		stockInfo.Price = strings.TrimSpace(priceElement.Text())
	}

	// Try to extract image
	imageElement := doc.Find("img[src*='product'], img[src*='item'], .product-image img, .item-image img").First()
	if imageElement.Length() > 0 {
		if src, exists := imageElement.Attr("src"); exists {
			stockInfo.Image = src
		}
	}

	return stockInfo, nil
}
