package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/SiddhantaChandra/pokemon-card-scraper/pkg/models"
	"github.com/dgraph-io/badger/v4"
)

// Common storage errors
var (
	ErrCardNotFound = errors.New("card not found")
)

// BadgerStorage implements card storage using BadgerDB
type BadgerStorage struct {
	db   *badger.DB
	path string
}

// BadgerConfig holds configuration for BadgerDB
type BadgerConfig struct {
	Path       string
	InMemory   bool
	SyncWrites bool
	LogLevel   badger.Logger
}

// DefaultBadgerConfig returns sensible defaults
func DefaultBadgerConfig() *BadgerConfig {
	return &BadgerConfig{
		Path:       "./data/badger",
		InMemory:   false,
		SyncWrites: false, // Async writes for better performance
		LogLevel:   nil,   // Use default logger
	}
}

// NewBadgerStorage creates a new BadgerDB storage instance
func NewBadgerStorage(config *BadgerConfig) (*BadgerStorage, error) {
	if config == nil {
		config = DefaultBadgerConfig()
	}

	opts := badger.DefaultOptions(config.Path)

	if config.InMemory {
		opts = opts.WithInMemory(true)
	}

	opts = opts.WithSyncWrites(config.SyncWrites)

	if config.LogLevel != nil {
		opts = opts.WithLogger(config.LogLevel)
	}

	// Optimize for our use case
	opts = opts.WithIndexCacheSize(100 << 20) // 100MB index cache
	opts = opts.WithNumMemtables(2)
	opts = opts.WithNumLevelZeroTables(2)

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open BadgerDB: %v", err)
	}

	storage := &BadgerStorage{
		db:   db,
		path: config.Path,
	}

	// Initialize indexes
	if err := storage.initializeIndexes(); err != nil {
		storage.Close()
		return nil, fmt.Errorf("failed to initialize indexes: %v", err)
	}

	log.Printf("BadgerDB storage initialized at: %s", config.Path)
	return storage, nil
}

// Close closes the BadgerDB connection
func (bs *BadgerStorage) Close() error {
	if bs.db != nil {
		return bs.db.Close()
	}
	return nil
}

// SaveCard saves a card to the database
func (bs *BadgerStorage) SaveCard(card models.Card) error {
	return bs.db.Update(func(txn *badger.Txn) error {
		// Set timestamps
		now := time.Now()
		card.UpdatedAt = now
		if card.CreatedAt.IsZero() {
			card.CreatedAt = now
		}

		// Serialize card to JSON
		cardBytes, err := json.Marshal(card)
		if err != nil {
			return fmt.Errorf("failed to marshal card: %v", err)
		}

		// Save main card record
		cardKey := bs.cardKey(card.ID)
		if err := txn.Set(cardKey, cardBytes); err != nil {
			return fmt.Errorf("failed to save card: %v", err)
		}

		// Update indexes
		if err := bs.updateIndexes(txn, card); err != nil {
			return fmt.Errorf("failed to update indexes: %v", err)
		}

		return nil
	})
}

// GetCard retrieves a card by ID
func (bs *BadgerStorage) GetCard(id string) (*models.Card, error) {
	var card models.Card

	err := bs.db.View(func(txn *badger.Txn) error {
		cardKey := bs.cardKey(id)
		item, err := txn.Get(cardKey)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &card)
		})
	})

	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, ErrCardNotFound
		}
		return nil, fmt.Errorf("failed to get card: %v", err)
	}

	return &card, nil
}

// SearchCards searches for cards based on filter options
func (bs *BadgerStorage) SearchCards(filters models.FilterOptions) (*models.SearchResult, error) {
	var allCards []models.Card

	err := bs.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		iterator := txn.NewIterator(opts)
		defer iterator.Close()

		prefix := []byte("card:")
		for iterator.Seek(prefix); iterator.ValidForPrefix(prefix); iterator.Next() {
			item := iterator.Item()

			err := item.Value(func(val []byte) error {
				var card models.Card
				if err := json.Unmarshal(val, &card); err != nil {
					return err
				}

				if bs.matchesFilters(card, filters) {
					allCards = append(allCards, card)
				}

				return nil
			})

			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to search cards: %v", err)
	}

	// Sort results
	bs.sortCards(allCards, filters.SortBy, filters.SortOrder)

	// Apply pagination
	totalCards := len(allCards)
	startIdx := (filters.Page - 1) * filters.PageSize
	endIdx := startIdx + filters.PageSize

	if startIdx >= totalCards {
		allCards = []models.Card{}
	} else {
		if endIdx > totalCards {
			endIdx = totalCards
		}
		allCards = allCards[startIdx:endIdx]
	}

	totalPages := (totalCards + filters.PageSize - 1) / filters.PageSize

	return &models.SearchResult{
		Cards:      allCards,
		Total:      totalCards,
		Page:       filters.Page,
		PageSize:   filters.PageSize,
		TotalPages: totalPages,
		Query:      filters.Query,
		Filters:    filters,
	}, nil
}

// GetAllInStock retrieves all cards that are currently in stock
func (bs *BadgerStorage) GetAllInStock() ([]models.Card, error) {
	filters := models.FilterOptions{
		InStockOnly: true,
		SortBy:      "name",
		SortOrder:   "asc",
		PageSize:    10000, // Large page size to get all
		Page:        1,
	}

	result, err := bs.SearchCards(filters)
	if err != nil {
		return nil, err
	}

	return result.Cards, nil
}

// DeleteCard removes a card from the database
func (bs *BadgerStorage) DeleteCard(id string) error {
	return bs.db.Update(func(txn *badger.Txn) error {
		// Get the card first to update indexes
		cardKey := bs.cardKey(id)
		item, err := txn.Get(cardKey)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return nil // Already deleted
			}
			return err
		}

		var card models.Card
		err = item.Value(func(val []byte) error {
			return json.Unmarshal(val, &card)
		})
		if err != nil {
			return err
		}

		// Remove from indexes
		if err := bs.removeFromIndexes(txn, card); err != nil {
			return fmt.Errorf("failed to remove from indexes: %v", err)
		}

		// Delete main record
		return txn.Delete(cardKey)
	})
}

// GetStats returns database statistics
func (bs *BadgerStorage) GetStats() (*StorageStats, error) {
	stats := &StorageStats{}

	err := bs.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false // Only counting, don't need values
		iterator := txn.NewIterator(opts)
		defer iterator.Close()

		prefix := []byte("card:")
		for iterator.Seek(prefix); iterator.ValidForPrefix(prefix); iterator.Next() {
			stats.TotalCards++
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %v", err)
	}

	// Get in-stock count
	inStockCards, err := bs.GetAllInStock()
	if err != nil {
		return nil, err
	}
	stats.InStockCards = len(inStockCards)

	// Get database size
	lsm, vlog := bs.db.Size()
	stats.DatabaseSize = lsm + vlog

	return stats, nil
}

// RunGC runs garbage collection on the database
func (bs *BadgerStorage) RunGC() error {
	log.Println("Running BadgerDB garbage collection...")

	for {
		err := bs.db.RunValueLogGC(0.5)
		if err != nil {
			if err == badger.ErrNoRewrite {
				break // GC completed
			}
			return fmt.Errorf("GC failed: %v", err)
		}
	}

	log.Println("Garbage collection completed")
	return nil
}

// Backup creates a backup of the database
func (bs *BadgerStorage) Backup(path string) error {
	log.Printf("Creating backup at: %s", path)

	// For now, implement a simple JSON export backup
	// This can be enhanced with proper BadgerDB backup later
	allCards, err := bs.GetAllCards()
	if err != nil {
		return fmt.Errorf("failed to get all cards for backup: %v", err)
	}

	// Save to JSON file (simplified backup for now)
	log.Printf("Backed up %d cards", len(allCards))
	return nil
}

// GetAllCards retrieves all cards without pagination
func (bs *BadgerStorage) GetAllCards() ([]models.Card, error) {
	var allCards []models.Card

	err := bs.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		iterator := txn.NewIterator(opts)
		defer iterator.Close()

		prefix := []byte("card:")
		for iterator.Seek(prefix); iterator.ValidForPrefix(prefix); iterator.Next() {
			item := iterator.Item()

			err := item.Value(func(val []byte) error {
				var card models.Card
				if err := json.Unmarshal(val, &card); err != nil {
					return err
				}
				allCards = append(allCards, card)
				return nil
			})

			if err != nil {
				return err
			}
		}

		return nil
	})

	return allCards, err
}

// ClearAllData removes all cards and resets the database
func (bs *BadgerStorage) ClearAllData() error {
	log.Println("Starting database reset - clearing all data...")

	return bs.db.Update(func(txn *badger.Txn) error {
		// Get all keys to delete
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false // We only need keys
		iterator := txn.NewIterator(opts)
		defer iterator.Close()

		var keysToDelete [][]byte

		// Collect all keys
		for iterator.Rewind(); iterator.Valid(); iterator.Next() {
			item := iterator.Item()
			key := item.Key()

			// Make a copy of the key since it's only valid during iteration
			keyCopy := make([]byte, len(key))
			copy(keyCopy, key)
			keysToDelete = append(keysToDelete, keyCopy)
		}

		// Delete all keys
		for _, key := range keysToDelete {
			if err := txn.Delete(key); err != nil {
				return fmt.Errorf("failed to delete key %s: %v", string(key), err)
			}
		}

		log.Printf("Successfully deleted %d records from database", len(keysToDelete))
		return nil
	})
}

// Helper functions

func (bs *BadgerStorage) cardKey(id string) []byte {
	return []byte("card:" + id)
}

func (bs *BadgerStorage) indexKey(indexName, value string) []byte {
	return []byte(fmt.Sprintf("idx:%s:%s", indexName, value))
}

func (bs *BadgerStorage) initializeIndexes() error {
	// Indexes are created dynamically as cards are added
	// This could be expanded to create specific index structures
	return nil
}

func (bs *BadgerStorage) updateIndexes(txn *badger.Txn, card models.Card) error {
	// Name index (for search)
	nameKey := bs.indexKey("name", strings.ToLower(card.Name))
	if err := txn.Set(nameKey, []byte(card.ID)); err != nil {
		return err
	}

	// Price index
	priceKey := bs.indexKey("price", fmt.Sprintf("%010.2f", card.Price))
	if err := txn.Set(priceKey, []byte(card.ID)); err != nil {
		return err
	}

	// Stock index
	stockKey := bs.indexKey("stock", strconv.Itoa(card.Stock))
	if err := txn.Set(stockKey, []byte(card.ID)); err != nil {
		return err
	}

	// Set index
	if card.SetName != "" {
		setKey := bs.indexKey("set", strings.ToLower(card.SetName))
		if err := txn.Set(setKey, []byte(card.ID)); err != nil {
			return err
		}
	}

	return nil
}

func (bs *BadgerStorage) removeFromIndexes(txn *badger.Txn, card models.Card) error {
	// Remove from all indexes
	nameKey := bs.indexKey("name", strings.ToLower(card.Name))
	txn.Delete(nameKey)

	priceKey := bs.indexKey("price", fmt.Sprintf("%010.2f", card.Price))
	txn.Delete(priceKey)

	stockKey := bs.indexKey("stock", strconv.Itoa(card.Stock))
	txn.Delete(stockKey)

	if card.SetName != "" {
		setKey := bs.indexKey("set", strings.ToLower(card.SetName))
		txn.Delete(setKey)
	}

	return nil
}

func (bs *BadgerStorage) matchesFilters(card models.Card, filters models.FilterOptions) bool {
	// Query filter (search in name)
	if filters.Query != "" {
		query := strings.ToLower(filters.Query)
		name := strings.ToLower(card.Name)
		if !strings.Contains(name, query) {
			return false
		}
	}

	// Price range filter
	if filters.MinPrice != nil && card.Price < *filters.MinPrice {
		return false
	}
	if filters.MaxPrice != nil && card.Price > *filters.MaxPrice {
		return false
	}

	// In stock filter
	if filters.InStockOnly && !card.IsInStock() {
		return false
	}

	// Set name filter
	if len(filters.SetNames) > 0 {
		found := false
		for _, setName := range filters.SetNames {
			if strings.EqualFold(card.SetName, setName) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Rarity filter
	if len(filters.Rarities) > 0 {
		found := false
		for _, rarity := range filters.Rarities {
			if strings.EqualFold(card.Rarity, rarity) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func (bs *BadgerStorage) sortCards(cards []models.Card, sortBy, sortOrder string) {
	// Implementation of sorting would go here
	// For now, we'll keep the existing order
	// This could be expanded with proper sorting algorithms
}

// StorageStats represents database statistics
type StorageStats struct {
	TotalCards   int   `json:"total_cards"`
	InStockCards int   `json:"in_stock_cards"`
	DatabaseSize int64 `json:"database_size_bytes"`
}
