//go:build !noaudio

package whisper

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/your-org/hema-replay-system/pkg/speech/types"
)

// ModelManager manages whisper model loading and lifecycle
type ModelManager struct {
	config     types.WhisperConfig
	models     map[types.ModelSize]*WhisperWrapper
	modelPaths map[types.ModelSize]string
	mu         sync.RWMutex
	logger     zerolog.Logger
}

// NewModelManager creates a new model manager
func NewModelManager(config types.WhisperConfig, logger zerolog.Logger) *ModelManager {
	return &ModelManager{
		config:     config,
		models:     make(map[types.ModelSize]*WhisperWrapper),
		modelPaths: make(map[types.ModelSize]string),
		logger:     logger.With().Str("component", "model_manager").Logger(),
	}
}

// LoadModel loads a whisper model of specified size
func (mm *ModelManager) LoadModel(modelSize types.ModelSize) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	// Check if model is already loaded
	if _, exists := mm.models[modelSize]; exists {
		return nil
	}

	modelPath := mm.getModelPath(modelSize)
	if !mm.modelExists(modelPath) {
		return fmt.Errorf("model file not found: %s", modelPath)
	}

	mm.logger.Info().
		Str("model_size", modelSize.String()).
		Str("model_path", modelPath).
		Msg("Loading whisper model")

	startTime := time.Now()
	wrapper, err := NewWhisperWrapper(modelPath, mm.logger)
	if err != nil {
		return fmt.Errorf("failed to load model %s: %w", modelSize.String(), err)
	}

	loadTime := time.Since(startTime)
	mm.models[modelSize] = wrapper
	mm.modelPaths[modelSize] = modelPath

	mm.logger.Info().
		Str("model_size", modelSize.String()).
		Dur("load_time", loadTime).
		Msg("Whisper model loaded successfully")

	return nil
}

// GetModel returns a loaded model wrapper
func (mm *ModelManager) GetModel(modelSize types.ModelSize) (*WhisperWrapper, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	wrapper, exists := mm.models[modelSize]
	if !exists {
		return nil, fmt.Errorf("model %s not loaded", modelSize.String())
	}

	return wrapper, nil
}

// UnloadModel unloads a specific model
func (mm *ModelManager) UnloadModel(modelSize types.ModelSize) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	wrapper, exists := mm.models[modelSize]
	if !exists {
		return nil
	}

	wrapper.Close()
	delete(mm.models, modelSize)
	delete(mm.modelPaths, modelSize)

	mm.logger.Info().
		Str("model_size", modelSize.String()).
		Msg("Whisper model unloaded")

	return nil
}

// UnloadAllModels unloads all loaded models
func (mm *ModelManager) UnloadAllModels() {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	for modelSize, wrapper := range mm.models {
		wrapper.Close()
		mm.logger.Info().
			Str("model_size", modelSize.String()).
			Msg("Whisper model unloaded")
	}

	mm.models = make(map[types.ModelSize]*WhisperWrapper)
	mm.modelPaths = make(map[types.ModelSize]string)
}

// getModelPath returns the file path for a model size
func (mm *ModelManager) getModelPath(modelSize types.ModelSize) string {
	baseDir := filepath.Dir(mm.config.ModelPath)
	filename := fmt.Sprintf("ggml-%s.bin", modelSize.String())
	return filepath.Join(baseDir, filename)
}

// modelExists checks if a model file exists
func (mm *ModelManager) modelExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetLoadedModels returns a list of currently loaded models
func (mm *ModelManager) GetLoadedModels() []types.ModelSize {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	models := make([]types.ModelSize, 0, len(mm.models))
	for modelSize := range mm.models {
		models = append(models, modelSize)
	}

	return models
}

// GetModelInfo returns information about a loaded model
func (mm *ModelManager) GetModelInfo(modelSize types.ModelSize) (map[string]interface{}, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	_, exists := mm.models[modelSize]
	if !exists {
		return nil, fmt.Errorf("model %s not loaded", modelSize.String())
	}

	path := mm.modelPaths[modelSize]
	stat, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get model file info: %w", err)
	}

	return map[string]interface{}{
		"model_size":    modelSize.String(),
		"model_path":    path,
		"file_size":     stat.Size(),
		"modified_time": stat.ModTime(),
		"loaded":        true,
	}, nil
}
