//go:build !noaudio
// +build !noaudio

package capture

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"github.com/your-org/hema-replay-system/pkg/audio/buffer"
	"github.com/your-org/hema-replay-system/pkg/audio/internal"
	"github.com/your-org/hema-replay-system/pkg/audio/processing"
	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

// Constants for async processing
const (
	// Channel buffer sizes (bounded to prevent unbounded memory growth)
	RawAudioChannelSize = 20 // ~400ms buffer at 50ms frames

	// Processing thresholds
	MaxProcessingLag = 500 * time.Millisecond
	DropThreshold    = 10 // Enable fast path if queue > 10
)

// AudioFrame represents a captured audio frame with metadata
type AudioFrame struct {
	Data      []float32
	Timestamp time.Time
	Sequence  uint64 // For ordering validation
}

// ProcessingStats tracks async processing metrics
type ProcessingStats struct {
	FramesQueued    uint64
	FramesProcessed uint64
	FramesDropped   uint64
	QueueDepth      int
	ProcessingLag   time.Duration
}

// Global frame pool to reduce GC pressure
var audioFramePool = sync.Pool{
	New: func() interface{} {
		// Pre-allocate with typical frame size
		// Will be resized as needed
		return &AudioFrame{
			Data: make([]float32, 0, 4096), // 2048 samples * 2 channels
		}
	},
}

// Helper functions for pool management
func getAudioFrame(size int) *AudioFrame {
	frame := audioFramePool.Get().(*AudioFrame)
	if cap(frame.Data) < size {
		frame.Data = make([]float32, size)
	} else {
		frame.Data = frame.Data[:size]
	}
	return frame
}

func putAudioFrame(frame *AudioFrame) {
	// Reset frame before returning to pool
	frame.Timestamp = time.Time{}
	frame.Sequence = 0
	frame.Data = frame.Data[:0]
	audioFramePool.Put(frame)
}

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

	// Async processing channels (bounded for backpressure control)
	rawAudioChan  chan *AudioFrame // From capture to processing
	processedChan chan *AudioFrame // From processing to buffer

	// Synchronization
	processingWG   sync.WaitGroup
	processingDone chan struct{}

	// Metrics
	frameSequence   uint64 // Atomic counter
	processingStats ProcessingStats
	statsLock       sync.RWMutex

	// Performance flags
	fastPathEnabled uint32 // Atomic flag (0 = false, 1 = true)
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

// initChannels initializes the bounded channels for async processing
func (ce *CaptureEngine) initChannels() {
	// Create bounded channels for backpressure control
	ce.rawAudioChan = make(chan *AudioFrame, RawAudioChannelSize)
	ce.processingDone = make(chan struct{})
}

// initPerformance sets up performance optimizations
func (ce *CaptureEngine) initPerformance() {
	// Set GOMAXPROCS to use all available CPU cores
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Initialize fast path flag
	atomic.StoreUint32(&ce.fastPathEnabled, 0) // Start with full processing

	// Log CPU configuration
	ce.logger.Info().
		Int("cpu_cores", runtime.NumCPU()).
		Int("gomaxprocs", runtime.GOMAXPROCS(0)).
		Msg("Performance configuration initialized")
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

	// Initialize performance settings
	ce.initPerformance()

	// Initialize channels with bounded sizes
	ce.initChannels()

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

	// Start processing goroutine
	ce.processingWG.Add(1)
	go ce.processingLoop(ctx)

	// Start capture goroutine
	go ce.captureLoop(ctx)

	// Start monitoring
	go ce.monitoringLoop(ctx)

	ce.logger.Info().
		Int("channel_buffer_size", RawAudioChannelSize).
		Int("drop_threshold", DropThreshold).
		Msg("Async audio processing started")

	return nil
}

func (ce *CaptureEngine) Stop() error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if !ce.running {
		return nil
	}

	// Signal stop
	close(ce.stopChan)
	ce.running = false

	// Stop audio stream first
	if ce.stream != nil {
		if err := ce.stream.Stop(); err != nil {
			ce.logger.Error().Err(err).Msg("Failed to stop audio stream")
		}
		if err := ce.stream.Close(); err != nil {
			ce.logger.Error().Err(err).Msg("Failed to close audio stream")
		}
		ce.stream = nil
	}

	// Wait for capture to finish
	// (This will close rawAudioChan)

	// Wait for processing to complete
	ce.processingWG.Wait()

	// Clean up any remaining frames in the pool
	// (The pool will be garbage collected automatically)

	if err := ce.deviceManager.Stop(); err != nil {
		ce.logger.Error().Err(err).Msg("Failed to stop device manager")
		return err
	}

	ce.logger.Info().Msg("Async audio processing stopped")
	return nil
}

func (ce *CaptureEngine) GetStats() types.CaptureStats {
	stats := ce.metrics.GetStats()
	stats.DeviceHealth = ce.deviceManager.GetHealth()

	bufferStats := ce.buffer.GetStats()
	stats.BufferUtilization = bufferStats.UtilizationPercent

	return stats
}

// captureLoop - Non-blocking capture with memory pool and frame dropping
func (ce *CaptureEngine) captureLoop(ctx context.Context) {
	defer func() {
		close(ce.rawAudioChan)
		ce.mu.Lock()
		ce.running = false
		ce.mu.Unlock()
		ce.logger.Info().Msg("Capture loop exited")
	}()

	frameSize := ce.config.FramesPerBuffer * ce.config.Channels
	audioBuffer := make([]float32, frameSize)

	// Track consecutive errors for stream health
	consecutiveErrors := 0
	const maxConsecutiveErrors = 10

	ce.logger.Info().Int("frame_size", frameSize).Msg("Starting async capture loop")

	for {
		select {
		case <-ctx.Done():
			ce.logger.Info().Msg("Capture loop stopped: context cancelled")
			return
		case <-ce.stopChan:
			ce.logger.Info().Msg("Capture loop stopped: stop signal received")
			return
		default:
			// Check if stream is still active
			if ce.stream == nil || !ce.stream.IsActive() {
				ce.logger.Error().Msg("Audio stream is not active, attempting recovery")
				if err := ce.attemptStreamRecovery(ctx); err != nil {
					ce.logger.Error().Err(err).Msg("Failed to recover audio stream")
					// Wait before retry
					select {
					case <-time.After(1 * time.Second):
					case <-ctx.Done():
						return
					case <-ce.stopChan:
						return
					}
					continue
				}
			}

			// Read from PortAudio (blocking but fast)
			if err := ce.stream.Read(audioBuffer); err != nil {
				consecutiveErrors++
				ce.logger.Error().
					Err(err).
					Int("consecutive_errors", consecutiveErrors).
					Msg("Stream read error")

				// Check if we've hit too many consecutive errors
				if consecutiveErrors >= maxConsecutiveErrors {
					ce.logger.Error().Msg("Too many consecutive read errors, attempting stream recovery")
					if recoveryErr := ce.attemptStreamRecovery(ctx); recoveryErr != nil {
						ce.logger.Error().Err(recoveryErr).Msg("Stream recovery failed")
						// Wait longer before next attempt
						select {
						case <-time.After(5 * time.Second):
						case <-ctx.Done():
							return
						case <-ce.stopChan:
							return
						}
					}
					consecutiveErrors = 0
				}

				ce.handleCaptureError(err)
				continue
			}

			// Reset error counter on successful read
			consecutiveErrors = 0

			// Get frame from pool (reduces GC pressure)
			frame := getAudioFrame(len(audioBuffer))
			frame.Timestamp = time.Now()
			frame.Sequence = atomic.AddUint64(&ce.frameSequence, 1)
			copy(frame.Data, audioBuffer)

			// Check queue depth for fast path decision
			queueDepth := len(ce.rawAudioChan)
			if queueDepth > DropThreshold {
				atomic.StoreUint32(&ce.fastPathEnabled, 1)
			} else if queueDepth < DropThreshold/2 {
				atomic.StoreUint32(&ce.fastPathEnabled, 0)
			}

			// Send to processing (non-blocking with frame dropping)
			select {
			case ce.rawAudioChan <- frame:
				ce.updateStats("frameQueued")
			default:
				// Channel full - implement frame dropping strategy
				select {
				case oldFrame := <-ce.rawAudioChan:
					// Return old frame to pool
					putAudioFrame(oldFrame)
					ce.updateStats("frameDropped")
				default:
				}
				// Try again
				select {
				case ce.rawAudioChan <- frame:
					ce.updateStats("frameQueued")
				default:
					// Frame couldn't be queued - return to pool
					putAudioFrame(frame)
					ce.updateStats("frameDropped")
				}
			}
		}
	}
}

// processingLoop - Async processing with fast path support
func (ce *CaptureEngine) processingLoop(ctx context.Context) {
	defer ce.processingWG.Done()

	// Clean up frames on exit
	defer func() {
		// Drain channel and return frames to pool
		for frame := range ce.rawAudioChan {
			putAudioFrame(frame)
		}
		ce.logger.Info().Msg("Processing loop exited")
	}()

	ce.logger.Info().Msg("Starting async processing loop")

	for {
		select {
		case <-ctx.Done():
			ce.logger.Info().Msg("Processing loop stopped: context cancelled")
			return
		case frame, ok := <-ce.rawAudioChan:
			if !ok {
				ce.logger.Info().Msg("Processing loop stopped: channel closed")
				return
			}

			processingStart := time.Now()

			// Check fast path flag - skip heavy processing if backed up
			fastPath := atomic.LoadUint32(&ce.fastPathEnabled) == 1

			// Process audio (VAD, quality assessment, etc.)
			processedData := frame.Data
			if ce.processor != nil && !fastPath {
				var err error
				processedData, err = ce.processor.Process(frame.Data, frame.Timestamp)
				if err != nil {
					ce.logger.Warn().
						Err(err).
						Uint64("sequence", frame.Sequence).
						Bool("fast_path", fastPath).
						Msg("Audio processing failed")
					processedData = frame.Data
				}
			} else if fastPath {
				// Fast path - minimal processing only
				ce.logger.Debug().
					Uint64("sequence", frame.Sequence).
					Int("queue_depth", len(ce.rawAudioChan)).
					Msg("Fast path enabled - skipping heavy processing")
			}

			// Write to buffer
			if err := ce.buffer.Write(processedData, frame.Timestamp); err != nil {
				ce.logger.Error().
					Err(err).
					Uint64("sequence", frame.Sequence).
					Msg("Failed to write to ring buffer")
				ce.metrics.IncrementDroppedSamples(int64(len(processedData)))
			} else {
				ce.metrics.IncrementProcessedSamples(int64(len(processedData)))
			}

			// Return frame to pool
			putAudioFrame(frame)

			// Update processing metrics
			processingTime := time.Since(processingStart)
			ce.updateProcessingStats(frame.Sequence, processingTime)
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

			// Check for processing lag
			queueDepth := len(ce.rawAudioChan)
			if ce.getProcessingLag() > MaxProcessingLag {
				ce.logger.Warn().
					Dur("lag", ce.getProcessingLag()).
					Int("queue_depth", queueDepth).
					Bool("fast_path", atomic.LoadUint32(&ce.fastPathEnabled) == 1).
					Msg("Processing falling behind")
			}

			// Report async processing metrics
			ce.reportAsyncMetrics()
		}
	}
}

// reportAsyncMetrics reports detailed async processing statistics
func (ce *CaptureEngine) reportAsyncMetrics() {
	ce.statsLock.RLock()
	defer ce.statsLock.RUnlock()

	// Calculate drop rate
	total := ce.processingStats.FramesQueued
	dropped := ce.processingStats.FramesDropped
	dropRate := float64(0)
	if total > 0 {
		dropRate = float64(dropped) / float64(total) * 100
	}

	ce.logger.Debug().
		Uint64("frames_queued", ce.processingStats.FramesQueued).
		Uint64("frames_processed", ce.processingStats.FramesProcessed).
		Uint64("frames_dropped", ce.processingStats.FramesDropped).
		Float64("drop_rate_pct", dropRate).
		Int("queue_depth", len(ce.rawAudioChan)).
		Dur("processing_lag", ce.processingStats.ProcessingLag).
		Bool("fast_path", atomic.LoadUint32(&ce.fastPathEnabled) == 1).
		Int("cpu_cores_used", runtime.NumCPU()).
		Msg("Async processing metrics")
}

// updateStats updates processing statistics atomically
func (ce *CaptureEngine) updateStats(statType string) {
	ce.statsLock.Lock()
	defer ce.statsLock.Unlock()

	switch statType {
	case "frameQueued":
		ce.processingStats.FramesQueued++
	case "frameProcessed":
		ce.processingStats.FramesProcessed++
	case "frameDropped":
		ce.processingStats.FramesDropped++
	}

	ce.processingStats.QueueDepth = len(ce.rawAudioChan)
}

// updateProcessingStats updates processing statistics with latency
func (ce *CaptureEngine) updateProcessingStats(sequence uint64, processingTime time.Duration) {
	ce.statsLock.Lock()
	defer ce.statsLock.Unlock()

	ce.processingStats.FramesProcessed++
	ce.processingStats.ProcessingLag = processingTime
	ce.processingStats.QueueDepth = len(ce.rawAudioChan)
}

// getProcessingLag returns current processing lag
func (ce *CaptureEngine) getProcessingLag() time.Duration {
	ce.statsLock.RLock()
	defer ce.statsLock.RUnlock()
	return ce.processingStats.ProcessingLag
}

func (ce *CaptureEngine) handleCaptureError(err error) {
	ce.logger.Error().Err(err).Msg("Audio capture error")

	select {
	case ce.errorChan <- err:
	default:
	}

	time.Sleep(10 * time.Millisecond)
}

// attemptStreamRecovery tries to recover from stream failures
func (ce *CaptureEngine) attemptStreamRecovery(ctx context.Context) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	ce.logger.Info().Msg("Attempting audio stream recovery")

	// Close existing stream if it exists
	if ce.stream != nil {
		if err := ce.stream.Stop(); err != nil {
			ce.logger.Warn().Err(err).Msg("Error stopping broken stream")
		}
		if err := ce.stream.Close(); err != nil {
			ce.logger.Warn().Err(err).Msg("Error closing broken stream")
		}
		ce.stream = nil
	}

	// Re-initialize device if needed
	device := ce.deviceManager.GetCurrentDevice()
	if device == nil {
		ce.logger.Info().Msg("Re-initializing audio device")
		// Try to refresh device list
		if err := ce.deviceManager.RefreshDevices(); err != nil {
			return fmt.Errorf("failed to refresh devices: %w", err)
		}

		device = ce.deviceManager.GetCurrentDevice()
		if device == nil {
			return fmt.Errorf("no audio device available after refresh")
		}
	}

	// Open new stream
	stream, err := ce.deviceManager.portAudio.OpenStream(device, ce.config)
	if err != nil {
		return fmt.Errorf("failed to open new audio stream: %w", err)
	}

	// Start the new stream
	if err := stream.Start(); err != nil {
		stream.Close()
		return fmt.Errorf("failed to start new audio stream: %w", err)
	}

	ce.stream = stream
	ce.logger.Info().Msg("Audio stream recovery successful")

	return nil
}
