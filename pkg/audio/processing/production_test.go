//go:build !noaudio

package processing

import (
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"github.com/your-org/hema-replay-system/internal/config"
)

// TestAudioProcessor_ProductionConfig tests with the exact production configuration
func TestAudioProcessor_ProductionConfig(t *testing.T) {
	// Load the real production config
	cfg, err := config.LoadConfig("../../../config/settings.yaml")
	if err != nil {
		t.Skipf("Skipping production config test - could not load settings.yaml: %v", err)
		return
	}

	// Create the exact same logger setup as production
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.InfoLevel)
	processorLogger := logger.With().Str("component", "audio_processor").Logger()

	t.Logf("Production config loaded: preprocessing=%v, vad_type=%s, vad_mode=%d",
		cfg.Audio.Processing.EnablePreprocessing,
		cfg.Audio.Processing.VADType,
		cfg.Audio.Processing.VADMode)

	// Create processor with exact production config and logger
	processor, err := NewAudioProcessor(
		cfg.Audio.Processing,
		int(cfg.Audio.Device.SampleRate),
		int(cfg.Audio.Device.Channels),
		processorLogger)
	require.NoError(t, err, "Failed to create AudioProcessor with production config")
	defer processor.Close()

	t.Log("AudioProcessor created with production configuration")

	// Simulate the exact capture engine loop behavior
	frameSize := int(cfg.Audio.Device.FramesPerBuffer * cfg.Audio.Device.Channels)
	audioBuffer := make([]float32, frameSize)

	t.Logf("Testing with production frame size: %d samples", frameSize)

	// Test multiple processing cycles like the capture engine does
	for i := 0; i < 100; i++ {
		// Generate audio data (start with silence, then add voice-like content)
		for j := range audioBuffer {
			if i < 50 {
				// First 50 iterations: mostly silence (like when nobody speaks)
				audioBuffer[j] = 0.001 * float32((j+i)%7-3) / 3.0
			} else {
				// Last 50 iterations: voice-like activity (this should trigger any freeze)
				audioBuffer[j] = 0.2 * float32((j*13+i*7)%23-11) / 11.0
			}
		}

		// This is the exact call that freezes in production
		startTime := time.Now()
		processedData, err := processor.Process(audioBuffer, startTime)
		processingTime := time.Since(startTime)

		require.NoError(t, err, "Processing iteration %d should not fail", i+1)
		require.Len(t, processedData, frameSize, "Should maintain frame size")

		// Test VAD detection
		voiceDetected := processor.DetectVoiceActivity(processedData)

		// Log transition point
		if i == 50 {
			t.Logf("Transition to voice-like audio at iteration %d", i+1)
		}

		// Check for freeze-like behavior
		if processingTime > 500*time.Millisecond {
			t.Errorf("Processing took too long on iteration %d: %v (possible freeze)", i+1, processingTime)
			return
		}

		// Log periodically
		if i%20 == 0 || i == 50 {
			t.Logf("Iteration %d: processing_time=%v, voice_detected=%v", i+1, processingTime, voiceDetected)
		}
	}

	t.Log("Production configuration test completed successfully - no freeze detected")
}

// TestAudioProcessor_MemoryPressure tests processor under memory pressure conditions
func TestAudioProcessor_MemoryPressure(t *testing.T) {
	// Load production config
	cfg, err := config.LoadConfig("../../../config/settings.yaml")
	if err != nil {
		t.Skipf("Skipping memory pressure test - could not load config: %v", err)
		return
	}

	logger := zerolog.Nop() // Silent to avoid logger issues

	processor, err := NewAudioProcessor(
		cfg.Audio.Processing,
		int(cfg.Audio.Device.SampleRate),
		int(cfg.Audio.Device.Channels),
		logger)
	require.NoError(t, err)
	defer processor.Close()

	// Create memory pressure by allocating lots of memory
	memoryHog := make([][]byte, 1000)
	for i := range memoryHog {
		memoryHog[i] = make([]byte, 1024*1024) // 1MB each
	}

	t.Log("Created memory pressure (~1GB), testing processor...")

	frameSize := int(cfg.Audio.Device.FramesPerBuffer * cfg.Audio.Device.Channels)
	audioBuffer := make([]float32, frameSize)

	// Test processing under memory pressure
	for i := 0; i < 50; i++ {
		// Generate voice-like audio
		for j := range audioBuffer {
			audioBuffer[j] = 0.3 * float32((j*11+i*5)%19-9) / 9.0
		}

		startTime := time.Now()
		processedData, err := processor.Process(audioBuffer, startTime)
		processingTime := time.Since(startTime)

		require.NoError(t, err, "Processing under memory pressure should not fail")
		require.NotEmpty(t, processedData, "Should return processed data")

		if processingTime > 1*time.Second {
			t.Errorf("Processing under memory pressure took too long: %v", processingTime)
			break
		}

		if i%10 == 0 {
			t.Logf("Memory pressure iteration %d: processing_time=%v", i+1, processingTime)
		}
	}

	// Clean up memory
	memoryHog = nil

	t.Log("Memory pressure test completed")
}
