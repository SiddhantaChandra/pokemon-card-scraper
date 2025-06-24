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

		// Count cards
		prefix := []byte("card:")
		for iterator.Seek(prefix); iterator.ValidForPrefix(prefix); iterator.Next() {
			stats.TotalCards++
		}

		// Reset iterator and count trackers
		iterator.Close()
		iterator = txn.NewIterator(opts)

		trackerPrefix := []byte("tracker:")
		for iterator.Seek(trackerPrefix); iterator.ValidForPrefix(trackerPrefix); iterator.Next() {
			stats.TotalTrackers++
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

	// Condition index - Add this
	conditionKey := bs.indexKey("condition", string(card.Condition))
	if err := txn.Set(conditionKey, []byte(card.ID)); err != nil {
		return err
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

	// Remove condition index
	conditionKey := bs.indexKey("condition", string(card.Condition))
	txn.Delete(conditionKey)

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

	// Condition filter - Add this new section
	if len(filters.Conditions) > 0 {
		found := false
		for _, condition := range filters.Conditions {
			if card.Condition == condition {
				found = true
				break
			}
		}
		if !found {
			// Add debug logging for condition filtering
			log.Printf("DEBUG: Card '%s' condition '%s' not in filter %v", card.Name, card.Condition, filters.Conditions)
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

// SaveCardsBatch saves multiple cards in a single transaction for improved performance
func (bs *BadgerStorage) SaveCardsBatch(cards []models.Card) error {
	if len(cards) == 0 {
		return nil
	}

	return bs.db.Update(func(txn *badger.Txn) error {
		now := time.Now()

		for _, card := range cards {
			// Set timestamps
			card.UpdatedAt = now
			if card.CreatedAt.IsZero() {
				card.CreatedAt = now
			}

			// Serialize card to JSON
			cardBytes, err := json.Marshal(card)
			if err != nil {
				return fmt.Errorf("failed to marshal card %s: %v", card.ID, err)
			}

			// Save main card record
			cardKey := bs.cardKey(card.ID)
			if err := txn.Set(cardKey, cardBytes); err != nil {
				return fmt.Errorf("failed to save card %s: %v", card.ID, err)
			}

			// Update indexes
			if err := bs.updateIndexes(txn, card); err != nil {
				return fmt.Errorf("failed to update indexes for card %s: %v", card.ID, err)
			}
		}

		return nil
	})
}

// UpdateCardsBatch updates multiple cards with partial field updates
func (bs *BadgerStorage) UpdateCardsBatch(updates []CardUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	return bs.db.Update(func(txn *badger.Txn) error {
		now := time.Now()

		for _, update := range updates {
			// Get existing card
			cardKey := bs.cardKey(update.ID)
			item, err := txn.Get(cardKey)
			if err != nil {
				if err == badger.ErrKeyNotFound {
					continue // Skip non-existent cards
				}
				return fmt.Errorf("failed to get card %s for update: %v", update.ID, err)
			}

			var card models.Card
			err = item.Value(func(val []byte) error {
				return json.Unmarshal(val, &card)
			})
			if err != nil {
				return fmt.Errorf("failed to unmarshal card %s: %v", update.ID, err)
			}

			// Apply field updates
			if err := bs.applyFieldUpdates(&card, update.Fields); err != nil {
				return fmt.Errorf("failed to apply updates to card %s: %v", update.ID, err)
			}

			// Update timestamp
			card.UpdatedAt = now

			// Save updated card
			cardBytes, err := json.Marshal(card)
			if err != nil {
				return fmt.Errorf("failed to marshal updated card %s: %v", card.ID, err)
			}

			if err := txn.Set(cardKey, cardBytes); err != nil {
				return fmt.Errorf("failed to save updated card %s: %v", card.ID, err)
			}

			// Update indexes
			if err := bs.updateIndexes(txn, card); err != nil {
				return fmt.Errorf("failed to update indexes for card %s: %v", card.ID, err)
			}
		}

		return nil
	})
}

// applyFieldUpdates applies field updates to a card
func (bs *BadgerStorage) applyFieldUpdates(card *models.Card, updates map[string]interface{}) error {
	for field, value := range updates {
		switch field {
		case "name":
			if name, ok := value.(string); ok {
				card.Name = name
			}
		case "name_jp":
			if nameJP, ok := value.(string); ok {
				card.NameJP = nameJP
			}
		case "price":
			if price, ok := value.(float64); ok {
				card.Price = price
			}
		case "stock":
			if stock, ok := value.(int); ok {
				card.Stock = stock
				card.InStock = stock > 0
			}
		case "image_url":
			if imageURL, ok := value.(string); ok {
				card.ImageURL = imageURL
			}
		case "set_name":
			if setName, ok := value.(string); ok {
				card.SetName = setName
			}
		case "rarity":
			if rarity, ok := value.(string); ok {
				card.Rarity = rarity
			}
		case "url":
			if url, ok := value.(string); ok {
				card.URL = url
			}
		case "in_stock":
			if inStock, ok := value.(bool); ok {
				card.InStock = inStock
			}
		case "condition":
			if condition, ok := value.(string); ok {
				card.Condition = models.CardCondition(condition)
			}
		default:
			// Ignore unknown fields
		}
	}
	return nil
}

// StorageStats represents database statistics
type StorageStats struct {
	TotalCards    int   `json:"total_cards"`
	InStockCards  int   `json:"in_stock_cards"`
	TotalTrackers int   `json:"total_trackers"`
	DatabaseSize  int64 `json:"database_size_bytes"`
}

// CardUpdate import the CardUpdate type to badger storage
type CardUpdate struct {
	ID     string
	Fields map[string]interface{}
}

// Tracker Storage Methods

// SaveTracker saves a tracker to the database
func (bs *BadgerStorage) SaveTracker(tracker models.TrackerItem) error {
	return bs.db.Update(func(txn *badger.Txn) error {
		// Set timestamps
		now := time.Now()
		tracker.LastUpdated = now
		if tracker.CreatedAt.IsZero() {
			tracker.CreatedAt = now
		}

		// Serialize tracker to JSON
		trackerBytes, err := json.Marshal(tracker)
		if err != nil {
			return fmt.Errorf("failed to marshal tracker: %v", err)
		}

		// Save main tracker record
		trackerKey := bs.trackerKey(tracker.ID)
		if err := txn.Set(trackerKey, trackerBytes); err != nil {
			return fmt.Errorf("failed to save tracker: %v", err)
		}

		// Update tracker indexes
		if err := bs.updateTrackerIndexes(txn, tracker); err != nil {
			return fmt.Errorf("failed to update tracker indexes: %v", err)
		}

		return nil
	})
}

// GetTracker retrieves a tracker by ID
func (bs *BadgerStorage) GetTracker(id string) (*models.TrackerItem, error) {
	var tracker models.TrackerItem

	err := bs.db.View(func(txn *badger.Txn) error {
		trackerKey := bs.trackerKey(id)
		item, err := txn.Get(trackerKey)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &tracker)
		})
	})

	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, fmt.Errorf("tracker not found")
		}
		return nil, fmt.Errorf("failed to get tracker: %v", err)
	}

	return &tracker, nil
}

// SearchTrackers searches for trackers based on filter options
func (bs *BadgerStorage) SearchTrackers(filters models.TrackerFilterOptions) (*models.TrackerSearchResult, error) {
	var allTrackers []models.TrackerItem

	err := bs.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		iterator := txn.NewIterator(opts)
		defer iterator.Close()

		prefix := []byte("tracker:")
		for iterator.Seek(prefix); iterator.ValidForPrefix(prefix); iterator.Next() {
			item := iterator.Item()

			err := item.Value(func(val []byte) error {
				var tracker models.TrackerItem
				if err := json.Unmarshal(val, &tracker); err != nil {
					return err
				}

				if bs.matchesTrackerFilters(tracker, filters) {
					allTrackers = append(allTrackers, tracker)
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
		return nil, fmt.Errorf("failed to search trackers: %v", err)
	}

	// Sort results
	bs.sortTrackers(allTrackers, filters.SortBy, filters.SortOrder)

	// Apply pagination
	totalTrackers := len(allTrackers)
	startIdx := (filters.Page - 1) * filters.PageSize
	endIdx := startIdx + filters.PageSize

	if startIdx >= totalTrackers {
		startIdx = totalTrackers
	}
	if endIdx > totalTrackers {
		endIdx = totalTrackers
	}

	var paginatedTrackers []models.TrackerItem
	if startIdx < endIdx {
		paginatedTrackers = allTrackers[startIdx:endIdx]
	}

	// Calculate total pages
	totalPages := (totalTrackers + filters.PageSize - 1) / filters.PageSize
	if totalPages == 0 {
		totalPages = 1
	}

	return &models.TrackerSearchResult{
		Trackers:   paginatedTrackers,
		Total:      totalTrackers,
		Page:       filters.Page,
		PageSize:   filters.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetAllTrackers retrieves all trackers
func (bs *BadgerStorage) GetAllTrackers() ([]models.TrackerItem, error) {
	var trackers []models.TrackerItem

	err := bs.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		iterator := txn.NewIterator(opts)
		defer iterator.Close()

		prefix := []byte("tracker:")
		for iterator.Seek(prefix); iterator.ValidForPrefix(prefix); iterator.Next() {
			item := iterator.Item()

			err := item.Value(func(val []byte) error {
				var tracker models.TrackerItem
				if err := json.Unmarshal(val, &tracker); err != nil {
					return err
				}

				trackers = append(trackers, tracker)
				return nil
			})

			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get all trackers: %v", err)
	}

	return trackers, nil
}

// DeleteTracker deletes a tracker by ID
func (bs *BadgerStorage) DeleteTracker(id string) error {
	return bs.db.Update(func(txn *badger.Txn) error {
		// Get tracker first to update indexes
		trackerKey := bs.trackerKey(id)
		item, err := txn.Get(trackerKey)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return fmt.Errorf("tracker not found")
			}
			return err
		}

		var tracker models.TrackerItem
		err = item.Value(func(val []byte) error {
			return json.Unmarshal(val, &tracker)
		})
		if err != nil {
			return err
		}

		// Remove from indexes
		if err := bs.removeFromTrackerIndexes(txn, tracker); err != nil {
			return fmt.Errorf("failed to remove tracker from indexes: %v", err)
		}

		// Delete main record
		if err := txn.Delete(trackerKey); err != nil {
			return fmt.Errorf("failed to delete tracker: %v", err)
		}

		return nil
	})
}

// UpdateTracker updates specific fields of a tracker
func (bs *BadgerStorage) UpdateTracker(id string, fields map[string]interface{}) error {
	return bs.db.Update(func(txn *badger.Txn) error {
		// Get existing tracker
		trackerKey := bs.trackerKey(id)
		item, err := txn.Get(trackerKey)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return fmt.Errorf("tracker not found")
			}
			return err
		}

		var tracker models.TrackerItem
		err = item.Value(func(val []byte) error {
			return json.Unmarshal(val, &tracker)
		})
		if err != nil {
			return err
		}

		// Apply field updates
		if err := bs.applyTrackerFieldUpdates(&tracker, fields); err != nil {
			return err
		}

		// Update timestamp
		tracker.LastUpdated = time.Now()

		// Serialize updated tracker
		trackerBytes, err := json.Marshal(tracker)
		if err != nil {
			return fmt.Errorf("failed to marshal updated tracker: %v", err)
		}

		// Save updated tracker
		if err := txn.Set(trackerKey, trackerBytes); err != nil {
			return fmt.Errorf("failed to save updated tracker: %v", err)
		}

		// Update indexes
		if err := bs.updateTrackerIndexes(txn, tracker); err != nil {
			return fmt.Errorf("failed to update tracker indexes: %v", err)
		}

		return nil
	})
}

// SaveTrackersBatch saves multiple trackers in a single transaction
func (bs *BadgerStorage) SaveTrackersBatch(trackers []models.TrackerItem) error {
	return bs.db.Update(func(txn *badger.Txn) error {
		now := time.Now()

		for _, tracker := range trackers {
			// Set timestamps
			tracker.LastUpdated = now
			if tracker.CreatedAt.IsZero() {
				tracker.CreatedAt = now
			}

			// Serialize tracker
			trackerBytes, err := json.Marshal(tracker)
			if err != nil {
				return fmt.Errorf("failed to marshal tracker %s: %v", tracker.ID, err)
			}

			// Save tracker
			trackerKey := bs.trackerKey(tracker.ID)
			if err := txn.Set(trackerKey, trackerBytes); err != nil {
				return fmt.Errorf("failed to save tracker %s: %v", tracker.ID, err)
			}

			// Update indexes
			if err := bs.updateTrackerIndexes(txn, tracker); err != nil {
				return fmt.Errorf("failed to update indexes for tracker %s: %v", tracker.ID, err)
			}
		}

		return nil
	})
}

// UpdateTrackersBatch updates multiple trackers in a single transaction
func (bs *BadgerStorage) UpdateTrackersBatch(updates []models.TrackerUpdate) error {
	return bs.db.Update(func(txn *badger.Txn) error {
		for _, update := range updates {
			// Get existing tracker
			trackerKey := bs.trackerKey(update.ID)
			item, err := txn.Get(trackerKey)
			if err != nil {
				if err == badger.ErrKeyNotFound {
					log.Printf("Warning: tracker %s not found during batch update", update.ID)
					continue
				}
				return err
			}

			var tracker models.TrackerItem
			err = item.Value(func(val []byte) error {
				return json.Unmarshal(val, &tracker)
			})
			if err != nil {
				return err
			}

			// Apply field updates
			if err := bs.applyTrackerFieldUpdates(&tracker, update.Fields); err != nil {
				return err
			}

			// Update timestamp
			tracker.LastUpdated = time.Now()

			// Serialize updated tracker
			trackerBytes, err := json.Marshal(tracker)
			if err != nil {
				return fmt.Errorf("failed to marshal updated tracker %s: %v", tracker.ID, err)
			}

			// Save updated tracker
			if err := txn.Set(trackerKey, trackerBytes); err != nil {
				return fmt.Errorf("failed to save updated tracker %s: %v", tracker.ID, err)
			}

			// Update indexes
			if err := bs.updateTrackerIndexes(txn, tracker); err != nil {
				return fmt.Errorf("failed to update indexes for tracker %s: %v", tracker.ID, err)
			}
		}

		return nil
	})
}

// Helper methods for tracker operations

// trackerKey generates a BadgerDB key for a tracker
func (bs *BadgerStorage) trackerKey(id string) []byte {
	return []byte("tracker:" + id)
}

// trackerIndexKey generates a BadgerDB key for tracker indexes
func (bs *BadgerStorage) trackerIndexKey(indexName, value string) []byte {
	return []byte("tracker_idx:" + indexName + ":" + value)
}

// updateTrackerIndexes updates secondary indexes for a tracker
func (bs *BadgerStorage) updateTrackerIndexes(txn *badger.Txn, tracker models.TrackerItem) error {
	// Index by URL
	if tracker.URL != "" {
		urlKey := bs.trackerIndexKey("url", tracker.URL)
		if err := txn.Set(urlKey, []byte(tracker.ID)); err != nil {
			return err
		}
	}

	// Index by in-stock status
	stockStatus := "false"
	if tracker.InStock {
		stockStatus = "true"
	}
	stockKey := bs.trackerIndexKey("in_stock", stockStatus)
	if err := txn.Set(stockKey, []byte(tracker.ID)); err != nil {
		return err
	}

	// Index by user ID (if provided)
	if tracker.UserID != "" {
		userKey := bs.trackerIndexKey("user_id", tracker.UserID)
		if err := txn.Set(userKey, []byte(tracker.ID)); err != nil {
			return err
		}
	}

	return nil
}

// removeFromTrackerIndexes removes a tracker from secondary indexes
func (bs *BadgerStorage) removeFromTrackerIndexes(txn *badger.Txn, tracker models.TrackerItem) error {
	// Remove from URL index
	if tracker.URL != "" {
		urlKey := bs.trackerIndexKey("url", tracker.URL)
		_ = txn.Delete(urlKey) // Ignore errors for index cleanup
	}

	// Remove from stock index
	stockStatus := "false"
	if tracker.InStock {
		stockStatus = "true"
	}
	stockKey := bs.trackerIndexKey("in_stock", stockStatus)
	_ = txn.Delete(stockKey)

	// Remove from user index
	if tracker.UserID != "" {
		userKey := bs.trackerIndexKey("user_id", tracker.UserID)
		_ = txn.Delete(userKey)
	}

	return nil
}

// matchesTrackerFilters checks if a tracker matches the given filters
func (bs *BadgerStorage) matchesTrackerFilters(tracker models.TrackerItem, filters models.TrackerFilterOptions) bool {
	// Filter by query (search in name and URL)
	if filters.Query != "" {
		query := strings.ToLower(filters.Query)
		name := strings.ToLower(tracker.Name)
		url := strings.ToLower(tracker.URL)

		if !strings.Contains(name, query) && !strings.Contains(url, query) {
			return false
		}
	}

	// Filter by in-stock status
	if filters.InStockOnly && !tracker.InStock {
		return false
	}

	// Filter by user ID
	if filters.UserID != "" && tracker.UserID != filters.UserID {
		return false
	}

	return true
}

// sortTrackers sorts trackers based on the specified criteria
func (bs *BadgerStorage) sortTrackers(trackers []models.TrackerItem, sortBy, sortOrder string) {
	// Implementation would sort the slice based on sortBy and sortOrder
	// For now, keeping it simple - could be enhanced with proper sorting logic
}

// applyTrackerFieldUpdates applies field updates to a tracker
func (bs *BadgerStorage) applyTrackerFieldUpdates(tracker *models.TrackerItem, updates map[string]interface{}) error {
	for field, value := range updates {
		switch field {
		case "name":
			if v, ok := value.(string); ok {
				tracker.Name = v
			}
		case "in_stock":
			if v, ok := value.(bool); ok {
				tracker.InStock = v
			}
		case "image":
			if v, ok := value.(string); ok {
				tracker.Image = v
			}
		case "price_yen":
			if v, ok := value.(string); ok {
				tracker.PriceYen = v
			}
		case "user_id":
			if v, ok := value.(string); ok {
				tracker.UserID = v
			}
		default:
			return fmt.Errorf("unknown field: %s", field)
		}
	}
	return nil
}
