package storage

import (
	"sync"
	"time"
)

// PoolStatsData resource pool statistics data
type PoolStatsData struct {
	Timestamp time.Time              `json:"timestamp"`
	Stats     map[string]interface{} `json:"stats"`
}

// PoolStatsStorage resource pool statistics storage (in-memory, keeps only latest data)
type PoolStatsStorage struct {
	mu     sync.RWMutex
	latest *PoolStatsData // only stores the latest statistics entry
}

var (
	globalPoolStatsStorage *PoolStatsStorage
	once                   sync.Once
)

// GetPoolStatsStorage returns the global pool statistics storage singleton
func GetPoolStatsStorage() *PoolStatsStorage {
	once.Do(func() {
		globalPoolStatsStorage = &PoolStatsStorage{
			latest: nil,
		}
	})
	return globalPoolStatsStorage
}

// AddStats adds statistics data (only keeps the latest, overwrites previous)
func (s *PoolStatsStorage) AddStats(stats map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// overwrite with latest data directly
	s.latest = &PoolStatsData{
		Timestamp: time.Now(),
		Stats:     stats,
	}
}

// GetLatestStats returns the latest statistics data
func (s *PoolStatsStorage) GetLatestStats() *PoolStatsData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.latest == nil {
		return nil
	}

	// return a copy of the latest data
	latest := *s.latest
	return &latest
}

// GetAllStats returns all statistics data (only the latest entry)
func (s *PoolStatsStorage) GetAllStats() []PoolStatsData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.latest == nil {
		return []PoolStatsData{}
	}

	// return only the latest entry
	return []PoolStatsData{*s.latest}
}

// GetStatsByTimeRange returns statistics within the given time range (latest only, if within range)
func (s *PoolStatsStorage) GetStatsByTimeRange(start, end time.Time) []PoolStatsData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.latest == nil {
		return []PoolStatsData{}
	}

	// check if the latest data falls within the time range
	if s.latest.Timestamp.After(start) && s.latest.Timestamp.Before(end) {
		return []PoolStatsData{*s.latest}
	}

	return []PoolStatsData{}
}

// GetStatsCount returns the number of stored entries (0 or 1 since only latest is kept)
func (s *PoolStatsStorage) GetStatsCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.latest == nil {
		return 0
	}
	return 1
}
