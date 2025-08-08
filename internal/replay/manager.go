package replay

import (
	"context"
	"fmt"
	"time"

	"github.com/your-org/hema-replay-system/internal/config"
	"github.com/your-org/hema-replay-system/internal/obs"
	"github.com/your-org/hema-replay-system/pkg/logger"
)

// Manager provides a high-level interface for replay operations,
// coordinating between the buffer and queue components.
type Manager struct {
	config    config.ReplayConfig
	obsClient *obs.Client
	logger    *logger.Logger
	queue     *Queue

	// State
	running   bool
	startTime time.Time
}

type ManagerStatus struct {
	Running      bool
	StartTime    time.Time
	Uptime       time.Duration
	QueueInfo    QueueInfo
	IsReady      bool
	LastActivity time.Time
}

// NewManager creates a new replay manager with the given configuration.
func NewManager(config config.ReplayConfig, obsClient *obs.Client, logger *logger.Logger) *Manager {
	return &Manager{
		config:    config,
		obsClient: obsClient,
		logger:    logger,
		queue:     NewQueue(config, obsClient, logger),
	}
}

func (m *Manager) Start(ctx context.Context) error {
	if m.running {
		return fmt.Errorf("replay manager already running")
	}

	if err := m.queue.Start(ctx); err != nil {
		return fmt.Errorf("failed to start replay queue: %w", err)
	}

	m.running = true
	m.startTime = time.Now()

	m.logger.Info().Msg("Replay manager started")
	return nil
}

func (m *Manager) Stop(ctx context.Context) error {
	if !m.running {
		return nil
	}

	if err := m.queue.Stop(ctx); err != nil {
		m.logger.Error().Err(err).Msg("Failed to stop replay queue")
	}

	m.running = false
	m.logger.Info().Msg("Replay manager stopped")
	return nil
}

// TriggerReplay queues a new replay request with the given message.
func (m *Manager) TriggerReplay(message string) error {
	if !m.running {
		return fmt.Errorf("replay manager not running")
	}

	request := ReplayRequest{
		Message: message,
	}

	return m.queue.AddRequest(request)
}

func (m *Manager) ProcessQueue(ctx context.Context) error {
	if !m.running {
		return fmt.Errorf("replay manager not running")
	}

	// The queue processes itself, this is mainly for status checks
	return nil
}

func (m *Manager) GetStatus() ManagerStatus {
	if !m.running {
		return ManagerStatus{
			Running:   false,
			StartTime: m.startTime,
		}
	}

	queueInfo := m.queue.GetQueueInfo()

	return ManagerStatus{
		Running:      true,
		StartTime:    m.startTime,
		Uptime:       time.Since(m.startTime),
		QueueInfo:    queueInfo,
		IsReady:      m.queue.buffer.IsReady(),
		LastActivity: time.Now(), // This would be updated with actual activity
	}
}

func (m *Manager) GetRecentResults(limit int) []ReplayResult {
	return m.queue.GetResults(limit)
}

func (m *Manager) ClearResults() {
	m.queue.ClearCompleted()
}

func (m *Manager) IsRunning() bool {
	return m.running
}

func (m *Manager) IsReady() bool {
	return m.running && m.queue.buffer.IsReady()
}
