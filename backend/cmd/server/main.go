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

	// Initialize scraper
	log.Println("Setting up scraper...")
	scraperConfig := scraper.DefaultScraperConfig()
	cardScraper := scraper.NewScraper(scraperConfig)

	// Set up scraper callbacks to save cards to storage
	cardScraper.SetCardFoundCallback(func(card models.Card) {
		if err := cachedStorage.SaveCard(card); err != nil {
			log.Printf("Failed to save card %s: %v", card.Name, err)
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

	server, err := api.NewServer(cachedStorage, cardScraper, serverConfig)
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
