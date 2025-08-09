package engine

import (
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/your-org/hema-replay-system/pkg/speech/types"
)

// CacheEntry represents a cached transcription result
type CacheEntry struct {
	Result    *types.TranscriptionResult
	ExpiresAt time.Time
}

// ResultCache provides caching for transcription results
type ResultCache struct {
	cache   map[string]*CacheEntry
	maxSize int
	ttl     time.Duration
	mu      sync.RWMutex
	logger  zerolog.Logger

	// Statistics
	hits      int64
	misses    int64
	evictions int64
}

// NewResultCache creates a new result cache
func NewResultCache(maxSize int, ttl time.Duration, logger zerolog.Logger) *ResultCache {
	cache := &ResultCache{
		cache:   make(map[string]*CacheEntry),
		maxSize: maxSize,
		ttl:     ttl,
		logger:  logger.With().Str("component", "result_cache").Logger(),
	}

	// Start cleanup goroutine
	go cache.cleanupExpired()

	return cache
}

// Get retrieves a cached result
func (rc *ResultCache) Get(key string) *types.TranscriptionResult {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	entry, exists := rc.cache[key]
	if !exists {
		rc.misses++
		return nil
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		rc.misses++
		// Don't delete here to avoid write lock, cleanup goroutine will handle it
		return nil
	}

	rc.hits++
	return entry.Result
}

// Set stores a result in the cache
func (rc *ResultCache) Set(key string, result *types.TranscriptionResult) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Check if we need to evict entries
	if len(rc.cache) >= rc.maxSize {
		rc.evictOldest()
	}

	entry := &CacheEntry{
		Result:    result,
		ExpiresAt: time.Now().Add(rc.ttl),
	}

	rc.cache[key] = entry
}

// evictOldest removes the oldest entry from the cache
func (rc *ResultCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range rc.cache {
		if oldestKey == "" || entry.ExpiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.ExpiresAt
		}
	}

	if oldestKey != "" {
		delete(rc.cache, oldestKey)
		rc.evictions++
	}
}

// cleanupExpired removes expired entries periodically
func (rc *ResultCache) cleanupExpired() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rc.mu.Lock()
		now := time.Now()

		for key, entry := range rc.cache {
			if now.After(entry.ExpiresAt) {
				delete(rc.cache, key)
				rc.evictions++
			}
		}

		rc.mu.Unlock()
	}
}

// Clear removes all entries from the cache
func (rc *ResultCache) Clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.cache = make(map[string]*CacheEntry)
}

// GetStats returns cache statistics
func (rc *ResultCache) GetStats() map[string]interface{} {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	var hitRate float64
	total := rc.hits + rc.misses
	if total > 0 {
		hitRate = float64(rc.hits) / float64(total) * 100
	}

	return map[string]interface{}{
		"size":      len(rc.cache),
		"max_size":  rc.maxSize,
		"hits":      rc.hits,
		"misses":    rc.misses,
		"hit_rate":  hitRate,
		"evictions": rc.evictions,
		"ttl":       rc.ttl,
	}
}
