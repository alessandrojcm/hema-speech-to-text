package integration

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/your-org/hema-replay-system/pkg/commentary/engine"
	"github.com/your-org/hema-replay-system/pkg/commentary/types"
	speechtypes "github.com/your-org/hema-replay-system/pkg/speech/types"
)

// IntegrationManager manages the simplified integration between speech recognition and commentary generation
type IntegrationManager struct {
	// Dependencies
	commentaryGenerator *engine.CommentaryGenerator

	// Configuration
	config *IntegrationConfig
	logger zerolog.Logger

	// Communication channels
	speechInput      <-chan *speechtypes.TranscriptionResult
	commentaryOutput chan<- *types.Commentary
	errorOutput      chan<- error

	// State management
	mu      sync.RWMutex
	active  bool
	metrics *IntegrationMetrics

	// Processing control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// IntegrationConfig configures the simplified integration behavior
type IntegrationConfig struct {
	// Processing settings
	MaxConcurrentRequests int           `mapstructure:"max_concurrent_requests"`
	ProcessingTimeout     time.Duration `mapstructure:"processing_timeout"`

	// Basic filtering settings
	MinConfidenceThreshold float32 `mapstructure:"min_confidence_threshold"`

	// Output settings
	EnableMetrics         bool          `mapstructure:"enable_metrics"`
	MetricsUpdateInterval time.Duration `mapstructure:"metrics_update_interval"`
}

// DefaultIntegrationConfig returns simplified default configuration
func DefaultIntegrationConfig() *IntegrationConfig {
	return &IntegrationConfig{
		MaxConcurrentRequests:  3,
		ProcessingTimeout:      5 * time.Second,
		MinConfidenceThreshold: 0.6,
		EnableMetrics:          true,
		MetricsUpdateInterval:  30 * time.Second,
	}
}

// IntegrationMetrics tracks simplified integration performance
type IntegrationMetrics struct {
	// Processing metrics
	TotalTranscriptions     uint64 `json:"total_transcriptions"`
	ProcessedTranscriptions uint64 `json:"processed_transcriptions"`
	FilteredTranscriptions  uint64 `json:"filtered_transcriptions"`
	FailedTranscriptions    uint64 `json:"failed_transcriptions"`

	// Timing metrics
	AvgProcessingTime time.Duration `json:"avg_processing_time"`
	MaxProcessingTime time.Duration `json:"max_processing_time"`

	// Quality metrics
	AvgInputConfidence  float32 `json:"avg_input_confidence"`
	AvgOutputConfidence float32 `json:"avg_output_confidence"`

	// Error metrics
	TimeoutErrors    uint64 `json:"timeout_errors"`
	GenerationErrors uint64 `json:"generation_errors"`

	// Timestamp
	LastUpdated time.Time `json:"last_updated"`
	StartTime   time.Time `json:"start_time"`
}

// NewIntegrationManager creates a new simplified integration manager
func NewIntegrationManager(
	generator *engine.CommentaryGenerator,
	config *IntegrationConfig,
	logger zerolog.Logger,
) *IntegrationManager {
	if config == nil {
		config = DefaultIntegrationConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &IntegrationManager{
		commentaryGenerator: generator,
		config:              config,
		logger:              logger.With().Str("component", "integration-manager").Logger(),
		ctx:                 ctx,
		cancel:              cancel,
		metrics: &IntegrationMetrics{
			StartTime:   time.Now(),
			LastUpdated: time.Now(),
		},
	}
}

// Start starts the integration manager
func (im *IntegrationManager) Start(
	speechInput <-chan *speechtypes.TranscriptionResult,
	commentaryOutput chan<- *types.Commentary,
	errorOutput chan<- error,
) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	if im.active {
		return fmt.Errorf("integration manager already active")
	}

	im.speechInput = speechInput
	im.commentaryOutput = commentaryOutput
	im.errorOutput = errorOutput

	// Start processing goroutine
	im.wg.Add(1)
	go im.processTranscriptions()

	// Start metrics updater if enabled
	if im.config.EnableMetrics {
		im.wg.Add(1)
		go im.updateMetrics()
	}

	im.active = true
	im.logger.Info().
		Int("max_concurrent", im.config.MaxConcurrentRequests).
		Dur("timeout", im.config.ProcessingTimeout).
		Float32("min_confidence", im.config.MinConfidenceThreshold).
		Msg("Simplified integration manager started")

	return nil
}

// Stop stops the integration manager
func (im *IntegrationManager) Stop() error {
	im.mu.Lock()
	if !im.active {
		im.mu.Unlock()
		return nil
	}

	im.active = false
	im.cancel()
	im.mu.Unlock()

	// Wait for goroutines to finish
	im.wg.Wait()

	im.logger.Info().Msg("Integration manager stopped")
	return nil
}

// processTranscriptions processes incoming speech transcriptions
func (im *IntegrationManager) processTranscriptions() {
	defer im.wg.Done()

	semaphore := make(chan struct{}, im.config.MaxConcurrentRequests)

	for {
		select {
		case transcription := <-im.speechInput:
			if transcription == nil {
				continue
			}

			// Acquire semaphore slot
			select {
			case semaphore <- struct{}{}:
				// Process transcription in goroutine
				im.wg.Add(1)
				go func(t *speechtypes.TranscriptionResult) {
					defer func() {
						<-semaphore // Release semaphore slot
						im.wg.Done()
					}()

					im.processTranscription(t)
				}(transcription)

			case <-im.ctx.Done():
				return
			}

		case <-im.ctx.Done():
			return
		}
	}
}

// processTranscription processes a single transcription with simplified flow
func (im *IntegrationManager) processTranscription(transcription *speechtypes.TranscriptionResult) {
	startTime := time.Now()

	// Update metrics
	im.mu.Lock()
	im.metrics.TotalTranscriptions++
	im.metrics.AvgInputConfidence = updateAverage(
		im.metrics.AvgInputConfidence,
		float32(transcription.Confidence),
		im.metrics.TotalTranscriptions,
	)
	im.mu.Unlock()

	// Apply basic confidence filter only
	if !im.shouldProcess(transcription) {
		im.mu.Lock()
		im.metrics.FilteredTranscriptions++
		im.mu.Unlock()
		return
	}

	// Create simple commentary request (no context enrichment)
	request := &types.CommentaryRequest{
		Input: types.TranscriptionInput{
			Text:       transcription.Text,
			Confidence: float32(transcription.Confidence),
			Timestamp:  transcription.ProcessedAt,
		},
		MaxLatency: im.config.ProcessingTimeout,
	}

	// Generate commentary
	ctx, cancel := context.WithTimeout(im.ctx, im.config.ProcessingTimeout)
	defer cancel()

	response, err := im.commentaryGenerator.Generate(ctx, request)

	processingTime := time.Since(startTime)

	if err != nil {
		im.handleError(fmt.Errorf("commentary generation failed: %w", err))

		im.mu.Lock()
		im.metrics.FailedTranscriptions++
		if ctx.Err() == context.DeadlineExceeded {
			im.metrics.TimeoutErrors++
		} else {
			im.metrics.GenerationErrors++
		}
		im.mu.Unlock()
		return
	}

	if !response.Success {
		im.handleError(fmt.Errorf("commentary generation unsuccessful: %s", response.Error))

		im.mu.Lock()
		im.metrics.FailedTranscriptions++
		im.metrics.GenerationErrors++
		im.mu.Unlock()
		return
	}

	// Update metrics
	im.mu.Lock()
	im.metrics.ProcessedTranscriptions++
	im.metrics.AvgProcessingTime = updateAverageDuration(
		im.metrics.AvgProcessingTime,
		processingTime,
		im.metrics.ProcessedTranscriptions,
	)

	if processingTime > im.metrics.MaxProcessingTime {
		im.metrics.MaxProcessingTime = processingTime
	}

	if response.Commentary != nil {
		im.metrics.AvgOutputConfidence = updateAverage(
			im.metrics.AvgOutputConfidence,
			response.Commentary.Confidence,
			im.metrics.ProcessedTranscriptions,
		)
	}
	im.mu.Unlock()

	// Send commentary to output channel
	select {
	case im.commentaryOutput <- response.Commentary:
		im.logger.Debug().
			Str("input", transcription.Text).
			Str("commentary", response.Commentary.Text).
			Dur("processing_time", processingTime).
			Str("source", response.Commentary.Source).
			Msg("Commentary generated and sent")

	case <-im.ctx.Done():
		return
	}
}

// shouldProcess determines if a transcription should be processed (basic confidence check only)
func (im *IntegrationManager) shouldProcess(transcription *speechtypes.TranscriptionResult) bool {
	// Only check confidence threshold
	if float32(transcription.Confidence) < im.config.MinConfidenceThreshold {
		im.logger.Debug().
			Str("text", transcription.Text).
			Float32("confidence", float32(transcription.Confidence)).
			Float32("threshold", im.config.MinConfidenceThreshold).
			Msg("Transcription filtered by confidence threshold")
		return false
	}

	return true
}

// handleError handles integration errors
func (im *IntegrationManager) handleError(err error) {
	im.logger.Error().Err(err).Msg("Integration error")

	select {
	case im.errorOutput <- err:
		// Error sent successfully
	case <-im.ctx.Done():
		// Context cancelled, don't block
	default:
		// Error channel full, log and continue
		im.logger.Warn().Err(err).Msg("Error channel full, dropping error")
	}
}

// updateMetrics periodically updates metrics
func (im *IntegrationManager) updateMetrics() {
	defer im.wg.Done()

	ticker := time.NewTicker(im.config.MetricsUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			im.mu.Lock()
			im.metrics.LastUpdated = time.Now()
			im.mu.Unlock()

		case <-im.ctx.Done():
			return
		}
	}
}

// GetMetrics returns current integration metrics
func (im *IntegrationManager) GetMetrics() *IntegrationMetrics {
	im.mu.RLock()
	defer im.mu.RUnlock()

	// Return a copy
	metrics := *im.metrics
	return &metrics
}

// GetStatus returns the current status of the integration manager
func (im *IntegrationManager) GetStatus() map[string]interface{} {
	im.mu.RLock()
	defer im.mu.RUnlock()

	return map[string]interface{}{
		"active":  im.active,
		"metrics": im.metrics,
		"config":  im.config,
	}
}

// Utility functions

func updateAverage(current float32, newValue float32, count uint64) float32 {
	if count <= 1 {
		return newValue
	}
	return (current*float32(count-1) + newValue) / float32(count)
}

func updateAverageDuration(current time.Duration, newValue time.Duration, count uint64) time.Duration {
	if count <= 1 {
		return newValue
	}
	return time.Duration((int64(current)*int64(count-1) + int64(newValue)) / int64(count))
}
