//go:build noaudio

package whisper

import (
	"fmt"

	"github.com/rs/zerolog"
	"github.com/your-org/hema-replay-system/pkg/speech/types"
)

// Wrapper wraps whisper.cpp functionality (stub implementation)
type Wrapper struct {
	logger zerolog.Logger
}

// NewWrapper creates a new whisper wrapper (stub)
func NewWrapper(config types.WhisperConfig, logger zerolog.Logger) (*Wrapper, error) {
	return &Wrapper{
		logger: logger.With().Str("component", "whisper_wrapper_stub").Logger(),
	}, nil
}

// Transcribe transcribes audio data (stub)
func (w *Wrapper) Transcribe(audioData []float32, params types.WhisperParams) (*types.TranscriptionResult, error) {
	w.logger.Debug().Msg("Stub: Transcribe called")
	return nil, fmt.Errorf("whisper not available in noaudio build")
}

// Close closes the wrapper (stub)
func (w *Wrapper) Close() {
	w.logger.Debug().Msg("Stub: Close called")
}
