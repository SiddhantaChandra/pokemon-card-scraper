package storage

import (
	"container/list"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/SiddhantaChandra/pokemon-card-scraper/pkg/models"
)

// CacheEntry represents a cached search result
type CacheEntry struct {
	Key        string
	Result     *models.SearchResult
	Timestamp  time.Time
	AccessTime time.Time
}

// LRUCache implements an LRU cache for search results
type LRUCache struct {
	capacity int
	cache    map[string]*list.Element
	order    *list.List
	mu       sync.RWMutex
	stats    CacheStats
}

// CacheStats tracks cache performance metrics
type CacheStats struct {
	Hits         int64 `json:"hits"`
	Misses       int64 `json:"misses"`
	Evictions    int64 `json:"evictions"`
	TotalEntries int   `json:"total_entries"`
}

// CacheConfig holds configuration for the cache
type CacheConfig struct {
	Capacity int           // Maximum number of entries
	TTL      time.Duration // Time to live for entries
}

// DefaultCacheConfig returns sensible defaults
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		Capacity: 1000,             // 1000 search results
		TTL:      10 * time.Minute, // 10 minutes TTL
	}
}

// NewLRUCache creates a new LRU cache
func NewLRUCache(config *CacheConfig) *LRUCache {
	if config == nil {
		config = DefaultCacheConfig()
	}

	return &LRUCache{
		capacity: config.Capacity,
		cache:    make(map[string]*list.Element),
		order:    list.New(),
		stats:    CacheStats{},
	}
}

// Get retrieves a value from the cache
func (c *LRUCache) Get(key string) (*models.SearchResult, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	element, exists := c.cache[key]
	if !exists {
		c.stats.Misses++
		return nil, false
	}

	entry := element.Value.(*CacheEntry)

	// Check if entry has expired
	if time.Since(entry.Timestamp) > 10*time.Minute { // Default TTL
		c.removeElement(element)
		c.stats.Misses++
		return nil, false
	}

	// Move to front (most recently used)
	c.order.MoveToFront(element)
	entry.AccessTime = time.Now()

	c.stats.Hits++
	return entry.Result, true
}

// Put adds or updates a value in the cache
func (c *LRUCache) Put(key string, result *models.SearchResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if key already exists
	if element, exists := c.cache[key]; exists {
		// Update existing entry
		entry := element.Value.(*CacheEntry)
		entry.Result = result
		entry.Timestamp = time.Now()
		entry.AccessTime = time.Now()
		c.order.MoveToFront(element)
		return
	}

	// Create new entry
	entry := &CacheEntry{
		Key:        key,
		Result:     result,
		Timestamp:  time.Now(),
		AccessTime: time.Now(),
	}

	element := c.order.PushFront(entry)
	c.cache[key] = element

	// Check capacity and evict if necessary
	if c.order.Len() > c.capacity {
		c.evictLRU()
	}

	c.stats.TotalEntries = len(c.cache)
}

// Delete removes a specific key from the cache
func (c *LRUCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, exists := c.cache[key]; exists {
		c.removeElement(element)
	}
}

// Clear removes all entries from the cache
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*list.Element)
	c.order = list.New()
	c.stats.TotalEntries = 0
}

// Size returns the current number of entries in the cache
func (c *LRUCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

// GetStats returns cache performance statistics
func (c *LRUCache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := c.stats
	stats.TotalEntries = len(c.cache)
	return stats
}

// GetHitRatio returns the cache hit ratio as a percentage
func (c *LRUCache) GetHitRatio() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.stats.Hits + c.stats.Misses
	if total == 0 {
		return 0
	}
	return float64(c.stats.Hits) / float64(total) * 100
}

// CleanupExpired removes expired entries from the cache
func (c *LRUCache) CleanupExpired(ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	var toRemove []*list.Element

	// Iterate from back (oldest) to front
	for element := c.order.Back(); element != nil; element = element.Prev() {
		entry := element.Value.(*CacheEntry)
		if now.Sub(entry.Timestamp) > ttl {
			toRemove = append(toRemove, element)
		} else {
			break // Since we're iterating from oldest, we can stop here
		}
	}

	// Remove expired entries
	for _, element := range toRemove {
		c.removeElement(element)
	}

	c.stats.TotalEntries = len(c.cache)
}

// Private helper methods

func (c *LRUCache) evictLRU() {
	if c.order.Len() == 0 {
		return
	}

	// Remove least recently used (back of list)
	element := c.order.Back()
	if element != nil {
		c.removeElement(element)
		c.stats.Evictions++
	}
}

func (c *LRUCache) removeElement(element *list.Element) {
	entry := element.Value.(*CacheEntry)
	delete(c.cache, entry.Key)
	c.order.Remove(element)
}

// SearchCache provides caching for search operations
type SearchCache struct {
	cache *LRUCache
	ttl   time.Duration
}

// NewSearchCache creates a new search cache
func NewSearchCache(config *CacheConfig) *SearchCache {
	if config == nil {
		config = DefaultCacheConfig()
	}

	return &SearchCache{
		cache: NewLRUCache(config),
		ttl:   config.TTL,
	}
}

// GetSearchResult retrieves a cached search result
func (sc *SearchCache) GetSearchResult(filters models.FilterOptions) (*models.SearchResult, bool) {
	key := sc.generateCacheKey(filters)
	return sc.cache.Get(key)
}

// CacheSearchResult stores a search result in the cache
func (sc *SearchCache) CacheSearchResult(filters models.FilterOptions, result *models.SearchResult) {
	key := sc.generateCacheKey(filters)
	sc.cache.Put(key, result)
}

// InvalidateByPattern removes cache entries that match a pattern
func (sc *SearchCache) InvalidateByPattern(pattern string) {
	// For now, clear all cache. Could be optimized to pattern matching
	sc.cache.Clear()
}

// InvalidateAll clears all cached results
func (sc *SearchCache) InvalidateAll() {
	sc.cache.Clear()
}

// GetStats returns cache statistics
func (sc *SearchCache) GetStats() CacheStats {
	return sc.cache.GetStats()
}

// StartCleanupRoutine starts a background goroutine to clean up expired entries
func (sc *SearchCache) StartCleanupRoutine() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute) // Clean up every 5 minutes
		defer ticker.Stop()

		for range ticker.C {
			sc.cache.CleanupExpired(sc.ttl)
		}
	}()
}

// generateCacheKey creates a unique key for the filter options
func (sc *SearchCache) generateCacheKey(filters models.FilterOptions) string {
	// Normalize filters for consistent caching
	normalizedFilters := filters

	// Sort arrays for consistent hashing
	// (Implementation would sort SetNames and Rarities arrays)

	// Marshal to JSON for hashing
	data, err := json.Marshal(normalizedFilters)
	if err != nil {
		// Fallback to a simple string representation
		return fmt.Sprintf("query:%s,page:%d,size:%d",
			filters.Query, filters.Page, filters.PageSize)
	}

	// Generate MD5 hash
	hash := md5.Sum(data)
	return fmt.Sprintf("%x", hash)
}

// CachedStorage wraps a storage implementation with caching
type CachedStorage struct {
	storage Storage
	cache   *SearchCache
}

// Storage interface that the cached storage wraps
type Storage interface {
	SaveCard(card models.Card) error
	GetCard(id string) (*models.Card, error)
	SearchCards(filters models.FilterOptions) (*models.SearchResult, error)
	GetAllInStock() ([]models.Card, error)
	GetAllCards() ([]models.Card, error)
	DeleteCard(id string) error
	ClearAllData() error
}

// NewCachedStorage creates a new cached storage wrapper
func NewCachedStorage(storage Storage, cacheConfig *CacheConfig) *CachedStorage {
	searchCache := NewSearchCache(cacheConfig)
	searchCache.StartCleanupRoutine()

	return &CachedStorage{
		storage: storage,
		cache:   searchCache,
	}
}

// SaveCard saves a card and invalidates relevant cache entries
func (cs *CachedStorage) SaveCard(card models.Card) error {
	err := cs.storage.SaveCard(card)
	if err != nil {
		return err
	}

	// Invalidate cache since data has changed
	cs.cache.InvalidateAll()
	return nil
}

// GetCard retrieves a card (no caching for individual cards)
func (cs *CachedStorage) GetCard(id string) (*models.Card, error) {
	return cs.storage.GetCard(id)
}

// SearchCards performs cached search
func (cs *CachedStorage) SearchCards(filters models.FilterOptions) (*models.SearchResult, error) {
	// Try cache first
	if result, found := cs.cache.GetSearchResult(filters); found {
		return result, nil
	}

	// Cache miss, query storage
	result, err := cs.storage.SearchCards(filters)
	if err != nil {
		return nil, err
	}

	// Cache the result
	cs.cache.CacheSearchResult(filters, result)
	return result, nil
}

// GetAllInStock retrieves all in-stock cards (cached)
func (cs *CachedStorage) GetAllInStock() ([]models.Card, error) {
	filters := models.FilterOptions{
		InStockOnly: true,
		SortBy:      "name",
		SortOrder:   "asc",
		PageSize:    10000,
		Page:        1,
	}

	result, err := cs.SearchCards(filters)
	if err != nil {
		return nil, err
	}

	return result.Cards, nil
}

// GetAllCards retrieves all cards (delegated to storage, no caching)
func (cs *CachedStorage) GetAllCards() ([]models.Card, error) {
	return cs.storage.GetAllCards()
}

// DeleteCard deletes a card and invalidates cache
func (cs *CachedStorage) DeleteCard(id string) error {
	err := cs.storage.DeleteCard(id)
	if err != nil {
		return err
	}

	// Invalidate cache since data has changed
	cs.cache.InvalidateAll()
	return nil
}

// ClearAllData removes all cards and resets the database (preserves tracker data)
func (cs *CachedStorage) ClearAllData() error {
	// First clear the underlying storage (cards only, preserves tracker data)
	if clearable, ok := cs.storage.(interface{ ClearAllData() error }); ok {
		err := clearable.ClearAllData()
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("underlying storage does not support ClearAllData operation")
	}

	// Clear all cache entries (only card search cache, not tracker data)
	cs.cache.InvalidateAll()
	return nil
}
