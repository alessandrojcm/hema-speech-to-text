package scene

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-org/hema-replay-system/internal/config"
	"github.com/your-org/hema-replay-system/internal/obs"
	"github.com/your-org/hema-replay-system/pkg/logger"
)

func TestNewManager(t *testing.T) {
	sceneConfig := config.SceneConfig{
		MainScene:   "Main",
		ReplayScene: "Replay",
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

	manager := NewManager(sceneConfig, obsClient, logger)
	require.NotNil(t, manager)
	assert.Equal(t, sceneConfig, manager.config)
	assert.Equal(t, SceneMain, manager.state)
	assert.False(t, manager.running)
}

func TestManager_GetState(t *testing.T) {
	sceneConfig := config.SceneConfig{
		MainScene:   "Main",
		ReplayScene: "Replay",
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

	manager := NewManager(sceneConfig, obsClient, logger)

	// Test initial state
	assert.Equal(t, SceneMain, manager.GetState())

	// Test state changes
	manager.currentScene = "Main"
	manager.determineState()
	assert.Equal(t, SceneMain, manager.GetState())

	manager.currentScene = "Replay"
	manager.determineState()
	assert.Equal(t, SceneReplay, manager.GetState())

	manager.currentScene = "Unknown"
	manager.determineState()
	assert.Equal(t, SceneError, manager.GetState())
}

func TestManager_SceneChecks(t *testing.T) {
	sceneConfig := config.SceneConfig{
		MainScene:   "Main",
		ReplayScene: "Replay",
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

	manager := NewManager(sceneConfig, obsClient, logger)

	// Test main scene check
	manager.currentScene = "Main"
	assert.True(t, manager.IsInMainScene())
	assert.False(t, manager.IsInReplayScene())

	// Test replay scene check
	manager.currentScene = "Replay"
	assert.False(t, manager.IsInMainScene())
	assert.True(t, manager.IsInReplayScene())

	// Test other scene
	manager.currentScene = "Other"
	assert.False(t, manager.IsInMainScene())
	assert.False(t, manager.IsInReplayScene())
}

func TestManager_GetStatus(t *testing.T) {
	sceneConfig := config.SceneConfig{
		MainScene:   "Main",
		ReplayScene: "Replay",
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

	manager := NewManager(sceneConfig, obsClient, logger)

	// Set some test state
	manager.running = true
	manager.currentScene = "Main"
	manager.state = SceneMain
	manager.transitionCount = 5
	manager.errorCount = 1

	status := manager.GetStatus()
	assert.True(t, status.Running)
	assert.Equal(t, "Main", status.CurrentScene)
	assert.Equal(t, SceneMain, status.State)
	assert.Equal(t, 5, status.TransitionCount)
	assert.Equal(t, 1, status.ErrorCount)
}

func TestManager_IsRunning(t *testing.T) {
	sceneConfig := config.SceneConfig{
		MainScene:   "Main",
		ReplayScene: "Replay",
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

	manager := NewManager(sceneConfig, obsClient, logger)

	assert.False(t, manager.IsRunning())

	manager.running = true
	assert.True(t, manager.IsRunning())
}

func TestSceneTransition(t *testing.T) {
	transition := SceneTransition{
		FromScene: "Main",
		ToScene:   "Replay",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(100 * time.Millisecond),
		Reason:    "test transition",
	}

	assert.Equal(t, "Main", transition.FromScene)
	assert.Equal(t, "Replay", transition.ToScene)
	assert.Equal(t, "test transition", transition.Reason)
	assert.True(t, transition.EndTime.After(transition.StartTime))
}

// Integration tests that require OBS Studio
func TestManager_Start_Integration(t *testing.T) {
	t.Skip("Integration test - requires running OBS Studio")

	// This test would require a running OBS Studio instance
	// In a real test environment, you might want to mock the OBS client
}

func TestManager_SwitchScene_Integration(t *testing.T) {
	t.Skip("Integration test - requires running OBS Studio")

	// This test would require a running OBS Studio instance
	// In a real test environment, you might want to mock the OBS client
}

func TestManager_ValidateScenes_Integration(t *testing.T) {
	t.Skip("Integration test - requires running OBS Studio")

	// This test would require a running OBS Studio instance
	// In a real test environment, you might want to mock the OBS client
}
