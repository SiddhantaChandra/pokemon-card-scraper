package scraper

import (
	"fmt"
	"net/url"
	"strings"
)

// SearchParams holds parameters for building search URLs
type SearchParams struct {
	Page        int
	MaxPages    int // Limit scraping to this many pages (0 = no limit)
	InStockOnly bool
	MinPrice    string
	MaxPrice    string
	SortBy      string
}

// BuildSearchURL creates a properly formatted search URL based on the given parameters
func BuildSearchURL(params SearchParams) string {
	baseURL := "https://torecacamp-pokemon.com/search"

	u, _ := url.Parse(baseURL)
	q := u.Query()

	// Required parameters from analysis
	q.Set("sort_by", "relevance")
	if params.SortBy != "" {
		q.Set("sort_by", params.SortBy)
	}
	q.Set("q", ".")
	q.Set("type", "product")
	q.Set("options[prefix]", "last")
	q.Set("options[unavailable_products]", "last")

	// Filters
	if params.InStockOnly {
		q.Set("filter.v.availability", "1")
	}

	if params.MinPrice != "" {
		q.Set("filter.v.price.gte", params.MinPrice)
	}

	if params.MaxPrice != "" {
		q.Set("filter.v.price.lte", params.MaxPrice)
	}

	// Page number (only add if > 1)
	if params.Page > 1 {
		q.Set("page", fmt.Sprintf("%d", params.Page))
	}

	u.RawQuery = q.Encode()
	return u.String()
}

// BuildProductURL creates an absolute URL from a relative product path
func BuildProductURL(baseURL, relativePath string) string {
	if relativePath == "" {
		return ""
	}

	// If already absolute, return as-is
	if strings.HasPrefix(relativePath, "http://") || strings.HasPrefix(relativePath, "https://") {
		return relativePath
	}

	// Ensure baseURL doesn't end with slash and relativePath doesn't start with slash
	if baseURL[len(baseURL)-1] == '/' {
		baseURL = baseURL[:len(baseURL)-1]
	}
	if len(relativePath) > 0 && relativePath[0] != '/' {
		relativePath = "/" + relativePath
	}

	return baseURL + relativePath
}

// GetBaseURL returns the base website URL
func GetBaseURL() string {
	return "https://torecacamp-pokemon.com"
}
