//go:build nollm

package engine

import (
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/your-org/hema-replay-system/pkg/llm/types"
)

// LlamaEngine stub implementation for builds without LLM support
type LlamaEngine struct {
	logger zerolog.Logger
}

// NewLlamaEngine creates a new LlamaEngine stub instance
func NewLlamaEngine(config *types.LLMConfig, logger zerolog.Logger) (*LlamaEngine, error) {
	engine := &LlamaEngine{
		logger: logger.With().Str("component", "llama_engine_stub").Logger(),
	}

	engine.logger.Warn().Msg("LlamaEngine stub initialized - LLM functionality disabled")
	return engine, nil
}

// Generate returns an error indicating LLM is not available
func (e *LlamaEngine) Generate(request types.GenerationRequest) (*types.GenerationResponse, error) {
	e.logger.Debug().Msg("Stub: Generate called")
	return nil, fmt.Errorf("llm generation not available in nollm build")
}

// GetStatus returns a stub status
func (e *LlamaEngine) GetStatus() types.EngineStatus {
	return types.EngineStatus{
		Ready:       false,
		ModelLoaded: false,
		ModelPath:   "stub",
		LastError:   "LLM not available in nollm build",
	}
}

// IsReady always returns false for stub
func (e *LlamaEngine) IsReady() bool {
	return false
}

// Close does nothing for stub
func (e *LlamaEngine) Close() error {
	e.logger.Debug().Msg("Stub: Close called")
	return nil
}
