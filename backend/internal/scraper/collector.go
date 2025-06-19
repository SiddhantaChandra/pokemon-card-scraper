package scraper

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
)

// CollectorConfig holds configuration for the scraper
type CollectorConfig struct {
	UserAgent          string
	DelayMin           time.Duration
	DelayMax           time.Duration
	ConcurrentRequests int
	RequestTimeout     time.Duration
	RetryAttempts      int
	EnableDebug        bool
}

// DefaultCollectorConfig returns sensible default configuration
func DefaultCollectorConfig() *CollectorConfig {
	return &CollectorConfig{
		UserAgent:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		DelayMin:           500 * time.Millisecond, // Reduced from 1s to 500ms
		DelayMax:           1 * time.Second,        // Reduced from 2.5s to 1s
		ConcurrentRequests: 10,                     // Increased from 2 to 10
		RequestTimeout:     30 * time.Second,
		RetryAttempts:      3,
		EnableDebug:        false,
	}
}

// NewCollector creates and configures a new Colly collector
func NewCollector(config *CollectorConfig) *colly.Collector {
	if config == nil {
		config = DefaultCollectorConfig()
	}

	var c *colly.Collector
	if config.EnableDebug {
		c = colly.NewCollector(
			colly.Debugger(&debug.LogDebugger{}),
			colly.UserAgent(config.UserAgent),
		)
	} else {
		c = colly.NewCollector(
			colly.UserAgent(config.UserAgent),
		)
	}

	// Set allowed domains
	c.AllowedDomains = []string{"torecacamp-pokemon.com"}

	// Configure delays
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*torecacamp-pokemon.com*",
		Parallelism: config.ConcurrentRequests,
		Delay:       randomDelay(config.DelayMin, config.DelayMax),
	})

	// Set timeout
	c.SetRequestTimeout(config.RequestTimeout)

	// Setup retry logic
	setupRetryLogic(c, config.RetryAttempts)

	// Setup error handling
	setupErrorHandling(c)

	// Setup request/response logging
	setupLogging(c, config.EnableDebug)

	// Add common headers
	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.5")
		r.Headers.Set("Accept-Encoding", "gzip, deflate")
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")
	})

	return c
}

// randomDelay generates a random delay between min and max duration
func randomDelay(min, max time.Duration) time.Duration {
	if min >= max {
		return min
	}
	diff := max - min
	return min + time.Duration(rand.Int63n(int64(diff)))
}

// setupRetryLogic configures retry mechanism for failed requests
func setupRetryLogic(c *colly.Collector, maxRetries int) {
	c.OnError(func(r *colly.Response, err error) {
		retries := r.Ctx.GetAny("retries")
		if retries == nil {
			retries = 0
		}

		retryCount := retries.(int)
		if retryCount < maxRetries {
			log.Printf("Retrying request to %s (attempt %d/%d): %v",
				r.Request.URL, retryCount+1, maxRetries, err)

			// Add delay before retry
			time.Sleep(randomDelay(1*time.Second, 3*time.Second))

			// Clone request with incremented retry count
			newCtx := colly.NewContext()
			newCtx.Put("retries", retryCount+1)
			r.Request.Retry()
		} else {
			log.Printf("Max retries exceeded for %s: %v", r.Request.URL, err)
		}
	})
}

// setupErrorHandling configures error handling for the collector
func setupErrorHandling(c *colly.Collector) {
	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Error scraping %s: %v", r.Request.URL, err)

		// Log response status if available
		if r != nil {
			log.Printf("Response status: %d", r.StatusCode)
		}
	})

	c.OnResponse(func(r *colly.Response) {
		// Log non-2xx status codes
		if r.StatusCode < 200 || r.StatusCode >= 300 {
			log.Printf("Non-2xx response from %s: %d", r.Request.URL, r.StatusCode)
		}
	})
}

// setupLogging configures request/response logging
func setupLogging(c *colly.Collector, debug bool) {
	if debug {
		c.OnRequest(func(r *colly.Request) {
			log.Printf("Visiting: %s", r.URL)
		})

		c.OnResponse(func(r *colly.Response) {
			log.Printf("Received response from: %s (status: %d, size: %d bytes)",
				r.Request.URL, r.StatusCode, len(r.Body))
		})
	}

	c.OnScraped(func(r *colly.Response) {
		if debug {
			log.Printf("Finished scraping: %s", r.Request.URL)
		}
	})
}

// CreateProductCollector creates a collector specifically for product pages
func CreateProductCollector(config *CollectorConfig) *colly.Collector {
	c := NewCollector(config)

	// Add specific rate limiting for product pages
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*torecacamp-pokemon.com*",
		Parallelism: 1,                       // More conservative for product pages
		Delay:       1500 * time.Millisecond, // Reduced from 3s to 1.5s
	})

	return c
}

// CreateSearchCollector creates a collector specifically for search pages
func CreateSearchCollector(config *CollectorConfig) *colly.Collector {
	c := NewCollector(config)

	// Search pages can handle slightly more aggressive scraping
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*torecacamp-pokemon.com*",
		Parallelism: config.ConcurrentRequests,
		Delay:       randomDelay(config.DelayMin, config.DelayMax),
	})

	return c
}

// ValidateCollector performs basic validation of collector setup
func ValidateCollector(c *colly.Collector) error {
	if c == nil {
		return fmt.Errorf("collector is nil")
	}

	if len(c.AllowedDomains) == 0 {
		return fmt.Errorf("no allowed domains configured")
	}

	return nil
}
