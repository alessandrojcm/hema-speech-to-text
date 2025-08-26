package engine

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"

	"github.com/your-org/hema-replay-system/pkg/commentary/prompt"
	"github.com/your-org/hema-replay-system/pkg/commentary/templates"
	"github.com/your-org/hema-replay-system/pkg/commentary/types"
	llmtypes "github.com/your-org/hema-replay-system/pkg/llm/types"
)

// CommentaryGenerator orchestrates the commentary generation pipeline
type CommentaryGenerator struct {
	// Dependencies
	llmEngine     llmtypes.EngineInterface
	promptBuilder *prompt.Builder
	validator     *QualityValidator
	fallback      *FallbackGenerator

	// Configuration
	config *types.CommentaryConfig
	logger zerolog.Logger

	// State management
	mu             sync.RWMutex
	active         bool
	metrics        *types.GenerationMetrics
	activeRequests int64

	// Request management
	requestChan chan *generationRequest
	workers     []*worker
	workerCount int

	// Shutdown management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// generationRequest represents an internal generation request with channels for response
type generationRequest struct {
	request    *types.CommentaryRequest
	response   chan *types.CommentaryResponse
	ctx        context.Context
	startTime  time.Time
	logEntries []types.LogEntry
}

// worker represents a generation worker
type worker struct {
	id        int
	generator *CommentaryGenerator
	logger    zerolog.Logger
}

// NewCommentaryGenerator creates a new commentary generator
func NewCommentaryGenerator(
	llmEngine llmtypes.EngineInterface,
	promptBuilder *prompt.Builder,
	config *types.CommentaryConfig,
	logger zerolog.Logger,
) (*CommentaryGenerator, error) {
	if config == nil {
		config = types.DefaultCommentaryConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	generator := &CommentaryGenerator{
		llmEngine:     llmEngine,
		promptBuilder: promptBuilder,
		config:        config,
		logger:        logger.With().Str("component", "commentary-generator").Logger(),
		ctx:           ctx,
		cancel:        cancel,
		requestChan:   make(chan *generationRequest, config.ConcurrentRequests*2),
		workerCount:   config.ConcurrentRequests,
		metrics: &types.GenerationMetrics{
			CollectionStart: time.Now(),
			LastUpdated:     time.Now(),
		},
	}

	// Initialize validator
	validator, err := NewQualityValidator(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create validator: %w", err)
	}
	generator.validator = validator

	// Initialize fallback generator
	fallback, err := NewFallbackGenerator(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create fallback generator: %w", err)
	}
	generator.fallback = fallback

	return generator, nil
}

// Start starts the commentary generator
func (g *CommentaryGenerator) Start() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.active {
		return fmt.Errorf("generator already active")
	}

	// Create and start workers
	g.workers = make([]*worker, g.workerCount)
	for i := 0; i < g.workerCount; i++ {
		worker := &worker{
			id:        i,
			generator: g,
			logger:    g.logger.With().Int("worker_id", i).Logger(),
		}
		g.workers[i] = worker

		g.wg.Add(1)
		go worker.run()
	}

	g.active = true
	g.logger.Info().
		Int("workers", g.workerCount).
		Msg("Commentary generator started")

	return nil
}

// Stop stops the commentary generator
func (g *CommentaryGenerator) Stop() error {
	g.mu.Lock()
	if !g.active {
		g.mu.Unlock()
		return nil
	}

	g.active = false
	g.cancel()
	g.mu.Unlock()

	// Close request channel to signal workers to stop
	close(g.requestChan)

	// Wait for workers to finish
	g.wg.Wait()

	g.logger.Info().Msg("Commentary generator stopped")
	return nil
}

// Generate generates commentary for the given request
func (g *CommentaryGenerator) Generate(ctx context.Context, request *types.CommentaryRequest) (*types.CommentaryResponse, error) {
	startTime := time.Now()

	if err := g.validateRequest(request); err != nil {
		return g.errorResponse(err, startTime), nil
	}

	// Check if generator is active
	g.mu.RLock()
	active := g.active
	g.mu.RUnlock()

	if !active {
		return g.errorResponse(fmt.Errorf("generator not active"), startTime), nil
	}

	// Create internal request
	genReq := &generationRequest{
		request:   request,
		response:  make(chan *types.CommentaryResponse, 1),
		ctx:       ctx,
		startTime: startTime,
		logEntries: []types.LogEntry{
			{
				Step:      "request_received",
				Timestamp: startTime,
				Success:   true,
				Message:   "Commentary generation request received",
			},
		},
	}

	// Apply timeout
	timeout := request.MaxLatency
	if timeout == 0 {
		timeout = g.config.MaxLatency
	}

	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Submit request to worker pool
	select {
	case g.requestChan <- genReq:
		// Request queued successfully
		atomic.AddInt64(&g.activeRequests, 1)
		defer atomic.AddInt64(&g.activeRequests, -1)

	case <-requestCtx.Done():
		return g.errorResponse(fmt.Errorf("request queue timeout"), startTime), nil
	}

	// Wait for response
	select {
	case response := <-genReq.response:
		// Update metrics
		g.updateMetrics(response, startTime)
		return response, nil

	case <-requestCtx.Done():
		return g.errorResponse(fmt.Errorf("generation timeout"), startTime), nil
	}
}

// worker.run processes generation requests
func (w *worker) run() {
	defer w.generator.wg.Done()

	w.logger.Debug().Msg("Worker started")

	for request := range w.generator.requestChan {
		response := w.processRequest(request)

		select {
		case request.response <- response:
			// Response sent successfully
		case <-request.ctx.Done():
			// Request context cancelled, skip response
			w.logger.Debug().Msg("Request context cancelled, skipping response")
		}
	}

	w.logger.Debug().Msg("Worker stopped")
}

// processRequest processes a single generation request
func (w *worker) processRequest(request *generationRequest) *types.CommentaryResponse {
	startTime := request.startTime

	// Add processing step
	request.logEntries = append(request.logEntries, types.LogEntry{
		Step:      "processing_started",
		Timestamp: time.Now(),
		Success:   true,
		Message:   fmt.Sprintf("Worker %d processing request", w.id),
	})

	// Step 1: Build prompt
	promptStartTime := time.Now()
	promptReq := &prompt.BuildRequest{
		Transcription: request.request.Input.Text,
		Confidence:    request.request.Input.Confidence,
		Timestamp:     request.request.Input.Timestamp,
		TemplateID:    request.request.Input.Text, // Will be selected automatically
		ExtraContext:  request.request.Input.ExtraData,
	}

	if request.request.Input.AudioMetrics != nil {
		promptReq.AudioMetrics = &templates.AudioMetrics{
			Volume:       1.0 - request.request.Input.AudioMetrics.BackgroundNoise, // Convert background noise to volume
			Clarity:      request.request.Input.AudioMetrics.Clarity,
			NoiseLevel:   request.request.Input.AudioMetrics.BackgroundNoise,
			VoicePresent: request.request.Input.AudioMetrics.VoiceDetection,
		}
	}

	if request.request.Input.AudioMetrics != nil {
		promptReq.AudioMetrics = &templates.AudioMetrics{
			Volume:       1.0 - request.request.Input.AudioMetrics.BackgroundNoise, // Convert background noise to volume
			Clarity:      request.request.Input.AudioMetrics.Clarity,
			NoiseLevel:   request.request.Input.AudioMetrics.BackgroundNoise,
			VoicePresent: request.request.Input.AudioMetrics.VoiceDetection,
		}
	}

	promptResult, err := w.generator.promptBuilder.Build(promptReq)
	if err != nil {
		request.logEntries = append(request.logEntries, types.LogEntry{
			Step:      "prompt_build_failed",
			Timestamp: time.Now(),
			Duration:  time.Since(promptStartTime),
			Success:   false,
			Message:   err.Error(),
		})

		// Try fallback
		return w.generateFallback(request, fmt.Errorf("prompt build failed: %w", err))
	}

	request.logEntries = append(request.logEntries, types.LogEntry{
		Step:      "prompt_built",
		Timestamp: time.Now(),
		Duration:  time.Since(promptStartTime),
		Success:   true,
		Message:   fmt.Sprintf("Prompt built using template %s", promptResult.TemplateID),
		Data:      map[string]interface{}{"template_id": promptResult.TemplateID, "prompt_length": promptResult.PromptLength},
	})

	// Step 2: Generate with LLM
	if w.generator.llmEngine == nil {
		// No LLM engine available, go directly to fallback
		return w.generateFallback(request, fmt.Errorf("no LLM engine available"))
	}

	llmStartTime := time.Now()
	llmReq := llmtypes.GenerationRequest{
		Prompt:      promptResult.Prompt,
		MaxTokens:   150, // Default for commentary
		Temperature: 0.7,
		TopP:        0.9,
		TopK:        40,
		Timeout:     15 * time.Second, // Increased from 2s to allow for LLM generation
	}

	llmResp, err := w.generator.llmEngine.Generate(llmReq)
	if err != nil {
		request.logEntries = append(request.logEntries, types.LogEntry{
			Step:      "llm_generation_failed",
			Timestamp: time.Now(),
			Duration:  time.Since(llmStartTime),
			Success:   false,
			Message:   err.Error(),
		})

		// Try fallback
		return w.generateFallback(request, fmt.Errorf("LLM generation failed: %w", err))
	}

	request.logEntries = append(request.logEntries, types.LogEntry{
		Step:      "llm_generated",
		Timestamp: time.Now(),
		Duration:  time.Since(llmStartTime),
		Success:   true,
		Message:   fmt.Sprintf("Generated %d tokens", llmResp.TokenCount),
		Data:      map[string]interface{}{"token_count": llmResp.TokenCount, "tokens_per_second": llmResp.Metadata.TokensPerSecond},
	})

	// Step 3: Validate quality
	validationStartTime := time.Now()
	validation := w.generator.validator.Validate(llmResp.Text, request.request.Input)

	request.logEntries = append(request.logEntries, types.LogEntry{
		Step:      "quality_validated",
		Timestamp: time.Now(),
		Duration:  time.Since(validationStartTime),
		Success:   validation.IsValid,
		Message:   fmt.Sprintf("Quality validation: %v (score: %.2f)", validation.IsValid, validation.Confidence),
		Data:      map[string]interface{}{"quality_score": validation.Confidence, "issues": validation.Issues},
	})

	if !validation.IsValid && w.generator.config.EnableFallback {
		// Try fallback
		return w.generateFallback(request, fmt.Errorf("quality validation failed: %v", validation.Issues))
	}

	// Step 4: Create commentary result
	commentary := &types.Commentary{
		ID:          generateCommentaryID(),
		Text:        llmResp.Text,
		DisplayText: w.generator.formatForDisplay(llmResp.Text),
		Source:      "llm",
		Confidence:  validation.Confidence,
		Timestamp:   time.Now(),

		InputText:       request.request.Input.Text,
		InputConfidence: request.request.Input.Confidence,
		TemplateID:      promptResult.TemplateID,
		PromptUsed:      promptResult.Prompt,

		GenerationLatency: llmResp.Latency,
		CacheHit:          false,

		QualityScore:     validation.Confidence,
		RelevanceScore:   validation.Relevance,
		ValidationPassed: validation.IsValid,

		Metadata: types.CommentaryMetadata{
			Category:        promptResult.Metadata["template_category"].(string),
			Priority:        promptResult.Metadata["template_priority"].(string),
			ProcessingSteps: extractSteps(request.logEntries),
			ExtraData:       make(map[string]string),
		},
	}

	// Add match context if available
	if request.request.Input.Context != nil {
		commentary.Metadata.MatchContext = request.request.Input.Context
	}

	// Add audio quality if available
	if request.request.Input.AudioMetrics != nil {
		commentary.Metadata.AudioQuality = request.request.Input.AudioMetrics
	}

	totalLatency := time.Since(startTime)

	return &types.CommentaryResponse{
		Commentary:    commentary,
		Success:       true,
		Latency:       totalLatency,
		Timestamp:     time.Now(),
		ProcessingLog: request.logEntries,
	}
}

// generateFallback generates fallback commentary when primary generation fails
func (w *worker) generateFallback(request *generationRequest, originalErr error) *types.CommentaryResponse {
	if !w.generator.config.EnableFallback {
		return w.generator.errorResponse(originalErr, request.startTime)
	}

	w.logger.Warn().
		Err(originalErr).
		Str("input", request.request.Input.Text).
		Msg("Primary generation failed, using fallback")

	fallbackStartTime := time.Now()
	fallbackCommentary, err := w.generator.fallback.Generate(request.request.Input)

	request.logEntries = append(request.logEntries, types.LogEntry{
		Step:      "fallback_generated",
		Timestamp: time.Now(),
		Duration:  time.Since(fallbackStartTime),
		Success:   err == nil,
		Message:   "Fallback generation attempted",
		Data:      map[string]interface{}{"original_error": originalErr.Error()},
	})

	if err != nil {
		return w.generator.errorResponse(fmt.Errorf("fallback failed: %w", err), request.startTime)
	}

	// Mark as fallback source
	fallbackCommentary.Source = "fallback"
	fallbackCommentary.Metadata.Fallbacks = []string{originalErr.Error()}

	totalLatency := time.Since(request.startTime)

	return &types.CommentaryResponse{
		Commentary:    fallbackCommentary,
		Success:       true,
		Latency:       totalLatency,
		Timestamp:     time.Now(),
		ProcessingLog: request.logEntries,
	}
}

// Helper functions

func (g *CommentaryGenerator) validateRequest(request *types.CommentaryRequest) error {
	if request == nil {
		return fmt.Errorf("request cannot be nil")
	}
	if request.Input.Text == "" {
		return fmt.Errorf("input text cannot be empty")
	}
	if request.Input.Confidence < 0 || request.Input.Confidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1")
	}
	return nil
}

func (g *CommentaryGenerator) errorResponse(err error, startTime time.Time) *types.CommentaryResponse {
	return &types.CommentaryResponse{
		Success:   false,
		Error:     err.Error(),
		Latency:   time.Since(startTime),
		Timestamp: time.Now(),
	}
}

func (g *CommentaryGenerator) updateMetrics(response *types.CommentaryResponse, startTime time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.metrics.TotalRequests++

	if response.Success {
		g.metrics.SuccessfulReqs++

		if response.Commentary != nil {
			switch response.Commentary.Source {
			case "llm":
				g.metrics.LLMGenerated++
			case "cache":
				g.metrics.CacheHits++
			case "fallback":
				g.metrics.FallbackUsed++
			}

			// Update quality metrics
			g.metrics.AvgQualityScore = updateAverage(g.metrics.AvgQualityScore, response.Commentary.QualityScore, g.metrics.SuccessfulReqs)
			g.metrics.AvgRelevanceScore = updateAverage(g.metrics.AvgRelevanceScore, response.Commentary.RelevanceScore, g.metrics.SuccessfulReqs)

			if !response.Commentary.ValidationPassed {
				g.metrics.ValidationFailures++
			}
		}
	} else {
		g.metrics.FailedReqs++
	}

	// Update latency metrics
	latency := response.Latency
	g.metrics.AvgLatency = updateAverageDuration(g.metrics.AvgLatency, latency, g.metrics.TotalRequests)

	if latency > g.metrics.MaxLatency {
		g.metrics.MaxLatency = latency
	}

	g.metrics.ActiveRequests = int(atomic.LoadInt64(&g.activeRequests))
	g.metrics.LastUpdated = time.Now()
}

func (g *CommentaryGenerator) formatForDisplay(text string) string {
	// Basic formatting for OBS overlay
	// In a real implementation, this would include proper text formatting,
	// line breaks, length limits, etc.
	if len(text) > 120 {
		text = text[:117] + "..."
	}
	return text
}

// GetMetrics returns current metrics
func (g *CommentaryGenerator) GetMetrics() *types.GenerationMetrics {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Create a copy to avoid race conditions
	metrics := *g.metrics
	return &metrics
}

// GetStatus returns the current status of the generator
func (g *CommentaryGenerator) GetStatus() map[string]interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()

	status := map[string]interface{}{
		"active":          g.active,
		"worker_count":    g.workerCount,
		"active_requests": atomic.LoadInt64(&g.activeRequests),
		"queue_size":      len(g.requestChan),
		"llm_ready":       g.llmEngine.IsReady(),
	}

	if g.llmEngine != nil {
		status["llm_status"] = g.llmEngine.GetStatus()
	}

	return status
}

// Utility functions

func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

func generateCommentaryID() string {
	return fmt.Sprintf("comm_%d", time.Now().UnixNano())
}

func extractSteps(logEntries []types.LogEntry) []string {
	steps := make([]string, len(logEntries))
	for i, entry := range logEntries {
		steps[i] = entry.Step
	}
	return steps
}

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
