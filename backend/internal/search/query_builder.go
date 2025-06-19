package search

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/SiddhantaChandra/pokemon-card-scraper/pkg/models"
)

// QueryBuilder helps build search queries and convert between different query formats
type QueryBuilder struct{}

// NewQueryBuilder creates a new query builder instance
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{}
}

// BuildFilterOptions converts HTTP query parameters to FilterOptions
func (qb *QueryBuilder) BuildFilterOptions(values url.Values) (models.FilterOptions, error) {
	filterOpts := models.FilterOptions{
		Page:     1,
		PageSize: 20,
	}

	// Parse page
	if pageStr := values.Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			filterOpts.Page = page
		}
	}

	// Parse page size
	if pageSizeStr := values.Get("page_size"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 && pageSize <= 100 {
			filterOpts.PageSize = pageSize
		}
	}

	// Parse price range
	if minPriceStr := values.Get("min_price"); minPriceStr != "" {
		if minPrice, err := strconv.ParseFloat(minPriceStr, 64); err == nil && minPrice >= 0 {
			filterOpts.MinPrice = &minPrice
		}
	}

	if maxPriceStr := values.Get("max_price"); maxPriceStr != "" {
		if maxPrice, err := strconv.ParseFloat(maxPriceStr, 64); err == nil && maxPrice >= 0 {
			filterOpts.MaxPrice = &maxPrice
		}
	}

	// Parse in stock filter
	if values.Get("in_stock") == "true" || values.Get("in_stock_only") == "true" {
		filterOpts.InStockOnly = true
	}

	return filterOpts, nil
}

// BuildSearchOptions converts HTTP query parameters to SearchOptions
func (qb *QueryBuilder) BuildSearchOptions(values url.Values) (SearchOptions, error) {
	options := SearchOptions{
		Page:     1,
		PageSize: 20,
	}

	// Parse query string
	options.Query = strings.TrimSpace(values.Get("q"))

	// Parse page
	if pageStr := values.Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			options.Page = page
		}
	}

	// Parse page size
	if pageSizeStr := values.Get("page_size"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 && pageSize <= 100 {
			options.PageSize = pageSize
		}
	}

	// Parse price range
	if minPriceStr := values.Get("min_price"); minPriceStr != "" {
		if minPrice, err := strconv.ParseFloat(minPriceStr, 64); err == nil && minPrice >= 0 {
			options.MinPrice = &minPrice
		}
	}

	if maxPriceStr := values.Get("max_price"); maxPriceStr != "" {
		if maxPrice, err := strconv.ParseFloat(maxPriceStr, 64); err == nil && maxPrice >= 0 {
			options.MaxPrice = &maxPrice
		}
	}

	// Parse in stock filter
	if values.Get("in_stock") == "true" || values.Get("in_stock_only") == "true" {
		options.InStockOnly = true
	}

	// Parse set name filter
	options.SetName = strings.TrimSpace(values.Get("set"))

	// Parse rarity filter
	options.Rarity = strings.TrimSpace(values.Get("rarity"))

	// Parse conditions filter
	if conditionsStr := values.Get("conditions"); conditionsStr != "" {
		conditions := strings.Split(conditionsStr, ",")
		for _, c := range conditions {
			if c != "" {
				options.Conditions = append(options.Conditions, models.CardCondition(c))
			}
		}
	}

	// Parse sort option
	sortBy := strings.TrimSpace(values.Get("sort"))
	if qb.isValidSortOption(sortBy) {
		options.SortBy = sortBy
	}

	return options, nil
}

// isValidSortOption checks if the sort option is valid
func (qb *QueryBuilder) isValidSortOption(sortBy string) bool {
	validSorts := []string{
		"price_asc", "price_desc",
		"name_asc", "name_desc",
		"date_asc", "date_desc",
		"stock_desc",
	}

	for _, valid := range validSorts {
		if sortBy == valid {
			return true
		}
	}
	return false
}

// BuildQueryURL creates a URL with query parameters from SearchOptions
func (qb *QueryBuilder) BuildQueryURL(baseURL string, options SearchOptions) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}

	values := url.Values{}

	// Add query string
	if options.Query != "" {
		values.Set("q", options.Query)
	}

	// Add pagination
	if options.Page > 1 {
		values.Set("page", strconv.Itoa(options.Page))
	}
	if options.PageSize != 20 {
		values.Set("page_size", strconv.Itoa(options.PageSize))
	}

	// Add price filters
	if options.MinPrice != nil {
		values.Set("min_price", fmt.Sprintf("%.2f", *options.MinPrice))
	}
	if options.MaxPrice != nil {
		values.Set("max_price", fmt.Sprintf("%.2f", *options.MaxPrice))
	}

	// Add stock filter
	if options.InStockOnly {
		values.Set("in_stock", "true")
	}

	// Add set filter
	if options.SetName != "" {
		values.Set("set", options.SetName)
	}

	// Add rarity filter
	if options.Rarity != "" {
		values.Set("rarity", options.Rarity)
	}

	// Add conditions filter
	if len(options.Conditions) > 0 {
		conditionStrs := make([]string, len(options.Conditions))
		for i, condition := range options.Conditions {
			conditionStrs[i] = string(condition)
		}
		values.Set("conditions", strings.Join(conditionStrs, ","))
	}

	// Add sort option
	if options.SortBy != "" {
		values.Set("sort", options.SortBy)
	}

	u.RawQuery = values.Encode()
	return u.String()
}

// ParseQuery parses a raw query string into structured search terms
func (qb *QueryBuilder) ParseQuery(rawQuery string) *ParsedQuery {
	parsed := &ParsedQuery{
		Terms:     make([]string, 0),
		Phrases:   make([]string, 0),
		Modifiers: make(map[string]string),
	}

	if rawQuery == "" {
		return parsed
	}

	// Split by spaces but preserve quoted phrases
	terms := qb.splitPreservingQuotes(rawQuery)

	for _, term := range terms {
		term = strings.TrimSpace(term)
		if term == "" {
			continue
		}

		// Check for modifiers (e.g., "set:base" or "price:>100")
		if strings.Contains(term, ":") {
			parts := strings.SplitN(term, ":", 2)
			if len(parts) == 2 {
				key := strings.ToLower(strings.TrimSpace(parts[0]))
				value := strings.TrimSpace(parts[1])
				parsed.Modifiers[key] = value
				continue
			}
		}

		// Check if it's a quoted phrase
		if strings.HasPrefix(term, "\"") && strings.HasSuffix(term, "\"") && len(term) > 2 {
			phrase := term[1 : len(term)-1]
			parsed.Phrases = append(parsed.Phrases, phrase)
		} else {
			parsed.Terms = append(parsed.Terms, term)
		}
	}

	return parsed
}

// splitPreservingQuotes splits a string by spaces while preserving quoted phrases
func (qb *QueryBuilder) splitPreservingQuotes(s string) []string {
	var result []string
	var current strings.Builder
	inQuotes := false

	for _, r := range s {
		switch r {
		case '"':
			inQuotes = !inQuotes
			current.WriteRune(r)
		case ' ':
			if inQuotes {
				current.WriteRune(r)
			} else {
				if current.Len() > 0 {
					result = append(result, current.String())
					current.Reset()
				}
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// ApplyQueryModifiers applies parsed query modifiers to SearchOptions
func (qb *QueryBuilder) ApplyQueryModifiers(options *SearchOptions, parsed *ParsedQuery) error {
	for key, value := range parsed.Modifiers {
		switch key {
		case "set":
			options.SetName = value
		case "rarity":
			options.Rarity = value
		case "price":
			if err := qb.parsePriceModifier(value, options); err != nil {
				return fmt.Errorf("invalid price modifier '%s': %v", value, err)
			}
		case "sort":
			if qb.isValidSortOption(value) {
				options.SortBy = value
			}
		case "stock":
			if value == "available" || value == "true" || value == "yes" {
				options.InStockOnly = true
			}
		}
	}

	// Combine regular terms and phrases for the main query
	allTerms := append(parsed.Terms, parsed.Phrases...)
	if len(allTerms) > 0 {
		options.Query = strings.Join(allTerms, " ")
	}

	return nil
}

// parsePriceModifier parses price modifiers like ">100", "<500", "100-500"
func (qb *QueryBuilder) parsePriceModifier(value string, options *SearchOptions) error {
	value = strings.TrimSpace(value)

	// Check for range (e.g., "100-500")
	if strings.Contains(value, "-") {
		parts := strings.Split(value, "-")
		if len(parts) == 2 {
			minStr := strings.TrimSpace(parts[0])
			maxStr := strings.TrimSpace(parts[1])

			if minStr != "" {
				minPrice, err := strconv.ParseFloat(minStr, 64)
				if err != nil {
					return err
				}
				options.MinPrice = &minPrice
			}

			if maxStr != "" {
				maxPrice, err := strconv.ParseFloat(maxStr, 64)
				if err != nil {
					return err
				}
				options.MaxPrice = &maxPrice
			}

			return nil
		}
	}

	// Check for comparison operators
	if strings.HasPrefix(value, ">") {
		priceStr := strings.TrimSpace(value[1:])
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			return err
		}
		options.MinPrice = &price
		return nil
	}

	if strings.HasPrefix(value, "<") {
		priceStr := strings.TrimSpace(value[1:])
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			return err
		}
		options.MaxPrice = &price
		return nil
	}

	// Default: exact price match (convert to small range)
	price, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return err
	}
	options.MinPrice = &price
	options.MaxPrice = &price

	return nil
}

// ParsedQuery represents a parsed search query
type ParsedQuery struct {
	Terms     []string          `json:"terms"`
	Phrases   []string          `json:"phrases"`
	Modifiers map[string]string `json:"modifiers"`
}

// GetSortOptions returns available sort options with descriptions
func (qb *QueryBuilder) GetSortOptions() []SortOption {
	return []SortOption{
		{Value: "date_desc", Label: "Newest First", Description: "Sort by date added (newest first)"},
		{Value: "date_asc", Label: "Oldest First", Description: "Sort by date added (oldest first)"},
		{Value: "price_asc", Label: "Price: Low to High", Description: "Sort by price (ascending)"},
		{Value: "price_desc", Label: "Price: High to Low", Description: "Sort by price (descending)"},
		{Value: "name_asc", Label: "Name: A to Z", Description: "Sort by name (ascending)"},
		{Value: "name_desc", Label: "Name: Z to A", Description: "Sort by name (descending)"},
		{Value: "stock_desc", Label: "Most in Stock", Description: "Sort by stock quantity (descending)"},
	}
}

// SortOption represents a sorting option
type SortOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// ValidateSearchOptions validates SearchOptions and returns any errors
func (qb *QueryBuilder) ValidateSearchOptions(options SearchOptions) error {
	// Validate page
	if options.Page < 1 {
		return fmt.Errorf("page must be >= 1")
	}

	// Validate page size
	if options.PageSize < 1 || options.PageSize > 100 {
		return fmt.Errorf("page_size must be between 1 and 100")
	}

	// Validate price range
	if options.MinPrice != nil && *options.MinPrice < 0 {
		return fmt.Errorf("min_price must be >= 0")
	}

	if options.MaxPrice != nil && *options.MaxPrice < 0 {
		return fmt.Errorf("max_price must be >= 0")
	}

	if options.MinPrice != nil && options.MaxPrice != nil && *options.MinPrice > *options.MaxPrice {
		return fmt.Errorf("min_price must be <= max_price")
	}

	// Validate sort option
	if options.SortBy != "" && !qb.isValidSortOption(options.SortBy) {
		return fmt.Errorf("invalid sort option: %s", options.SortBy)
	}

	return nil
}
