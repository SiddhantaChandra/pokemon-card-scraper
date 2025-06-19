package search

import (
	"sort"
	"strconv"
	"strings"

	"github.com/SiddhantaChandra/pokemon-card-scraper/pkg/models"
)

// SearchEngine provides search functionality for Pokemon cards
type SearchEngine struct {
	storage Storage
}

// Storage interface defines the methods needed by the search engine
type Storage interface {
	GetAllCards() ([]models.Card, error)
	SearchCards(filters models.FilterOptions) (*models.SearchResult, error)
}

// NewSearchEngine creates a new search engine instance
func NewSearchEngine(storage Storage) *SearchEngine {
	return &SearchEngine{
		storage: storage,
	}
}

// SearchOptions represents search parameters
type SearchOptions struct {
	Query       string
	MinPrice    *float64
	MaxPrice    *float64
	InStockOnly bool
	SetName     string
	Rarity      string
	Conditions  []models.CardCondition
	SortBy      string // price_asc, price_desc, name_asc, name_desc, date_asc, date_desc
	Page        int
	PageSize    int
}

// SearchCards performs a comprehensive search with filters and sorting
func (se *SearchEngine) SearchCards(options SearchOptions) (*models.SearchResult, error) {
	// Convert SearchOptions to FilterOptions for storage layer
	filterOpts := models.FilterOptions{
		InStockOnly: options.InStockOnly,
		Page:        options.Page,
		PageSize:    options.PageSize,
	}

	if options.MinPrice != nil {
		filterOpts.MinPrice = options.MinPrice
	}
	if options.MaxPrice != nil {
		filterOpts.MaxPrice = options.MaxPrice
	}

	// Get all cards matching basic filters from storage
	allCards, err := se.storage.GetAllCards()
	if err != nil {
		return nil, err
	}

	// Apply additional filters and full-text search
	filteredCards := se.applyFilters(allCards, options)

	// Apply full-text search if query is provided
	if options.Query != "" {
		filteredCards = se.performFullTextSearch(filteredCards, options.Query)
	}

	// Sort results
	se.sortCards(filteredCards, options.SortBy)

	// Apply pagination
	totalItems := len(filteredCards)
	totalPages := (totalItems + options.PageSize - 1) / options.PageSize

	if options.Page > totalPages {
		options.Page = totalPages
	}
	if options.Page < 1 {
		options.Page = 1
	}

	startIdx := (options.Page - 1) * options.PageSize
	endIdx := startIdx + options.PageSize
	if endIdx > totalItems {
		endIdx = totalItems
	}

	paginatedCards := make([]models.Card, 0)
	if startIdx < totalItems {
		paginatedCards = filteredCards[startIdx:endIdx]
	}

	// Convert SearchOptions to FilterOptions for the result
	resultFilters := models.FilterOptions{
		Query:       options.Query,
		MinPrice:    options.MinPrice,
		MaxPrice:    options.MaxPrice,
		InStockOnly: options.InStockOnly,
		Conditions:  options.Conditions,
		SortBy:      options.SortBy,
		Page:        options.Page,
		PageSize:    options.PageSize,
	}

	if options.SetName != "" {
		resultFilters.SetNames = []string{options.SetName}
	}
	if options.Rarity != "" {
		resultFilters.Rarities = []string{options.Rarity}
	}

	return &models.SearchResult{
		Cards:      paginatedCards,
		Total:      totalItems,
		Page:       options.Page,
		PageSize:   options.PageSize,
		TotalPages: totalPages,
		Query:      options.Query,
		Filters:    resultFilters,
	}, nil
}

// applyFilters applies all non-text search filters
func (se *SearchEngine) applyFilters(cards []models.Card, options SearchOptions) []models.Card {
	filtered := make([]models.Card, 0)

	for _, card := range cards {
		// Stock filter
		if options.InStockOnly && card.Stock <= 0 {
			continue
		}

		// Price range filter
		if options.MinPrice != nil && card.Price < *options.MinPrice {
			continue
		}
		if options.MaxPrice != nil && card.Price > *options.MaxPrice {
			continue
		}

		// Set name filter
		if options.SetName != "" && !strings.Contains(strings.ToLower(card.SetName), strings.ToLower(options.SetName)) {
			continue
		}

		// Rarity filter
		if options.Rarity != "" && !strings.Contains(strings.ToLower(card.Rarity), strings.ToLower(options.Rarity)) {
			continue
		}

		// Condition filter
		if len(options.Conditions) > 0 {
			found := false
			for _, condition := range options.Conditions {
				if card.Condition == condition {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		filtered = append(filtered, card)
	}

	return filtered
}

// performFullTextSearch performs full-text search on card fields
func (se *SearchEngine) performFullTextSearch(cards []models.Card, query string) []models.Card {
	if query == "" {
		return cards
	}

	searchTerms := strings.Fields(strings.ToLower(query))
	if len(searchTerms) == 0 {
		return cards
	}

	results := make([]models.Card, 0)

	for _, card := range cards {
		if se.matchesSearch(card, searchTerms) {
			results = append(results, card)
		}
	}

	return results
}

// matchesSearch checks if a card matches the search terms
func (se *SearchEngine) matchesSearch(card models.Card, searchTerms []string) bool {
	// Create searchable text from card fields
	cardText := strings.ToLower(strings.Join([]string{
		card.Name,
		card.NameJP,
		card.SetName,
		card.Rarity,
	}, " "))

	// Check if all search terms are found in the card text
	for _, term := range searchTerms {
		if !strings.Contains(cardText, term) {
			return false
		}
	}

	return true
}

// sortCards sorts the cards based on the specified sort option
func (se *SearchEngine) sortCards(cards []models.Card, sortBy string) {
	switch sortBy {
	case "price_asc":
		sort.Slice(cards, func(i, j int) bool {
			return cards[i].Price < cards[j].Price
		})
	case "price_desc":
		sort.Slice(cards, func(i, j int) bool {
			return cards[i].Price > cards[j].Price
		})
	case "name_asc":
		sort.Slice(cards, func(i, j int) bool {
			return strings.ToLower(cards[i].Name) < strings.ToLower(cards[j].Name)
		})
	case "name_desc":
		sort.Slice(cards, func(i, j int) bool {
			return strings.ToLower(cards[i].Name) > strings.ToLower(cards[j].Name)
		})
	case "date_asc":
		sort.Slice(cards, func(i, j int) bool {
			return cards[i].CreatedAt.Before(cards[j].CreatedAt)
		})
	case "date_desc":
		sort.Slice(cards, func(i, j int) bool {
			return cards[i].CreatedAt.After(cards[j].CreatedAt)
		})
	case "stock_desc":
		sort.Slice(cards, func(i, j int) bool {
			return cards[i].Stock > cards[j].Stock
		})
	default:
		// Default sort by date descending (newest first)
		sort.Slice(cards, func(i, j int) bool {
			return cards[i].CreatedAt.After(cards[j].CreatedAt)
		})
	}
}

// GetSuggestions provides search suggestions based on partial query
func (se *SearchEngine) GetSuggestions(partialQuery string, limit int) ([]string, error) {
	if partialQuery == "" || limit <= 0 {
		return []string{}, nil
	}

	allCards, err := se.storage.GetAllCards()
	if err != nil {
		return nil, err
	}

	suggestionMap := make(map[string]int)
	queryLower := strings.ToLower(partialQuery)

	// Collect suggestions from card names and set names
	for _, card := range allCards {
		// Check card name
		if strings.Contains(strings.ToLower(card.Name), queryLower) {
			suggestionMap[card.Name]++
		}

		// Check Japanese name
		if strings.Contains(strings.ToLower(card.NameJP), queryLower) {
			suggestionMap[card.NameJP]++
		}

		// Check set name
		if strings.Contains(strings.ToLower(card.SetName), queryLower) {
			suggestionMap[card.SetName]++
		}

		// Extract words from names for partial matches
		nameWords := strings.Fields(card.Name)
		for _, word := range nameWords {
			if len(word) > 2 && strings.Contains(strings.ToLower(word), queryLower) {
				suggestionMap[word]++
			}
		}
	}

	// Convert map to slice and sort by frequency
	type suggestion struct {
		text  string
		count int
	}

	suggestions := make([]suggestion, 0, len(suggestionMap))
	for text, count := range suggestionMap {
		suggestions = append(suggestions, suggestion{text: text, count: count})
	}

	// Sort by frequency (descending) then alphabetically
	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].count == suggestions[j].count {
			return suggestions[i].text < suggestions[j].text
		}
		return suggestions[i].count > suggestions[j].count
	})

	// Extract text and limit results
	results := make([]string, 0, limit)
	for i, s := range suggestions {
		if i >= limit {
			break
		}
		results = append(results, s.text)
	}

	return results, nil
}

// GetStats returns search statistics
func (se *SearchEngine) GetStats() (*SearchStats, error) {
	allCards, err := se.storage.GetAllCards()
	if err != nil {
		return nil, err
	}

	stats := &SearchStats{
		TotalCards:    len(allCards),
		InStockCards:  0,
		TotalSets:     make(map[string]int),
		TotalRarities: make(map[string]int),
		PriceRange: PriceRange{
			Min: 999999.0,
			Max: 0.0,
		},
	}

	for _, card := range allCards {
		// Count in-stock cards
		if card.Stock > 0 {
			stats.InStockCards++
		}

		// Count by set
		if card.SetName != "" {
			stats.TotalSets[card.SetName]++
		}

		// Count by rarity
		if card.Rarity != "" {
			stats.TotalRarities[card.Rarity]++
		}

		// Track price range
		if card.Price < stats.PriceRange.Min {
			stats.PriceRange.Min = card.Price
		}
		if card.Price > stats.PriceRange.Max {
			stats.PriceRange.Max = card.Price
		}
	}

	return stats, nil
}

// SearchStats represents search database statistics
type SearchStats struct {
	TotalCards    int            `json:"total_cards"`
	InStockCards  int            `json:"in_stock_cards"`
	TotalSets     map[string]int `json:"total_sets"`
	TotalRarities map[string]int `json:"total_rarities"`
	PriceRange    PriceRange     `json:"price_range"`
}

// PriceRange represents min and max prices
type PriceRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// ParsePriceFilter parses price filter strings
func ParsePriceFilter(minStr, maxStr string) (*float64, *float64, error) {
	var minPrice, maxPrice *float64

	if minStr != "" {
		min, err := strconv.ParseFloat(minStr, 64)
		if err != nil {
			return nil, nil, err
		}
		minPrice = &min
	}

	if maxStr != "" {
		max, err := strconv.ParseFloat(maxStr, 64)
		if err != nil {
			return nil, nil, err
		}
		maxPrice = &max
	}

	return minPrice, maxPrice, nil
}
