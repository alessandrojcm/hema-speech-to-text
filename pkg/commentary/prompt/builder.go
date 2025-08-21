package prompt

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/your-org/hema-replay-system/pkg/commentary/context"
	"github.com/your-org/hema-replay-system/pkg/commentary/templates"
)

// Builder handles dynamic prompt construction for commentary generation
type Builder struct {
	templateManager  *templates.TemplateManager
	templateSelector *templates.TemplateSelector
	contextManager   *context.ContextManager
	logger           zerolog.Logger
	config           *BuilderConfig
}

// BuilderConfig configures the prompt builder behavior
type BuilderConfig struct {
	EnableContextEnrichment bool    `mapstructure:"enable_context_enrichment"`
	MaxPromptLength         int     `mapstructure:"max_prompt_length"`
	DefaultTemplate         string  `mapstructure:"default_template"`
	ContextWeight           float32 `mapstructure:"context_weight"`
	IncludeTimestamp        bool    `mapstructure:"include_timestamp"`
	IncludeConfidence       bool    `mapstructure:"include_confidence"`
	IncludeAudioMetrics     bool    `mapstructure:"include_audio_metrics"`
	FallbackBehavior        string  `mapstructure:"fallback_behavior"` // "default", "simple", "none"
}

// DefaultBuilderConfig returns default configuration for the prompt builder
func DefaultBuilderConfig() *BuilderConfig {
	return &BuilderConfig{
		EnableContextEnrichment: true,
		MaxPromptLength:         500,
		DefaultTemplate:         "point_scored",
		ContextWeight:           1.0,
		IncludeTimestamp:        false,
		IncludeConfidence:       true,
		IncludeAudioMetrics:     false,
		FallbackBehavior:        "default",
	}
}

// NewBuilder creates a new prompt builder
func NewBuilder(
	templateManager *templates.TemplateManager,
	contextManager *context.ContextManager,
	config *BuilderConfig,
	logger zerolog.Logger,
) *Builder {
	if config == nil {
		config = DefaultBuilderConfig()
	}

	return &Builder{
		templateManager:  templateManager,
		templateSelector: templates.NewTemplateSelector(),
		contextManager:   contextManager,
		logger:           logger.With().Str("component", "prompt-builder").Logger(),
		config:           config,
	}
}

// BuildRequest represents a request to build a prompt
type BuildRequest struct {
	Transcription string                  `json:"transcription"`
	Confidence    float32                 `json:"confidence"`
	Timestamp     time.Time               `json:"timestamp"`
	AudioMetrics  *templates.AudioMetrics `json:"audio_metrics,omitempty"`
	TemplateID    string                  `json:"template_id,omitempty"` // Optional override
	ExtraContext  map[string]string       `json:"extra_context,omitempty"`
}

// BuildResult represents the result of prompt building
type BuildResult struct {
	Prompt       string                 `json:"prompt"`
	TemplateID   string                 `json:"template_id"`
	TemplateName string                 `json:"template_name"`
	UsedContext  map[string]string      `json:"used_context"`
	PromptLength int                    `json:"prompt_length"`
	BuildTime    time.Duration          `json:"build_time"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Build constructs a prompt based on the build request
func (b *Builder) Build(request *BuildRequest) (*BuildResult, error) {
	startTime := time.Now()

	if err := b.validateRequest(request); err != nil {
		return nil, fmt.Errorf("invalid build request: %w", err)
	}

	// Select template
	templateID := b.selectTemplate(request)

	template, exists := b.templateManager.GetTemplate(templateID)
	if !exists {
		return b.buildFallbackResult(request, fmt.Errorf("template %s not found", templateID))
	}

	// Build template data
	templateData, err := b.buildTemplateData(request)
	if err != nil {
		return b.buildFallbackResult(request, fmt.Errorf("failed to build template data: %w", err))
	}

	// Execute template
	prompt, err := b.templateManager.ExecuteTemplate(templateID, templateData)
	if err != nil {
		return b.buildFallbackResult(request, fmt.Errorf("failed to execute template: %w", err))
	}

	// Validate and trim prompt if necessary
	prompt = b.validateAndTrimPrompt(prompt)

	buildTime := time.Since(startTime)

	result := &BuildResult{
		Prompt:       prompt,
		TemplateID:   templateID,
		TemplateName: template.Name,
		UsedContext:  templateData.Context,
		PromptLength: len(prompt),
		BuildTime:    buildTime,
		Metadata:     make(map[string]interface{}),
	}

	// Add metadata
	result.Metadata["template_category"] = template.Metadata["category"]
	result.Metadata["template_priority"] = template.Metadata["priority"]
	result.Metadata["build_timestamp"] = startTime

	b.logger.Debug().
		Str("template_id", templateID).
		Str("template_name", template.Name).
		Int("prompt_length", len(prompt)).
		Dur("build_time", buildTime).
		Msg("Built prompt successfully")

	return result, nil
}

// selectTemplate selects the appropriate template for the request
func (b *Builder) selectTemplate(request *BuildRequest) string {
	// Use explicit template ID if provided
	if request.TemplateID != "" {
		if _, exists := b.templateManager.GetTemplate(request.TemplateID); exists {
			return request.TemplateID
		}
		b.logger.Warn().
			Str("requested_template", request.TemplateID).
			Msg("Requested template not found, using selector")
	}

	// Get match state for template selection
	matchState := b.contextManager.GetMatchState()

	// Use template selector
	selectedID := b.templateSelector.SelectTemplate(request.Transcription, matchState)

	// Verify selected template exists
	if _, exists := b.templateManager.GetTemplate(selectedID); !exists {
		b.logger.Warn().
			Str("selected_template", selectedID).
			Str("fallback_template", b.config.DefaultTemplate).
			Msg("Selected template not found, using default")
		return b.config.DefaultTemplate
	}

	return selectedID
}

// buildTemplateData constructs the data structure for template execution
func (b *Builder) buildTemplateData(request *BuildRequest) (*templates.TemplateData, error) {
	data := &templates.TemplateData{
		Transcription: request.Transcription,
		Confidence:    request.Confidence,
		Timestamp:     request.Timestamp,
		Context:       make(map[string]string),
	}

	// Add match state
	data.MatchState = b.contextManager.GetMatchState()

	// Add audio metrics if configured
	if b.config.IncludeAudioMetrics && request.AudioMetrics != nil {
		data.AudioMetrics = request.AudioMetrics
	}

	// Add recent calls context
	data.PreviousCalls = b.contextManager.GetRecentCalls(3)

	// Enrich context if enabled
	if b.config.EnableContextEnrichment {
		enrichedContext := b.contextManager.EnrichContext(request.Transcription)
		for k, v := range enrichedContext {
			data.Context[k] = v
		}
	}

	// Add extra context from request
	if request.ExtraContext != nil {
		for k, v := range request.ExtraContext {
			data.Context[k] = v
		}
	}

	// Add build-time context
	if b.config.IncludeTimestamp {
		data.Context["timestamp"] = request.Timestamp.Format(time.RFC3339)
	}

	if b.config.IncludeConfidence {
		data.Context["confidence"] = fmt.Sprintf("%.2f", request.Confidence)
	}

	return data, nil
}

// validateRequest validates the build request
func (b *Builder) validateRequest(request *BuildRequest) error {
	if request == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if request.Transcription == "" {
		return fmt.Errorf("transcription cannot be empty")
	}

	if request.Confidence < 0 || request.Confidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1, got %f", request.Confidence)
	}

	if request.Timestamp.IsZero() {
		request.Timestamp = time.Now()
	}

	return nil
}

// validateAndTrimPrompt validates and trims the prompt if necessary
func (b *Builder) validateAndTrimPrompt(prompt string) string {
	prompt = strings.TrimSpace(prompt)

	if len(prompt) > b.config.MaxPromptLength {
		// Trim to max length, trying to break at sentence boundaries
		truncated := prompt[:b.config.MaxPromptLength]

		// Try to find the last sentence boundary
		lastPeriod := strings.LastIndex(truncated, ".")
		lastExclamation := strings.LastIndex(truncated, "!")
		lastQuestion := strings.LastIndex(truncated, "?")

		lastSentence := lastPeriod
		if lastExclamation > lastSentence {
			lastSentence = lastExclamation
		}
		if lastQuestion > lastSentence {
			lastSentence = lastQuestion
		}

		if lastSentence > 0 && lastSentence > len(truncated)*3/4 {
			// If we found a sentence boundary in the last quarter, use it
			prompt = truncated[:lastSentence+1]
		} else {
			// Otherwise, just truncate and add ellipsis
			prompt = truncated + "..."
		}

		b.logger.Warn().
			Int("original_length", len(prompt)).
			Int("max_length", b.config.MaxPromptLength).
			Int("final_length", len(prompt)).
			Msg("Prompt was truncated")
	}

	return prompt
}

// buildFallbackResult builds a fallback result when template building fails
func (b *Builder) buildFallbackResult(request *BuildRequest, originalErr error) (*BuildResult, error) {
	b.logger.Error().Err(originalErr).Msg("Template building failed, using fallback")

	switch b.config.FallbackBehavior {
	case "simple":
		return b.buildSimpleFallback(request), nil
	case "none":
		return nil, originalErr
	default: // "default"
		return b.buildDefaultFallback(request), nil
	}
}

// buildSimpleFallback creates a very basic prompt
func (b *Builder) buildSimpleFallback(request *BuildRequest) *BuildResult {
	prompt := fmt.Sprintf("Comment on this HEMA judge call: \"%s\"", request.Transcription)

	return &BuildResult{
		Prompt:       prompt,
		TemplateID:   "simple_fallback",
		TemplateName: "Simple Fallback",
		UsedContext:  make(map[string]string),
		PromptLength: len(prompt),
		BuildTime:    0,
		Metadata: map[string]interface{}{
			"fallback": "simple",
		},
	}
}

// buildDefaultFallback creates a fallback using the default template
func (b *Builder) buildDefaultFallback(request *BuildRequest) *BuildResult {
	// Try to use the default template
	template, exists := b.templateManager.GetTemplate(b.config.DefaultTemplate)
	if !exists {
		return b.buildSimpleFallback(request)
	}

	// Create minimal template data
	data := &templates.TemplateData{
		Transcription: request.Transcription,
		Confidence:    request.Confidence,
		Timestamp:     request.Timestamp,
		Context:       make(map[string]string),
	}

	prompt, err := b.templateManager.ExecuteTemplate(b.config.DefaultTemplate, data)
	if err != nil {
		b.logger.Error().Err(err).Msg("Default template execution failed")
		return b.buildSimpleFallback(request)
	}

	return &BuildResult{
		Prompt:       prompt,
		TemplateID:   b.config.DefaultTemplate,
		TemplateName: template.Name,
		UsedContext:  data.Context,
		PromptLength: len(prompt),
		BuildTime:    0,
		Metadata: map[string]interface{}{
			"fallback": "default_template",
		},
	}
}

// BatchBuild builds multiple prompts in a batch
func (b *Builder) BatchBuild(requests []*BuildRequest) ([]*BuildResult, []error) {
	results := make([]*BuildResult, len(requests))
	errors := make([]error, len(requests))

	for i, request := range requests {
		result, err := b.Build(request)
		results[i] = result
		errors[i] = err
	}

	return results, errors
}

// GetStats returns builder statistics
func (b *Builder) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"template_count": len(b.templateManager.ListTemplates()),
		"config":         b.config,
	}
}
