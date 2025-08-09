//go:build noaudio

package whisper

import (
	"fmt"

	"github.com/rs/zerolog"
	"github.com/your-org/hema-replay-system/pkg/speech/types"
)

// ModelManager manages whisper models (stub implementation)
type ModelManager struct {
	config types.WhisperConfig
	logger zerolog.Logger
}

// NewModelManager creates a new model manager (stub)
func NewModelManager(config types.WhisperConfig, logger zerolog.Logger) *ModelManager {
	return &ModelManager{
		config: config,
		logger: logger.With().Str("component", "whisper_model_manager_stub").Logger(),
	}
}

// LoadModel loads a whisper model (stub)
func (mm *ModelManager) LoadModel(modelSize types.ModelSize) error {
	mm.logger.Debug().Str("model", modelSize.String()).Msg("Stub: LoadModel called")
	return fmt.Errorf("whisper not available in noaudio build")
}

// UnloadAllModels unloads all models (stub)
func (mm *ModelManager) UnloadAllModels() {
	mm.logger.Debug().Msg("Stub: UnloadAllModels called")
}

// GetModel gets a loaded model (stub)
func (mm *ModelManager) GetModel(modelSize types.ModelSize) (*Wrapper, error) {
	return nil, fmt.Errorf("whisper not available in noaudio build")
}

// GetLoadedModels returns list of loaded models (stub)
func (mm *ModelManager) GetLoadedModels() []string {
	return []string{}
}

// CheckHealth checks model manager health (stub)
func (mm *ModelManager) CheckHealth() error {
	return fmt.Errorf("whisper not available in noaudio build")
}
