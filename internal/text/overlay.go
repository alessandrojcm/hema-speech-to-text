// Package text provides text overlay management for OBS Studio,
// including formatting, display timing, and automatic clearing.
package text

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/your-org/hema-replay-system/internal/config"
	"github.com/your-org/hema-replay-system/internal/obs"
	"github.com/your-org/hema-replay-system/pkg/logger"
)

type OverlayStatus int

const (
	OverlayHidden OverlayStatus = iota
	OverlayVisible
	OverlayError
)

type TextRequest struct {
	ID       string
	Text     string
	Duration time.Duration
	Priority int
}

type ActiveOverlay struct {
	Request   TextRequest
	StartTime time.Time
	EndTime   time.Time
}

// Manager handles text overlay operations including display timing,
// formatting, and automatic clearing of text sources in OBS.
type Manager struct {
	config    config.TextConfig
	obsClient *obs.Client
	logger    *logger.Logger
	formatter *Formatter

	// State
	running       bool
	currentText   string
	status        OverlayStatus
	activeOverlay *ActiveOverlay
	mu            sync.RWMutex

	// Channels
	requestChan chan TextRequest
	stopChan    chan struct{}

	// Metrics
	totalRequests int
	displayCount  int
	errorCount    int
}

type ManagerStatus struct {
	Running       bool
	Status        OverlayStatus
	CurrentText   string
	ActiveOverlay *ActiveOverlay
	TotalRequests int
	DisplayCount  int
	ErrorCount    int
}

// NewManager creates a new text overlay manager with the given configuration.
func NewManager(config config.TextConfig, obsClient *obs.Client, logger *logger.Logger) *Manager {
	return &Manager{
		config:      config,
		obsClient:   obsClient,
		logger:      logger,
		formatter:   NewFormatter(config),
		requestChan: make(chan TextRequest, 10),
		stopChan:    make(chan struct{}),
		status:      OverlayHidden,
	}
}

func (m *Manager) Start(ctx context.Context) error {
	if m.running {
		return fmt.Errorf("text manager already running")
	}

	// Verify OBS connection
	if !m.obsClient.IsConnected() {
		return fmt.Errorf("OBS client not connected")
	}

	// Initialize text source (clear any existing text)
	if err := m.clearText(); err != nil {
		return fmt.Errorf("failed to initialize text source: %w", err)
	}

	m.running = true

	// Start request processor
	go m.processRequests(ctx)

	// Start overlay timer
	go m.manageOverlayTimer(ctx)

	m.logger.Info().
		Str("source_name", m.config.SourceName).
		Int("max_length", m.config.MaxLength).
		Msg("Text overlay manager started")

	return nil
}

func (m *Manager) Stop(ctx context.Context) error {
	if !m.running {
		return nil
	}

	close(m.stopChan)

	// Clear any active text
	if err := m.clearText(); err != nil {
		m.logger.Error().Err(err).Msg("Failed to clear text on shutdown")
	}

	m.running = false
	m.logger.Info().Msg("Text overlay manager stopped")
	return nil
}

// DisplayText shows the given text for the specified duration.
func (m *Manager) DisplayText(text string, duration time.Duration) error {
	if !m.running {
		return fmt.Errorf("text manager not running")
	}

	request := TextRequest{
		ID:       fmt.Sprintf("text_%d", time.Now().UnixNano()),
		Text:     text,
		Duration: duration,
		Priority: 1,
	}

	select {
	case m.requestChan <- request:
		return nil
	default:
		return fmt.Errorf("text request queue full")
	}
}

func (m *Manager) DisplayMessage(messageIndex int, duration time.Duration) error {
	if !m.running {
		return fmt.Errorf("text manager not running")
	}

	if messageIndex < 0 || messageIndex >= len(m.config.DefaultMessages) {
		return fmt.Errorf("invalid message index: %d", messageIndex)
	}

	text := m.config.DefaultMessages[messageIndex]
	return m.DisplayText(text, duration)
}

func (m *Manager) ClearText() error {
	if !m.running {
		return fmt.Errorf("text manager not running")
	}

	return m.clearText()
}

func (m *Manager) processRequests(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case request := <-m.requestChan:
			m.handleTextRequest(request)
		}
	}
}

func (m *Manager) handleTextRequest(request TextRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalRequests++

	// Format the text
	formattedText := m.formatter.Format(request.Text)

	// Update OBS text source
	if err := m.obsClient.UpdateTextSource(m.config.SourceName, formattedText); err != nil {
		m.errorCount++
		m.status = OverlayError
		m.logger.Error().
			Err(err).
			Str("text", request.Text).
			Msg("Failed to update text source")
		return
	}

	// Update state
	m.currentText = formattedText
	m.status = OverlayVisible
	m.displayCount++

	// Set active overlay
	m.activeOverlay = &ActiveOverlay{
		Request:   request,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(request.Duration),
	}

	m.logger.Info().
		Str("id", request.ID).
		Str("text", formattedText).
		Dur("duration", request.Duration).
		Msg("Text overlay displayed")
}

func (m *Manager) manageOverlayTimer(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.checkOverlayExpiration()
		}
	}
}

func (m *Manager) checkOverlayExpiration() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.activeOverlay == nil {
		return
	}

	if time.Now().After(m.activeOverlay.EndTime) {
		// Clear the text
		if err := m.obsClient.UpdateTextSource(m.config.SourceName, ""); err != nil {
			m.logger.Error().Err(err).Msg("Failed to clear expired text")
			return
		}

		m.logger.Debug().
			Str("id", m.activeOverlay.Request.ID).
			Msg("Text overlay expired and cleared")

		m.currentText = ""
		m.status = OverlayHidden
		m.activeOverlay = nil
	}
}

func (m *Manager) clearText() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.obsClient.UpdateTextSource(m.config.SourceName, ""); err != nil {
		m.errorCount++
		m.status = OverlayError
		return fmt.Errorf("failed to clear text source: %w", err)
	}

	m.currentText = ""
	m.status = OverlayHidden
	m.activeOverlay = nil

	return nil
}

func (m *Manager) GetStatus() ManagerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return ManagerStatus{
		Running:       m.running,
		Status:        m.status,
		CurrentText:   m.currentText,
		ActiveOverlay: m.activeOverlay,
		TotalRequests: m.totalRequests,
		DisplayCount:  m.displayCount,
		ErrorCount:    m.errorCount,
	}
}

func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

func (m *Manager) GetCurrentText() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentText
}

func (m *Manager) IsVisible() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status == OverlayVisible
}
