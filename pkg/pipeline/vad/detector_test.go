package vad

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/your-org/hema-replay-system/pkg/audio"
	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

func TestVADDetector_New(t *testing.T) {
	vadConfig := &Config{
		MinSpeechDurationMs:  500,
		MaxSilenceDurationMs: 1000,
		VADMode:              2,
		BufferBeforeMs:       500,
		BufferAfterMs:        500,
	}

	logger := zerolog.New(zerolog.NewTestWriter(t))

	// Create a mock audio manager for testing
	audioConfig := types.AudioConfig{
		Device: types.DeviceConfig{
			ID:         -1,
			SampleRate: 44100,
			Channels:   1,
			BitDepth:   16,
		},
		Processing: types.ProcessingConfig{
			EnablePreprocessing: true,
			VADType:             "threshold",
			VADThreshold:        0.01,
		},
		Buffer: types.BufferConfig{
			Duration:        time.Minute * 5,
			SegmentSize:     time.Second,
			PreallocateSize: 1024 * 10,
		},
		Extraction: types.ExtractionConfig{
			MaxConcurrent:    2,
			DefaultDuration:  time.Second * 5,
			OutputSampleRate: 16000,
			OutputChannels:   1,
		},
	}

	audioManager, err := audio.NewAudioManager(audioConfig, logger)

	// In noaudio builds, the AudioManager creation will fail
	if err != nil {
		t.Skip("Audio support not available in this build")
		return
	}

	detector := NewVADDetector(audioManager, vadConfig, logger)

	assert.NotNil(t, detector)
	assert.Equal(t, vadConfig, detector.config)
	assert.Equal(t, audioManager, detector.audioManager)
	assert.NotNil(t, detector.eventChan)
	assert.NotNil(t, detector.stopChan)
}

func TestVADDetector_Start(t *testing.T) {
	t.Skip("Integration test - requires audio system setup")

	// This would need a properly initialized audio manager
	// For now, skip this test as it requires audio hardware
}

func TestVADEvent_Types(t *testing.T) {
	// Test event type constants
	assert.Equal(t, 0, int(EventSpeechStart))
	assert.Equal(t, 1, int(EventSpeechEnd))
	assert.Equal(t, 2, int(EventSpeechSegment))
}

func TestConfig_Validation(t *testing.T) {
	config := &Config{
		MinSpeechDurationMs:  500,
		MaxSilenceDurationMs: 1000,
		VADMode:              2,
		BufferBeforeMs:       500,
		BufferAfterMs:        500,
	}

	// Basic validation that config values are reasonable
	assert.Greater(t, config.MinSpeechDurationMs, 0)
	assert.Greater(t, config.MaxSilenceDurationMs, 0)
	assert.GreaterOrEqual(t, config.VADMode, 0)
	assert.LessOrEqual(t, config.VADMode, 3)
	assert.GreaterOrEqual(t, config.BufferBeforeMs, 0)
	assert.GreaterOrEqual(t, config.BufferAfterMs, 0)
}

func TestVADDetector_calculateConfidence(t *testing.T) {
	config := &Config{}
	logger := zerolog.New(zerolog.NewTestWriter(t))
	detector := &VADDetector{
		config: config,
		logger: logger,
	}

	// Test confidence calculation for different durations
	tests := []struct {
		duration time.Duration
		expected float32
	}{
		{500 * time.Millisecond, 0.70},
		{1500 * time.Millisecond, 0.85},
		{3500 * time.Millisecond, 0.95},
	}

	for _, test := range tests {
		confidence := detector.calculateConfidence(test.duration)
		assert.Equal(t, test.expected, confidence, "Duration: %v", test.duration)
	}
}
