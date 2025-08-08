package replay

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-org/hema-replay-system/internal/config"
	"github.com/your-org/hema-replay-system/internal/obs"
	"github.com/your-org/hema-replay-system/pkg/logger"
)

func TestNewBuffer(t *testing.T) {
	replayConfig := config.ReplayConfig{
		BufferDuration: 60 * time.Second,
		PreRollSeconds: 5,
		MinInterval:    15 * time.Second,
		QueueSize:      10,
	}

	obsConfig := config.OBSConfig{
		Host: "localhost",
		Port: 4455,
	}

	logger, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "console",
	})
	require.NoError(t, err)

	obsClient, err := obs.NewClient(obsConfig, logger)
	require.NoError(t, err)

	buffer := NewBuffer(replayConfig, obsClient, logger)
	require.NotNil(t, buffer)
	assert.Equal(t, BufferStopped, buffer.status)
	assert.Equal(t, 0, buffer.saveCount)
	assert.Equal(t, 0, buffer.errorCount)
}

func TestBuffer_GetInfo(t *testing.T) {
	replayConfig := config.ReplayConfig{
		BufferDuration: 60 * time.Second,
		PreRollSeconds: 5,
		MinInterval:    15 * time.Second,
		QueueSize:      10,
	}

	obsConfig := config.OBSConfig{
		Host: "localhost",
		Port: 4455,
	}

	logger, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "console",
	})
	require.NoError(t, err)

	obsClient, err := obs.NewClient(obsConfig, logger)
	require.NoError(t, err)

	buffer := NewBuffer(replayConfig, obsClient, logger)

	info := buffer.GetInfo()
	assert.Equal(t, BufferStopped, info.Status)
	assert.Equal(t, 0, info.SaveCount)
	assert.Equal(t, 0, info.ErrorCount)
	assert.False(t, info.IsActive)
	assert.False(t, info.CanSave)
}

func TestBuffer_Reset(t *testing.T) {
	replayConfig := config.ReplayConfig{
		BufferDuration: 60 * time.Second,
		PreRollSeconds: 5,
		MinInterval:    15 * time.Second,
		QueueSize:      10,
	}

	obsConfig := config.OBSConfig{
		Host: "localhost",
		Port: 4455,
	}

	logger, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "console",
	})
	require.NoError(t, err)

	obsClient, err := obs.NewClient(obsConfig, logger)
	require.NoError(t, err)

	buffer := NewBuffer(replayConfig, obsClient, logger)

	// Simulate some activity
	buffer.saveCount = 5
	buffer.errorCount = 2
	buffer.avgSaveTime = 100 * time.Millisecond
	buffer.lastSaved = time.Now()

	buffer.Reset()

	assert.Equal(t, 0, buffer.saveCount)
	assert.Equal(t, 0, buffer.errorCount)
	assert.Equal(t, time.Duration(0), buffer.avgSaveTime)
	assert.True(t, buffer.lastSaved.IsZero())
}

func TestBuffer_IsReady(t *testing.T) {
	replayConfig := config.ReplayConfig{
		BufferDuration: 60 * time.Second,
		PreRollSeconds: 5,
		MinInterval:    1 * time.Second, // Short interval for testing
		QueueSize:      10,
	}

	obsConfig := config.OBSConfig{
		Host: "localhost",
		Port: 4455,
	}

	logger, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "console",
	})
	require.NoError(t, err)

	obsClient, err := obs.NewClient(obsConfig, logger)
	require.NoError(t, err)

	buffer := NewBuffer(replayConfig, obsClient, logger)

	// Buffer not started
	assert.False(t, buffer.IsReady())

	// Simulate started buffer
	buffer.status = BufferStarted
	buffer.lastSaved = time.Now().Add(-2 * time.Second) // Past the minimum interval

	// Still not ready because OBS not connected
	assert.False(t, buffer.IsReady())
}

// Integration tests that require OBS Studio
func TestBuffer_StartStop_Integration(t *testing.T) {
	t.Skip("Integration test - requires running OBS Studio")

	// This test would require a running OBS Studio instance
	// In a real test environment, you might want to mock the OBS client
}

func TestBuffer_Save_Integration(t *testing.T) {
	t.Skip("Integration test - requires running OBS Studio")

	// This test would require a running OBS Studio instance
	// In a real test environment, you might want to mock the OBS client
}
