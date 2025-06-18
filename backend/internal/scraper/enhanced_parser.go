package scraper

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/SiddhantaChandra/pokemon-card-scraper/pkg/models"
)

// PageInfo holds the results from parsing a product page
type PageInfo struct {
	Cards      []models.Card
	TotalPages int
	TotalItems int
}

// ParseProductPage parses a search results page using the specific selectors
func ParseProductPage(doc *goquery.Selection, baseURL string) (*PageInfo, error) {
	result := &PageInfo{}
	
	// Parse products using the specific selector
	doc.Find(".product-item.product-item--vertical").Each(func(i int, s *goquery.Selection) {
		card := parseProductItem(s, baseURL)
		if card != nil {
			result.Cards = append(result.Cards, *card)
		}
	})
	
	// Parse pagination info (only needed once)
	result.TotalPages = parseTotalPages(doc)
	result.TotalItems = parseTotalItems(doc)
	
	return result, nil
}

// parseTotalPages extracts total pages from pagination info
func parseTotalPages(doc *goquery.Selection) int {
	// Method 1: Look for page count text "1042ページ中1ページ目"
	pageCountText := doc.Find(".pagination__page-count").Text()
	if pageCountText != "" {
		// Extract number before "ページ中"
		re := regexp.MustCompile(`(\d+)ページ中`)
		matches := re.FindStringSubmatch(pageCountText)
		if len(matches) > 1 {
			if totalPages, err := strconv.Atoi(matches[1]); err == nil {
				return totalPages
			}
		}
	}
	
	// Method 2: Look for highest page number in pagination links
	var maxPage int
	doc.Find(".pagination__nav-item").Each(func(i int, s *goquery.Selection) {
		pageText := strings.TrimSpace(s.Text())
		if pageText != "…" { // Skip ellipsis
			if page, err := strconv.Atoi(pageText); err == nil && page > maxPage {
				maxPage = page
			}
		}
	})
	
	// Fallback: look for any pagination elements
	if maxPage == 0 {
		doc.Find(".pagination a").Each(func(i int, s *goquery.Selection) {
			pageText := strings.TrimSpace(s.Text())
			if page, err := strconv.Atoi(pageText); err == nil && page > maxPage {
				maxPage = page
			}
		})
	}
	
	return maxPage
}

// parseTotalItems extracts total item count from the page
func parseTotalItems(doc *goquery.Selection) int {
	// Look for "25001の結果のうち" or similar text
	countText := doc.Find(".collection__showing-count").Text()
	if countText == "" {
		countText = doc.Find(".collection__mobile-active-filters-results").Text()
	}
	
	if countText != "" {
		// Extract number before "の結果" or "結果"
		re := regexp.MustCompile(`(\d+).*結果`)
		matches := re.FindStringSubmatch(countText)
		if len(matches) > 1 {
			if total, err := strconv.Atoi(matches[1]); err == nil {
				return total
			}
		}
	}
	
	// Try alternative patterns
	selectors := []string{
		".collection-header__results",
		".product-count",
		".results-count",
	}
	
	for _, selector := range selectors {
		text := doc.Find(selector).Text()
		if text != "" {
			// Look for numbers in the text
			re := regexp.MustCompile(`(\d+)`)
			matches := re.FindAllString(text, -1)
			if len(matches) > 0 {
				if count, err := strconv.Atoi(matches[len(matches)-1]); err == nil {
					return count
				}
			}
		}
	}
	
	return 0
}

// parseProductItem extracts card data from a product item element
func parseProductItem(s *goquery.Selection, baseURL string) *models.Card {
	// Extract product URL using the specific selector
	linkElement := s.Find(".product-item__image-wrapper a")
	productURL, exists := linkElement.Attr("href")
	if !exists {
		// Fallback selector
		productURL, exists = s.Find("a").First().Attr("href")
		if !exists {
			return nil
		}
	}
	
	// Make absolute URL
	productURL = BuildProductURL(baseURL, productURL)
	
	// Extract card name using the specific selector
	name := strings.TrimSpace(s.Find(".product-item__title").Text())
	if name == "" {
		// Fallback selectors
		name = strings.TrimSpace(s.Find(".product-title").Text())
		if name == "" {
			name = strings.TrimSpace(s.Find("h3").Text())
		}
	}
	
	if name == "" {
		return nil // Skip if no name found
	}
	
	// Extract price using the specific selector
	priceText := s.Find(".price").Text()
	price := parsePrice(priceText)
	
	// Extract stock info using the specific selector
	stockText := s.Find(".product-item__inventory").Text()
	stock := parseStockText(stockText)
	
	// Extract image URL using the specific selector
	imgElement := s.Find(".product-item__primary-image")
	imageURL := extractImageURL(imgElement)
	
	// Generate unique ID from URL
	cardID := generateCardID(productURL)
	
	return &models.Card{
		ID:        cardID,
		Name:      name,
		Price:     price,
		Stock:     stock,
		URL:       productURL,
		ImageURL:  imageURL,
		InStock:   stock > 0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// parsePrice extracts numeric price from Japanese price text
func parsePrice(priceText string) float64 {
	if priceText == "" {
		return 0
	}
	
	// Clean the price text
	priceText = strings.ReplaceAll(priceText, "¥", "")
	priceText = strings.ReplaceAll(priceText, ",", "")
	priceText = strings.ReplaceAll(priceText, "販売価格", "")
	priceText = strings.ReplaceAll(priceText, "円", "")
	priceText = strings.TrimSpace(priceText)
	
	// Extract numeric value
	re := regexp.MustCompile(`(\d+(?:\.\d+)?)`)
	matches := re.FindStringSubmatch(priceText)
	if len(matches) > 1 {
		if price, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return price
		}
	}
	
	return 0
}

// parseStockText extracts stock quantity from Japanese stock text
func parseStockText(stockText string) int {
	if stockText == "" {
		return 0
	}
	
	stockText = strings.TrimSpace(stockText)
	
	// Pattern: "在庫 X個" or "残りX個のみ"
	re := regexp.MustCompile(`(\d+)個`)
	matches := re.FindStringSubmatch(stockText)
	
	if len(matches) > 1 {
		if stock, err := strconv.Atoi(matches[1]); err == nil {
			return stock
		}
	}
	
	// Check for common stock status indicators
	lowerText := strings.ToLower(stockText)
	if strings.Contains(lowerText, "在庫") || strings.Contains(lowerText, "available") {
		// If it mentions stock but no number, assume 1
		return 1
	}
	
	// If no stock info found, assume out of stock
	return 0
}

// extractImageURL gets the image URL from an img element
func extractImageURL(imgElement *goquery.Selection) string {
	// Try different image source attributes
	imageURL, exists := imgElement.Attr("data-srcset")
	if !exists || imageURL == "" {
		imageURL, exists = imgElement.Attr("srcset")
	}
	if !exists || imageURL == "" {
		imageURL, _ = imgElement.Attr("data-src")
	}
	if imageURL == "" {
		imageURL, _ = imgElement.Attr("src")
	}
	
	// Parse first image from srcset if present
	if imageURL != "" && strings.Contains(imageURL, " ") {
		parts := strings.Split(imageURL, " ")
		if len(parts) > 0 {
			imageURL = strings.TrimSpace(parts[0])
		}
	}
	
	// Make absolute URL if needed
	if imageURL != "" && strings.HasPrefix(imageURL, "//") {
		imageURL = "https:" + imageURL
	} else if imageURL != "" && strings.HasPrefix(imageURL, "/") {
		imageURL = "https://torecacamp-pokemon.com" + imageURL
	}
	
	return imageURL
}

// generateCardID creates a unique ID from the product URL
func generateCardID(productURL string) string {
	if productURL == "" {
		return ""
	}
	
	// Extract the last part of the URL path as ID
	parts := strings.Split(productURL, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" && parts[i] != "products" {
			return parts[i]
		}
	}
	
	// Fallback: use timestamp
	return strconv.FormatInt(time.Now().UnixNano(), 36)
} 