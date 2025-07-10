package obs

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-org/hema-replay-system/internal/config"
	"github.com/your-org/hema-replay-system/pkg/logger"
)

func TestNewClient(t *testing.T) {
	config := config.OBSConfig{
		Host: "localhost",
		Port: 4455,
	}
	
	logger, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "console",
	})
	require.NoError(t, err)
	
	client, err := NewClient(config, logger)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, config, client.config)
	assert.False(t, client.IsConnected())
}

func TestClient_Connect(t *testing.T) {
	// Note: This test requires OBS Studio to be running
	// In a real test environment, you might want to mock the OBS client
	t.Skip("Integration test - requires running OBS Studio")
	
	config := config.OBSConfig{
		Host: "localhost",
		Port: 4455,
	}
	
	logger, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "console",
	})
	require.NoError(t, err)
	
	client, err := NewClient(config, logger)
	require.NoError(t, err)
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	err = client.Connect(ctx)
	require.NoError(t, err)
	assert.True(t, client.IsConnected())
	
	defer client.Disconnect()
}

func TestClient_SceneOperations(t *testing.T) {
	// Mock test - in a real implementation, you would mock the OBS client
	t.Skip("Integration test - requires running OBS Studio")
	
	// Test implementation would go here
	// This would test GetCurrentScene, SetCurrentScene, GetSceneList
}

func TestClient_ReplayBufferOperations(t *testing.T) {
	// Mock test - in a real implementation, you would mock the OBS client
	t.Skip("Integration test - requires running OBS Studio")
	
	// Test implementation would go here
	// This would test StartReplayBuffer, StopReplayBuffer, SaveReplayBuffer
}

func TestClient_TextSourceOperations(t *testing.T) {
	// Mock test - in a real implementation, you would mock the OBS client
	t.Skip("Integration test - requires running OBS Studio")
	
	// Test implementation would go here
	// This would test UpdateTextSource, SetSourceVisibility
}

// Mock tests for unit testing without OBS dependency
func TestClient_IsConnected(t *testing.T) {
	config := config.OBSConfig{
		Host: "localhost",
		Port: 4455,
	}
	
	logger, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "console",
	})
	require.NoError(t, err)
	
	client, err := NewClient(config, logger)
	require.NoError(t, err)
	
	assert.False(t, client.IsConnected())
	
	// Simulate connection
	client.mu.Lock()
	client.connected = true
	client.mu.Unlock()
	
	assert.True(t, client.IsConnected())
}

func TestClient_GetStatus(t *testing.T) {
	config := config.OBSConfig{
		Host: "localhost",
		Port: 4455,
	}
	
	logger, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "console",
	})
	require.NoError(t, err)
	
	client, err := NewClient(config, logger)
	require.NoError(t, err)
	
	status := client.GetStatus()
	assert.False(t, status.Connected)
	
	// Simulate connection
	client.mu.Lock()
	client.connected = true
	client.mu.Unlock()
	
	status = client.GetStatus()
	assert.True(t, status.Connected)
}