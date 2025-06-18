package scraper

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/SiddhantaChandra/pokemon-card-scraper/pkg/models"
)

// ProductParser handles parsing of individual product pages and search results
type ProductParser struct {
	baseURL string
}

// NewProductParser creates a new product parser
func NewProductParser(baseURL string) *ProductParser {
	return &ProductParser{
		baseURL: baseURL,
	}
}

// ParseSearchResults extracts product information from search result pages
func (p *ProductParser) ParseSearchResults(c *colly.Collector, onProduct func(models.Card)) {
	// Parse individual product items in search results
	c.OnHTML(".product-item, .grid-item, .product-grid-item", func(e *colly.HTMLElement) {
		card := p.extractProductFromElement(e)
		if card.ID != "" {
			onProduct(card)
		}
	})

	// Handle pagination
	c.OnHTML(".pagination a, .next, .page-next", func(e *colly.HTMLElement) {
		nextURL := e.Attr("href")
		if nextURL != "" {
			// Convert relative URL to absolute
			if !strings.HasPrefix(nextURL, "http") {
				nextURL = p.baseURL + nextURL
			}
			e.Request.Visit(nextURL)
		}
	})
}

// ParseProductDetails extracts detailed information from individual product pages
func (p *ProductParser) ParseProductDetails(c *colly.Collector, onProduct func(models.Card)) {
	c.OnHTML("body", func(e *colly.HTMLElement) {
		card := p.extractDetailedProduct(e)
		if card.ID != "" {
			onProduct(card)
		}
	})
}

// extractProductFromElement extracts basic product info from search result elements
func (p *ProductParser) extractProductFromElement(e *colly.HTMLElement) models.Card {
	card := models.Card{
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Extract product URL and generate ID
	productURL := e.ChildAttr("a", "href")
	if productURL != "" {
		if !strings.HasPrefix(productURL, "http") {
			productURL = p.baseURL + productURL
		}
		card.URL = productURL
		card.ID = p.generateIDFromURL(productURL)
	}

	// Extract product name (try multiple selectors)
	name := p.tryExtractText(e, []string{
		".product-title",
		".product-name", 
		".grid-product__title",
		"h3 a",
		"h2 a",
		".product-link",
	})
	card.Name = strings.TrimSpace(name)

	// Extract price
	priceText := p.tryExtractText(e, []string{
		".price",
		".product-price",
		".grid-product__price",
		".money",
		".price-item",
	})
	card.Price = p.parsePrice(priceText)

	// Extract image URL
	imageURL := p.tryExtractAttr(e, []string{
		"img",
		".product-image img",
		".grid-product__image img",
	}, "src")
	if imageURL != "" && !strings.HasPrefix(imageURL, "http") {
		imageURL = p.baseURL + imageURL
	}
	card.ImageURL = imageURL

	// Extract stock status
	stockText := p.tryExtractText(e, []string{
		".stock-status",
		".availability",
		".inventory",
		".product-availability",
	})
	card.Stock = p.parseStock(stockText)
	card.InStock = card.Stock > 0

	// Try to extract set name from title or other elements
	card.SetName = p.extractSetName(card.Name)

	// Try to extract rarity
	rarityText := p.tryExtractText(e, []string{
		".rarity",
		".product-rarity",
		".card-rarity",
	})
	card.Rarity = strings.TrimSpace(rarityText)

	return card
}

// extractDetailedProduct extracts comprehensive product info from product detail pages
func (p *ProductParser) extractDetailedProduct(e *colly.HTMLElement) models.Card {
	card := models.Card{
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		URL:       e.Request.URL.String(),
		ID:        p.generateIDFromURL(e.Request.URL.String()),
	}

	// Extract detailed product information
	card.Name = strings.TrimSpace(p.tryExtractText(e, []string{
		"h1.product-title",
		"h1.product-name",
		".product-single__title",
		"h1",
	}))

	// Japanese name might be in parentheses or separate element
	card.NameJP = p.extractJapaneseName(e, card.Name)

	// Extract price with more precision
	priceText := p.tryExtractText(e, []string{
		".product-price .money",
		".price-item--sale",
		".price-item--regular",
		".product-single__price",
	})
	card.Price = p.parsePrice(priceText)

	// Extract detailed image
	imageURL := p.tryExtractAttr(e, []string{
		".product-single__photo img",
		".product-image-main img",
		".product-gallery img",
	}, "src")
	if imageURL != "" && !strings.HasPrefix(imageURL, "http") {
		imageURL = p.baseURL + imageURL
	}
	card.ImageURL = imageURL

	// Extract detailed stock information
	stockText := p.tryExtractText(e, []string{
		".product-inventory",
		".product-stock",
		".availability-text",
	})
	card.Stock = p.parseStock(stockText)
	card.InStock = card.Stock > 0

	// Extract set information
	card.SetName = p.extractSetNameDetailed(e, card.Name)

	// Extract rarity information
	card.Rarity = p.extractRarityDetailed(e)

	return card
}

// tryExtractText tries multiple CSS selectors to extract text
func (p *ProductParser) tryExtractText(e *colly.HTMLElement, selectors []string) string {
	for _, selector := range selectors {
		text := strings.TrimSpace(e.ChildText(selector))
		if text != "" {
			return text
		}
	}
	return ""
}

// tryExtractAttr tries multiple CSS selectors to extract an attribute
func (p *ProductParser) tryExtractAttr(e *colly.HTMLElement, selectors []string, attr string) string {
	for _, selector := range selectors {
		value := strings.TrimSpace(e.ChildAttr(selector, attr))
		if value != "" {
			return value
		}
	}
	return ""
}

// parsePrice extracts numeric price from price text
func (p *ProductParser) parsePrice(priceText string) float64 {
	if priceText == "" {
		return 0
	}

	// Remove currency symbols and clean the string
	re := regexp.MustCompile(`[^\d.,]`)
	cleanPrice := re.ReplaceAllString(priceText, "")
	
	// Handle different decimal separators
	cleanPrice = strings.ReplaceAll(cleanPrice, ",", ".")
	
	if price, err := strconv.ParseFloat(cleanPrice, 64); err == nil {
		return price
	}
	
	return 0
}

// parseStock extracts stock quantity from stock text
func (p *ProductParser) parseStock(stockText string) int {
	if stockText == "" {
		return 0
	}

	stockText = strings.ToLower(stockText)
	
	// Check for out of stock indicators
	if strings.Contains(stockText, "out of stock") ||
	   strings.Contains(stockText, "sold out") ||
	   strings.Contains(stockText, "unavailable") {
		return 0
	}

	// Check for in stock indicators
	if strings.Contains(stockText, "in stock") ||
	   strings.Contains(stockText, "available") {
		return 1 // Default to 1 if we know it's in stock but don't have exact quantity
	}

	// Try to extract numeric quantity
	re := regexp.MustCompile(`\d+`)
	matches := re.FindAllString(stockText, -1)
	if len(matches) > 0 {
		if quantity, err := strconv.Atoi(matches[0]); err == nil {
			return quantity
		}
	}

	return 0
}

// generateIDFromURL creates a unique ID from the product URL
func (p *ProductParser) generateIDFromURL(productURL string) string {
	if productURL == "" {
		return ""
	}

	// Parse URL and extract meaningful parts
	u, err := url.Parse(productURL)
	if err != nil {
		return ""
	}

	// Extract product ID from path or query parameters
	path := strings.Trim(u.Path, "/")
	pathParts := strings.Split(path, "/")
	
	// Try to find a product ID in the URL
	for _, part := range pathParts {
		if strings.Contains(part, "product") || len(part) > 10 {
			return part
		}
	}

	// Fallback: use the last part of the path
	if len(pathParts) > 0 {
		return pathParts[len(pathParts)-1]
	}

	return ""
}

// extractSetName tries to extract Pokemon set name from the product name
func (p *ProductParser) extractSetName(productName string) string {
	// Common Pokemon set patterns
	setPatterns := []string{
		`\b([A-Z][A-Za-z\s]+(?:Base Set|Jungle|Fossil|Team Rocket|Gym Heroes|Gym Challenge))\b`,
		`\b(Sword\s*&\s*Shield[^,]*)\b`,
		`\b(Sun\s*&\s*Moon[^,]*)\b`,
		`\b(XY[^,]*)\b`,
		`\b(Black\s*&\s*White[^,]*)\b`,
	}

	for _, pattern := range setPatterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(productName); len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}

	return ""
}

// extractSetNameDetailed extracts set name from detailed product page
func (p *ProductParser) extractSetNameDetailed(e *colly.HTMLElement, productName string) string {
	// First try to find set name in specific elements
	setName := p.tryExtractText(e, []string{
		".product-set",
		".card-set",
		".set-name",
		".product-meta .set",
	})

	if setName != "" {
		return setName
	}

	// Fallback to extracting from product name
	return p.extractSetName(productName)
}

// extractRarityDetailed extracts rarity information from detailed product page
func (p *ProductParser) extractRarityDetailed(e *colly.HTMLElement) string {
	rarity := p.tryExtractText(e, []string{
		".product-rarity",
		".card-rarity",
		".rarity-text",
		".product-meta .rarity",
	})

	if rarity != "" {
		return rarity
	}

	// Try to extract from product description
	description := p.tryExtractText(e, []string{
		".product-description",
		".product-single__description",
	})

	rarityPatterns := []string{
		`(?i)\b(common|uncommon|rare|ultra rare|secret rare|rainbow rare)\b`,
	}

	for _, pattern := range rarityPatterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(description); len(matches) > 1 {
			return strings.Title(strings.ToLower(matches[1]))
		}
	}

	return ""
}

// extractJapaneseName tries to extract Japanese name from various sources
func (p *ProductParser) extractJapaneseName(e *colly.HTMLElement, englishName string) string {
	// Try specific Japanese name selectors
	japName := p.tryExtractText(e, []string{
		".japanese-name",
		".name-jp",
		".product-name-jp",
	})

	if japName != "" {
		return japName
	}

	// Try to extract from parentheses in main name
	re := regexp.MustCompile(`\(([^)]+)\)`)
	if matches := re.FindStringSubmatch(englishName); len(matches) > 1 {
		potential := matches[1]
		// Simple check for Japanese characters (Hiragana, Katakana, Kanji)
		if regexp.MustCompile(`[\p{Hiragana}\p{Katakana}\p{Han}]`).MatchString(potential) {
			return potential
		}
	}

	return ""
}

// GetTotalPages extracts the total number of pages from pagination
func (p *ProductParser) GetTotalPages(doc *goquery.Document) int {
	// Try to find pagination info
	paginationSelectors := []string{
		".pagination .page:last-child",
		".pagination-info",
		".page-count",
	}

	for _, selector := range paginationSelectors {
		element := doc.Find(selector)
		if element.Length() > 0 {
			text := element.Text()
			re := regexp.MustCompile(`\d+`)
			matches := re.FindAllString(text, -1)
			if len(matches) > 0 {
				if pages, err := strconv.Atoi(matches[len(matches)-1]); err == nil {
					return pages
				}
			}
		}
	}

	return 1 // Default to 1 page if we can't determine
} 