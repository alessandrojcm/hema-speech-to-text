//go:build !noaudio

package processing

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

// TestWebRTCVAD_DirectFrameProcessing tests WebRTC VAD with exact frame sizes
// This simulates the exact conditions in the production pipeline
func TestWebRTCVAD_DirectFrameProcessing(t *testing.T) {
	// Test with the exact configuration from settings.yaml
	vadDetector, err := NewWebRTCVAD(16000, 1) // 16kHz, mode 1 (less aggressive)
	require.NoError(t, err, "Failed to create WebRTC VAD")
	defer vadDetector.Close()

	// Test frame sizes that WebRTC VAD expects for 16kHz
	// WebRTC VAD expects exactly: 160, 320, or 480 samples for 16kHz (10ms, 20ms, 30ms)
	testFrames := []struct {
		name     string
		samples  int
		duration string
	}{
		{"10ms frame", 160, "10ms"},
		{"20ms frame", 320, "20ms"},
		{"30ms frame", 480, "30ms"},
	}

	for _, tf := range testFrames {
		t.Run(tf.name, func(t *testing.T) {
			// Generate white noise for the exact frame size
			samples := generateWhiteNoise(tf.samples)

			// Test ProcessFrame (which should not freeze)
			voiceDetected, err := vadDetector.ProcessFrame(samples)
			require.NoError(t, err, "ProcessFrame should not error for valid frame size")

			t.Logf("Frame %s (%d samples): voice_detected=%v", tf.duration, tf.samples, voiceDetected)

			// Test DetectVoice (the main method used in production)
			voiceDetected2 := vadDetector.DetectVoice(samples)
			t.Logf("DetectVoice %s (%d samples): voice_detected=%v", tf.duration, tf.samples, voiceDetected2)
		})
	}
}

// TestWebRTCVAD_ProductionScenario tests the exact scenario that might be causing freezes
func TestWebRTCVAD_ProductionScenario(t *testing.T) {
	// Create the same configuration as the production settings.yaml
	config := types.ProcessingConfig{
		EnablePreprocessing: true,
		Normalization:       true,
		HighpassFilter:      80.0,
		LowpassFilter:       8000.0,
		VADThreshold:        0.01,
		ResamplerType:       "gosamplerate",
		VADType:             "webrtc",
		WAVExporterType:     "goaudio",
		ResamplerQuality:    0,
		VADMode:             1, // Less aggressive (from settings.yaml)
	}

	logger := zerolog.New(&zerolog.ConsoleWriter{Out: nil}) // Silent logger

	// Create processor with the exact same setup as production
	processor, err := NewAudioProcessor(config, 16000, 1, logger)
	require.NoError(t, err, "Failed to create AudioProcessor with production config")
	defer processor.Close()

	// Simulate the audio processing pipeline
	// 1. Generate audio chunk (like what comes from PortAudio)
	chunkSize := 1024 // Same as frames_per_buffer in settings.yaml
	audioChunk := generateWhiteNoise(chunkSize)

	// 2. Process the audio (like AudioManager.Process does)
	processedAudio, err := processor.Process(audioChunk, time.Now())
	require.NoError(t, err, "Audio processing should not fail")
	require.NotEmpty(t, processedAudio, "Processed audio should not be empty")

	// 3. Detect voice activity (this is where freezing might occur)
	voiceDetected := processor.DetectVoiceActivity(processedAudio)
	t.Logf("Production scenario test: voice_detected=%v for %d samples", voiceDetected, len(processedAudio))

	// 4. Test multiple successive calls (like the continuous loop in production)
	for i := 0; i < 10; i++ {
		chunk := generateWhiteNoise(chunkSize)
		processed, err := processor.Process(chunk, time.Now())
		require.NoError(t, err, "Successive processing should not fail")

		voice := processor.DetectVoiceActivity(processed)
		t.Logf("Call %d: voice_detected=%v", i+1, voice)
	}

	t.Log("Production scenario test completed successfully - no freezing detected")
}

// TestWebRTCVAD_EdgeCases tests edge cases that might cause freezing
func TestWebRTCVAD_EdgeCases(t *testing.T) {
	vadDetector, err := NewWebRTCVAD(16000, 1)
	require.NoError(t, err, "Failed to create WebRTC VAD")
	defer vadDetector.Close()

	testCases := []struct {
		name    string
		samples []float32
		desc    string
	}{
		{
			name:    "Empty samples",
			samples: []float32{},
			desc:    "Empty slice should not freeze",
		},
		{
			name:    "Single sample",
			samples: []float32{0.5},
			desc:    "Very small sample should not freeze",
		},
		{
			name:    "Exact 160 samples",
			samples: generateWhiteNoise(160),
			desc:    "Perfect WebRTC frame size should not freeze",
		},
		{
			name:    "Odd frame size",
			samples: generateWhiteNoise(159), // 159 is not a valid WebRTC frame size
			desc:    "Invalid frame size should not freeze (should fall back)",
		},
		{
			name:    "Large chunk",
			samples: generateWhiteNoise(1600), // 100ms of audio
			desc:    "Large chunk should be processed in smaller frames",
		},
		{
			name:    "All zeros",
			samples: make([]float32, 320), // 320 zeros
			desc:    "Silent audio should not freeze",
		},
		{
			name:    "All max values",
			samples: generateMaxNoise(480), // All values at max
			desc:    "Saturated audio should not freeze",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// This should complete quickly and not freeze
			voiceDetected := vadDetector.DetectVoice(tc.samples)
			t.Logf("%s: voice_detected=%v for %d samples", tc.desc, voiceDetected, len(tc.samples))
		})
	}
}

// Helper function to generate samples at maximum level
func generateMaxNoise(size int) []float32 {
	samples := make([]float32, size)
	for i := range samples {
		if i%2 == 0 {
			samples[i] = 1.0
		} else {
			samples[i] = -1.0
		}
	}
	return samples
}
