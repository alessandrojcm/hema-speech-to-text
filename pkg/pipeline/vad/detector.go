package vad

import (
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/your-org/hema-replay-system/pkg/audio"
	"github.com/your-org/hema-replay-system/pkg/audio/processing"
)

// VADDetector wraps the existing audio.Manager's VAD capabilities
type VADDetector struct {
	audioManager *audio.AudioManager
	processor    *processing.AudioProcessor

	// Detection state
	isActive      bool
	activityStart time.Time
	silenceStart  time.Time
	lastLogTime   time.Time

	// Configuration
	config *Config

	// Output
	eventChan chan VADEvent

	// Control
	stopChan chan struct{}
	mu       sync.RWMutex
	logger   zerolog.Logger
}

type Config struct {
	MinSpeechDurationMs  int `mapstructure:"min_speech_duration_ms"`  // Min duration to trigger
	MaxSilenceDurationMs int `mapstructure:"max_silence_duration_ms"` // Max silence before end
	VADMode              int `mapstructure:"vad_mode"`                // 0-3 (WebRTC VAD mode)
	BufferBeforeMs       int `mapstructure:"buffer_before_ms"`        // Audio before speech
	BufferAfterMs        int `mapstructure:"buffer_after_ms"`         // Audio after speech
}

type VADEvent struct {
	Type       EventType
	StartTime  time.Time
	EndTime    time.Time
	Duration   time.Duration
	Confidence float32

	// Audio segment boundaries for extraction
	BufferStart time.Time
	BufferEnd   time.Time
}

type EventType int

const (
	EventSpeechStart EventType = iota
	EventSpeechEnd
	EventSpeechSegment // Complete segment ready for processing
)

func NewVADDetector(audioManager *audio.AudioManager, config *Config, logger zerolog.Logger) *VADDetector {
	return &VADDetector{
		audioManager: audioManager,
		config:       config,
		eventChan:    make(chan VADEvent, 10),
		stopChan:     make(chan struct{}),
		logger:       logger.With().Str("component", "vad_detector").Logger(),
	}
}

// GetProcessor returns the internal audio processor for testing access
func (v *VADDetector) GetProcessor() *processing.AudioProcessor {
	return v.processor
}

// Events returns the event channel for receiving VAD events
func (v *VADDetector) Events() <-chan VADEvent {
	return v.eventChan
}

// Stop stops the VAD detector
func (v *VADDetector) Stop() {
	close(v.stopChan)
	close(v.eventChan)
}

// sendEvent sends a VAD event to the event channel
func (v *VADDetector) sendEvent(event VADEvent) {
	select {
	case v.eventChan <- event:
	default:
		v.logger.Warn().Msg("VAD event channel full, dropping event")
	}
}

// resetState resets the internal VAD detection state
func (v *VADDetector) resetState() {
	v.isActive = false
	v.activityStart = time.Time{}
	v.silenceStart = time.Time{}
}

// calculateConfidence calculates confidence score based on speech duration
func (v *VADDetector) calculateConfidence(duration time.Duration) float32 {
	// Simple confidence based on duration
	// Longer, clearer speech = higher confidence
	if duration > 3*time.Second {
		return 0.95
	} else if duration > 1*time.Second {
		return 0.85
	} else {
		return 0.70
	}
}
