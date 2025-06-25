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
	"github.com/SiddhantaChandra/pokemon-card-scraper/internal/tracker"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Server represents the API server
type Server struct {
	router                *gin.Engine
	storage               storage.Storage
	scraper               scraper.ScraperInterface
	searchEngine          *search.SearchEngine
	handlers              *Handlers
	trackerHandlers       *TrackerHandlers
	simpleTrackerHandlers *SimplifiedTrackerHandlers
	trackerManager        *tracker.TrackerManager
	config                *ServerConfig
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port          string                 `json:"port"`
	AllowOrigins  []string               `json:"allow_origins"`
	Debug         bool                   `json:"debug"`
	TrackerConfig *tracker.TrackerConfig `json:"tracker"`
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
		Debug:         false,
		TrackerConfig: tracker.DefaultTrackerConfig(),
	}
}

// NewServer creates a new API server instance
func NewServer(storage storage.Storage, scraperInstance scraper.ScraperInterface, config *ServerConfig) (*Server, error) {
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
		scraper:      scraperInstance,
		searchEngine: searchEngine,
		config:       config,
	}

	// Initialize handlers
	server.handlers = NewHandlers(storage, scraperInstance, searchEngine)

	// Initialize simplified tracker handlers (always available)
	server.simpleTrackerHandlers = NewSimplifiedTrackerHandlers(storage)
	log.Println("Simplified tracker handlers initialized")

	// Initialize tracker system
	if config.TrackerConfig != nil && config.TrackerConfig.Enabled {
		log.Println("Initializing tracker system...")

		// Create tracker manager (pass storage as interface{})
		trackerManager, err := tracker.NewTrackerManager(config.TrackerConfig, storage)
		if err != nil {
			log.Printf("Warning: Failed to initialize tracker manager: %v", err)
			log.Println("Tracker functionality will be disabled")
		} else {
			server.trackerManager = trackerManager
			log.Println("Tracker system initialized successfully")

			// Note: TrackerHandlers initialization is skipped due to interface constraints
			// This would require extending the Storage interface to include tracker methods
			log.Println("Tracker handlers disabled - interface constraints")
		}
	} else {
		log.Println("Tracker system is disabled in configuration")
	}

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
			scrape.POST("/pause", s.handlers.PauseScrape)     // POST /api/scrape/pause
			scrape.POST("/resume", s.handlers.ResumeScrape)   // POST /api/scrape/resume
			scrape.POST("/restart", s.handlers.RestartScrape) // POST /api/scrape/restart
		}

		// Statistics endpoints
		api.GET("/stats", s.handlers.GetStats)              // GET /api/stats
		api.GET("/sort-options", s.handlers.GetSortOptions) // GET /api/sort-options

		// Tracker endpoints (using simplified handlers for demo/testing)
		log.Println("Setting up simplified tracker routes")
		tracker := api.Group("/tracker")
		{
			tracker.POST("/add", s.simpleTrackerHandlers.AddTracker)                     // POST /api/tracker/add
			tracker.GET("/list", s.simpleTrackerHandlers.GetTrackers)                    // GET /api/tracker/list
			tracker.GET("/:id", s.simpleTrackerHandlers.GetTracker)                      // GET /api/tracker/:id
			tracker.DELETE("/:id", s.simpleTrackerHandlers.DeleteTracker)                // DELETE /api/tracker/:id
			tracker.POST("/check-now", s.simpleTrackerHandlers.CheckNow)                 // POST /api/tracker/check-now
			tracker.GET("/status", s.simpleTrackerHandlers.GetTrackerStatus)             // GET /api/tracker/status
			tracker.GET("/options", s.simpleTrackerHandlers.GetTrackerOptions)           // GET /api/tracker/options
			tracker.POST("/test-notification", s.simpleTrackerHandlers.TestNotification) // POST /api/tracker/test-notification
		}

		// Full tracker endpoints (now enabled if tracker is initialized)
		if s.trackerHandlers != nil {
			log.Println("Setting up full tracker routes")
			trackerFull := api.Group("/tracker-full")
			{
				trackerFull.POST("/add", s.trackerHandlers.AddTracker)           // POST /api/tracker-full/add
				trackerFull.GET("/list", s.trackerHandlers.GetTrackers)          // GET /api/tracker-full/list
				trackerFull.GET("/:id", s.trackerHandlers.GetTracker)            // GET /api/tracker-full/:id
				trackerFull.PUT("/:id", s.trackerHandlers.UpdateTracker)         // PUT /api/tracker-full/:id
				trackerFull.DELETE("/:id", s.trackerHandlers.DeleteTracker)      // DELETE /api/tracker-full/:id
				trackerFull.POST("/check-now", s.trackerHandlers.CheckNow)       // POST /api/tracker-full/check-now
				trackerFull.GET("/status", s.trackerHandlers.GetTrackerStatus)   // GET /api/tracker-full/status
				trackerFull.GET("/options", s.trackerHandlers.GetTrackerOptions) // GET /api/tracker-full/options
			}
		}
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
	// Start tracker system if enabled
	if s.trackerManager != nil && s.trackerManager.IsEnabled() {
		log.Println("Starting tracker system...")
		if err := s.trackerManager.Start(); err != nil {
			log.Printf("Warning: Failed to start tracker system: %v", err)
		} else {
			log.Println("Tracker system started successfully")
		}
	}

	srv := &http.Server{
		Addr:    ":" + s.config.Port,
		Handler: s.router,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on port %s", s.config.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Stop tracker system
	if s.trackerManager != nil && s.trackerManager.IsRunning() {
		log.Println("Stopping tracker system...")
		if err := s.trackerManager.Stop(); err != nil {
			log.Printf("Error stopping tracker system: %v", err)
		}
	}

	// Create context with timeout for server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
		return err
	}

	log.Println("Server exited")
	return nil
}

// GetRouter returns the Gin router (useful for testing)
func (s *Server) GetRouter() *gin.Engine {
	return s.router
}

// GetTrackerManager returns the tracker manager (may be nil)
func (s *Server) GetTrackerManager() *tracker.TrackerManager {
	return s.trackerManager
}
