package integration

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/your-org/hema-replay-system/pkg/audio"
	"github.com/your-org/hema-replay-system/pkg/audio/types"
	"github.com/your-org/hema-replay-system/pkg/speech/engine"
	speechTypes "github.com/your-org/hema-replay-system/pkg/speech/types"
)

// AudioSpeechBridge bridges the audio system with speech recognition
type AudioSpeechBridge struct {
	audioManager  *audio.AudioManager
	speechManager *engine.SpeechManager
	config        BridgeConfig
	logger        zerolog.Logger
}

// BridgeConfig contains configuration for the audio-speech bridge
type BridgeConfig struct {
	AutoTranscribe      bool          `mapstructure:"auto_transcribe"`
	TranscribeDuration  time.Duration `mapstructure:"transcribe_duration"`
	ConfidenceThreshold float64       `mapstructure:"confidence_threshold"`
	MaxConcurrent       int           `mapstructure:"max_concurrent"`
}

// NewAudioSpeechBridge creates a new audio-speech bridge
func NewAudioSpeechBridge(
	audioManager *audio.AudioManager,
	speechManager *engine.SpeechManager,
	config BridgeConfig,
	logger zerolog.Logger,
) *AudioSpeechBridge {
	return &AudioSpeechBridge{
		audioManager:  audioManager,
		speechManager: speechManager,
		config:        config,
		logger:        logger.With().Str("component", "audio_speech_bridge").Logger(),
	}
}

// TranscribeRecentAudio extracts and transcribes recent audio
func (asb *AudioSpeechBridge) TranscribeRecentAudio(ctx context.Context, duration time.Duration) (*speechTypes.TranscriptionResult, error) {
	// Extract audio from the audio manager
	extractionReq := types.ExtractionRequest{
		Duration: duration,
		EndTime:  time.Now(),
		Format:   "raw", // Use raw format for speech processing
	}

	audioSegment, err := asb.audioManager.ExtractAudio(ctx, extractionReq)
	if err != nil {
		return nil, fmt.Errorf("failed to extract audio: %w", err)
	}

	// Transcribe the audio segment
	result, err := asb.speechManager.TranscribeAudio(ctx, audioSegment)
	if err != nil {
		return nil, fmt.Errorf("failed to transcribe audio: %w", err)
	}

	asb.logger.Info().
		Dur("audio_duration", duration).
		Float64("confidence", result.Confidence).
		Str("text", result.Text).
		Msg("Audio transcribed successfully")

	return result, nil
}

// TranscribeAudioSegment transcribes a specific audio segment
func (asb *AudioSpeechBridge) TranscribeAudioSegment(ctx context.Context, segment *types.AudioSegment) (*speechTypes.TranscriptionResult, error) {
	return asb.speechManager.TranscribeAudio(ctx, segment)
}

// StartContinuousTranscription starts continuous transcription of audio
func (asb *AudioSpeechBridge) StartContinuousTranscription(ctx context.Context, callback func(*speechTypes.TranscriptionResult)) error {
	if !asb.config.AutoTranscribe {
		return fmt.Errorf("auto transcription not enabled")
	}

	ticker := time.NewTicker(asb.config.TranscribeDuration)
	defer ticker.Stop()

	asb.logger.Info().
		Dur("interval", asb.config.TranscribeDuration).
		Msg("Starting continuous transcription")

	for {
		select {
		case <-ctx.Done():
			asb.logger.Info().Msg("Stopping continuous transcription")
			return ctx.Err()

		case <-ticker.C:
			go func() {
				result, err := asb.TranscribeRecentAudio(ctx, asb.config.TranscribeDuration)
				if err != nil {
					asb.logger.Error().
						Err(err).
						Msg("Failed to transcribe audio in continuous mode")
					return
				}

				if result.Confidence >= asb.config.ConfidenceThreshold {
					callback(result)
				}
			}()
		}
	}
}

// GetCombinedStats returns combined statistics from both systems
func (asb *AudioSpeechBridge) GetCombinedStats() map[string]interface{} {
	audioStats := asb.audioManager.GetStats()
	speechStats := asb.speechManager.GetStats()

	return map[string]interface{}{
		"audio_system":  audioStats,
		"speech_system": speechStats,
		"bridge_config": asb.config,
	}
}
