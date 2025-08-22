package integration

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/your-org/hema-replay-system/pkg/commentary/engine"
	"github.com/your-org/hema-replay-system/pkg/commentary/types"
	speechtypes "github.com/your-org/hema-replay-system/pkg/speech/types"
)

// IntegrationManager manages the integration between speech recognition and commentary generation
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

	// Match context tracking
	matchState  *types.MatchContext
	callHistory []TranscriptionEvent
	maxHistory  int

	// Processing control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// IntegrationConfig configures the integration behavior
type IntegrationConfig struct {
	// Processing settings
	EnableContextEnrichment bool          `mapstructure:"enable_context_enrichment"`
	MaxConcurrentRequests   int           `mapstructure:"max_concurrent_requests"`
	ProcessingTimeout       time.Duration `mapstructure:"processing_timeout"`

	// Filtering settings
	MinConfidenceThreshold float32       `mapstructure:"min_confidence_threshold"`
	EnableDuplicateFilter  bool          `mapstructure:"enable_duplicate_filter"`
	DuplicateTimeWindow    time.Duration `mapstructure:"duplicate_time_window"`

	// Commentary settings
	DefaultQuality     types.QualityLevel `mapstructure:"default_quality"`
	DefaultCachePolicy types.CachePolicy  `mapstructure:"default_cache_policy"`

	// Match context settings
	EnableScoreTracking   bool `mapstructure:"enable_score_tracking"`
	EnableActionHistory   bool `mapstructure:"enable_action_history"`
	HistoryRetentionCount int  `mapstructure:"history_retention_count"`

	// Output settings
	EnableMetrics         bool          `mapstructure:"enable_metrics"`
	MetricsUpdateInterval time.Duration `mapstructure:"metrics_update_interval"`
}

// DefaultIntegrationConfig returns default configuration
func DefaultIntegrationConfig() *IntegrationConfig {
	return &IntegrationConfig{
		EnableContextEnrichment: true,
		MaxConcurrentRequests:   3,
		ProcessingTimeout:       5 * time.Second,
		MinConfidenceThreshold:  0.6,
		EnableDuplicateFilter:   true,
		DuplicateTimeWindow:     2 * time.Second,
		DefaultQuality:          types.QualityLevelBalanced,
		DefaultCachePolicy:      types.CachePolicyDefault,
		EnableScoreTracking:     true,
		EnableActionHistory:     true,
		HistoryRetentionCount:   10,
		EnableMetrics:           true,
		MetricsUpdateInterval:   30 * time.Second,
	}
}

// TranscriptionEvent represents a transcription with enriched context
type TranscriptionEvent struct {
	Transcription  *speechtypes.TranscriptionResult
	Timestamp      time.Time
	Processed      bool
	Commentary     *types.Commentary
	ProcessingTime time.Duration
}

// IntegrationMetrics tracks integration performance
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
	TimeoutErrors       uint64 `json:"timeout_errors"`
	QualityFilterErrors uint64 `json:"quality_filter_errors"`
	GenerationErrors    uint64 `json:"generation_errors"`

	// Match context
	ScoreUpdates       uint64 `json:"score_updates"`
	ContextEnrichments uint64 `json:"context_enrichments"`

	// Timestamp
	LastUpdated time.Time `json:"last_updated"`
	StartTime   time.Time `json:"start_time"`
}

// NewIntegrationManager creates a new integration manager
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
		maxHistory:          config.HistoryRetentionCount,
		ctx:                 ctx,
		cancel:              cancel,
		metrics: &IntegrationMetrics{
			StartTime:   time.Now(),
			LastUpdated: time.Now(),
		},
		matchState: &types.MatchContext{
			ScoreRed:      0,
			ScoreBlue:     0,
			Period:        1,
			TimeRemaining: 3 * time.Minute, // Default match time
			MatchPhase:    "opening",
			Intensity:     "medium",
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
		Msg("Integration manager started")

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

// processTranscription processes a single transcription
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

	// Apply filters
	if !im.shouldProcess(transcription) {
		im.mu.Lock()
		im.metrics.FilteredTranscriptions++
		im.mu.Unlock()
		return
	}

	// Create transcription event
	event := &TranscriptionEvent{
		Transcription: transcription,
		Timestamp:     time.Now(),
		Processed:     false,
	}

	// Add to history
	im.addToHistory(event)

	// Update match context
	if im.config.EnableScoreTracking {
		im.updateMatchContext(transcription)
	}

	// Create commentary request
	request := &types.CommentaryRequest{
		Input: types.TranscriptionInput{
			Text:       transcription.Text,
			Confidence: float32(transcription.Confidence),
			Timestamp:  transcription.ProcessedAt, // Use ProcessedAt instead of Timestamp
			Context:    im.getMatchContext(),
		},
		MaxLatency:  im.config.ProcessingTimeout,
		Quality:     im.config.DefaultQuality,
		CachePolicy: im.config.DefaultCachePolicy,
	}

	// Add basic audio metrics derived from metadata
	// Since AudioQuality is not available directly, we derive from metadata
	if transcription.Metadata.AudioQuality > 0 {
		request.Input.AudioMetrics = &types.AudioQuality{
			SignalToNoise:   float32(transcription.Metadata.AudioQuality),
			Clarity:         float32(transcription.Metadata.AudioQuality),
			VoiceDetection:  true,                                               // Assume voice was detected if we got transcription
			BackgroundNoise: 1.0 - float32(transcription.Metadata.AudioQuality), // Inverse of quality
			Distortion:      0.2,                                                // Default low distortion
		}
	}

	// Generate commentary
	ctx, cancel := context.WithTimeout(im.ctx, im.config.ProcessingTimeout)
	defer cancel()

	response, err := im.commentaryGenerator.Generate(ctx, request)

	processingTime := time.Since(startTime)
	event.ProcessingTime = processingTime

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

	// Update event and metrics
	event.Commentary = response.Commentary
	event.Processed = true

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

// shouldProcess determines if a transcription should be processed
func (im *IntegrationManager) shouldProcess(transcription *speechtypes.TranscriptionResult) bool {
	// Check confidence threshold
	if float32(transcription.Confidence) < im.config.MinConfidenceThreshold {
		im.logger.Debug().
			Str("text", transcription.Text).
			Float32("confidence", float32(transcription.Confidence)).
			Float32("threshold", im.config.MinConfidenceThreshold).
			Msg("Transcription filtered by confidence threshold")
		return false
	}

	// Check for duplicates if enabled
	if im.config.EnableDuplicateFilter {
		if im.isDuplicate(transcription) {
			im.logger.Debug().
				Str("text", transcription.Text).
				Msg("Transcription filtered as duplicate")
			return false
		}
	}

	return true
}

// isDuplicate checks if a transcription is a duplicate of recent ones
func (im *IntegrationManager) isDuplicate(transcription *speechtypes.TranscriptionResult) bool {
	im.mu.RLock()
	defer im.mu.RUnlock()

	cutoffTime := time.Now().Add(-im.config.DuplicateTimeWindow)

	for i := len(im.callHistory) - 1; i >= 0; i-- {
		event := im.callHistory[i]

		// Stop checking if we've gone beyond the time window
		if event.Timestamp.Before(cutoffTime) {
			break
		}

		// Check for similar text
		if im.isSimilarText(transcription.Text, event.Transcription.Text) {
			return true
		}
	}

	return false
}

// isSimilarText checks if two texts are similar enough to be considered duplicates
func (im *IntegrationManager) isSimilarText(text1, text2 string) bool {
	// Simple similarity check - could be enhanced with more sophisticated algorithms
	if text1 == text2 {
		return true
	}

	// Check if one text is contained in the other (for partial transcriptions)
	if len(text1) > len(text2) {
		return len(text2) > 5 && strings.Contains(text1, text2)
	} else {
		return len(text1) > 5 && strings.Contains(text2, text1)
	}
}

// updateMatchContext updates the match state based on transcription content
func (im *IntegrationManager) updateMatchContext(transcription *speechtypes.TranscriptionResult) {
	im.mu.Lock()
	defer im.mu.Unlock()

	text := strings.ToLower(transcription.Text)

	// Update score based on keywords
	if strings.Contains(text, "point") {
		if strings.Contains(text, "left") || strings.Contains(text, "red") {
			im.matchState.ScoreRed++
			im.matchState.LastScorer = "red"
			im.metrics.ScoreUpdates++
		} else if strings.Contains(text, "right") || strings.Contains(text, "blue") {
			im.matchState.ScoreBlue++
			im.matchState.LastScorer = "blue"
			im.metrics.ScoreUpdates++
		}
	}

	// Update last action
	if strings.Contains(text, "halt") {
		im.matchState.LastAction = "halt"
	} else if strings.Contains(text, "double") {
		im.matchState.LastAction = "double_hit"
	} else if strings.Contains(text, "point") {
		im.matchState.LastAction = "point_scored"
	} else {
		im.matchState.LastAction = "unknown"
	}

	// Update match phase based on score
	totalScore := im.matchState.ScoreRed + im.matchState.ScoreBlue
	if totalScore == 0 {
		im.matchState.MatchPhase = "opening"
	} else if totalScore < 8 {
		im.matchState.MatchPhase = "middle"
	} else {
		im.matchState.MatchPhase = "closing"
	}

	// Add to recent actions
	if len(im.matchState.RecentActions) >= 5 {
		im.matchState.RecentActions = im.matchState.RecentActions[1:]
	}
	im.matchState.RecentActions = append(im.matchState.RecentActions, transcription.Text)
}

// addToHistory adds a transcription event to the call history
func (im *IntegrationManager) addToHistory(event *TranscriptionEvent) {
	im.mu.Lock()
	defer im.mu.Unlock()

	im.callHistory = append(im.callHistory, *event)

	// Keep only recent history
	if len(im.callHistory) > im.maxHistory {
		im.callHistory = im.callHistory[1:]
	}
}

// getMatchContext returns a copy of the current match context
func (im *IntegrationManager) getMatchContext() *types.MatchContext {
	im.mu.RLock()
	defer im.mu.RUnlock()

	// Return a copy to avoid race conditions
	context := *im.matchState
	return &context
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
		"active":            im.active,
		"match_state":       im.matchState,
		"call_history_size": len(im.callHistory),
		"metrics":           im.metrics,
		"config":            im.config,
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
