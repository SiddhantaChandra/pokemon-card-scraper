package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/SiddhantaChandra/pokemon-card-scraper/pkg/models"
	"github.com/dgraph-io/badger/v4"
)

// Common tracker storage errors
var (
	ErrTrackerNotFound      = errors.New("tracker not found")
	ErrTrackerURLExists     = errors.New("tracker URL already exists")
	ErrInvalidTrackerStatus = errors.New("invalid tracker status")
)

// TrackerStorage interface defines methods for tracker persistence
type TrackerStorage interface {
	SaveTracker(tracker models.TrackerEntry) error
	GetTracker(id string) (*models.TrackerEntry, error)
	GetAllTrackers() ([]models.TrackerEntry, error)
	GetActiveTrackers() ([]models.TrackerEntry, error)
	SearchTrackers(filters models.TrackerFilterOptions) (*models.TrackerSearchResult, error)
	UpdateTrackerStatus(id string, inStock bool, price float64, imageURL string) error
	UpdateTracker(tracker models.TrackerEntry) error
	DeleteTracker(id string) error
	GetTrackerByURL(url string) (*models.TrackerEntry, error)
	GetTrackerStats() (*models.TrackerStats, error)
	SaveTrackersBatch(trackers []models.TrackerEntry) error
}

// Extend BadgerStorage to include tracker functionality
func (bs *BadgerStorage) SaveTracker(tracker models.TrackerEntry) error {
	// Validate tracker
	if err := tracker.Validate(); err != nil {
		return fmt.Errorf("invalid tracker: %v", err)
	}

	// Check if URL already exists
	existingTracker, err := bs.GetTrackerByURL(tracker.URL)
	if err != nil && err != ErrTrackerNotFound {
		return fmt.Errorf("failed to check existing URL: %v", err)
	}
	if existingTracker != nil && existingTracker.ID != tracker.ID {
		return ErrTrackerURLExists
	}

	return bs.db.Update(func(txn *badger.Txn) error {
		// Set timestamps
		now := time.Now()
		tracker.UpdatedAt = now
		if tracker.CreatedAt.IsZero() {
			tracker.CreatedAt = now
		}

		// Generate ID if not provided
		if tracker.ID == "" {
			tracker.ID = bs.generateTrackerID()
		}

		// Set default status if not provided
		if tracker.Status == "" {
			tracker.Status = models.TrackerStatusActive
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
func (bs *BadgerStorage) GetTracker(id string) (*models.TrackerEntry, error) {
	var tracker models.TrackerEntry

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
			return nil, ErrTrackerNotFound
		}
		return nil, fmt.Errorf("failed to get tracker: %v", err)
	}

	return &tracker, nil
}

// GetAllTrackers retrieves all trackers without pagination
func (bs *BadgerStorage) GetAllTrackers() ([]models.TrackerEntry, error) {
	var allTrackers []models.TrackerEntry

	err := bs.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		iterator := txn.NewIterator(opts)
		defer iterator.Close()

		prefix := []byte("tracker:")
		for iterator.Seek(prefix); iterator.ValidForPrefix(prefix); iterator.Next() {
			item := iterator.Item()

			err := item.Value(func(val []byte) error {
				var tracker models.TrackerEntry
				if err := json.Unmarshal(val, &tracker); err != nil {
					return err
				}
				allTrackers = append(allTrackers, tracker)
				return nil
			})

			if err != nil {
				return err
			}
		}

		return nil
	})

	return allTrackers, err
}

// GetActiveTrackers retrieves all active trackers
func (bs *BadgerStorage) GetActiveTrackers() ([]models.TrackerEntry, error) {
	filters := models.TrackerFilterOptions{
		Status:   models.TrackerStatusActive,
		PageSize: 10000, // Large page size to get all
		Page:     1,
	}

	result, err := bs.SearchTrackers(filters)
	if err != nil {
		return nil, err
	}

	return result.Trackers, nil
}

// SearchTrackers searches for trackers based on filter options
func (bs *BadgerStorage) SearchTrackers(filters models.TrackerFilterOptions) (*models.TrackerSearchResult, error) {
	var allTrackers []models.TrackerEntry

	err := bs.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		iterator := txn.NewIterator(opts)
		defer iterator.Close()

		prefix := []byte("tracker:")
		for iterator.Seek(prefix); iterator.ValidForPrefix(prefix); iterator.Next() {
			item := iterator.Item()

			err := item.Value(func(val []byte) error {
				var tracker models.TrackerEntry
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
		allTrackers = []models.TrackerEntry{}
	} else {
		if endIdx > totalTrackers {
			endIdx = totalTrackers
		}
		allTrackers = allTrackers[startIdx:endIdx]
	}

	totalPages := (totalTrackers + filters.PageSize - 1) / filters.PageSize

	return &models.TrackerSearchResult{
		Trackers:   allTrackers,
		Total:      totalTrackers,
		Page:       filters.Page,
		PageSize:   filters.PageSize,
		TotalPages: totalPages,
	}, nil
}

// UpdateTrackerStatus updates tracker stock status, price, and image
func (bs *BadgerStorage) UpdateTrackerStatus(id string, inStock bool, price float64, imageURL string) error {
	return bs.db.Update(func(txn *badger.Txn) error {
		// Get existing tracker
		trackerKey := bs.trackerKey(id)
		item, err := txn.Get(trackerKey)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrTrackerNotFound
			}
			return fmt.Errorf("failed to get tracker for status update: %v", err)
		}

		var tracker models.TrackerEntry
		err = item.Value(func(val []byte) error {
			return json.Unmarshal(val, &tracker)
		})
		if err != nil {
			return fmt.Errorf("failed to unmarshal tracker: %v", err)
		}

		// Update fields
		tracker.InStock = inStock
		tracker.Price = price
		if imageURL != "" {
			tracker.ImageURL = imageURL
		}
		tracker.LastChecked = time.Now()
		tracker.UpdatedAt = time.Now()

		// Save updated tracker
		trackerBytes, err := json.Marshal(tracker)
		if err != nil {
			return fmt.Errorf("failed to marshal updated tracker: %v", err)
		}

		if err := txn.Set(trackerKey, trackerBytes); err != nil {
			return fmt.Errorf("failed to save updated tracker: %v", err)
		}

		// Update indexes
		return bs.updateTrackerIndexes(txn, tracker)
	})
}

// UpdateTracker updates an entire tracker entry
func (bs *BadgerStorage) UpdateTracker(tracker models.TrackerEntry) error {
	if err := tracker.Validate(); err != nil {
		return fmt.Errorf("invalid tracker: %v", err)
	}

	return bs.db.Update(func(txn *badger.Txn) error {
		// Check if tracker exists
		trackerKey := bs.trackerKey(tracker.ID)
		_, err := txn.Get(trackerKey)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrTrackerNotFound
			}
			return fmt.Errorf("failed to check tracker existence: %v", err)
		}

		// Update timestamp
		tracker.UpdatedAt = time.Now()

		// Serialize tracker to JSON
		trackerBytes, err := json.Marshal(tracker)
		if err != nil {
			return fmt.Errorf("failed to marshal tracker: %v", err)
		}

		// Save updated tracker
		if err := txn.Set(trackerKey, trackerBytes); err != nil {
			return fmt.Errorf("failed to save tracker: %v", err)
		}

		// Update indexes
		return bs.updateTrackerIndexes(txn, tracker)
	})
}

// DeleteTracker removes a tracker from the database
func (bs *BadgerStorage) DeleteTracker(id string) error {
	return bs.db.Update(func(txn *badger.Txn) error {
		// Get the tracker first to update indexes
		trackerKey := bs.trackerKey(id)
		item, err := txn.Get(trackerKey)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return nil // Already deleted
			}
			return err
		}

		var tracker models.TrackerEntry
		err = item.Value(func(val []byte) error {
			return json.Unmarshal(val, &tracker)
		})
		if err != nil {
			return err
		}

		// Remove from indexes
		if err := bs.removeFromTrackerIndexes(txn, tracker); err != nil {
			return fmt.Errorf("failed to remove from tracker indexes: %v", err)
		}

		// Delete main record
		return txn.Delete(trackerKey)
	})
}

// GetTrackerByURL retrieves a tracker by its URL
func (bs *BadgerStorage) GetTrackerByURL(url string) (*models.TrackerEntry, error) {
	var tracker *models.TrackerEntry

	err := bs.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		iterator := txn.NewIterator(opts)
		defer iterator.Close()

		prefix := []byte("tracker:")
		for iterator.Seek(prefix); iterator.ValidForPrefix(prefix); iterator.Next() {
			item := iterator.Item()

			err := item.Value(func(val []byte) error {
				var t models.TrackerEntry
				if err := json.Unmarshal(val, &t); err != nil {
					return err
				}

				if t.URL == url {
					tracker = &t
					return nil
				}

				return nil
			})

			if err != nil {
				return err
			}

			if tracker != nil {
				break
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to search tracker by URL: %v", err)
	}

	if tracker == nil {
		return nil, ErrTrackerNotFound
	}

	return tracker, nil
}

// GetTrackerStats returns statistics about trackers
func (bs *BadgerStorage) GetTrackerStats() (*models.TrackerStats, error) {
	stats := &models.TrackerStats{}

	err := bs.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		iterator := txn.NewIterator(opts)
		defer iterator.Close()

		prefix := []byte("tracker:")
		for iterator.Seek(prefix); iterator.ValidForPrefix(prefix); iterator.Next() {
			item := iterator.Item()

			err := item.Value(func(val []byte) error {
				var tracker models.TrackerEntry
				if err := json.Unmarshal(val, &tracker); err != nil {
					return err
				}

				stats.TotalTrackers++

				switch tracker.Status {
				case models.TrackerStatusActive:
					stats.ActiveTrackers++
					if tracker.InStock {
						stats.InStockTrackers++
					} else {
						stats.OutStockTrackers++
					}
				case models.TrackerStatusPaused:
					stats.PausedTrackers++
				}

				return nil
			})

			if err != nil {
				return err
			}
		}

		return nil
	})

	return stats, err
}

// SaveTrackersBatch saves multiple trackers in a single transaction
func (bs *BadgerStorage) SaveTrackersBatch(trackers []models.TrackerEntry) error {
	if len(trackers) == 0 {
		return nil
	}

	return bs.db.Update(func(txn *badger.Txn) error {
		now := time.Now()

		for i, tracker := range trackers {
			// Validate tracker
			if err := tracker.Validate(); err != nil {
				return fmt.Errorf("invalid tracker at index %d: %v", i, err)
			}

			// Set timestamps
			tracker.UpdatedAt = now
			if tracker.CreatedAt.IsZero() {
				tracker.CreatedAt = now
			}

			// Generate ID if not provided
			if tracker.ID == "" {
				tracker.ID = bs.generateTrackerID()
			}

			// Set default status if not provided
			if tracker.Status == "" {
				tracker.Status = models.TrackerStatusActive
			}

			// Serialize tracker to JSON
			trackerBytes, err := json.Marshal(tracker)
			if err != nil {
				return fmt.Errorf("failed to marshal tracker %s: %v", tracker.ID, err)
			}

			// Save main tracker record
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

// Helper functions for tracker storage

func (bs *BadgerStorage) trackerKey(id string) []byte {
	return []byte("tracker:" + id)
}

func (bs *BadgerStorage) trackerIndexKey(indexName, value string) []byte {
	return []byte(fmt.Sprintf("tracker_idx:%s:%s", indexName, value))
}

func (bs *BadgerStorage) generateTrackerID() string {
	return fmt.Sprintf("tracker_%d", time.Now().UnixNano())
}

func (bs *BadgerStorage) updateTrackerIndexes(txn *badger.Txn, tracker models.TrackerEntry) error {
	// URL index (for duplicate prevention)
	urlKey := bs.trackerIndexKey("url", tracker.URL)
	if err := txn.Set(urlKey, []byte(tracker.ID)); err != nil {
		return err
	}

	// Status index
	statusKey := bs.trackerIndexKey("status", string(tracker.Status))
	if err := txn.Set(statusKey, []byte(tracker.ID)); err != nil {
		return err
	}

	// Stock index
	stockKey := bs.trackerIndexKey("stock", strconv.FormatBool(tracker.InStock))
	if err := txn.Set(stockKey, []byte(tracker.ID)); err != nil {
		return err
	}

	// User index (for future multi-user support)
	if tracker.UserID != "" {
		userKey := bs.trackerIndexKey("user", tracker.UserID)
		if err := txn.Set(userKey, []byte(tracker.ID)); err != nil {
			return err
		}
	}

	return nil
}

func (bs *BadgerStorage) removeFromTrackerIndexes(txn *badger.Txn, tracker models.TrackerEntry) error {
	// Remove from all indexes
	urlKey := bs.trackerIndexKey("url", tracker.URL)
	txn.Delete(urlKey)

	statusKey := bs.trackerIndexKey("status", string(tracker.Status))
	txn.Delete(statusKey)

	stockKey := bs.trackerIndexKey("stock", strconv.FormatBool(tracker.InStock))
	txn.Delete(stockKey)

	if tracker.UserID != "" {
		userKey := bs.trackerIndexKey("user", tracker.UserID)
		txn.Delete(userKey)
	}

	return nil
}

func (bs *BadgerStorage) matchesTrackerFilters(tracker models.TrackerEntry, filters models.TrackerFilterOptions) bool {
	// Status filter
	if filters.Status != "" && tracker.Status != filters.Status {
		return false
	}

	// Stock filter
	if filters.InStock != nil && tracker.InStock != *filters.InStock {
		return false
	}

	// User filter
	if filters.UserID != "" && tracker.UserID != filters.UserID {
		return false
	}

	return true
}

func (bs *BadgerStorage) sortTrackers(trackers []models.TrackerEntry, sortBy, sortOrder string) {
	// Implementation of sorting would go here
	// For now, we'll keep the existing order
	// This could be expanded with proper sorting algorithms based on sortBy field
}
