package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	// Server configuration
	Server ServerConfig `json:"server"`

	// Database configuration
	Database DatabaseConfig `json:"database"`

	// Monitor configuration
	Monitor MonitorConfig `json:"monitor"`

	// Discord configuration
	Discord DiscordConfig `json:"discord"`

	// WebDriver configuration
	WebDriver WebDriverConfig `json:"webdriver"`
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port         int    `json:"port"`
	Host         string `json:"host"`
	Environment  string `json:"environment"`
	LogLevel     string `json:"log_level"`
	EnableCORS   bool   `json:"enable_cors"`
	TrustedProxy string `json:"trusted_proxy"`
}

// DatabaseConfig holds database-related configuration
type DatabaseConfig struct {
	Path       string `json:"path"`
	InMemory   bool   `json:"in_memory"`
	SyncWrites bool   `json:"sync_writes"`
	BackupPath string `json:"backup_path"`
}

// MonitorConfig holds monitoring service configuration
type MonitorConfig struct {
	// Monitoring intervals
	CheckInterval       time.Duration `json:"check_interval"`
	BatchSize           int           `json:"batch_size"`
	MaxConcurrentChecks int           `json:"max_concurrent_checks"`

	// Retry configuration
	MaxRetries             int           `json:"max_retries"`
	RetryBackoffMultiplier float64       `json:"retry_backoff_multiplier"`
	InitialRetryDelay      time.Duration `json:"initial_retry_delay"`

	// Health check configuration
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	StaleThreshold      time.Duration `json:"stale_threshold"`

	// Auto-start monitoring
	AutoStart bool `json:"auto_start"`
}

// DiscordConfig holds Discord notification configuration
type DiscordConfig struct {
	WebhookURL string        `json:"webhook_url"`
	Enabled    bool          `json:"enabled"`
	Timeout    time.Duration `json:"timeout"`
	RateLimit  time.Duration `json:"rate_limit"`
	MaxRetries int           `json:"max_retries"`
	RetryDelay time.Duration `json:"retry_delay"`

	// Bot configuration
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`

	// Notification preferences
	NotifyOnInStock     bool `json:"notify_on_in_stock"`
	NotifyOnOutStock    bool `json:"notify_on_out_stock"`
	NotifyOnErrors      bool `json:"notify_on_errors"`
	NotifyOnPriceChange bool `json:"notify_on_price_change"`
}

// WebDriverConfig holds WebDriver configuration
type WebDriverConfig struct {
	PoolSize            int           `json:"pool_size"`
	Timeout             time.Duration `json:"timeout"`
	UserAgent           string        `json:"user_agent"`
	MaxRetries          int           `json:"max_retries"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
}

// LoadConfig loads configuration from environment variables and .env file
func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	config := &Config{}

	// Load each section
	var err error

	config.Server, err = loadServerConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load server config: %v", err)
	}

	config.Database, err = loadDatabaseConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load database config: %v", err)
	}

	config.Monitor, err = loadMonitorConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load monitor config: %v", err)
	}

	config.Discord, err = loadDiscordConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load discord config: %v", err)
	}

	config.WebDriver, err = loadWebDriverConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load webdriver config: %v", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %v", err)
	}

	return config, nil
}

// loadServerConfig loads server configuration from environment
func loadServerConfig() (ServerConfig, error) {
	config := ServerConfig{
		Port:         8080,
		Host:         "0.0.0.0",
		Environment:  "development",
		LogLevel:     "info",
		EnableCORS:   true,
		TrustedProxy: "",
	}

	if port := os.Getenv("PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Port = p
		}
	}

	if host := os.Getenv("HOST"); host != "" {
		config.Host = host
	}

	if env := os.Getenv("ENVIRONMENT"); env != "" {
		config.Environment = env
	}

	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		config.LogLevel = logLevel
	}

	if cors := os.Getenv("ENABLE_CORS"); cors == "false" {
		config.EnableCORS = false
	}

	if proxy := os.Getenv("TRUSTED_PROXY"); proxy != "" {
		config.TrustedProxy = proxy
	}

	return config, nil
}

// loadDatabaseConfig loads database configuration from environment
func loadDatabaseConfig() (DatabaseConfig, error) {
	config := DatabaseConfig{
		Path:       "./data/badger",
		InMemory:   false,
		SyncWrites: false,
		BackupPath: "./data/backup",
	}

	if path := os.Getenv("DB_PATH"); path != "" {
		config.Path = path
	}

	if inMemory := os.Getenv("DB_IN_MEMORY"); inMemory == "true" {
		config.InMemory = true
	}

	if syncWrites := os.Getenv("DB_SYNC_WRITES"); syncWrites == "true" {
		config.SyncWrites = true
	}

	if backupPath := os.Getenv("DB_BACKUP_PATH"); backupPath != "" {
		config.BackupPath = backupPath
	}

	return config, nil
}

// loadMonitorConfig loads monitor configuration from environment
func loadMonitorConfig() (MonitorConfig, error) {
	config := MonitorConfig{
		CheckInterval:          1 * time.Hour,
		BatchSize:              10,
		MaxConcurrentChecks:    5,
		MaxRetries:             3,
		RetryBackoffMultiplier: 2.0,
		InitialRetryDelay:      30 * time.Second,
		HealthCheckInterval:    10 * time.Minute,
		StaleThreshold:         6 * time.Hour,
		AutoStart:              false,
	}

	if interval := os.Getenv("MONITOR_INTERVAL"); interval != "" {
		if d, err := time.ParseDuration(interval); err == nil {
			config.CheckInterval = d
		}
	}

	if batchSize := os.Getenv("MONITOR_BATCH_SIZE"); batchSize != "" {
		if b, err := strconv.Atoi(batchSize); err == nil {
			config.BatchSize = b
		}
	}

	if maxConcurrent := os.Getenv("MAX_CONCURRENT_MONITORS"); maxConcurrent != "" {
		if m, err := strconv.Atoi(maxConcurrent); err == nil {
			config.MaxConcurrentChecks = m
		}
	}

	if maxRetries := os.Getenv("MONITOR_MAX_RETRIES"); maxRetries != "" {
		if m, err := strconv.Atoi(maxRetries); err == nil {
			config.MaxRetries = m
		}
	}

	if backoff := os.Getenv("MONITOR_RETRY_BACKOFF"); backoff != "" {
		if b, err := strconv.ParseFloat(backoff, 64); err == nil {
			config.RetryBackoffMultiplier = b
		}
	}

	if delay := os.Getenv("MONITOR_INITIAL_RETRY_DELAY"); delay != "" {
		if d, err := time.ParseDuration(delay); err == nil {
			config.InitialRetryDelay = d
		}
	}

	if healthInterval := os.Getenv("MONITOR_HEALTH_CHECK_INTERVAL"); healthInterval != "" {
		if d, err := time.ParseDuration(healthInterval); err == nil {
			config.HealthCheckInterval = d
		}
	}

	if staleThreshold := os.Getenv("MONITOR_STALE_THRESHOLD"); staleThreshold != "" {
		if d, err := time.ParseDuration(staleThreshold); err == nil {
			config.StaleThreshold = d
		}
	}

	if autoStart := os.Getenv("MONITOR_AUTO_START"); autoStart == "true" {
		config.AutoStart = true
	}

	return config, nil
}

// loadDiscordConfig loads Discord configuration from environment
func loadDiscordConfig() (DiscordConfig, error) {
	config := DiscordConfig{
		WebhookURL:          "",
		Enabled:             false,
		Timeout:             10 * time.Second,
		RateLimit:           5 * time.Minute,
		MaxRetries:          3,
		RetryDelay:          30 * time.Second,
		Username:            "Pokemon Card Monitor",
		AvatarURL:           "",
		NotifyOnInStock:     true,
		NotifyOnOutStock:    true,
		NotifyOnErrors:      true,
		NotifyOnPriceChange: true,
	}

	if webhookURL := os.Getenv("DISCORD_WEBHOOK_URL"); webhookURL != "" {
		config.WebhookURL = webhookURL
		config.Enabled = true
	}

	if enabled := os.Getenv("DISCORD_ENABLED"); enabled == "false" {
		config.Enabled = false
	}

	if timeout := os.Getenv("DISCORD_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			config.Timeout = d
		}
	}

	if rateLimit := os.Getenv("DISCORD_RATE_LIMIT"); rateLimit != "" {
		if d, err := time.ParseDuration(rateLimit); err == nil {
			config.RateLimit = d
		}
	}

	if maxRetries := os.Getenv("DISCORD_MAX_RETRIES"); maxRetries != "" {
		if m, err := strconv.Atoi(maxRetries); err == nil {
			config.MaxRetries = m
		}
	}

	if retryDelay := os.Getenv("DISCORD_RETRY_DELAY"); retryDelay != "" {
		if d, err := time.ParseDuration(retryDelay); err == nil {
			config.RetryDelay = d
		}
	}

	if username := os.Getenv("DISCORD_USERNAME"); username != "" {
		config.Username = username
	}

	if avatarURL := os.Getenv("DISCORD_AVATAR_URL"); avatarURL != "" {
		config.AvatarURL = avatarURL
	}

	if notifyInStock := os.Getenv("DISCORD_NOTIFY_IN_STOCK"); notifyInStock == "false" {
		config.NotifyOnInStock = false
	}

	if notifyOutStock := os.Getenv("DISCORD_NOTIFY_OUT_STOCK"); notifyOutStock == "false" {
		config.NotifyOnOutStock = false
	}

	if notifyErrors := os.Getenv("DISCORD_NOTIFY_ERRORS"); notifyErrors == "false" {
		config.NotifyOnErrors = false
	}

	if notifyPriceChange := os.Getenv("DISCORD_NOTIFY_PRICE_CHANGE"); notifyPriceChange == "false" {
		config.NotifyOnPriceChange = false
	}

	return config, nil
}

// loadWebDriverConfig loads WebDriver configuration from environment
func loadWebDriverConfig() (WebDriverConfig, error) {
	config := WebDriverConfig{
		PoolSize:            3,
		Timeout:             30 * time.Second,
		UserAgent:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		MaxRetries:          3,
		HealthCheckInterval: 5 * time.Minute,
	}

	if poolSize := os.Getenv("WEBDRIVER_POOL_SIZE"); poolSize != "" {
		if p, err := strconv.Atoi(poolSize); err == nil {
			config.PoolSize = p
		}
	}

	if timeout := os.Getenv("WEBDRIVER_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			config.Timeout = d
		}
	}

	if userAgent := os.Getenv("WEBDRIVER_USER_AGENT"); userAgent != "" {
		config.UserAgent = userAgent
	}

	if maxRetries := os.Getenv("WEBDRIVER_MAX_RETRIES"); maxRetries != "" {
		if m, err := strconv.Atoi(maxRetries); err == nil {
			config.MaxRetries = m
		}
	}

	if healthInterval := os.Getenv("WEBDRIVER_HEALTH_CHECK_INTERVAL"); healthInterval != "" {
		if d, err := time.ParseDuration(healthInterval); err == nil {
			config.HealthCheckInterval = d
		}
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Server validation
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Server.Host == "" {
		return fmt.Errorf("server host cannot be empty")
	}

	// Monitor validation
	if c.Monitor.CheckInterval < 1*time.Minute {
		return fmt.Errorf("monitor check interval cannot be less than 1 minute")
	}

	if c.Monitor.BatchSize < 1 {
		return fmt.Errorf("monitor batch size must be at least 1")
	}

	if c.Monitor.MaxConcurrentChecks < 1 {
		return fmt.Errorf("max concurrent checks must be at least 1")
	}

	// Discord validation
	if c.Discord.Enabled && c.Discord.WebhookURL == "" {
		return fmt.Errorf("Discord webhook URL is required when Discord is enabled")
	}

	// WebDriver validation
	if c.WebDriver.PoolSize < 1 {
		return fmt.Errorf("WebDriver pool size must be at least 1")
	}

	if c.WebDriver.Timeout < 1*time.Second {
		return fmt.Errorf("WebDriver timeout cannot be less than 1 second")
	}

	return nil
}

// IsDevelopment returns true if running in development environment
func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development"
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}

// GetServerAddress returns the full server address
func (c *Config) GetServerAddress() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// PrintConfig prints configuration summary (excluding sensitive data)
func (c *Config) PrintConfig() {
	fmt.Println("=== Configuration Summary ===")
	fmt.Printf("Server: %s:%d (env: %s)\n", c.Server.Host, c.Server.Port, c.Server.Environment)
	fmt.Printf("Database: %s (in-memory: %v)\n", c.Database.Path, c.Database.InMemory)
	fmt.Printf("Monitor: interval=%v, batch=%d, concurrent=%d\n",
		c.Monitor.CheckInterval, c.Monitor.BatchSize, c.Monitor.MaxConcurrentChecks)
	fmt.Printf("Discord: enabled=%v\n", c.Discord.Enabled)
	fmt.Printf("WebDriver: pool=%d, timeout=%v\n", c.WebDriver.PoolSize, c.WebDriver.Timeout)
	fmt.Println("=============================")
}
