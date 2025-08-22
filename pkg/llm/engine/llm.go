package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/openai/openai-go/v2/packages/param"
	"github.com/rs/zerolog"
	"github.com/your-org/hema-replay-system/pkg/llm/types"
)

const systemPrompt = `You are a Historical European Martial Arts (HEMA) expert assisting on a live tournament.
Your task is to listen to calls made by the main judge and translate them to layman terms so the audience on the livestream can understand. 

Explanations should be concise and to the point. Preferable one sentence as they will be used in the replay.

This tournament is loosely based in the irish ruleset, here are the basic rules:
1. Fencers identified by red and blue.
2. Point areas are defined between deep and shallow. Shallow areas are worth 3 points and deep areas are worth 5 points
3. Certain strikes with certain weapons are worth 1 point. For example, a strike with a buckler is worth 1 point. Pushing the opponent out of the ring is also worth one point.
4. Daggers are special, they separate deep and shallow. But a shallow target with a dagger is worth 1 points and a deep is worth 3 points
5. Afterblows occurs when one participants hits back their opponent right after the opponent hits them (and they are in tempo).
Afterblows are scored by subtracting. That is, if red hits back blue after blue hits red, red gets a deep but blue gets a shallow.
That is 5 points to red and 3 points to blue. The points are subtracted (5 - 3) = 2 points for red.
If both fencers hit the same area they get zero points.
6. Shallow targets are to the arms, hands and the outside part of the legs. Deep targets are to the inside of the legs, the head and the torso.
8. Fights last up to 9 exchanges or 3 mins or 20 points cap, whichever comes first.

Here are some important rules for your task:
1. You are acting as a translator, you will not add any additional unnecessary commentaries.
2. If you cannot understand the input (for example the judge did not mention colours of the fencers), say nothing.
3. Your explanation needs to be as short and accurate as possible.
4. Unless the judge specifically calls the body part that was hit, DO NOT mention it.

Here are some examples:

<example>
Judge: red hit to the head
Explanation: Red hits blue on the head, so they get 5 points.
</example>

<example>
Judge: blue shallow target red
Explanation: Blue gets 3 points (shallow target)..
</example>

<example>
Judge: Blue ring out
Explanation: Red gets 1 point as they pushed blue out of the ring.
</example>

<example>
Judge: blow afterblow no points
Explanation: Both fencers hit the same area, so no points.
</example>

<example>
Judge: red shallow target, blue afterflow deep target.
Explanation: Red hits with a shallow but blue hits right back with a deep target, so blue gets 3 points.
</example>

Now transcribe the judge call below
`

// ModelEngine wraps go-llama.cpp bindings for text generation
type ModelEngine struct {
	mu        sync.RWMutex
	logger    zerolog.Logger
	isReady   bool
	client    openai.Client
	ctx       context.Context
	startTime time.Time
	config    *types.LLMConfig

	// Metrics
	totalRequests  uint64
	successfulReqs uint64
	totalLatency   time.Duration
	activeRequests int
	lastError      string
	memoryUsage    uint64
}

// NewLlmEngine creates a new ModelEngine instance
func NewLlmEngine(config *types.LLMConfig, ctx context.Context, logger zerolog.Logger) (*ModelEngine, error) {
	engine := &ModelEngine{
		logger: logger.With().Str("component", "llama_engine").Logger(),
		client: openai.NewClient(
			option.WithBaseURL(config.Endpoint),
			option.WithRequestTimeout(config.Timeout)),
		ctx: ctx,
	}
	engine.config = config
	engine.startTime = time.Now()
	engine.isReady = true
	engine.logger.Info().
		Str("endpoint", config.Endpoint).
		Str("model", config.ModelID).
		Msg("ModelEngine initialized successfully")

	return engine, nil
}

// Generate generates text based on the given request
func (e *ModelEngine) Generate(request types.GenerationRequest) (*types.GenerationResponse, error) {
	if !e.IsReady() {
		return nil, types.ErrEngineNotReady
	}

	if request.Prompt == "" {
		return nil, types.ErrInvalidPrompt
	}

	e.mu.Lock()
	e.activeRequests++
	e.totalRequests++
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		e.activeRequests--
		e.mu.Unlock()
	}()

	startTime := time.Now()

	response, err := e.generateText(request)
	if err != nil {
		e.recordError(err)
		return nil, fmt.Errorf("generation failed: %w", err)
	}

	latency := time.Since(startTime)

	e.mu.Lock()
	e.successfulReqs++
	e.totalLatency += latency
	e.mu.Unlock()

	return response, nil
}

// generateText performs the actual text generation
func (e *ModelEngine) generateText(request types.GenerationRequest) (*types.GenerationResponse, error) {
	startTime := time.Now()

	e.mu.RLock()
	defer e.mu.RUnlock()

	// Generate with timeout
	resultChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		response, err := e.client.Chat.Completions.New(e.ctx, openai.ChatCompletionNewParams{
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage(systemPrompt),
				openai.UserMessage(request.Prompt),
			},
			MaxCompletionTokens: param.NewOpt[int64](int64(request.MaxTokens)),
			Temperature:         param.NewOpt[float64](float64(request.Temperature)),
			Model:               e.config.ModelID,
			TopP:                param.NewOpt[float64](float64(request.TopP)),
		}, option.WithRequestTimeout(request.Timeout))
		if err != nil {
			errChan <- err
			return
		}
		resultChan <- response.Choices[0].Message.Content
	}()

	select {
	case text := <-resultChan:
		// Process successful generation
		processingTime := time.Since(startTime)

		return &types.GenerationResponse{
			Text:         text,
			TokenCount:   len(text) / 4, // Rough estimate, 4 chars per token
			FinishReason: "stop",
			Latency:      processingTime,
			Metadata: types.GenerationMetadata{
				ModelName:        "Qwen3",
				TokensPerSecond:  float64(len(text)/4) / processingTime.Seconds(),
				PromptTokens:     len(request.Prompt) / 4, // Rough estimate
				CompletionTokens: len(text) / 4,           // Rough estimate
				TotalTokens:      (len(request.Prompt) + len(text)) / 4,
				ProcessingTime:   processingTime,
			},
			Timestamp: time.Now(),
		}, nil

	case err := <-errChan:
		return nil, err
	}
}

// GetStatus returns the current status of the engine
func (e *ModelEngine) GetStatus() types.EngineStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var successRate float64
	if e.totalRequests > 0 {
		successRate = float64(e.successfulReqs) / float64(e.totalRequests)
	}

	var avgLatency time.Duration
	if e.successfulReqs > 0 {
		avgLatency = e.totalLatency / time.Duration(e.successfulReqs)
	}

	return types.EngineStatus{
		Ready:          e.isReady,
		ModelLoaded:    e.isReady,
		ModelPath:      "Qwen3",
		MemoryUsage:    e.memoryUsage,
		ActiveRequests: e.activeRequests,
		TotalRequests:  e.totalRequests,
		SuccessRate:    successRate,
		AvgLatency:     avgLatency,
		LastError:      e.lastError,
		Uptime:         time.Since(e.startTime),
	}
}

// IsReady returns true if the engine is ready to generate text
func (e *ModelEngine) IsReady() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.isReady
}

// Close releases resources used by the engine
func (e *ModelEngine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.isReady {
		return nil
	}
	e.isReady = false
	e.logger.Info().Msg("ModelEngine closed")
	return nil
}

// recordError records an error and updates metrics
func (e *ModelEngine) recordError(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.lastError = err.Error()
	e.logger.Error().Err(err).Msg("Generation error")
}
