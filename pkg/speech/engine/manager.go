package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/your-org/hema-replay-system/pkg/audio/debug"
	audioTypes "github.com/your-org/hema-replay-system/pkg/audio/types"
	"github.com/your-org/hema-replay-system/pkg/speech/internal"
	speechTypes "github.com/your-org/hema-replay-system/pkg/speech/types"
	"github.com/your-org/hema-replay-system/pkg/speech/whisper"
	// ❌ preprocessing removed - using pkg/audio/processing instead
)

// SpeechManager manages the complete speech recognition pipeline
type SpeechManager struct {
	config       speechTypes.SpeechConfig
	modelManager *whisper.ModelManager
	// ❌ preprocessor removed - using pkg/audio/processing instead
	cache    *ResultCache
	pipeline *ProcessingPipeline
	metrics  *internal.SpeechMetrics

	// Concurrency control
	semaphore   chan struct{}
	activeTasks map[string]*TranscriptionTask
	taskMutex   sync.RWMutex

	// Metrics and monitoring
	totalRequests  int64
	successfulReqs int64
	failedReqs     int64
	avgProcessTime time.Duration

	mu      sync.RWMutex
	running bool
	logger  zerolog.Logger
}

// TranscriptionTask represents an active transcription task
type TranscriptionTask struct {
	ID        string
	Request   speechTypes.TranscriptionRequest
	StartTime time.Time
	Done      chan *speechTypes.TranscriptionResult
	Error     chan error
	Context   context.Context
	Cancel    context.CancelFunc
}

// NewSpeechManager creates a new speech recognition manager
func NewSpeechManager(config speechTypes.SpeechConfig, logger zerolog.Logger) (*SpeechManager, error) {
	modelManager := whisper.NewModelManager(config.Whisper, logger)

	cache := NewResultCache(config.Performance.CacheSize, config.Performance.CacheTTL, logger)
	pipeline, err := NewProcessingPipeline(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create processing pipeline: %w", err)
	}

	// Initialize metrics collection
	metrics := internal.NewSpeechMetrics(logger)

	// Set up pipeline dependencies
	pipeline.SetModelManager(modelManager)

	return &SpeechManager{
		config:       config,
		modelManager: modelManager,
		cache:        cache,
		pipeline:     pipeline,
		metrics:      metrics,
		semaphore:    make(chan struct{}, config.Performance.MaxConcurrent),
		activeTasks:  make(map[string]*TranscriptionTask),
		logger:       logger.With().Str("component", "speech_manager").Logger(),
	}, nil
}

// SetDebugSaver sets the debug saver for the speech processing pipeline
func (sm *SpeechManager) SetDebugSaver(debugSaver *debug.SegmentSaver) {
	if sm.pipeline != nil {
		sm.pipeline.SetDebugSaver(debugSaver)
	}
}

// Start initializes the speech recognition system
func (sm *SpeechManager) Start(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.running {
		return fmt.Errorf("speech manager already running")
	}

	// Load default model
	if err := sm.modelManager.LoadModel(sm.config.Whisper.ModelSize); err != nil {
		return fmt.Errorf("failed to load default model: %w", err)
	}

	sm.running = true
	sm.logger.Info().Msg("Speech recognition manager started")

	return nil
}

// Stop shuts down the speech recognition system
func (sm *SpeechManager) Stop() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.running {
		return nil
	}

	// Cancel all active tasks
	sm.taskMutex.Lock()
	for _, task := range sm.activeTasks {
		task.Cancel()
	}
	sm.taskMutex.Unlock()

	// Unload all models
	sm.modelManager.UnloadAllModels()

	sm.running = false
	sm.logger.Info().Msg("Speech recognition manager stopped")

	return nil
}

// TranscribeAudio transcribes an audio segment
func (sm *SpeechManager) TranscribeAudio(ctx context.Context, audioSegment *audioTypes.AudioSegment) (*speechTypes.TranscriptionResult, error) {
	if !sm.running {
		return nil, fmt.Errorf("speech manager not running")
	}

	// Validate audio segment before processing
	if audioSegment == nil {
		return nil, fmt.Errorf("audio segment is nil")
	}

	if len(audioSegment.Data) == 0 {
		return nil, fmt.Errorf("audio segment data is empty")
	}

	// Check for minimum audio data (at least 100ms worth)
	minDuration := 100 * time.Millisecond
	if audioSegment.Duration < minDuration {
		sm.logger.Debug().
			Dur("duration", audioSegment.Duration).
			Dur("min_duration", minDuration).
			Int("data_length", len(audioSegment.Data)).
			Msg("Audio segment too short for speech recognition, skipping")

		// Return empty result instead of error for short segments
		return &speechTypes.TranscriptionResult{
			ID:          generateID(),
			Text:        "",
			Confidence:  0.0,
			Language:    sm.config.Whisper.Language,
			Duration:    time.Duration(0),
			Segments:    []speechTypes.TranscriptionSegment{},
			ProcessedAt: time.Now(),
			Metadata: speechTypes.TranscriptionMetadata{
				ModelUsed:      "skipped_insufficient_data",
				ProcessingTime: time.Duration(0),
				TokenCount:     0,
			},
		}, nil
	}

	// Create transcription request
	request := speechTypes.TranscriptionRequest{
		ID:                  generateID(),
		AudioSegment:        audioSegment,
		Language:            sm.config.Whisper.Language,
		ModelSize:           sm.config.Whisper.ModelSize,
		UseVocabulary:       true,
		ConfidenceThreshold: 0.7,
		MaxDuration:         sm.config.Performance.TimeoutDuration,
	}

	return sm.ProcessTranscriptionRequest(ctx, request)
}

// ProcessTranscriptionRequest processes a transcription request
func (sm *SpeechManager) ProcessTranscriptionRequest(ctx context.Context, request speechTypes.TranscriptionRequest) (*speechTypes.TranscriptionResult, error) {
	startTime := time.Now()

	// Check cache first
	if cached := sm.cache.Get(request.ID); cached != nil {
		sm.logger.Debug().Str("request_id", request.ID).Msg("Returning cached result")
		return cached, nil
	}

	// Acquire semaphore for concurrency control
	select {
	case sm.semaphore <- struct{}{}:
		defer func() { <-sm.semaphore }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Create task context with timeout
	taskCtx, cancel := context.WithTimeout(ctx, request.MaxDuration)
	defer cancel()

	// Create and register task
	task := &TranscriptionTask{
		ID:        request.ID,
		Request:   request,
		StartTime: startTime,
		Done:      make(chan *speechTypes.TranscriptionResult, 1),
		Error:     make(chan error, 1),
		Context:   taskCtx,
		Cancel:    cancel,
	}

	sm.registerTask(task)
	defer sm.unregisterTask(task.ID)

	// Process the request
	go sm.processTask(task)

	// Wait for result
	select {
	case result := <-task.Done:
		processingTime := time.Since(startTime)
		sm.updateMetrics(true, processingTime)

		// Cache the result
		sm.cache.Set(request.ID, result)

		sm.logger.Info().
			Str("request_id", request.ID).
			Dur("processing_time", processingTime).
			Float64("confidence", result.Confidence).
			Msg("Transcription completed successfully")

		return result, nil

	case err := <-task.Error:
		processingTime := time.Since(startTime)
		sm.updateMetrics(false, processingTime)

		sm.logger.Error().
			Err(err).
			Str("request_id", request.ID).
			Dur("processing_time", processingTime).
			Msg("Transcription failed")

		return nil, err

	case <-taskCtx.Done():
		sm.updateMetrics(false, time.Since(startTime))
		return nil, fmt.Errorf("transcription timeout for request %s", request.ID)
	}
}

// processTask processes a single transcription task
func (sm *SpeechManager) processTask(task *TranscriptionTask) {
	defer func() {
		if r := recover(); r != nil {
			sm.logger.Error().
				Interface("panic", r).
				Str("request_id", task.ID).
				Msg("Panic in transcription task")
			task.Error <- fmt.Errorf("transcription panic: %v", r)
		}
	}()

	result, err := sm.pipeline.Process(task.Context, task.Request)
	if err != nil {
		task.Error <- err
		return
	}

	task.Done <- result
}

// registerTask registers an active task
func (sm *SpeechManager) registerTask(task *TranscriptionTask) {
	sm.taskMutex.Lock()
	defer sm.taskMutex.Unlock()
	sm.activeTasks[task.ID] = task
}

// unregisterTask unregisters a completed task
func (sm *SpeechManager) unregisterTask(taskID string) {
	sm.taskMutex.Lock()
	defer sm.taskMutex.Unlock()
	delete(sm.activeTasks, taskID)
}

// updateMetrics updates processing metrics
func (sm *SpeechManager) updateMetrics(success bool, processingTime time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.totalRequests++
	if success {
		sm.successfulReqs++
	} else {
		sm.failedReqs++
	}

	// Update average processing time (simple moving average)
	if sm.avgProcessTime == 0 {
		sm.avgProcessTime = processingTime
	} else {
		sm.avgProcessTime = (sm.avgProcessTime + processingTime) / 2
	}
}

// GetStats returns current processing statistics
func (sm *SpeechManager) GetStats() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var successRate float64
	if sm.totalRequests > 0 {
		successRate = float64(sm.successfulReqs) / float64(sm.totalRequests) * 100
	}

	sm.taskMutex.RLock()
	activeTasks := len(sm.activeTasks)
	sm.taskMutex.RUnlock()

	return map[string]interface{}{
		"total_requests":      sm.totalRequests,
		"successful_requests": sm.successfulReqs,
		"failed_requests":     sm.failedReqs,
		"success_rate":        successRate,
		"avg_processing_time": sm.avgProcessTime,
		"active_tasks":        activeTasks,
		"is_running":          sm.running,
		"loaded_models":       sm.modelManager.GetLoadedModels(),
		"cache_stats":         sm.cache.GetStats(),
	}
}

// generateID generates a unique ID for requests
func generateID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}
