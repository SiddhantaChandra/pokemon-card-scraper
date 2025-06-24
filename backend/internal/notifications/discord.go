package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/SiddhantaChandra/pokemon-card-scraper/pkg/models"
)

// DiscordNotifier handles Discord webhook notifications
type DiscordNotifier struct {
	webhookURL  string
	client      *http.Client
	rateLimiter *RateLimiter
	mu          sync.RWMutex

	// Statistics
	sentCount  int64
	errorCount int64
	lastSent   *time.Time
}

// DiscordConfig holds configuration for Discord notifications
type DiscordConfig struct {
	WebhookURL string        `json:"webhook_url"` // Discord webhook URL
	Timeout    time.Duration `json:"timeout"`     // HTTP request timeout
	RateLimit  time.Duration `json:"rate_limit"`  // Rate limit between messages
	MaxRetries int           `json:"max_retries"` // Maximum retry attempts
	RetryDelay time.Duration `json:"retry_delay"` // Delay between retries

	// Embed configuration
	Username  string `json:"username"`   // Bot username
	AvatarURL string `json:"avatar_url"` // Bot avatar URL
	Color     int    `json:"color"`      // Embed color

	// Notification preferences
	NotifyOnInStock     bool `json:"notify_on_in_stock"`     // Send notifications when items come in stock
	NotifyOnOutStock    bool `json:"notify_on_out_stock"`    // Send notifications when items go out of stock
	NotifyOnErrors      bool `json:"notify_on_errors"`       // Send error notifications
	NotifyOnPriceChange bool `json:"notify_on_price_change"` // Send price change notifications
}

// DefaultDiscordConfig returns sensible defaults
func DefaultDiscordConfig() *DiscordConfig {
	return &DiscordConfig{
		Timeout:             10 * time.Second,
		RateLimit:           5 * time.Minute,
		MaxRetries:          3,
		RetryDelay:          30 * time.Second,
		Username:            "Pokemon Card Monitor",
		AvatarURL:           "",
		Color:               0x00FF00, // Green
		NotifyOnInStock:     true,
		NotifyOnOutStock:    true,
		NotifyOnErrors:      true,
		NotifyOnPriceChange: true,
	}
}

// NewDiscordNotifier creates a new Discord notification service
func NewDiscordNotifier(config *DiscordConfig) (*DiscordNotifier, error) {
	if config == nil {
		config = DefaultDiscordConfig()
	}

	if config.WebhookURL == "" {
		return nil, fmt.Errorf("webhook URL is required")
	}

	client := &http.Client{
		Timeout: config.Timeout,
	}

	rateLimiter := NewRateLimiter(config.RateLimit)

	return &DiscordNotifier{
		webhookURL:  config.WebhookURL,
		client:      client,
		rateLimiter: rateLimiter,
	}, nil
}

// SendStockAlert sends a stock status change notification
func (dn *DiscordNotifier) SendStockAlert(item models.TrackerItem, oldStatus, newStatus bool) error {
	// Check rate limit
	if !dn.rateLimiter.Allow() {
		log.Printf("Rate limit exceeded, skipping notification for %s", item.Name)
		return nil
	}

	var embed DiscordEmbed
	if newStatus && !oldStatus {
		// Item came in stock
		embed = dn.createInStockEmbed(item)
	} else if !newStatus && oldStatus {
		// Item went out of stock
		embed = dn.createOutOfStockEmbed(item)
	} else {
		// No status change, skip notification
		return nil
	}

	payload := DiscordWebhookPayload{
		Username:  "Pokemon Card Monitor",
		AvatarURL: "",
		Embeds:    []DiscordEmbed{embed},
	}

	return dn.sendWebhook(payload)
}

// SendErrorAlert sends an error notification
func (dn *DiscordNotifier) SendErrorAlert(item models.TrackerItem, errorMsg string) error {
	// Check rate limit
	if !dn.rateLimiter.Allow() {
		log.Printf("Rate limit exceeded, skipping error notification for %s", item.Name)
		return nil
	}

	embed := dn.createErrorEmbed(item, errorMsg)

	payload := DiscordWebhookPayload{
		Username:  "Pokemon Card Monitor",
		AvatarURL: "",
		Embeds:    []DiscordEmbed{embed},
	}

	return dn.sendWebhook(payload)
}

// SendPriceChangeAlert sends a price change notification
func (dn *DiscordNotifier) SendPriceChangeAlert(item models.TrackerItem, oldPrice, newPrice string) error {
	// Check rate limit
	if !dn.rateLimiter.Allow() {
		log.Printf("Rate limit exceeded, skipping price change notification for %s", item.Name)
		return nil
	}

	embed := dn.createPriceChangeEmbed(item, oldPrice, newPrice)

	payload := DiscordWebhookPayload{
		Username:  "Pokemon Card Monitor",
		AvatarURL: "",
		Embeds:    []DiscordEmbed{embed},
	}

	return dn.sendWebhook(payload)
}

// createInStockEmbed creates an embed for in-stock notifications
func (dn *DiscordNotifier) createInStockEmbed(item models.TrackerItem) DiscordEmbed {
	embed := DiscordEmbed{
		Title:       "🟢 Item In Stock!",
		Description: fmt.Sprintf("**%s** is now available!", item.Name),
		Color:       0x00FF00, // Green
		Timestamp:   time.Now().Format(time.RFC3339),
		Fields: []DiscordEmbedField{
			{
				Name:   "Product",
				Value:  item.Name,
				Inline: false,
			},
			{
				Name:   "URL",
				Value:  fmt.Sprintf("[View Product](%s)", item.URL),
				Inline: false,
			},
		},
	}

	if item.PriceYen != "" {
		embed.Fields = append(embed.Fields, DiscordEmbedField{
			Name:   "Price",
			Value:  item.PriceYen,
			Inline: true,
		})
	}

	if item.Image != "" {
		embed.Thumbnail = &DiscordEmbedImage{
			URL: item.Image,
		}
	}

	embed.Footer = &DiscordEmbedFooter{
		Text: "Pokemon Card Monitor",
	}

	return embed
}

// createOutOfStockEmbed creates an embed for out-of-stock notifications
func (dn *DiscordNotifier) createOutOfStockEmbed(item models.TrackerItem) DiscordEmbed {
	embed := DiscordEmbed{
		Title:       "🔴 Item Out of Stock",
		Description: fmt.Sprintf("**%s** is no longer available.", item.Name),
		Color:       0xFF0000, // Red
		Timestamp:   time.Now().Format(time.RFC3339),
		Fields: []DiscordEmbedField{
			{
				Name:   "Product",
				Value:  item.Name,
				Inline: false,
			},
			{
				Name:   "URL",
				Value:  fmt.Sprintf("[View Product](%s)", item.URL),
				Inline: false,
			},
		},
	}

	if item.Image != "" {
		embed.Thumbnail = &DiscordEmbedImage{
			URL: item.Image,
		}
	}

	embed.Footer = &DiscordEmbedFooter{
		Text: "Pokemon Card Monitor",
	}

	return embed
}

// createErrorEmbed creates an embed for error notifications
func (dn *DiscordNotifier) createErrorEmbed(item models.TrackerItem, errorMsg string) DiscordEmbed {
	embed := DiscordEmbed{
		Title:       "⚠️ Monitoring Error",
		Description: fmt.Sprintf("Error monitoring **%s**", item.Name),
		Color:       0xFFFF00, // Yellow
		Timestamp:   time.Now().Format(time.RFC3339),
		Fields: []DiscordEmbedField{
			{
				Name:   "Product",
				Value:  item.Name,
				Inline: false,
			},
			{
				Name:   "Error",
				Value:  errorMsg,
				Inline: false,
			},
			{
				Name:   "URL",
				Value:  fmt.Sprintf("[View Product](%s)", item.URL),
				Inline: false,
			},
		},
	}

	embed.Footer = &DiscordEmbedFooter{
		Text: "Pokemon Card Monitor",
	}

	return embed
}

// createPriceChangeEmbed creates an embed for price change notifications
func (dn *DiscordNotifier) createPriceChangeEmbed(item models.TrackerItem, oldPrice, newPrice string) DiscordEmbed {
	embed := DiscordEmbed{
		Title:       "💰 Price Change",
		Description: fmt.Sprintf("Price changed for **%s**", item.Name),
		Color:       0x0099FF, // Blue
		Timestamp:   time.Now().Format(time.RFC3339),
		Fields: []DiscordEmbedField{
			{
				Name:   "Product",
				Value:  item.Name,
				Inline: false,
			},
			{
				Name:   "Old Price",
				Value:  oldPrice,
				Inline: true,
			},
			{
				Name:   "New Price",
				Value:  newPrice,
				Inline: true,
			},
			{
				Name:   "URL",
				Value:  fmt.Sprintf("[View Product](%s)", item.URL),
				Inline: false,
			},
		},
	}

	if item.Image != "" {
		embed.Thumbnail = &DiscordEmbedImage{
			URL: item.Image,
		}
	}

	embed.Footer = &DiscordEmbedFooter{
		Text: "Pokemon Card Monitor",
	}

	return embed
}

// sendWebhook sends a webhook payload to Discord
func (dn *DiscordNotifier) sendWebhook(payload DiscordWebhookPayload) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", dn.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := dn.client.Do(req)
	if err != nil {
		dn.incrementErrorCount()
		return fmt.Errorf("failed to send webhook: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		dn.incrementErrorCount()
		return fmt.Errorf("webhook failed with status: %d", resp.StatusCode)
	}

	dn.incrementSentCount()
	log.Printf("Discord notification sent successfully")

	return nil
}

// GetStats returns notification statistics
func (dn *DiscordNotifier) GetStats() NotificationStats {
	dn.mu.RLock()
	defer dn.mu.RUnlock()

	return NotificationStats{
		SentCount:  dn.sentCount,
		ErrorCount: dn.errorCount,
		LastSent:   dn.lastSent,
	}
}

func (dn *DiscordNotifier) incrementSentCount() {
	dn.mu.Lock()
	defer dn.mu.Unlock()

	dn.sentCount++
	now := time.Now()
	dn.lastSent = &now
}

func (dn *DiscordNotifier) incrementErrorCount() {
	dn.mu.Lock()
	defer dn.mu.Unlock()

	dn.errorCount++
}

// Discord webhook structures

// DiscordWebhookPayload represents a Discord webhook payload
type DiscordWebhookPayload struct {
	Username  string         `json:"username,omitempty"`
	AvatarURL string         `json:"avatar_url,omitempty"`
	Content   string         `json:"content,omitempty"`
	Embeds    []DiscordEmbed `json:"embeds,omitempty"`
}

// DiscordEmbed represents a Discord embed
type DiscordEmbed struct {
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	URL         string              `json:"url,omitempty"`
	Color       int                 `json:"color,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
	Footer      *DiscordEmbedFooter `json:"footer,omitempty"`
	Image       *DiscordEmbedImage  `json:"image,omitempty"`
	Thumbnail   *DiscordEmbedImage  `json:"thumbnail,omitempty"`
	Author      *DiscordEmbedAuthor `json:"author,omitempty"`
	Fields      []DiscordEmbedField `json:"fields,omitempty"`
}

// DiscordEmbedFooter represents a Discord embed footer
type DiscordEmbedFooter struct {
	Text    string `json:"text"`
	IconURL string `json:"icon_url,omitempty"`
}

// DiscordEmbedImage represents a Discord embed image
type DiscordEmbedImage struct {
	URL string `json:"url"`
}

// DiscordEmbedAuthor represents a Discord embed author
type DiscordEmbedAuthor struct {
	Name    string `json:"name"`
	URL     string `json:"url,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

// DiscordEmbedField represents a Discord embed field
type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// NotificationStats represents notification statistics
type NotificationStats struct {
	SentCount  int64      `json:"sent_count"`
	ErrorCount int64      `json:"error_count"`
	LastSent   *time.Time `json:"last_sent"`
}

// RateLimiter provides simple rate limiting for notifications
type RateLimiter struct {
	interval time.Duration
	lastSent time.Time
	mu       sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(interval time.Duration) *RateLimiter {
	return &RateLimiter{
		interval: interval,
	}
}

// Allow checks if an action is allowed based on rate limiting
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	if now.Sub(rl.lastSent) >= rl.interval {
		rl.lastSent = now
		return true
	}

	return false
}
