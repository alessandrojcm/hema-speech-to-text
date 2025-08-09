//go:build !noaudio
// +build !noaudio

package audio

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/your-org/hema-replay-system/pkg/audio/buffer"
	"github.com/your-org/hema-replay-system/pkg/audio/capture"
	"github.com/your-org/hema-replay-system/pkg/audio/processing"
	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

type AudioManager struct {
	config        types.AudioConfig
	captureEngine *capture.CaptureEngine
	ringBuffer    *buffer.RingBuffer
	processor     *processing.AudioProcessor
	deviceManager *capture.DeviceManager
	extractor     *AudioExtractor

	mu      sync.RWMutex
	running bool
	health  types.SystemHealth

	statsChan  chan types.CaptureStats
	healthChan chan types.SystemHealth
	errorChan  chan error

	logger zerolog.Logger
}

type AudioExtractor struct {
	config      types.ExtractionConfig
	ringBuffer  *buffer.RingBuffer
	activeTasks map[string]*ExtractionTask
	taskMutex   sync.RWMutex
	semaphore   chan struct{}
	logger      zerolog.Logger
}

type ExtractionTask struct {
	ID        string
	Request   types.ExtractionRequest
	StartTime time.Time
	Done      chan *types.AudioSegment
	Error     chan error
}

func NewAudioManager(config types.AudioConfig, logger zerolog.Logger) (*AudioManager, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	ringBuffer := buffer.NewRingBuffer(
		config.Buffer,
		config.Device.SampleRate,
		config.Device.Channels,
		config.Device.BitDepth,
	)

	processor := processing.NewAudioProcessor(
		config.Processing,
		config.Device.SampleRate,
		config.Device.Channels,
		logger,
	)

	captureEngine := capture.NewCaptureEngine(
		config.Device,
		config.Processing,
		ringBuffer,
		logger,
	)

	deviceManager := capture.NewDeviceManager(config.Device, logger)

	extractor := NewAudioExtractor(config.Extraction, ringBuffer, logger)

	return &AudioManager{
		config:        config,
		captureEngine: captureEngine,
		ringBuffer:    ringBuffer,
		processor:     processor,
		deviceManager: deviceManager,
		extractor:     extractor,
		statsChan:     make(chan types.CaptureStats, 1),
		healthChan:    make(chan types.SystemHealth, 1),
		errorChan:     make(chan error, 10),
		logger:        logger.With().Str("component", "audio_manager").Logger(),
	}, nil
}

func (am *AudioManager) Start(ctx context.Context) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if am.running {
		return types.ErrAlreadyRunning
	}

	if err := am.deviceManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start device manager: %w", err)
	}

	if err := am.captureEngine.Start(ctx); err != nil {
		return fmt.Errorf("failed to start capture engine: %w", err)
	}

	go am.healthMonitorLoop(ctx)

	am.running = true
	am.logger.Info().Msg("Audio manager started")

	return nil
}

func (am *AudioManager) Stop() error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if !am.running {
		return nil
	}

	if err := am.captureEngine.Stop(); err != nil {
		am.logger.Error().Err(err).Msg("Failed to stop capture engine")
	}

	if err := am.deviceManager.Stop(); err != nil {
		am.logger.Error().Err(err).Msg("Failed to stop device manager")
	}

	am.running = false
	am.logger.Info().Msg("Audio manager stopped")

	return nil
}

func (am *AudioManager) ExtractAudio(ctx context.Context, req types.ExtractionRequest) (*types.AudioSegment, error) {
	if !am.running {
		return nil, types.ErrNotRunning
	}

	return am.extractor.Extract(ctx, req)
}

func (am *AudioManager) GetHealth() types.SystemHealth {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.health
}

func (am *AudioManager) GetStats() types.CaptureStats {
	return am.captureEngine.GetStats()
}

func (am *AudioManager) healthMonitorLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			am.updateSystemHealth()
		}
	}
}

func (am *AudioManager) updateSystemHealth() {
	am.mu.Lock()
	defer am.mu.Unlock()

	captureStats := am.captureEngine.GetStats()
	bufferStats := am.ringBuffer.GetStats()
	deviceHealth := am.deviceManager.GetHealth()

	processorStats := types.ProcessorStats{
		ProcessedSamples:  captureStats.SamplesProcessed,
		ProcessingLatency: captureStats.AverageLatency,
		// TODO: placeholders while the real stats are implemented
		QualityScore:       0.8,
		VoiceDetectionRate: 0.7,
		NoiseReductionGain: 0.3,
	}

	overallStatus := am.calculateOverallStatus(captureStats, bufferStats, deviceHealth, processorStats)

	am.health = types.SystemHealth{
		CaptureHealth:   captureStats,
		BufferHealth:    bufferStats,
		DeviceHealth:    deviceHealth,
		ProcessorHealth: processorStats,
		OverallStatus:   overallStatus,
		LastUpdate:      time.Now(),
	}

	select {
	case am.healthChan <- am.health:
	default:
	}
}

func (am *AudioManager) calculateOverallStatus(
	capture types.CaptureStats,
	buffer types.BufferStats,
	device types.DeviceHealth,
	processor types.ProcessorStats,
) types.HealthStatus {
	if !device.IsConnected {
		return types.HealthStatusFailed
	}

	if device.ErrorCount > 10 {
		return types.HealthStatusCritical
	}

	if capture.DroppedSamples > capture.SamplesProcessed/10 {
		return types.HealthStatusWarning
	}

	if buffer.UtilizationPercent > 90.0 {
		return types.HealthStatusWarning
	}

	return types.HealthStatusHealthy
}

func NewAudioExtractor(config types.ExtractionConfig, ringBuffer *buffer.RingBuffer, logger zerolog.Logger) *AudioExtractor {
	return &AudioExtractor{
		config:      config,
		ringBuffer:  ringBuffer,
		activeTasks: make(map[string]*ExtractionTask),
		semaphore:   make(chan struct{}, config.MaxConcurrent),
		logger:      logger.With().Str("component", "audio_extractor").Logger(),
	}
}

func (ae *AudioExtractor) Extract(ctx context.Context, req types.ExtractionRequest) (*types.AudioSegment, error) {
	select {
	case ae.semaphore <- struct{}{}:
		defer func() { <-ae.semaphore }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	if req.Duration == 0 {
		req.Duration = ae.config.DefaultDuration
	}

	if req.EndTime.IsZero() {
		req.EndTime = time.Now()
	}

	segment, err := ae.ringBuffer.Extract(req.Duration, req.EndTime)
	if err != nil {
		return nil, fmt.Errorf("failed to extract audio: %w", err)
	}

	if req.Format == "wav" {
		wavData, err := processing.ConvertToWAV(segment, ae.config.OutputSampleRate, ae.config.OutputChannels)
		if err != nil {
			ae.logger.Warn().Err(err).Msg("Failed to convert to WAV format")
		} else {
			segment.Data = make([]float32, len(wavData)/4)
			for i := 0; i < len(wavData)/4; i++ {
				segment.Data[i] = float32(wavData[i*4]) / 32768.0
			}
		}
	}

	return segment, nil
}
