package main

import (
	"log"
	"os"

	"github.com/SiddhantaChandra/pokemon-card-scraper/internal/api"
	"github.com/SiddhantaChandra/pokemon-card-scraper/internal/scraper"
	"github.com/SiddhantaChandra/pokemon-card-scraper/internal/storage"
	"github.com/SiddhantaChandra/pokemon-card-scraper/pkg/models"
	"github.com/joho/godotenv"
)

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
	server, err := api.NewServer(cachedStorage, scraperInstance, serverConfig)
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

	if err := server.RunWithGracefulShutdown(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
