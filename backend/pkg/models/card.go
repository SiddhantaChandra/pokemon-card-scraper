package models

import (
	"fmt"
	"time"
)

// CardCondition represents the condition/quality of a card
type CardCondition string

const (
	ConditionPerfect CardCondition = "Perfect"
	ConditionAPlus   CardCondition = "A+"
	ConditionA       CardCondition = "A"
	ConditionAMinus  CardCondition = "A-"
	ConditionBPlus   CardCondition = "B+"
	ConditionB       CardCondition = "B"
	ConditionBMinus  CardCondition = "B-"
	ConditionC       CardCondition = "C"
	ConditionD       CardCondition = "D"
)

// Card represents a Pokemon card with all its metadata
type Card struct {
	ID        string        `json:"id" db:"id"`
	Name      string        `json:"name" db:"name"`
	NameJP    string        `json:"name_jp" db:"name_jp"`
	Price     float64       `json:"price" db:"price"`
	Stock     int           `json:"stock" db:"stock"`
	ImageURL  string        `json:"image_url" db:"image_url"`
	SetName   string        `json:"set_name" db:"set_name"`
	Rarity    string        `json:"rarity" db:"rarity"`
	URL       string        `json:"url" db:"url"`
	Condition CardCondition `json:"condition" db:"condition"`
	// Additional fields for internal tracking
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	InStock   bool      `json:"in_stock" db:"in_stock"`
}

// SearchResult represents the result of a search operation
type SearchResult struct {
	Cards      []Card        `json:"cards"`
	Total      int           `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	TotalPages int           `json:"total_pages"`
	Query      string        `json:"query"`
	Filters    FilterOptions `json:"filters"`
}

// FilterOptions represents search and filter parameters
type FilterOptions struct {
	Query       string          `json:"query" form:"q"`
	MinPrice    *float64        `json:"min_price" form:"min_price"`
	MaxPrice    *float64        `json:"max_price" form:"max_price"`
	InStockOnly bool            `json:"in_stock_only" form:"in_stock_only"`
	SetNames    []string        `json:"set_names" form:"set_names"`
	Rarities    []string        `json:"rarities" form:"rarities"`
	Conditions  []CardCondition `json:"conditions" form:"conditions"`
	SortBy      string          `json:"sort_by" form:"sort_by"`       // name, price, date_added, stock
	SortOrder   string          `json:"sort_order" form:"sort_order"` // asc, desc
	Page        int             `json:"page" form:"page"`
	PageSize    int             `json:"page_size" form:"page_size"`
}

// DefaultFilterOptions returns filter options with sensible defaults
func DefaultFilterOptions() FilterOptions {
	return FilterOptions{
		InStockOnly: true,
		SortBy:      "name",
		SortOrder:   "asc",
		Page:        1,
		PageSize:    50,
	}
}

// IsInStock returns true if the card is available
func (c *Card) IsInStock() bool {
	return c.Stock > 0
}

// FormattedPrice returns the price formatted as a string with currency
func (c *Card) FormattedPrice() string {
	return fmt.Sprintf("¥%.2f", c.Price)
}
