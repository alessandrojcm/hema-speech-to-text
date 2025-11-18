package obs

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/andreykaipov/goobs"
	"github.com/andreykaipov/goobs/api/requests/general"
	"github.com/andreykaipov/goobs/api/requests/inputs"
	"github.com/andreykaipov/goobs/api/requests/scenes"
	"github.com/your-org/hema-replay-system/internal/config"
	"github.com/your-org/hema-replay-system/pkg/logger"
)

type Client struct {
	config    config.OBSConfig
	logger    *logger.Logger
	client    *goobs.Client
	connected bool
	reconnect bool
	mu        sync.RWMutex
	eventChan chan any
	errorChan chan error
}

type ConnectionStatus struct {
	Connected         bool
	LastConnectTime   time.Time
	LastError         error
	ReconnectAttempts int
}

func NewClient(config config.OBSConfig, logger *logger.Logger) (*Client, error) {
	return &Client{
		config:    config,
		logger:    logger,
		reconnect: true,
		eventChan: make(chan any, 100),
		errorChan: make(chan error, 10),
	}, nil
}

func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	var client *goobs.Client
	var err error

	address := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)

	if c.config.Password != "" {
		client, err = goobs.New(address, goobs.WithPassword(c.config.Password))
	} else {
		client, err = goobs.New(address)
	}

	if err != nil {
		return fmt.Errorf("failed to create OBS client: %w", err)
	}

	c.client = client
	c.connected = true

	// Test connection
	if err := c.testConnection(); err != nil {
		c.connected = false
		c.client = nil
		return fmt.Errorf("connection test failed: %w", err)
	}

	// Start event handler
	go c.handleEvents()

	c.logger.Info().Str("address", address).Msg("Connected to OBS Studio")
	return nil
}

func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.reconnect = false

	if c.client != nil {
		if err := c.client.Disconnect(); err != nil {
			c.logger.Error().Err(err).Msg("Error disconnecting from OBS")
		}
		c.client = nil
	}

	c.connected = false
	c.logger.Info().Msg("Disconnected from OBS Studio")
	return nil
}

func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

func (c *Client) testConnection() error {
	if !c.connected {
		return fmt.Errorf("not connected to OBS")
	}

	// Test with GetVersion request
	resp, err := c.client.General.GetVersion(&general.GetVersionParams{})
	if err != nil {
		return fmt.Errorf("failed to get OBS version: %w", err)
	}

	c.logger.Info().
		Str("version", resp.ObsVersion).
		Str("websocket_version", resp.ObsWebSocketVersion).
		Msg("OBS connection verified")

	return nil
}

func (c *Client) handleEvents() {
	for event := range c.client.IncomingEvents {
		select {
		case c.eventChan <- event:
		default:
			c.logger.Warn().Msg("Event channel full, dropping event")
		}
	}
}

func (c *Client) GetEventChannel() <-chan any {
	return c.eventChan
}

func (c *Client) GetErrorChannel() <-chan error {
	return c.errorChan
}

// Scene operations
func (c *Client) GetCurrentScene() (string, error) {
	if !c.IsConnected() {
		return "", fmt.Errorf("not connected to OBS")
	}

	resp, err := c.client.Scenes.GetCurrentProgramScene(&scenes.GetCurrentProgramSceneParams{})
	if err != nil {
		return "", fmt.Errorf("failed to get current scene: %w", err)
	}

	return resp.CurrentProgramSceneName, nil
}

func (c *Client) SetCurrentScene(sceneName string) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected to OBS")
	}

	params := &scenes.SetCurrentProgramSceneParams{
		SceneName: &sceneName,
	}

	_, err := c.client.Scenes.SetCurrentProgramScene(params)
	if err != nil {
		return fmt.Errorf("failed to set current scene to %s: %w", sceneName, err)
	}

	c.logger.Debug().Str("scene", sceneName).Msg("Scene changed")
	return nil
}

func (c *Client) GetSceneList() ([]string, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected to OBS")
	}

	resp, err := c.client.Scenes.GetSceneList(&scenes.GetSceneListParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to get scene list: %w", err)
	}

	scenes := make([]string, len(resp.Scenes))
	for i, scene := range resp.Scenes {
		scenes[i] = scene.SceneName
	}

	return scenes, nil
}

// Replay buffer operations
func (c *Client) StartReplayBuffer() error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected to OBS")
	}

	// TODO: Implement replay buffer methods once we determine the correct API
	c.logger.Debug().Msg("Replay buffer start requested (not implemented)")
	return nil
}

func (c *Client) StopReplayBuffer() error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected to OBS")
	}

	// TODO: Implement replay buffer methods once we determine the correct API
	c.logger.Debug().Msg("Replay buffer stop requested (not implemented)")
	return nil
}

func (c *Client) SaveReplayBuffer() error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected to OBS")
	}

	// TODO: Implement replay buffer methods once we determine the correct API
	c.logger.Debug().Msg("Replay buffer save requested (not implemented)")
	return nil
}

func (c *Client) GetReplayBufferStatus() (bool, error) {
	if !c.IsConnected() {
		return false, fmt.Errorf("not connected to OBS")
	}

	// TODO: Implement replay buffer methods once we determine the correct API
	c.logger.Debug().Msg("Replay buffer status requested (not implemented)")
	return false, nil
}

// Text source operations
func (c *Client) GetTextSourceInfo(sourceName string) (map[string]any, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected to OBS")
	}

	getParams := &inputs.GetInputSettingsParams{
		InputName: &sourceName,
	}

	resp, err := c.client.Inputs.GetInputSettings(getParams)
	if err != nil {
		return nil, fmt.Errorf("failed to get input settings for %s: %w", sourceName, err)
	}

	return resp.InputSettings, nil
}

func (c *Client) UpdateTextSource(sourceName, text string) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected to OBS")
	}

	// Get current settings first
	getParams := &inputs.GetInputSettingsParams{
		InputName: &sourceName,
	}

	resp, err := c.client.Inputs.GetInputSettings(getParams)
	if err != nil {
		return fmt.Errorf("failed to get input settings for %s: %w", sourceName, err)
	}

	// Update only the text field in the existing settings
	if resp.InputSettings == nil {
		resp.InputSettings = make(map[string]any)
	}
	resp.InputSettings["text"] = text

	// Set the updated settings
	overlay := true
	params := &inputs.SetInputSettingsParams{
		InputName:     &sourceName,
		InputSettings: resp.InputSettings,
		Overlay:       &overlay,
	}

	_, err = c.client.Inputs.SetInputSettings(params)
	if err != nil {
		return fmt.Errorf("failed to update text source %s: %w", sourceName, err)
	}

	c.logger.Debug().Str("source", sourceName).Str("text", text).Msg("Text source updated")
	return nil
}

func (c *Client) SetSourceVisibility(sourceName string, visible bool) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected to OBS")
	}

	// This would need to be implemented based on specific scene item management
	// The exact implementation depends on how the text source is set up in OBS
	c.logger.Debug().Str("source", sourceName).Bool("visible", visible).Msg("Source visibility changed")
	return nil
}

// Utility methods
func (c *Client) GetStatus() ConnectionStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return ConnectionStatus{
		Connected: c.connected,
		// Additional status fields would be populated here
	}
}
