package tracker

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// TrackerScraper interface defines methods for checking URLs
type TrackerScraper interface {
	CheckURL(url string) (inStock bool, price float64, imageURL string, err error)
	CheckURLWithContext(ctx context.Context, url string) (inStock bool, price float64, imageURL string, err error)
	Close() error
}

// ChromeDPScraper implements TrackerScraper using ChromeDP
type ChromeDPScraper struct {
	allocator context.Context
	cancel    context.CancelFunc
	timeout   time.Duration
}

// ScraperConfig holds configuration for the scraper
type ScraperConfig struct {
	Timeout   time.Duration `json:"timeout"`
	UserAgent string        `json:"user_agent"`
	Headless  bool          `json:"headless"`
}

// DefaultScraperConfig returns default scraper configuration
func DefaultScraperConfig() *ScraperConfig {
	return &ScraperConfig{
		Timeout:   30 * time.Second,
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		Headless:  true,
	}
}

// NewChromeDPScraper creates a new ChromeDP-based scraper
func NewChromeDPScraper(config *ScraperConfig) (*ChromeDPScraper, error) {
	if config == nil {
		config = DefaultScraperConfig()
	}

	// Create Chrome options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", config.Headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.UserAgent(config.UserAgent),
	)

	// Create allocator context
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)

	scraper := &ChromeDPScraper{
		allocator: allocCtx,
		cancel:    cancel,
		timeout:   config.Timeout,
	}

	return scraper, nil
}

// CheckURL checks a single URL for stock status, price, and image
func (s *ChromeDPScraper) CheckURL(url string) (bool, float64, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	return s.CheckURLWithContext(ctx, url)
}

// CheckURLWithContext checks a URL with a custom context
func (s *ChromeDPScraper) CheckURLWithContext(ctx context.Context, url string) (bool, float64, string, error) {
	// Create browser context
	taskCtx, cancel := chromedp.NewContext(s.allocator)
	defer cancel()

	// Combine with timeout context
	taskCtx, cancel = context.WithTimeout(taskCtx, s.timeout)
	defer cancel()

	var inStock bool
	var price float64
	var imageURL string

	// Determine the website type and use appropriate scraping logic
	if s.isJapanesePokemonSite(url) {
		var err error
		inStock, price, imageURL, err = s.scrapeJapanesePokemonSite(taskCtx, url)
		if err != nil {
			return false, 0, "", fmt.Errorf("failed to scrape Japanese Pokemon site: %v", err)
		}
	} else {
		// Generic scraping logic for other sites
		var err error
		inStock, price, imageURL, err = s.scrapeGenericSite(taskCtx, url)
		if err != nil {
			return false, 0, "", fmt.Errorf("failed to scrape generic site: %v", err)
		}
	}

	return inStock, price, imageURL, nil
}

// scrapeJapanesePokemonSite scrapes Japanese Pokemon card sites (like the one in tracker.py)
func (s *ChromeDPScraper) scrapeJapanesePokemonSite(ctx context.Context, url string) (bool, float64, string, error) {
	var inStock bool
	var price float64
	var imageURL string

	// Navigate to the page and extract information
	err := chromedp.Run(ctx,
		// Navigate to URL
		chromedp.Navigate(url),

		// Wait for page to load
		chromedp.WaitVisible("body", chromedp.ByQuery),

		// Check for add to cart buttons (stock indicator)
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Try to find add button elements
			var addButtons []string
			err := chromedp.Evaluate(`
				Array.from(document.querySelectorAll('.product-form__add-button')).map(btn => btn.className)
			`, &addButtons).Do(ctx)

			if err != nil {
				log.Printf("Warning: Could not find add buttons: %v", err)
				// Try alternative selectors
				var hasAddButton bool
				chromedp.Evaluate(`
					document.querySelector('button[type="submit"]') !== null || 
					document.querySelector('.add-to-cart') !== null ||
					document.querySelector('[data-action="add-to-cart"]') !== null
				`, &hasAddButton).Do(ctx)
				inStock = hasAddButton
			} else {
				// Check if any button is not disabled
				for _, className := range addButtons {
					if !strings.Contains(className, "button--disabled") {
						inStock = true
						break
					}
				}
			}
			return nil
		}),

		// Extract image URL
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Try multiple selectors for product images
			selectors := []string{
				".product-gallery__image",
				".product-image img",
				".product-photo img",
				"img[data-zoom]",
				".main-image img",
			}

			for _, selector := range selectors {
				var imgSrc string
				err := chromedp.AttributeValue(selector, "data-zoom", &imgSrc, nil).Do(ctx)
				if err == nil && imgSrc != "" {
					imageURL = imgSrc
					return nil
				}

				err = chromedp.AttributeValue(selector, "src", &imgSrc, nil).Do(ctx)
				if err == nil && imgSrc != "" {
					imageURL = imgSrc
					return nil
				}

				err = chromedp.AttributeValue(selector, "data-srcset", &imgSrc, nil).Do(ctx)
				if err == nil && imgSrc != "" {
					// Extract first URL from srcset
					parts := strings.Split(imgSrc, " ")
					if len(parts) > 0 {
						imageURL = parts[0]
						return nil
					}
				}
			}
			return nil
		}),

		// Extract price
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Try multiple selectors for price
			selectors := []string{
				".product-form__info-content .price",
				".price",
				".product-price",
				"[data-price]",
				".price-current",
			}

			for _, selector := range selectors {
				var priceText string
				err := chromedp.Text(selector, &priceText, chromedp.ByQuery).Do(ctx)
				if err == nil && priceText != "" {
					// Extract numeric price from text
					extractedPrice := s.extractPrice(priceText)
					if extractedPrice > 0 {
						price = extractedPrice
						return nil
					}
				}
			}
			return nil
		}),
	)

	if err != nil {
		return false, 0, "", fmt.Errorf("chromedp execution failed: %v", err)
	}

	return inStock, price, imageURL, nil
}

// scrapeGenericSite provides generic scraping logic for other sites
func (s *ChromeDPScraper) scrapeGenericSite(ctx context.Context, url string) (bool, float64, string, error) {
	var inStock bool
	var price float64
	var imageURL string

	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible("body", chromedp.ByQuery),

		// Generic stock check
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Look for common add to cart indicators
			var hasStock bool
			chromedp.Evaluate(`
				// Check for common stock indicators
				var stockIndicators = [
					'button[type="submit"]',
					'.add-to-cart',
					'[data-action="add-to-cart"]',
					'.buy-now',
					'.purchase',
					'.in-stock'
				];
				
				for (var i = 0; i < stockIndicators.length; i++) {
					if (document.querySelector(stockIndicators[i])) {
						break;
					}
				}
				
				// Check for out of stock indicators
				var outOfStockText = document.body.innerText.toLowerCase();
				var hasOutOfStock = outOfStockText.includes('out of stock') || 
								   outOfStockText.includes('sold out') || 
								   outOfStockText.includes('unavailable');
				
				i < stockIndicators.length && !hasOutOfStock;
			`, &hasStock).Do(ctx)
			inStock = hasStock
			return nil
		}),

		// Generic image extraction
		chromedp.ActionFunc(func(ctx context.Context) error {
			var imgSrc string
			// Try common image selectors
			selectors := []string{
				".product-image img",
				".main-image img",
				"img[alt*='product']",
				"img[alt*='item']",
				".gallery img",
			}

			for _, selector := range selectors {
				err := chromedp.AttributeValue(selector, "src", &imgSrc, nil).Do(ctx)
				if err == nil && imgSrc != "" {
					imageURL = imgSrc
					break
				}
			}
			return nil
		}),

		// Generic price extraction
		chromedp.ActionFunc(func(ctx context.Context) error {
			var priceText string
			selectors := []string{
				".price",
				".cost",
				".amount",
				"[data-price]",
				".product-price",
			}

			for _, selector := range selectors {
				err := chromedp.Text(selector, &priceText, chromedp.ByQuery).Do(ctx)
				if err == nil && priceText != "" {
					extractedPrice := s.extractPrice(priceText)
					if extractedPrice > 0 {
						price = extractedPrice
						break
					}
				}
			}
			return nil
		}),
	)

	if err != nil {
		return false, 0, "", fmt.Errorf("generic scraping failed: %v", err)
	}

	return inStock, price, imageURL, nil
}

// extractPrice extracts numeric price from text (handles Japanese Yen and other currencies)
func (s *ChromeDPScraper) extractPrice(priceText string) float64 {
	// Clean the text
	cleanText := strings.TrimSpace(priceText)
	cleanText = strings.ReplaceAll(cleanText, "¥", "")
	cleanText = strings.ReplaceAll(cleanText, "$", "")
	cleanText = strings.ReplaceAll(cleanText, "円", "")
	cleanText = strings.ReplaceAll(cleanText, "販売価格", "")
	cleanText = strings.ReplaceAll(cleanText, ",", "")
	cleanText = strings.TrimSpace(cleanText)

	// Use regex to extract numbers
	re := regexp.MustCompile(`[\d.]+`)
	matches := re.FindAllString(cleanText, -1)

	if len(matches) > 0 {
		// Take the first numeric match
		priceStr := matches[0]
		if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
			return price
		}
	}

	return 0
}

// isJapanesePokemonSite checks if the URL is a Japanese Pokemon card site
func (s *ChromeDPScraper) isJapanesePokemonSite(url string) bool {
	// Add known Japanese Pokemon card sites
	japaneseSites := []string{
		"pokemoncard.co.jp",
		"tokyostore.jp",
		"pokemon-center.com",
		"cardshop-serra.com",
		// Add more as needed
	}

	urlLower := strings.ToLower(url)
	for _, site := range japaneseSites {
		if strings.Contains(urlLower, site) {
			return true
		}
	}

	// Also check for Japanese characters in URL
	return strings.Contains(url, "jp") ||
		regexp.MustCompile(`[\p{Hiragana}\p{Katakana}\p{Han}]`).MatchString(url)
}

// Close closes the scraper and cleans up resources
func (s *ChromeDPScraper) Close() error {
	if s.cancel != nil {
		s.cancel()
	}
	return nil
}

// ScraperResult represents the result of a single scraping operation
type ScraperResult struct {
	URL      string        `json:"url"`
	InStock  bool          `json:"in_stock"`
	Price    float64       `json:"price"`
	ImageURL string        `json:"image_url"`
	Error    string        `json:"error,omitempty"`
	Duration time.Duration `json:"duration"`
}

// BatchScraper provides functionality to scrape multiple URLs concurrently
type BatchScraper struct {
	scraper    TrackerScraper
	maxWorkers int
	timeout    time.Duration
}

// NewBatchScraper creates a new batch scraper
func NewBatchScraper(scraper TrackerScraper, maxWorkers int, timeout time.Duration) *BatchScraper {
	if maxWorkers <= 0 {
		maxWorkers = 5 // Default to 5 workers
	}
	if timeout <= 0 {
		timeout = 30 * time.Second // Default timeout
	}

	return &BatchScraper{
		scraper:    scraper,
		maxWorkers: maxWorkers,
		timeout:    timeout,
	}
}

// ScrapeURLs scrapes multiple URLs concurrently
func (bs *BatchScraper) ScrapeURLs(urls []string) []ScraperResult {
	if len(urls) == 0 {
		return []ScraperResult{}
	}

	// Create channels for work distribution
	urlChan := make(chan string, len(urls))
	resultChan := make(chan ScraperResult, len(urls))

	// Start workers
	for i := 0; i < bs.maxWorkers; i++ {
		go bs.worker(urlChan, resultChan)
	}

	// Send URLs to workers
	for _, url := range urls {
		urlChan <- url
	}
	close(urlChan)

	// Collect results
	results := make([]ScraperResult, 0, len(urls))
	for i := 0; i < len(urls); i++ {
		result := <-resultChan
		results = append(results, result)
	}

	return results
}

// worker processes URLs from the channel
func (bs *BatchScraper) worker(urlChan <-chan string, resultChan chan<- ScraperResult) {
	for url := range urlChan {
		start := time.Now()

		ctx, cancel := context.WithTimeout(context.Background(), bs.timeout)
		inStock, price, imageURL, err := bs.scraper.CheckURLWithContext(ctx, url)
		cancel()

		result := ScraperResult{
			URL:      url,
			InStock:  inStock,
			Price:    price,
			ImageURL: imageURL,
			Duration: time.Since(start),
		}

		if err != nil {
			result.Error = err.Error()
		}

		resultChan <- result
	}
}
