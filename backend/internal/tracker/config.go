package tracker

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

// TrackerConfig holds all tracker-related configuration
type TrackerConfig struct {
	Enabled     bool           `json:"enabled"`
	Worker      *WorkerConfig  `json:"worker"`
	Scraper     *ScraperConfig `json:"scraper"`
	Discord     *DiscordConfig `json:"discord"`
	Development bool           `json:"development"`
}

// DefaultTrackerConfig returns default tracker configuration
func DefaultTrackerConfig() *TrackerConfig {
	return &TrackerConfig{
		Enabled:     true,
		Worker:      DefaultWorkerConfig(),
		Scraper:     DefaultScraperConfig(),
		Discord:     DefaultDiscordConfig(),
		Development: false,
	}
}

// LoadTrackerConfigFromEnv loads tracker configuration from environment variables
func LoadTrackerConfigFromEnv() *TrackerConfig {
	config := ProductionDefaults()

	// Load basic settings
	if enabled := os.Getenv("TRACKER_ENABLED"); enabled != "" {
		config.Enabled = enabled == "true"
	}

	// Load worker settings
	if interval := os.Getenv("TRACKER_SCAN_INTERVAL"); interval != "" {
		if duration, err := time.ParseDuration(interval); err == nil {
			config.Worker.ScanInterval = duration
		}
	}

	if workers := os.Getenv("TRACKER_MAX_WORKERS"); workers != "" {
		if maxWorkers, err := strconv.Atoi(workers); err == nil && maxWorkers > 0 {
			config.Worker.MaxWorkers = maxWorkers
		}
	}

	if timeout := os.Getenv("TRACKER_TIMEOUT"); timeout != "" {
		if duration, err := time.ParseDuration(timeout); err == nil {
			config.Worker.TimeoutPerURL = duration
		}
	}

	// Load notification settings
	if webhookURL := os.Getenv("DISCORD_WEBHOOK_URL"); webhookURL != "" {
		config.Discord.WebhookURL = webhookURL
		config.Worker.EnableNotifications = true
	} else {
		config.Worker.EnableNotifications = false
	}

	return config
}

// IsValid checks if the tracker configuration is valid
func (tc *TrackerConfig) IsValid() error {
	if !tc.Enabled {
		return nil // Valid but disabled
	}

	if tc.Worker.ScanInterval < time.Minute {
		return fmt.Errorf("scan interval too short: %v (minimum 1 minute)", tc.Worker.ScanInterval)
	}

	if tc.Worker.MaxWorkers < 1 {
		return fmt.Errorf("max workers must be at least 1, got: %d", tc.Worker.MaxWorkers)
	}

	if tc.Worker.TimeoutPerURL < 5*time.Second {
		return fmt.Errorf("timeout per URL too short: %v (minimum 5 seconds)", tc.Worker.TimeoutPerURL)
	}

	if tc.Scraper.Timeout < 10*time.Second {
		return fmt.Errorf("scraper timeout too short: %v (minimum 10 seconds)", tc.Scraper.Timeout)
	}

	// Discord webhook URL is optional, but if notifications are enabled, warn
	if tc.Worker.EnableNotifications && tc.Discord.WebhookURL == "" {
		log.Println("WARNING: Notifications enabled but no Discord webhook URL configured")
	}

	return nil
}

// GetNotificationService returns the appropriate notification service based on configuration
func (tc *TrackerConfig) GetNotificationService() NotificationService {
	if !tc.Worker.EnableNotifications {
		return NewNoOpNotifier()
	}

	if tc.Discord.WebhookURL == "" {
		log.Println("WARNING: Notifications enabled but no Discord webhook URL - using NoOp notifier")
		return NewNoOpNotifier()
	}

	return NewDiscordNotifier(tc.Discord)
}

// GetDisplayConfig returns a sanitized config for display (removes sensitive info)
func (tc *TrackerConfig) GetDisplayConfig() *TrackerConfig {
	display := *tc

	// Sanitize Discord webhook URL for security
	if display.Discord.WebhookURL != "" {
		display.Discord.WebhookURL = "***configured***"
	}

	return &display
}

// ProductionDefaults returns production-optimized defaults
func ProductionDefaults() *TrackerConfig {
	config := DefaultTrackerConfig()

	// Production-optimized settings
	config.Worker.ScanInterval = 1 * time.Hour     // Check every hour
	config.Worker.MaxWorkers = 3                   // Conservative worker count
	config.Worker.TimeoutPerURL = 45 * time.Second // Longer timeout for stability
	config.Worker.RetryAttempts = 3                // More retries
	config.Worker.RetryDelay = 10 * time.Second    // Longer delay between retries

	config.Scraper.Timeout = 60 * time.Second // Longer timeout for scraping
	config.Scraper.Headless = true            // Always headless in production

	config.Discord.Timeout = 15 * time.Second // Reasonable Discord timeout

	config.Development = false

	return config
}

// DevelopmentDefaults returns development-optimized defaults
func DevelopmentDefaults() *TrackerConfig {
	config := DefaultTrackerConfig()

	// Development-optimized settings
	config.Worker.ScanInterval = 5 * time.Minute   // More frequent checks for testing
	config.Worker.MaxWorkers = 2                   // Fewer workers for dev
	config.Worker.TimeoutPerURL = 30 * time.Second // Shorter timeout
	config.Worker.RetryAttempts = 2                // Fewer retries
	config.Worker.RetryDelay = 5 * time.Second     // Shorter delay

	config.Scraper.Timeout = 30 * time.Second // Shorter timeout
	config.Scraper.Headless = true            // Still headless by default

	config.Discord.Timeout = 10 * time.Second // Shorter Discord timeout

	config.Development = true

	return config
}
