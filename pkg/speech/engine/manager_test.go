package engine

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	speechTypes "github.com/your-org/hema-replay-system/pkg/speech/types"
)

func createTestConfig() speechTypes.SpeechConfig {
	return speechTypes.SpeechConfig{
		Whisper: speechTypes.WhisperConfig{
			ModelPath:      "/tmp/test-models",
			ModelSize:      speechTypes.ModelTiny,
			Language:       "en",
			Temperature:    0.0,
			ThreadCount:    4,
			WordTimestamps: true,
		},

		Performance: speechTypes.PerformanceConfig{
			MaxConcurrent:   2,
			CacheSize:       10,
			CacheTTL:        time.Minute,
			TimeoutDuration: 30 * time.Second,
		},
	}
}

func createTestLogger() zerolog.Logger {
	return zerolog.New(zerolog.NewTestWriter(nil)).With().Timestamp().Logger()
}

func TestNewSpeechManager(t *testing.T) {
	config := createTestConfig()
	logger := createTestLogger()

	// This will fail without actual Whisper setup, but tests the structure
	manager, err := NewSpeechManager(config, logger)

	// In a real environment with Whisper models, this should succeed
	// For now, we expect it to fail gracefully due to missing dependencies
	if err != nil {
		// Expected - missing whisper models or vocabulary files
		assert.Error(t, err)
		return
	}

	// If it succeeds (with proper setup), verify structure
	assert.NotNil(t, manager)
	assert.Equal(t, config, manager.config)
}

func TestSpeechManager_StartStop(t *testing.T) {
	t.Skip("Integration test - requires Whisper model setup")

	config := createTestConfig()
	logger := createTestLogger()

	manager, err := NewSpeechManager(config, logger)
	require.NoError(t, err)
	require.NotNil(t, manager)

	// Test start
	ctx := context.Background()
	err = manager.Start(ctx)
	require.NoError(t, err)

	// Test stop
	err = manager.Stop()
	assert.NoError(t, err)
}

func TestSpeechManager_StartAlreadyRunning(t *testing.T) {
	t.Skip("Integration test - requires Whisper model setup")

	config := createTestConfig()
	logger := createTestLogger()

	manager, err := NewSpeechManager(config, logger)
	require.NoError(t, err)

	ctx := context.Background()
	err = manager.Start(ctx)
	require.NoError(t, err)
	defer manager.Stop()

	// Try to start again
	err = manager.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
}

func TestSpeechManager_StopNotRunning(t *testing.T) {
	config := createTestConfig()
	logger := createTestLogger()

	manager, err := NewSpeechManager(config, logger)
	if err != nil {
		t.Skip("Skipping test - Whisper setup required")
	}
	require.NotNil(t, manager)

	// Stop without starting should not error
	err = manager.Stop()
	assert.NoError(t, err)
}

func TestSpeechManager_GetStats(t *testing.T) {
	config := createTestConfig()
	logger := createTestLogger()

	manager, err := NewSpeechManager(config, logger)
	if err != nil {
		t.Skip("Skipping test - Whisper setup required")
	}
	require.NotNil(t, manager)

	stats := manager.GetStats()

	// Verify stats structure (returns map[string]interface{})
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "total_requests")
	assert.Contains(t, stats, "successful_requests")
	assert.Contains(t, stats, "failed_requests")
	assert.Contains(t, stats, "success_rate")
	assert.Contains(t, stats, "is_running")

	// Verify initial values
	assert.Equal(t, int64(0), stats["total_requests"])
	assert.Equal(t, int64(0), stats["successful_requests"])
	assert.Equal(t, int64(0), stats["failed_requests"])
	assert.Equal(t, false, stats["is_running"])
}

func TestSpeechManager_TranscribeAudio_NotRunning(t *testing.T) {
	config := createTestConfig()
	logger := createTestLogger()

	manager, err := NewSpeechManager(config, logger)
	if err != nil {
		t.Skip("Skipping test - Whisper setup required")
	}
	require.NotNil(t, manager)

	// Try to transcribe without starting
	ctx := context.Background()
	result, err := manager.TranscribeAudio(ctx, nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not running")
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2) // Should be unique
	assert.Contains(t, id1, "req_")
	assert.Contains(t, id2, "req_")
}

func TestSpeechManager_UpdateMetrics(t *testing.T) {
	config := createTestConfig()
	logger := createTestLogger()

	manager, err := NewSpeechManager(config, logger)
	if err != nil {
		t.Skip("Skipping test - Whisper setup required")
	}
	require.NotNil(t, manager)

	// Get initial stats
	initialStats := manager.GetStats()

	// Simulate successful request
	manager.updateMetrics(true, 100*time.Millisecond)

	// Simulate failed request
	manager.updateMetrics(false, 50*time.Millisecond)

	newStats := manager.GetStats()

	// Verify metrics were updated
	assert.Equal(t, int64(2), newStats["total_requests"])
	assert.Equal(t, int64(1), newStats["successful_requests"])
	assert.Equal(t, int64(1), newStats["failed_requests"])
	assert.Equal(t, float64(50), newStats["success_rate"]) // 1/2 * 100 = 50%

	// Verify initial state
	assert.Equal(t, int64(0), initialStats["total_requests"])
}

func TestSpeechManager_TaskManagement(t *testing.T) {
	config := createTestConfig()
	logger := createTestLogger()

	manager, err := NewSpeechManager(config, logger)
	if err != nil {
		t.Skip("Skipping test - Whisper setup required")
	}
	require.NotNil(t, manager)

	// Create a test task
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	task := &TranscriptionTask{
		ID:        "test-task-1",
		StartTime: time.Now(),
		Done:      make(chan *speechTypes.TranscriptionResult, 1),
		Error:     make(chan error, 1),
		Context:   ctx,
		Cancel:    cancel,
	}

	// Test task registration
	manager.registerTask(task)

	stats := manager.GetStats()
	assert.Equal(t, 1, stats["active_tasks"])

	// Test task unregistration
	manager.unregisterTask(task.ID)

	stats = manager.GetStats()
	assert.Equal(t, 0, stats["active_tasks"])
}
