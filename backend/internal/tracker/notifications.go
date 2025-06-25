package tracker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/SiddhantaChandra/pokemon-card-scraper/pkg/models"
)

// DiscordNotifier implements NotificationService for Discord webhooks
type DiscordNotifier struct {
	webhookURL string
	client     *http.Client
	username   string
	avatarURL  string
}

// DiscordConfig holds configuration for Discord notifications
type DiscordConfig struct {
	WebhookURL string        `json:"webhook_url"`
	Username   string        `json:"username"`
	AvatarURL  string        `json:"avatar_url"`
	Timeout    time.Duration `json:"timeout"`
}

// DefaultDiscordConfig returns default Discord notification configuration
func DefaultDiscordConfig() *DiscordConfig {
	return &DiscordConfig{
		Username:  "Pokemon Card Tracker",
		AvatarURL: "",
		Timeout:   10 * time.Second,
	}
}

// NewDiscordNotifier creates a new Discord notification service
func NewDiscordNotifier(config *DiscordConfig) *DiscordNotifier {
	if config == nil {
		config = DefaultDiscordConfig()
	}

	client := &http.Client{
		Timeout: config.Timeout,
	}

	return &DiscordNotifier{
		webhookURL: config.WebhookURL,
		client:     client,
		username:   config.Username,
		avatarURL:  config.AvatarURL,
	}
}

// DiscordMessage represents a Discord webhook message
type DiscordMessage struct {
	Content   string         `json:"content,omitempty"`
	Username  string         `json:"username,omitempty"`
	AvatarURL string         `json:"avatar_url,omitempty"`
	Embeds    []DiscordEmbed `json:"embeds,omitempty"`
}

// DiscordEmbed represents a Discord embed
type DiscordEmbed struct {
	Title       string            `json:"title,omitempty"`
	Description string            `json:"description,omitempty"`
	URL         string            `json:"url,omitempty"`
	Color       int               `json:"color,omitempty"`
	Timestamp   string            `json:"timestamp,omitempty"`
	Thumbnail   *DiscordThumbnail `json:"thumbnail,omitempty"`
	Image       *DiscordImage     `json:"image,omitempty"`
	Fields      []DiscordField    `json:"fields,omitempty"`
	Footer      *DiscordFooter    `json:"footer,omitempty"`
}

// DiscordThumbnail represents a Discord embed thumbnail
type DiscordThumbnail struct {
	URL string `json:"url"`
}

// DiscordImage represents a Discord embed image
type DiscordImage struct {
	URL string `json:"url"`
}

// DiscordField represents a Discord embed field
type DiscordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// DiscordFooter represents a Discord embed footer
type DiscordFooter struct {
	Text    string `json:"text"`
	IconURL string `json:"icon_url,omitempty"`
}

// SendStockAlert sends a stock change notification to Discord
func (dn *DiscordNotifier) SendStockAlert(tracker models.TrackerEntry, statusChanged bool) error {
	if dn.webhookURL == "" {
		return fmt.Errorf("Discord webhook URL not configured")
	}

	// Create the embed message
	embed := dn.createStockEmbed(tracker, statusChanged)

	message := DiscordMessage{
		Username:  dn.username,
		AvatarURL: dn.avatarURL,
		Embeds:    []DiscordEmbed{embed},
	}

	return dn.sendMessage(message)
}

// SendErrorAlert sends an error notification to Discord
func (dn *DiscordNotifier) SendErrorAlert(errorMessage string, err error) error {
	if dn.webhookURL == "" {
		return fmt.Errorf("Discord webhook URL not configured")
	}

	embed := DiscordEmbed{
		Title:       "🚨 Tracker Error",
		Description: errorMessage,
		Color:       0xFF0000, // Red color
		Timestamp:   time.Now().Format(time.RFC3339),
		Fields: []DiscordField{
			{
				Name:  "Error Details",
				Value: err.Error(),
			},
		},
		Footer: &DiscordFooter{
			Text: "Pokemon Card Tracker",
		},
	}

	message := DiscordMessage{
		Username:  dn.username,
		AvatarURL: dn.avatarURL,
		Embeds:    []DiscordEmbed{embed},
	}

	return dn.sendMessage(message)
}

// createStockEmbed creates a Discord embed for stock changes
func (dn *DiscordNotifier) createStockEmbed(tracker models.TrackerEntry, statusChanged bool) DiscordEmbed {
	var title string
	var color int
	var description string

	if statusChanged {
		if tracker.InStock {
			title = "🎉 Item Back in Stock!"
			color = 0x00FF00 // Green
			description = fmt.Sprintf("**%s** is now available!", tracker.Name)
		} else {
			title = "⚠️ Item Out of Stock"
			color = 0xFF8C00 // Orange
			description = fmt.Sprintf("**%s** is no longer available.", tracker.Name)
		}
	} else {
		title = "💰 Price Update"
		color = 0x0099FF // Blue
		description = fmt.Sprintf("Price updated for **%s**", tracker.Name)
	}

	fields := []DiscordField{
		{
			Name:   "Status",
			Value:  dn.getStatusText(tracker.InStock),
			Inline: true,
		},
		{
			Name:   "Price",
			Value:  tracker.FormattedPrice(),
			Inline: true,
		},
		{
			Name:   "Last Checked",
			Value:  tracker.TimeSinceLastCheck(),
			Inline: true,
		},
	}

	embed := DiscordEmbed{
		Title:       title,
		Description: description,
		URL:         tracker.URL,
		Color:       color,
		Timestamp:   time.Now().Format(time.RFC3339),
		Fields:      fields,
		Footer: &DiscordFooter{
			Text: "Pokemon Card Tracker",
		},
	}

	// Add thumbnail if image URL is available
	if tracker.ImageURL != "" {
		embed.Thumbnail = &DiscordThumbnail{
			URL: tracker.ImageURL,
		}
	}

	return embed
}

// getStatusText returns a formatted status text with emoji
func (dn *DiscordNotifier) getStatusText(inStock bool) string {
	if inStock {
		return "✅ In Stock"
	}
	return "❌ Out of Stock"
}

// sendMessage sends a message to Discord via webhook
func (dn *DiscordNotifier) sendMessage(message DiscordMessage) error {
	// Marshal message to JSON
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal Discord message: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", dn.webhookURL, bytes.NewBuffer(messageBytes))
	if err != nil {
		return fmt.Errorf("failed to create Discord webhook request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := dn.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Discord webhook: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Discord webhook returned status %d", resp.StatusCode)
	}

	log.Printf("Discord notification sent successfully")
	return nil
}

// Test sends a test notification to verify the webhook is working
func (dn *DiscordNotifier) Test() error {
	testTracker := models.TrackerEntry{
		ID:      "test",
		Name:    "Test Pokemon Card",
		URL:     "https://example.com/test-card",
		InStock: true,
		Price:   100.0,
	}

	return dn.SendStockAlert(testTracker, true)
}

// NoOpNotifier is a notification service that does nothing (for testing/disabled notifications)
type NoOpNotifier struct{}

// NewNoOpNotifier creates a new no-op notification service
func NewNoOpNotifier() *NoOpNotifier {
	return &NoOpNotifier{}
}

// SendStockAlert does nothing (no-op implementation)
func (n *NoOpNotifier) SendStockAlert(tracker models.TrackerEntry, statusChanged bool) error {
	log.Printf("NoOp notification: Stock alert for %s (in stock: %v)", tracker.Name, tracker.InStock)
	return nil
}

// SendErrorAlert does nothing (no-op implementation)
func (n *NoOpNotifier) SendErrorAlert(message string, err error) error {
	log.Printf("NoOp notification: Error alert - %s: %v", message, err)
	return nil
}

// MultiNotifier allows sending notifications to multiple services
type MultiNotifier struct {
	notifiers []NotificationService
}

// NewMultiNotifier creates a new multi-service notifier
func NewMultiNotifier(notifiers ...NotificationService) *MultiNotifier {
	return &MultiNotifier{
		notifiers: notifiers,
	}
}

// SendStockAlert sends stock alerts to all configured notifiers
func (mn *MultiNotifier) SendStockAlert(tracker models.TrackerEntry, statusChanged bool) error {
	var lastError error

	for _, notifier := range mn.notifiers {
		if err := notifier.SendStockAlert(tracker, statusChanged); err != nil {
			log.Printf("Failed to send stock alert via notifier: %v", err)
			lastError = err
		}
	}

	return lastError
}

// SendErrorAlert sends error alerts to all configured notifiers
func (mn *MultiNotifier) SendErrorAlert(message string, err error) error {
	var lastError error

	for _, notifier := range mn.notifiers {
		if err := notifier.SendErrorAlert(message, err); err != nil {
			log.Printf("Failed to send error alert via notifier: %v", err)
			lastError = err
		}
	}

	return lastError
}

// AddNotifier adds a new notifier to the multi-notifier
func (mn *MultiNotifier) AddNotifier(notifier NotificationService) {
	mn.notifiers = append(mn.notifiers, notifier)
}
