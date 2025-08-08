// Package replay provides replay buffer management and queue processing
// for the HEMA Tournament Replay System.
package replay

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/your-org/hema-replay-system/internal/config"
	"github.com/your-org/hema-replay-system/internal/obs"
	"github.com/your-org/hema-replay-system/pkg/logger"
)

// BufferStatus represents the current state of the replay buffer.
type BufferStatus int

const (
	BufferStopped BufferStatus = iota // Buffer is stopped
	BufferStarted                     // Buffer is running and ready
	BufferSaving                      // Buffer is currently saving a replay
	BufferError                       // Buffer encountered an error
)

// Buffer manages the OBS replay buffer functionality, including starting,
// stopping, and saving replay clips with rate limiting and metrics tracking.
type Buffer struct {
	config    config.ReplayConfig
	obsClient *obs.Client
	logger    *logger.Logger
	status    BufferStatus
	lastSaved time.Time
	mu        sync.RWMutex

	// Phase 1: Basic metrics only
	saveCount   int
	errorCount  int
	avgSaveTime time.Duration
}

// BufferInfo provides status and metrics information about the replay buffer.
type BufferInfo struct {
	Status      BufferStatus
	LastSaved   time.Time
	SaveCount   int
	ErrorCount  int
	IsActive    bool
	CanSave     bool
	AvgSaveTime time.Duration
}

// NewBuffer creates a new replay buffer manager with the given configuration.
func NewBuffer(config config.ReplayConfig, obsClient *obs.Client, logger *logger.Logger) *Buffer {
	return &Buffer{
		config:    config,
		obsClient: obsClient,
		logger:    logger,
		status:    BufferStopped,
	}
}

// Start initializes and starts the OBS replay buffer.
func (b *Buffer) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.status == BufferStarted {
		return nil // Already started
	}

	if !b.obsClient.IsConnected() {
		return fmt.Errorf("OBS client not connected")
	}

	if err := b.obsClient.StartReplayBuffer(); err != nil {
		b.status = BufferError
		b.errorCount++
		return fmt.Errorf("failed to start replay buffer: %w", err)
	}

	b.status = BufferStarted
	b.logger.Info().Dur("buffer_duration", b.config.BufferDuration).Msg("Replay buffer started")
	return nil
}

// Stop stops the OBS replay buffer.
func (b *Buffer) Stop(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.status == BufferStopped {
		return nil // Already stopped
	}

	if !b.obsClient.IsConnected() {
		return fmt.Errorf("OBS client not connected")
	}

	if err := b.obsClient.StopReplayBuffer(); err != nil {
		b.status = BufferError
		b.errorCount++
		return fmt.Errorf("failed to stop replay buffer: %w", err)
	}

	b.status = BufferStopped
	b.logger.Info().Msg("Replay buffer stopped")
	return nil
}

// Save triggers a replay buffer save, respecting minimum interval constraints.
func (b *Buffer) Save(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.status != BufferStarted {
		return fmt.Errorf("replay buffer not started")
	}

	// Check minimum interval
	if time.Since(b.lastSaved) < b.config.MinInterval {
		return fmt.Errorf("replay save too frequent, minimum interval is %v", b.config.MinInterval)
	}

	if !b.obsClient.IsConnected() {
		return fmt.Errorf("OBS client not connected")
	}

	b.status = BufferSaving
	saveStart := time.Now()

	if err := b.obsClient.SaveReplayBuffer(); err != nil {
		b.status = BufferStarted // Reset to started state
		b.errorCount++
		return fmt.Errorf("failed to save replay buffer: %w", err)
	}

	saveTime := time.Since(saveStart)
	b.lastSaved = time.Now()
	b.saveCount++

	// Update average save time
	if b.avgSaveTime == 0 {
		b.avgSaveTime = saveTime
	} else {
		b.avgSaveTime = (b.avgSaveTime + saveTime) / 2
	}

	b.status = BufferStarted
	b.logger.Info().
		Dur("save_time", saveTime).
		Int("save_count", b.saveCount).
		Dur("avg_save_time", b.avgSaveTime).
		Msg("Replay buffer saved")

	return nil
}

func (b *Buffer) GetInfo() BufferInfo {
	b.mu.RLock()
	defer b.mu.RUnlock()

	isActive, _ := b.obsClient.GetReplayBufferStatus()
	canSave := b.status == BufferStarted &&
		time.Since(b.lastSaved) >= b.config.MinInterval

	return BufferInfo{
		Status:      b.status,
		LastSaved:   b.lastSaved,
		SaveCount:   b.saveCount,
		ErrorCount:  b.errorCount,
		AvgSaveTime: b.avgSaveTime,
		IsActive:    isActive,
		CanSave:     canSave,
	}
}

func (b *Buffer) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.saveCount = 0
	b.errorCount = 0
	b.avgSaveTime = 0
	b.lastSaved = time.Time{}

	b.logger.Info().Msg("Replay buffer metrics reset")
}

func (b *Buffer) IsReady() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.status == BufferStarted &&
		b.obsClient.IsConnected() &&
		time.Since(b.lastSaved) >= b.config.MinInterval
}
