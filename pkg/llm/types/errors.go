package types

import "errors"

var (
	// Configuration errors
	ErrInvalidModelPath   = errors.New("invalid model path")
	ErrInvalidContextSize = errors.New("invalid context size: must be positive")
	ErrInvalidThreadCount = errors.New("invalid thread count: must be positive")
	ErrInvalidTemperature = errors.New("invalid temperature: must be between 0 and 2.0")
	ErrInvalidTopP        = errors.New("invalid top_p: must be between 0 and 1.0")
	ErrInvalidMaxTokens   = errors.New("invalid max_tokens: must be positive")
	ErrInvalidTimeout     = errors.New("invalid timeout: must be positive")

	// Model loading errors
	ErrModelNotFound        = errors.New("model file not found")
	ErrModelLoadFailed      = errors.New("failed to load model")
	ErrModelCorrupted       = errors.New("model file appears corrupted")
	ErrInsufficientMemory   = errors.New("insufficient memory to load model")
	ErrUnsupportedModelType = errors.New("unsupported model type")

	// Generation errors
	ErrGenerationFailed   = errors.New("text generation failed")
	ErrGenerationTimeout  = errors.New("text generation timed out")
	ErrInvalidPrompt      = errors.New("invalid prompt provided")
	ErrContextExceeded    = errors.New("prompt exceeds context size")
	ErrTokenizationFailed = errors.New("prompt tokenization failed")

	// Engine errors
	ErrEngineNotReady    = errors.New("llm engine not ready")
	ErrEngineShutdown    = errors.New("llm engine has been shut down")
	ErrConcurrentRequest = errors.New("concurrent request limit exceeded")

	// GPU errors
	ErrGPUNotAvailable = errors.New("gpu acceleration not available")
	ErrGPUMemoryFull   = errors.New("gpu memory insufficient")
	ErrGPUDriverError  = errors.New("gpu driver error")
)
