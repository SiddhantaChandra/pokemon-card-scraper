package main

import (
	"log"
	"os"
	"time"

	"github.com/SiddhantaChandra/pokemon-card-scraper/internal/api"
	"github.com/SiddhantaChandra/pokemon-card-scraper/internal/config"
	"github.com/SiddhantaChandra/pokemon-card-scraper/internal/monitor"
	"github.com/SiddhantaChandra/pokemon-card-scraper/internal/notifications"
	"github.com/SiddhantaChandra/pokemon-card-scraper/internal/scraper"
	"github.com/SiddhantaChandra/pokemon-card-scraper/internal/storage"
	"github.com/SiddhantaChandra/pokemon-card-scraper/pkg/models"
	"github.com/joho/godotenv"
)

// noOpNotifier is a no-operation notification service for fallback
type noOpNotifier struct{}

func (n *noOpNotifier) SendStockAlert(item models.TrackerItem, oldStatus, newStatus bool) error {
	return nil
}

func (n *noOpNotifier) SendErrorAlert(item models.TrackerItem, error string) error {
	return nil
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	log.Println("Initializing Pokemon Card Scraper API...")

	// Initialize storage
	log.Println("Setting up storage...")
	storageConfig := storage.DefaultBadgerConfig()

	// Override with environment variables if provided
	if dbPath := os.Getenv("DB_PATH"); dbPath != "" {
		storageConfig.Path = dbPath
	}

	badgerStorage, err := storage.NewBadgerStorage(storageConfig)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer badgerStorage.Close()

	// Wrap with cache
	cacheConfig := storage.DefaultCacheConfig()
	cachedStorage := storage.NewCachedStorage(badgerStorage, cacheConfig)

	// Initialize batch processor for improved performance
	log.Println("Setting up batch processor...")
	batchConfig := storage.DefaultBatchProcessorConfig()
	batchProcessor := storage.NewBatchProcessor(cachedStorage, batchConfig)
	defer batchProcessor.Close()

	// Initialize scraper based on environment variable
	useParallelScraper := os.Getenv("USE_PARALLEL_SCRAPER")
	if useParallelScraper == "" {
		useParallelScraper = "true" // Default to parallel scraper
	}

	var scraperInstance scraper.ScraperInterface

	if useParallelScraper == "true" {
		// Initialize parallel scraper for enhanced performance
		log.Println("Setting up parallel scraper...")
		parallelConfig := scraper.DefaultParallelScraperConfig()
		parallelConfig.PageWorkers = 5 // Use 5 concurrent workers to avoid rate limiting
		parallelScraper := scraper.NewParallelScraper(parallelConfig, batchProcessor)

		log.Printf("Parallel scraper configured with %d workers and %d collector pool size",
			parallelConfig.PageWorkers, parallelConfig.CollectorPoolSize)

		// Set up parallel scraper callbacks to save cards to batch processor
		parallelScraper.SetCardFoundCallback(func(card models.Card) {
			if err := batchProcessor.AddCard(card); err != nil {
				log.Printf("Failed to add card %s to batch: %v", card.Name, err)
			} else {
				log.Printf("Saved card: %s", card.Name)
			}
		})

		scraperInstance = parallelScraper
	} else {
		// Use regular scraper as fallback
		log.Println("Setting up regular scraper (parallel scraping disabled)...")
		scraperConfig := scraper.DefaultScraperConfig()
		regularScraper := scraper.NewScraper(scraperConfig)

		// Set up regular scraper callbacks
		regularScraper.SetCardFoundCallback(func(card models.Card) {
			if err := batchProcessor.AddCard(card); err != nil {
				log.Printf("Failed to add card %s to batch: %v", card.Name, err)
			} else {
				log.Printf("Saved card: %s", card.Name)
			}
		})

		scraperInstance = regularScraper
	}

	// Create a wrapper scraper for API compatibility (keep for potential fallback)
	log.Println("Setting up scraper wrapper...")
	scraperConfig := scraper.DefaultScraperConfig()
	cardScraper := scraper.NewScraper(scraperConfig)

	// Set up regular scraper callbacks for fallback compatibility
	cardScraper.SetCardFoundCallback(func(card models.Card) {
		if err := batchProcessor.AddCard(card); err != nil {
			log.Printf("Failed to add card %s to batch: %v", card.Name, err)
		} else {
			log.Printf("Saved card: %s", card.Name)
		}
	})

	// Initialize configuration
	log.Println("Loading configuration...")
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize Discord notifications
	log.Println("Setting up Discord notifications...")
	var notifier monitor.NotificationService
	if cfg.Discord.Enabled {
		// Convert config.DiscordConfig to notifications.DiscordConfig
		discordConfig := &notifications.DiscordConfig{
			WebhookURL:          cfg.Discord.WebhookURL,
			Timeout:             cfg.Discord.Timeout,
			RateLimit:           cfg.Discord.RateLimit,
			MaxRetries:          cfg.Discord.MaxRetries,
			RetryDelay:          cfg.Discord.RetryDelay,
			Username:            cfg.Discord.Username,
			AvatarURL:           cfg.Discord.AvatarURL,
			Color:               0x00FF00, // Green default color
			NotifyOnInStock:     cfg.Discord.NotifyOnInStock,
			NotifyOnOutStock:    cfg.Discord.NotifyOnOutStock,
			NotifyOnErrors:      cfg.Discord.NotifyOnErrors,
			NotifyOnPriceChange: cfg.Discord.NotifyOnPriceChange,
		}

		discordNotifier, err := notifications.NewDiscordNotifier(discordConfig)
		if err != nil {
			log.Printf("Failed to initialize Discord notifier: %v", err)
			// Use no-op notifier as fallback
			notifier = &noOpNotifier{}
		} else {
			notifier = discordNotifier
		}
	} else {
		log.Println("Discord notifications disabled")
		notifier = &noOpNotifier{}
	}

	// Initialize WebDriver pool
	log.Println("Setting up WebDriver pool...")

	// Convert config.WebDriverConfig to monitor.WebDriverPoolConfig
	webDriverConfig := &monitor.WebDriverPoolConfig{
		PoolSize:   cfg.WebDriver.PoolSize,
		Timeout:    cfg.WebDriver.Timeout,
		UserAgent:  cfg.WebDriver.UserAgent,
		MaxRetries: cfg.WebDriver.MaxRetries,
	}

	webDriverPool, err := monitor.NewHTTPWebDriverPool(webDriverConfig)
	if err != nil {
		log.Fatalf("Failed to initialize WebDriver pool: %v", err)
	}
	defer webDriverPool.Close()

	// Initialize stock monitor
	log.Println("Setting up stock monitor...")

	// Convert config.MonitorConfig to monitor.MonitorConfig
	monitorConfig := &monitor.MonitorConfig{
		CheckInterval:          cfg.Monitor.CheckInterval,
		BatchSize:              cfg.Monitor.BatchSize,
		MaxConcurrentChecks:    cfg.Monitor.MaxConcurrentChecks,
		MaxRetries:             cfg.Monitor.MaxRetries,
		RetryBackoffMultiplier: cfg.Monitor.RetryBackoffMultiplier,
		InitialRetryDelay:      cfg.Monitor.InitialRetryDelay,
		HealthCheckInterval:    cfg.Monitor.HealthCheckInterval,
		StaleThreshold:         cfg.Monitor.StaleThreshold,
		NotificationEnabled:    true, // Enable notifications since we have a notifier
		NotificationRateLimit:  5 * time.Minute,
		WebDriverPoolSize:      3,
		WebDriverTimeout:       30 * time.Second,
	}

	stockMonitor := monitor.NewStockMonitor(monitorConfig, cachedStorage, notifier, webDriverPool)
	defer stockMonitor.StopMonitoring()

	// Auto-start monitoring if configured
	if cfg.Monitor.AutoStart {
		log.Println("Auto-starting stock monitoring...")
		if err := stockMonitor.StartMonitoring(); err != nil {
			log.Printf("Failed to auto-start monitoring: %v", err)
		}
	}

	// Initialize API server
	log.Println("Setting up API server...")
	serverConfig := api.DefaultServerConfig()

	// Override with environment variables if provided
	if port := os.Getenv("PORT"); port != "" {
		serverConfig.Port = port
	}
	if debug := os.Getenv("DEBUG"); debug == "true" {
		serverConfig.Debug = true
	}

	// Create API server with parallel scraper as primary scraper
	server, err := api.NewServer(cachedStorage, scraperInstance, stockMonitor, serverConfig)
	if err != nil {
		log.Fatalf("Failed to create API server: %v", err)
	}

	// Start server with graceful shutdown
	log.Printf("Starting Pokemon Card Scraper API on port %s", serverConfig.Port)
	log.Println("Available endpoints:")
	log.Println("  GET  /health              - Health check")
	log.Println("  GET  /api/cards           - List all cards")
	log.Println("  GET  /api/cards/search    - Search cards")
	log.Println("  GET  /api/cards/:id       - Get single card")
	log.Println("  GET  /api/cards/suggestions - Get search suggestions")
	log.Println("  POST /api/scrape/start    - Start scraping")
	log.Println("  GET  /api/scrape/status   - Get scrape status")
	log.Println("  POST /api/scrape/stop     - Stop scraping")
	log.Println("  GET  /api/stats           - Get statistics")
	log.Println("  GET  /api/sort-options    - Get sort options")
	log.Println("  POST /api/tracker         - Add new tracker")
	log.Println("  GET  /api/tracker         - List all trackers")
	log.Println("  GET  /api/tracker/:id     - Get specific tracker")
	log.Println("  PUT  /api/tracker/:id     - Update tracker")
	log.Println("  DELETE /api/tracker/:id   - Delete tracker")
	log.Println("  POST /api/tracker/bulk    - Bulk add trackers")
	log.Println("  POST /api/monitor/start   - Start monitoring")
	log.Println("  POST /api/monitor/stop    - Stop monitoring")
	log.Println("  GET  /api/monitor/status  - Get monitor status")
	log.Println("  GET  /api/monitor/stats   - Get monitor stats")

	if err := server.RunWithGracefulShutdown(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
