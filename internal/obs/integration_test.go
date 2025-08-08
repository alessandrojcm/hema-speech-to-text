//go:build integration
// +build integration

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

// This file contains real integration tests that require OBS Studio to be running
// Run with: go test -tags=integration ./internal/obs/

func TestRealOBSConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := config.OBSConfig{
		Host:     "localhost",
		Port:     4455,
		Password: "", // Set this if your OBS has a password
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

	// This will fail if OBS is not running
	err = client.Connect(ctx)
	if err != nil {
		t.Skipf("OBS Studio not running or not accessible: %v", err)
	}
	defer client.Disconnect()

	// Test that we're actually connected
	assert.True(t, client.IsConnected())

	// Test getting scene list
	scenes, err := client.GetSceneList()
	require.NoError(t, err)
	assert.NotEmpty(t, scenes, "Expected at least one scene in OBS")

	// Test getting current scene
	currentScene, err := client.GetCurrentScene()
	require.NoError(t, err)
	assert.NotEmpty(t, currentScene, "Expected a current scene")

	// Test that current scene is in scene list
	assert.Contains(t, scenes, currentScene)
}

func TestRealOBSSceneOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

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
	if err != nil {
		t.Skipf("OBS Studio not running: %v", err)
	}
	defer client.Disconnect()

	// Get initial scene
	originalScene, err := client.GetCurrentScene()
	require.NoError(t, err)

	// Get scene list
	scenes, err := client.GetSceneList()
	require.NoError(t, err)
	require.Greater(t, len(scenes), 1, "Need at least 2 scenes to test scene switching")

	// Find a different scene to switch to
	var targetScene string
	for _, scene := range scenes {
		if scene != originalScene {
			targetScene = scene
			break
		}
	}
	require.NotEmpty(t, targetScene, "Need at least one scene different from current")

	// Switch to target scene
	err = client.SetCurrentScene(targetScene)
	require.NoError(t, err)

	// Verify scene changed
	currentScene, err := client.GetCurrentScene()
	require.NoError(t, err)
	assert.Equal(t, targetScene, currentScene)

	// Switch back to original scene
	err = client.SetCurrentScene(originalScene)
	require.NoError(t, err)

	// Verify we're back to original
	currentScene, err = client.GetCurrentScene()
	require.NoError(t, err)
	assert.Equal(t, originalScene, currentScene)
}

func TestRealOBSTextSource(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

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
	if err != nil {
		t.Skipf("OBS Studio not running: %v", err)
	}
	defer client.Disconnect()

	// Test updating a text source
	// Note: This requires a text source named "TestText" to exist in OBS
	testSourceName := "TestText"
	testText := "Integration test message"

	err = client.UpdateTextSource(testSourceName, testText)
	// We don't require no error here since the source might not exist
	if err != nil {
		t.Logf("Text source update failed (expected if source doesn't exist): %v", err)
	}
}
