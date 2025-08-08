// Package scene provides scene management functionality for OBS Studio,
// including automated scene switching for replay sequences.
package scene

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/your-org/hema-replay-system/internal/config"
	"github.com/your-org/hema-replay-system/internal/obs"
	"github.com/your-org/hema-replay-system/pkg/logger"
)

type SceneState int

const (
	SceneMain SceneState = iota
	SceneReplay
	SceneTransitioning
	SceneError
)

type SceneTransition struct {
	FromScene string
	ToScene   string
	StartTime time.Time
	EndTime   time.Time
	Reason    string
}

// Manager handles OBS scene transitions and state tracking,
// providing automated scene switching for replay sequences.
type Manager struct {
	config    config.SceneConfig
	obsClient *obs.Client
	logger    *logger.Logger

	// State
	running        bool
	currentScene   string
	state          SceneState
	lastTransition *SceneTransition
	mu             sync.RWMutex

	// Channels
	transitionChan chan SceneTransition
	stopChan       chan struct{}

	// Metrics
	transitionCount int
	errorCount      int
}

type ManagerStatus struct {
	Running         bool
	CurrentScene    string
	State           SceneState
	LastTransition  *SceneTransition
	TransitionCount int
	ErrorCount      int
	AvailableScenes []string
}

// NewManager creates a new scene manager with the given configuration.
func NewManager(config config.SceneConfig, obsClient *obs.Client, logger *logger.Logger) *Manager {
	return &Manager{
		config:         config,
		obsClient:      obsClient,
		logger:         logger,
		transitionChan: make(chan SceneTransition, 5),
		stopChan:       make(chan struct{}),
		state:          SceneMain,
	}
}

func (m *Manager) Start(ctx context.Context) error {
	if m.running {
		return fmt.Errorf("scene manager already running")
	}

	// Verify OBS connection
	if !m.obsClient.IsConnected() {
		return fmt.Errorf("OBS client not connected")
	}

	// Get current scene from OBS
	currentScene, err := m.obsClient.GetCurrentScene()
	if err != nil {
		return fmt.Errorf("failed to get current scene: %w", err)
	}

	m.currentScene = currentScene
	m.determineState()
	m.running = true

	// Start transition processor
	go m.processTransitions(ctx)

	m.logger.Info().
		Str("current_scene", m.currentScene).
		Str("main_scene", m.config.MainScene).
		Str("replay_scene", m.config.ReplayScene).
		Msg("Scene manager started")

	return nil
}

func (m *Manager) Stop(ctx context.Context) error {
	if !m.running {
		return nil
	}

	close(m.stopChan)

	// Return to main scene if not already there
	if m.currentScene != m.config.MainScene {
		if err := m.switchToMainScene("shutdown"); err != nil {
			m.logger.Error().Err(err).Msg("Failed to return to main scene on shutdown")
		}
	}

	m.running = false
	m.logger.Info().Msg("Scene manager stopped")
	return nil
}

// SwitchToReplayScene transitions to the configured replay scene.
func (m *Manager) SwitchToReplayScene(reason string) error {
	if !m.running {
		return fmt.Errorf("scene manager not running")
	}

	return m.switchScene(m.config.ReplayScene, reason)
}

// SwitchToMainScene transitions to the configured main scene.
func (m *Manager) SwitchToMainScene(reason string) error {
	if !m.running {
		return fmt.Errorf("scene manager not running")
	}

	return m.switchToMainScene(reason)
}

func (m *Manager) switchToMainScene(reason string) error {
	return m.switchScene(m.config.MainScene, reason)
}

func (m *Manager) switchScene(targetScene, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.currentScene == targetScene {
		m.logger.Debug().
			Str("scene", targetScene).
			Msg("Already in target scene")
		return nil
	}

	transition := SceneTransition{
		FromScene: m.currentScene,
		ToScene:   targetScene,
		StartTime: time.Now(),
		Reason:    reason,
	}

	// Perform the scene switch
	m.state = SceneTransitioning

	if err := m.obsClient.SetCurrentScene(targetScene); err != nil {
		m.errorCount++
		m.state = SceneError
		return fmt.Errorf("failed to switch to scene %s: %w", targetScene, err)
	}

	// Update state
	transition.EndTime = time.Now()
	m.currentScene = targetScene
	m.lastTransition = &transition
	m.transitionCount++
	m.determineState()

	m.logger.Info().
		Str("from_scene", transition.FromScene).
		Str("to_scene", transition.ToScene).
		Str("reason", reason).
		Dur("duration", transition.EndTime.Sub(transition.StartTime)).
		Msg("Scene transition completed")

	return nil
}

func (m *Manager) processTransitions(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case transition := <-m.transitionChan:
			m.handleTransition(transition)
		}
	}
}

func (m *Manager) handleTransition(transition SceneTransition) {
	// This could be used for queued transitions in the future
	// For Phase 1, we handle transitions synchronously
	m.logger.Debug().
		Str("from_scene", transition.FromScene).
		Str("to_scene", transition.ToScene).
		Msg("Processing scene transition")
}

func (m *Manager) determineState() {
	switch m.currentScene {
	case m.config.MainScene:
		m.state = SceneMain
	case m.config.ReplayScene:
		m.state = SceneReplay
	default:
		m.state = SceneError
	}
}

func (m *Manager) GetCurrentScene() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentScene
}

func (m *Manager) GetState() SceneState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

func (m *Manager) IsInMainScene() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentScene == m.config.MainScene
}

func (m *Manager) IsInReplayScene() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentScene == m.config.ReplayScene
}

func (m *Manager) GetStatus() ManagerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get available scenes from OBS
	availableScenes, _ := m.obsClient.GetSceneList()

	return ManagerStatus{
		Running:         m.running,
		CurrentScene:    m.currentScene,
		State:           m.state,
		LastTransition:  m.lastTransition,
		TransitionCount: m.transitionCount,
		ErrorCount:      m.errorCount,
		AvailableScenes: availableScenes,
	}
}

func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

func (m *Manager) ValidateScenes() error {
	if !m.obsClient.IsConnected() {
		return fmt.Errorf("OBS client not connected")
	}

	scenes, err := m.obsClient.GetSceneList()
	if err != nil {
		return fmt.Errorf("failed to get scene list: %w", err)
	}

	// Check if configured scenes exist
	mainExists := false
	replayExists := false

	for _, scene := range scenes {
		if scene == m.config.MainScene {
			mainExists = true
		}
		if scene == m.config.ReplayScene {
			replayExists = true
		}
	}

	if !mainExists {
		return fmt.Errorf("main scene '%s' not found in OBS", m.config.MainScene)
	}

	if !replayExists {
		return fmt.Errorf("replay scene '%s' not found in OBS", m.config.ReplayScene)
	}

	return nil
}

// Utility methods for integration with replay system
func (m *Manager) PrepareForReplay(reason string) error {
	return m.SwitchToReplayScene(reason)
}

func (m *Manager) ReturnFromReplay(reason string) error {
	return m.SwitchToMainScene(reason)
}
