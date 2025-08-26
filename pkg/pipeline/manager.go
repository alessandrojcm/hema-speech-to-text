package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/your-org/hema-replay-system/pkg/audio"
	"github.com/your-org/hema-replay-system/pkg/audio/types"
	"github.com/your-org/hema-replay-system/pkg/pipeline/vad"
	"github.com/your-org/hema-replay-system/pkg/speech/engine"
	speechTypes "github.com/your-org/hema-replay-system/pkg/speech/types"
)

// Manager orchestrates the complete pipeline from VAD detection to processing
type Manager struct {
	// Core components
	audioManager  *audio.AudioManager
	speechManager *engine.SpeechManager
	vadDetector   *vad.VADDetector

	// Pipeline state
	state         *StateManager
	eventBus      *EventBus
	metrics       *MetricsCollector
	segmentBuffer *SegmentBuffer

	// Configuration
	config *PipelineManagerConfig
	logger zerolog.Logger

	// Control
	ctx       context.Context
	cancel    context.CancelFunc
	errorChan chan error
	mu        sync.RWMutex
	running   bool
}

// Config holds pipeline configuration
type Config struct {
	Audio  types.AudioConfig        `mapstructure:"audio"`
	Speech speechTypes.SpeechConfig `mapstructure:"speech"`
	VAD    *vad.Config              `mapstructure:"vad"`

	// Pipeline settings
	MaxConcurrentRequests int           `mapstructure:"max_concurrent_requests"`
	ProcessingTimeout     time.Duration `mapstructure:"processing_timeout"`
	SegmentBufferSize     int           `mapstructure:"segment_buffer_size"`

	// Error handling
	MaxRetries      int           `mapstructure:"max_retries"`
	RetryDelay      time.Duration `mapstructure:"retry_delay"`
	FallbackEnabled bool          `mapstructure:"fallback_enabled"`

	// Metadata
	ShowMetadata    bool          `mapstructure:"show_metadata"`
	MetricsEnabled  bool          `mapstructure:"metrics_enabled"`
	MetricsInterval time.Duration `mapstructure:"metrics_interval"`
}

// NewManager creates a new pipeline manager
func NewManager(audioManager *audio.AudioManager, cfg *PipelineManagerConfig, logger zerolog.Logger) (*Manager, error) {
	// Initialize speech manager
	speechManager, err := engine.NewSpeechManager(cfg.Speech, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create speech manager: %w", err)
	}

	// Initialize VAD detector with the existing audio manager
	vadDetector := vad.NewVADDetector(audioManager, cfg.VAD, logger)

	return &Manager{
		audioManager:  audioManager,
		speechManager: speechManager,
		vadDetector:   vadDetector,
		state:         NewStateManager(),
		eventBus:      NewEventBus(),
		metrics:       NewMetricsCollector(),
		segmentBuffer: NewSegmentBuffer(cfg.Pipeline.SegmentBufferSize),
		config:        cfg,
		logger:        logger.With().Str("component", "pipeline_manager").Logger(),
		errorChan:     make(chan error, 10),
	}, nil
}

// Start starts the pipeline manager
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("pipeline manager already running")
	}

	m.ctx, m.cancel = context.WithCancel(ctx)

	// Start audio manager (skip if already running)
	if err := m.audioManager.Start(m.ctx); err != nil && err != types.ErrAlreadyRunning {
		return fmt.Errorf("failed to start audio manager: %w", err)
	}

	// Start speech manager
	if err := m.speechManager.Start(m.ctx); err != nil {
		return fmt.Errorf("failed to start speech manager: %w", err)
	}

	// Subscribe to VAD events
	m.subscribeToVADEvents()

	// Start VAD monitoring
	if err := m.vadDetector.Start(m.ctx); err != nil {
		return fmt.Errorf("failed to start VAD detector: %w", err)
	}

	// Start pipeline processing loop
	go m.processPipeline()

	// Start metrics collection if enabled
	if m.config.Pipeline.MetricsEnabled {
		go m.collectMetrics()
	}

	m.running = true
	m.state.Transition(EventStart)

	m.logger.Info().Msg("Pipeline manager started")
	return nil
}

// Stop stops the pipeline manager
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	if m.cancel != nil {
		m.cancel()
	}

	// Stop VAD detector
	m.vadDetector.Stop()

	// Stop speech manager
	if err := m.speechManager.Stop(); err != nil {
		m.logger.Error().Err(err).Msg("Failed to stop speech manager")
	}

	// Stop audio manager
	if err := m.audioManager.Stop(); err != nil {
		m.logger.Error().Err(err).Msg("Failed to stop audio manager")
	}

	m.running = false
	m.logger.Info().Msg("Pipeline manager stopped")
	return nil
}

// subscribeToVADEvents connects VAD events to pipeline processing
func (m *Manager) subscribeToVADEvents() {
	// Subscribe to VAD events
	m.eventBus.Subscribe(EventTypeVADSpeechStart, m.handleVADSpeechStart)
	m.eventBus.Subscribe(EventTypeVADSpeechEnd, m.handleVADSpeechEnd)
	m.eventBus.Subscribe(EventTypeVADSpeechSegment, m.handleVADSpeechSegment)

	// Start goroutine to listen to VAD events and publish to event bus
	go m.listenVADEvents()
}

// listenVADEvents listens to VAD events and publishes them to the event bus
func (m *Manager) listenVADEvents() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case vadEvent := <-m.vadDetector.Events():
			// Convert VAD events to pipeline events
			pipelineEvent := m.convertVADEvent(vadEvent)
			m.eventBus.Publish(pipelineEvent)
		}
	}
}

// convertVADEvent converts a VAD event to a pipeline event
func (m *Manager) convertVADEvent(vadEvent vad.VADEvent) PipelineEvent {
	switch vadEvent.Type {
	case vad.EventSpeechStart:
		return PipelineEvent{
			Type:      EventTypeVADSpeechStart,
			Timestamp: time.Now(),
			Data:      vadEvent,
		}
	case vad.EventSpeechEnd:
		return PipelineEvent{
			Type:      EventTypeVADSpeechEnd,
			Timestamp: time.Now(),
			Data:      vadEvent,
		}
	case vad.EventSpeechSegment:
		return PipelineEvent{
			Type:      EventTypeVADSpeechSegment,
			Timestamp: time.Now(),
			Data:      vadEvent,
		}
	default:
		return PipelineEvent{
			Type:      EventTypeError,
			Timestamp: time.Now(),
			Error:     fmt.Errorf("unknown VAD event type: %v", vadEvent.Type),
		}
	}
}

// handleVADSpeechStart handles the start of speech detection
func (m *Manager) handleVADSpeechStart(event PipelineEvent) {
	m.logger.Debug().Msg("Speech detection started")
	m.state.Transition(EventSpeechDetected)
}

// handleVADSpeechEnd handles the end of speech detection
func (m *Manager) handleVADSpeechEnd(event PipelineEvent) {
	m.logger.Debug().Msg("Speech detection ended")
}

// handleVADSpeechSegment handles a complete speech segment
func (m *Manager) handleVADSpeechSegment(event PipelineEvent) {
	vadEvent, ok := event.Data.(vad.VADEvent)
	if !ok {
		m.handleError(fmt.Errorf("invalid VAD event data"))
		return
	}

	m.logger.Debug().
		Dur("duration", vadEvent.Duration).
		Float32("confidence", vadEvent.Confidence).
		Msg("Speech segment detected")

	// Extract audio segment for processing
	go m.extractAndProcessSegment(vadEvent)
}

// extractAndProcessSegment extracts audio segment and processes it
func (m *Manager) extractAndProcessSegment(vadEvent vad.VADEvent) {
	startTime := time.Now()

	// Create extraction request based on VAD event
	extractionReq := types.ExtractionRequest{
		Duration: vadEvent.Duration,
		EndTime:  vadEvent.BufferEnd,
		Format:   "raw", // We want raw float32 samples for speech processing
	}

	// Extract audio segment
	segment, err := m.audioManager.ExtractAudio(m.ctx, extractionReq)
	if err != nil {
		m.handleError(fmt.Errorf("failed to extract audio segment: %w", err))
		return
	}

	// Note: VAD metadata will be tracked in the segment buffer and pipeline events
	// The AudioSegment doesn't have a Custom field, so we'll pass metadata through events

	// Store segment in buffer
	segmentID := m.segmentBuffer.Add(segment, vadEvent)

	// Record metrics
	extractionTime := time.Since(startTime)
	m.metrics.RecordProcessingTime("audio_extraction", extractionTime)

	// Publish segment ready event
	m.eventBus.Publish(PipelineEvent{
		Type:      EventTypeAudioSegmentReady,
		Timestamp: time.Now(),
		Data: AudioSegmentData{
			SegmentID: segmentID,
			Segment:   segment,
			VADEvent:  vadEvent,
		},
	})

	// Transition to processing state
	m.state.Transition(EventProcessingStart)
}

// processPipeline main processing loop
func (m *Manager) processPipeline() {
	for {
		select {
		case <-m.ctx.Done():
			return
		default:
			// Process any pending segments
			if segments := m.segmentBuffer.GetPending(); len(segments) > 0 {
				for _, segmentData := range segments {
					go m.processAudioSegment(segmentData)
				}
			}
			time.Sleep(10 * time.Millisecond) // Brief sleep to prevent busy loop
		}
	}
}

// processAudioSegment processes a single audio segment through speech recognition
func (m *Manager) processAudioSegment(segmentData *AudioSegmentData) {
	startTime := time.Now()

	// Process through speech recognition using the manager's TranscribeAudio method
	result, err := m.speechManager.TranscribeAudio(m.ctx, segmentData.Segment)
	if err != nil {
		m.handleError(fmt.Errorf("speech processing failed: %w", err))
		return
	}

	// VAD metadata is tracked in the segment buffer and events
	// The TranscriptionMetadata doesn't have a Custom field, so we pass it through events

	// Record metrics
	processingTime := time.Since(startTime)
	m.metrics.RecordProcessingTime("speech_processing", processingTime)

	// Mark segment as processed
	m.segmentBuffer.MarkProcessed(segmentData.SegmentID, result)

	// Publish transcript ready event
	m.eventBus.Publish(PipelineEvent{
		Type:      EventTypeTranscriptReady,
		Timestamp: time.Now(),
		Data: TranscriptData{
			SegmentID: segmentData.SegmentID,
			Result:    result,
			VADEvent:  segmentData.VADEvent,
		},
	})

	m.logger.Info().
		Str("transcript", result.Text).
		Float64("confidence", result.Confidence).
		Dur("processing_time", processingTime).
		Msg("Speech segment processed")
}

// collectMetrics periodically collects and logs metrics
func (m *Manager) collectMetrics() {
	ticker := time.NewTicker(m.config.Pipeline.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			stats := m.GetMetrics()
			m.logger.Info().
				Interface("metrics", stats).
				Msg("Pipeline metrics")
		}
	}
}

// GetMetrics returns current pipeline metrics
func (m *Manager) GetMetrics() map[string]interface{} {
	stats := m.metrics.GetStats()

	// Add audio manager metrics
	audioMetrics := m.audioManager.GetMetrics()
	stats["audio_total_extractions"] = audioMetrics.TotalExtractions
	stats["audio_failed_extractions"] = audioMetrics.FailedExtractions
	stats["audio_extraction_failure_rate"] = audioMetrics.ExtractionFailureRate

	// Add segment buffer metrics
	stats["segments_pending"] = m.segmentBuffer.GetPendingCount()
	stats["segments_processed"] = m.segmentBuffer.GetProcessedCount()

	// Add state information
	stats["current_state"] = m.state.Current().String()
	stats["pipeline_running"] = m.running

	return stats
}

// handleError handles pipeline errors
func (m *Manager) handleError(err error) {
	m.logger.Error().Err(err).Msg("Pipeline error")

	// Send to error channel for monitoring
	select {
	case m.errorChan <- err:
	default:
		// Don't block if channel is full
	}

	// Record error in metrics
	m.metrics.RecordError(err)

	// Publish error event
	m.eventBus.Publish(PipelineEvent{
		Type:      EventTypeError,
		Timestamp: time.Now(),
		Error:     err,
	})
}

// GetErrors returns the error channel for monitoring
func (m *Manager) GetErrors() <-chan error {
	return m.errorChan
}

// GetState returns the current pipeline state
func (m *Manager) GetState() State {
	return m.state.Current()
}
