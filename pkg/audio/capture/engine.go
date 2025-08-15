//go:build !noaudio
// +build !noaudio

package capture

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/your-org/hema-replay-system/pkg/audio/buffer"
	"github.com/your-org/hema-replay-system/pkg/audio/internal"
	"github.com/your-org/hema-replay-system/pkg/audio/processing"
	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

type CaptureEngine struct {
	mu            sync.RWMutex
	config        types.DeviceConfig
	deviceManager *DeviceManager
	stream        *internal.AudioStream
	buffer        *buffer.RingBuffer
	processor     *processing.AudioProcessor
	running       bool
	stopChan      chan struct{}
	errorChan     chan error
	statsChan     chan types.CaptureStats
	logger        zerolog.Logger
	metrics       *CaptureMetrics
}

type CaptureMetrics struct {
	mu               sync.RWMutex
	samplesProcessed int64
	samplesDropped   int64
	totalLatency     time.Duration
	latencyCount     int64
	lastUpdate       time.Time
	startTime        time.Time
}

func NewCaptureMetrics() *CaptureMetrics {
	return &CaptureMetrics{
		startTime:  time.Now(),
		lastUpdate: time.Now(),
	}
}

func (cm *CaptureMetrics) IncrementProcessedSamples(count int64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.samplesProcessed += count
	cm.lastUpdate = time.Now()
}

func (cm *CaptureMetrics) IncrementDroppedSamples(count int64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.samplesDropped += count
	cm.lastUpdate = time.Now()
}

func (cm *CaptureMetrics) RecordLatency(latency time.Duration) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.totalLatency += latency
	cm.latencyCount++
	cm.lastUpdate = time.Now()
}

func (cm *CaptureMetrics) GetStats() types.CaptureStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var avgLatency time.Duration
	if cm.latencyCount > 0 {
		avgLatency = cm.totalLatency / time.Duration(cm.latencyCount)
	}

	return types.CaptureStats{
		SamplesProcessed: cm.samplesProcessed,
		DroppedSamples:   cm.samplesDropped,
		AverageLatency:   avgLatency,
		LastUpdate:       cm.lastUpdate,
	}
}

func NewCaptureEngine(config types.DeviceConfig, processingConfig types.ProcessingConfig, ringBuffer *buffer.RingBuffer, logger zerolog.Logger) (*CaptureEngine, error) {
	deviceManager := NewDeviceManager(config, logger)
	processor, err := processing.NewAudioProcessor(processingConfig, config.SampleRate, config.Channels, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio processor: %w", err)
	}

	return &CaptureEngine{
		config:        config,
		deviceManager: deviceManager,
		buffer:        ringBuffer,
		processor:     processor,
		stopChan:      make(chan struct{}),
		errorChan:     make(chan error, 10),
		statsChan:     make(chan types.CaptureStats, 1),
		logger:        logger.With().Str("component", "capture_engine").Logger(),
		metrics:       NewCaptureMetrics(),
	}, nil
}
func (ce *CaptureEngine) Start(ctx context.Context) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if ce.running {
		return types.ErrAlreadyRunning
	}

	if err := ce.deviceManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start device manager: %w", err)
	}

	device := ce.deviceManager.GetCurrentDevice()
	if device == nil {
		return types.ErrDeviceNotFound
	}

	stream, err := ce.deviceManager.portAudio.OpenStream(device, ce.config)
	if err != nil {
		return fmt.Errorf("failed to open audio stream: %w", err)
	}

	if err := stream.Start(); err != nil {
		stream.Close()
		return fmt.Errorf("failed to start audio stream: %w", err)
	}

	ce.stream = stream
	ce.running = true

	go ce.captureLoop(ctx)
	go ce.monitoringLoop(ctx)

	ce.logger.Info().Msg("Audio capture started")
	return nil
}

func (ce *CaptureEngine) Stop() error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if !ce.running {
		return nil
	}

	close(ce.stopChan)
	ce.running = false

	if ce.stream != nil {
		if err := ce.stream.Stop(); err != nil {
			ce.logger.Error().Err(err).Msg("Failed to stop audio stream")
		}
		if err := ce.stream.Close(); err != nil {
			ce.logger.Error().Err(err).Msg("Failed to close audio stream")
		}
		ce.stream = nil
	}

	if err := ce.deviceManager.Stop(); err != nil {
		ce.logger.Error().Err(err).Msg("Failed to stop device manager")
		return err
	}

	ce.logger.Info().Msg("Audio capture stopped")
	return nil
}

func (ce *CaptureEngine) GetStats() types.CaptureStats {
	stats := ce.metrics.GetStats()
	stats.DeviceHealth = ce.deviceManager.GetHealth()

	bufferStats := ce.buffer.GetStats()
	stats.BufferUtilization = bufferStats.UtilizationPercent

	return stats
}

func (ce *CaptureEngine) captureLoop(ctx context.Context) {
	defer func() {
		ce.mu.Lock()
		ce.running = false
		ce.mu.Unlock()
	}()

	frameSize := ce.config.FramesPerBuffer * ce.config.Channels
	audioBuffer := make([]float32, frameSize)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ce.stopChan:
			return
		default:
			startTime := time.Now()
			timestamp := startTime

			if err := ce.stream.Read(audioBuffer); err != nil {
				ce.handleCaptureError(err)
				continue
			}

			processedData := audioBuffer
			if ce.processor != nil {
				var err error
				processedData, err = ce.processor.Process(audioBuffer, timestamp)
				if err != nil {
					ce.logger.Warn().Err(err).Msg("Audio processing failed")
					processedData = audioBuffer
				}
			}

			if err := ce.buffer.Write(processedData, timestamp); err != nil {
				ce.logger.Error().Err(err).Msg("Failed to write to ring buffer")
				ce.metrics.IncrementDroppedSamples(int64(len(processedData)))
			} else {
				ce.metrics.IncrementProcessedSamples(int64(len(processedData)))
			}

			latency := time.Since(startTime)
			ce.metrics.RecordLatency(latency)
		}
	}
}

func (ce *CaptureEngine) monitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ce.stopChan:
			return
		case <-ticker.C:
			stats := ce.GetStats()
			select {
			case ce.statsChan <- stats:
			default:
			}
		}
	}
}

func (ce *CaptureEngine) handleCaptureError(err error) {
	ce.logger.Error().Err(err).Msg("Audio capture error")

	select {
	case ce.errorChan <- err:
	default:
	}

	time.Sleep(10 * time.Millisecond)
}
