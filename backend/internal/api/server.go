package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SiddhantaChandra/pokemon-card-scraper/internal/scraper"
	"github.com/SiddhantaChandra/pokemon-card-scraper/internal/search"
	"github.com/SiddhantaChandra/pokemon-card-scraper/internal/storage"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Server represents the API server
type Server struct {
	router       *gin.Engine
	storage      storage.Storage
	scraper      *scraper.Scraper
	searchEngine *search.SearchEngine
	handlers     *Handlers
	config       *ServerConfig
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port         string   `json:"port"`
	AllowOrigins []string `json:"allow_origins"`
	Debug        bool     `json:"debug"`
}

// DefaultServerConfig returns default server configuration
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Port: "8080",
		AllowOrigins: []string{
			"http://localhost:3000", // Next.js development server
			"http://localhost:3001", // Alternative port
			"http://frontend:3000",  // Docker frontend service
		},
		Debug: false,
	}
}

// NewServer creates a new API server instance
func NewServer(storage storage.Storage, scraper *scraper.Scraper, config *ServerConfig) (*Server, error) {
	if config == nil {
		config = DefaultServerConfig()
	}

	// Set Gin mode
	if config.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router
	router := gin.New()

	// Initialize search engine
	searchEngine := search.NewSearchEngine(storage)

	// Create server instance
	server := &Server{
		router:       router,
		storage:      storage,
		scraper:      scraper,
		searchEngine: searchEngine,
		config:       config,
	}

	// Initialize handlers
	server.handlers = NewHandlers(storage, scraper, searchEngine)

	// Setup middleware
	server.setupMiddleware()

	// Setup routes
	server.setupRoutes()

	return server, nil
}

// setupMiddleware configures middleware for the server
func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.router.Use(gin.Recovery())

	// Logger middleware
	s.router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	}))

	// CORS middleware
	corsConfig := cors.Config{
		AllowOrigins:     s.config.AllowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	s.router.Use(cors.New(corsConfig))

	// Error handling middleware
	s.router.Use(s.errorHandlingMiddleware())

	// Request timeout middleware
	s.router.Use(s.timeoutMiddleware(30 * time.Second))
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", s.handlers.HealthCheck)

	// API group
	api := s.router.Group("/api")
	{
		// Image proxy endpoint
		api.GET("/proxy-image", s.handlers.ImageProxy) // GET /api/proxy-image

		// Database management endpoints
		database := api.Group("/database")
		{
			database.DELETE("/reset", s.handlers.ResetDatabase) // DELETE /api/database/reset
		}

		// Card endpoints
		cards := api.Group("/cards")
		{
			cards.GET("", s.handlers.GetAllCards)                // GET /api/cards
			cards.GET("/search", s.handlers.SearchCards)         // GET /api/cards/search
			cards.GET("/:id", s.handlers.GetCard)                // GET /api/cards/:id
			cards.GET("/suggestions", s.handlers.GetSuggestions) // GET /api/cards/suggestions
		}

		// Scraping endpoints
		scrape := api.Group("/scrape")
		{
			scrape.POST("/start", s.handlers.StartScrape)     // POST /api/scrape/start
			scrape.GET("/status", s.handlers.GetScrapeStatus) // GET /api/scrape/status
			scrape.POST("/stop", s.handlers.StopScrape)       // POST /api/scrape/stop
			scrape.POST("/restart", s.handlers.RestartScrape) // POST /api/scrape/restart
		}

		// Statistics endpoints
		api.GET("/stats", s.handlers.GetStats)              // GET /api/stats
		api.GET("/sort-options", s.handlers.GetSortOptions) // GET /api/sort-options
	}

	// Catch-all for 404
	s.router.NoRoute(s.handlers.NotFound)
}

// errorHandlingMiddleware handles panics and errors
func (s *Server) errorHandlingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Internal server error",
					"message": "An unexpected error occurred",
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}

// timeoutMiddleware adds request timeout
func (s *Server) timeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set context timeout
		ctx := c.Request.Context()
		if _, hasDeadline := ctx.Deadline(); !hasDeadline {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
			c.Request = c.Request.WithContext(ctx)
		}
		c.Next()
	}
}

// Run starts the server
func (s *Server) Run() error {
	log.Printf("Starting server on port %s", s.config.Port)
	log.Printf("Allowed origins: %v", s.config.AllowOrigins)
	log.Printf("Debug mode: %v", s.config.Debug)

	return s.router.Run(":" + s.config.Port)
}

// RunWithGracefulShutdown starts the server with graceful shutdown
func (s *Server) RunWithGracefulShutdown() error {
	srv := &http.Server{
		Addr:    ":" + s.config.Port,
		Handler: s.router,
	}

	// Channel to listen for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on port %s", s.config.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Println("Server started. Press CTRL+C to shutdown...")

	// Wait for interrupt signal
	<-quit
	log.Println("Shutting down server...")

	// Create context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
		return err
	}

	log.Println("Server shutdown complete")
	return nil
}

// GetRouter returns the Gin router (useful for testing)
func (s *Server) GetRouter() *gin.Engine {
	return s.router
}
