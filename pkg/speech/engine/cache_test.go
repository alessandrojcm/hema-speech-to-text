package engine

import (
	"fmt"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	speechTypes "github.com/your-org/hema-replay-system/pkg/speech/types"
)

func createTestResult(id, text string) *speechTypes.TranscriptionResult {
	return &speechTypes.TranscriptionResult{
		ID:         id,
		Text:       text,
		Confidence: 0.95,
		Language:   "en",
		Duration:   2 * time.Second,
		Segments:   []speechTypes.TranscriptionSegment{},
		Metadata: speechTypes.TranscriptionMetadata{
			ModelUsed:      "tiny",
			ProcessingTime: 100 * time.Millisecond,
			AudioQuality:   0.8,
		},
		ProcessedAt: time.Now(),
	}
}

func TestNewResultCache(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(nil))

	cache := NewResultCache(10, time.Minute, logger)

	assert.NotNil(t, cache)

	stats := cache.GetStats()
	assert.Equal(t, 0, stats["size"])
	assert.Equal(t, int64(0), stats["hits"])
	assert.Equal(t, int64(0), stats["misses"])
}

func TestResultCache_SetAndGet(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(nil))
	cache := NewResultCache(10, time.Minute, logger)

	result := createTestResult("test-1", "Hello world")

	// Set result
	cache.Set("test-1", result)

	// Get result
	retrieved := cache.Get("test-1")
	require.NotNil(t, retrieved)
	assert.Equal(t, result.ID, retrieved.ID)
	assert.Equal(t, result.Text, retrieved.Text)
	assert.Equal(t, result.Confidence, retrieved.Confidence)

	// Verify stats
	stats := cache.GetStats()
	assert.Equal(t, 1, stats["size"])
	assert.Equal(t, int64(1), stats["hits"])
	assert.Equal(t, int64(0), stats["misses"])
}

func TestResultCache_GetMiss(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(nil))
	cache := NewResultCache(10, time.Minute, logger)

	// Try to get non-existent result
	result := cache.Get("non-existent")
	assert.Nil(t, result)

	// Verify stats
	stats := cache.GetStats()
	assert.Equal(t, 0, stats["size"])
	assert.Equal(t, int64(0), stats["hits"])
	assert.Equal(t, int64(1), stats["misses"])
}

func TestResultCache_Eviction(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(nil))
	cache := NewResultCache(2, time.Minute, logger) // Small cache size

	// Add items beyond capacity
	result1 := createTestResult("test-1", "First")
	result2 := createTestResult("test-2", "Second")
	result3 := createTestResult("test-3", "Third")

	cache.Set("test-1", result1)
	cache.Set("test-2", result2)
	cache.Set("test-3", result3) // Should evict test-1

	// test-1 should be evicted
	assert.Nil(t, cache.Get("test-1"))

	// test-2 and test-3 should still be there
	assert.NotNil(t, cache.Get("test-2"))
	assert.NotNil(t, cache.Get("test-3"))

	// Verify size
	stats := cache.GetStats()
	assert.Equal(t, 2, stats["size"])
}

func TestResultCache_TTLExpiration(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(nil))
	cache := NewResultCache(10, 50*time.Millisecond, logger) // Short TTL

	result := createTestResult("test-1", "Hello world")
	cache.Set("test-1", result)

	// Should be available immediately
	assert.NotNil(t, cache.Get("test-1"))

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired now
	assert.Nil(t, cache.Get("test-1"))

	// Verify stats show the miss
	stats := cache.GetStats()
	assert.Equal(t, int64(1), stats["misses"])
}

func TestResultCache_Clear(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(nil))
	cache := NewResultCache(10, time.Minute, logger)

	// Add some items
	cache.Set("test-1", createTestResult("test-1", "First"))
	cache.Set("test-2", createTestResult("test-2", "Second"))

	stats := cache.GetStats()
	assert.Equal(t, 2, stats["size"])

	// Clear cache
	cache.Clear()

	// Verify empty
	assert.Nil(t, cache.Get("test-1"))
	assert.Nil(t, cache.Get("test-2"))

	stats = cache.GetStats()
	assert.Equal(t, 0, stats["size"])
}

func TestResultCache_AutoCleanup(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(nil))
	cache := NewResultCache(10, 50*time.Millisecond, logger)

	// Add items
	cache.Set("test-1", createTestResult("test-1", "First"))
	cache.Set("test-2", createTestResult("test-2", "Second"))

	initialStats := cache.GetStats()
	assert.Equal(t, 2, initialStats["size"])

	// Wait for expiration and auto-cleanup (cleanup runs every minute, but we can test expiration)
	time.Sleep(100 * time.Millisecond)

	// Try to get expired items - should return nil
	assert.Nil(t, cache.Get("test-1"))
	assert.Nil(t, cache.Get("test-2"))

	// Verify misses were recorded
	stats := cache.GetStats()
	assert.Equal(t, int64(2), stats["misses"])
}
func TestResultCache_ConcurrentAccess(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(nil))
	cache := NewResultCache(100, time.Minute, logger)

	// Test concurrent writes and reads
	done := make(chan bool, 20)

	// Start 10 writers
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			for j := 0; j < 10; j++ {
				key := fmt.Sprintf("test-%d-%d", id, j)
				result := createTestResult(key, fmt.Sprintf("Text %d-%d", id, j))
				cache.Set(key, result)
			}
		}(i)
	}

	// Start 10 readers
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			for j := 0; j < 10; j++ {
				key := fmt.Sprintf("test-%d-%d", id, j)
				cache.Get(key) // May or may not find it due to timing
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Verify cache is in a consistent state
	stats := cache.GetStats()
	assert.GreaterOrEqual(t, stats["size"].(int), 0)
	assert.LessOrEqual(t, stats["size"].(int), 100)
}

func TestResultCache_HitRate(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(nil))
	cache := NewResultCache(10, time.Minute, logger)

	result := createTestResult("test-1", "Hello world")
	cache.Set("test-1", result)

	// Multiple hits
	cache.Get("test-1")
	cache.Get("test-1")
	cache.Get("test-1")

	// One miss
	cache.Get("non-existent")

	stats := cache.GetStats()
	assert.Equal(t, int64(3), stats["hits"])
	assert.Equal(t, int64(1), stats["misses"])

	hitRate := stats["hit_rate"].(float64)
	assert.Equal(t, 75.0, hitRate) // 3/(3+1) * 100 = 75%
}
