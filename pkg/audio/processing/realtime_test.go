//go:build !noaudio

package processing

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

// TestAudioProcessor_RealTimeProcessingLoop simulates the exact capture engine scenario
func TestAudioProcessor_RealTimeProcessingLoop(t *testing.T) {
	// Create the exact same configuration as production
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
		VADMode:             1,
	}

	logger := zerolog.Nop() // No output to avoid console writer issues

	processor, err := NewAudioProcessor(config, 16000, 1, logger)
	require.NoError(t, err, "Failed to create AudioProcessor")
	defer processor.Close()

	t.Log("Starting real-time processing simulation...")

	// Simulate the exact capture engine loop
	frameSize := 1024 // Same as settings.yaml frames_per_buffer
	audioBuffer := make([]float32, frameSize)

	// Test different types of audio data that might trigger the freeze
	testCases := []struct {
		name      string
		generator func([]float32)
		desc      string
	}{
		{
			name: "silence",
			generator: func(buf []float32) {
				// Fill with near-silence (like when nobody is speaking)
				for i := range buf {
					buf[i] = 0.001 * float32(i%3-1) // Very quiet noise
				}
			},
			desc: "Low-level background noise (pre-speech)",
		},
		{
			name: "sudden_voice",
			generator: func(buf []float32) {
				// Simulate sudden voice activity (this might trigger the freeze!)
				for i := range buf {
					if i < len(buf)/4 {
						buf[i] = 0.001 * float32(i%3-1) // Start with silence
					} else {
						// Sudden voice-like signal
						buf[i] = 0.3 * float32((i*7)%11-5) / 5.0 // Voice-like pattern
					}
				}
			},
			desc: "Sudden voice activity after silence (freeze trigger?)",
		},
		{
			name: "continuous_voice",
			generator: func(buf []float32) {
				// Continuous voice-like signal
				for i := range buf {
					buf[i] = 0.25 * float32((i*13)%17-8) / 8.0 // Realistic voice levels
				}
			},
			desc: "Continuous voice activity",
		},
		{
			name: "voice_with_dynamics",
			generator: func(buf []float32) {
				// Variable amplitude voice (most realistic)
				for i := range buf {
					amp := 0.1 + 0.2*float32((i/100)%3)/3.0 // Varying amplitude
					buf[i] = amp * float32((i*11)%23-11) / 11.0
				}
			},
			desc: "Dynamic voice with amplitude changes",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Testing: %s", tc.desc)

			// Simulate capture engine loop for this audio type
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			loopCount := 0
			maxLoops := 1000 // Process ~1000 frames (about 23 seconds at 1024 samples/frame, 44.1kHz)

			for loopCount < maxLoops {
				select {
				case <-ctx.Done():
					t.Errorf("Test timed out after %d loops - potential freeze detected!", loopCount)
					return
				default:
					loopCount++

					// Generate the test audio
					tc.generator(audioBuffer)

					// This is the exact line that freezes in capture engine!
					startTime := time.Now()
					processedData, err := processor.Process(audioBuffer, startTime)
					processingTime := time.Since(startTime)

					// Check for freeze indicators
					if processingTime > 100*time.Millisecond {
						t.Errorf("Processing took too long: %v (potential freeze)", processingTime)
						return
					}

					require.NoError(t, err, "Processing should not fail on loop %d", loopCount)
					require.NotEmpty(t, processedData, "Should return processed data on loop %d", loopCount)

					// Test VAD detection (this might also freeze)
					voiceDetected := processor.DetectVoiceActivity(processedData)

					// Log periodically
					if loopCount%200 == 0 {
						t.Logf("Loop %d: processing_time=%v, voice_detected=%v", loopCount, processingTime, voiceDetected)
					}
				}
			}

			t.Logf("Successfully completed %d processing loops for %s", loopCount, tc.name)
		})
	}
}

// TestAudioProcessor_ThreadSafety tests if processor is thread-safe for concurrent use
func TestAudioProcessor_ThreadSafety(t *testing.T) {
	config := types.ProcessingConfig{
		EnablePreprocessing: true,
		Normalization:       true,
		HighpassFilter:      80.0,
		LowpassFilter:       8000.0,
		VADType:             "webrtc",
		VADMode:             1,
	}

	logger := zerolog.Nop()
	processor, err := NewAudioProcessor(config, 16000, 1, logger)
	require.NoError(t, err)
	defer processor.Close()

	// Test concurrent processing (simulate multiple audio threads)
	done := make(chan bool, 3)

	for i := 0; i < 3; i++ {
		go func(workerID int) {
			defer func() { done <- true }()

			samples := generateWhiteNoise(1024)
			for j := 0; j < 100; j++ {
				_, err := processor.Process(samples, time.Now())
				if err != nil {
					t.Errorf("Worker %d failed on iteration %d: %v", workerID, j, err)
					return
				}

				processor.DetectVoiceActivity(samples)
			}
		}(i)
	}

	// Wait for all workers
	timeout := time.After(10 * time.Second)
	for i := 0; i < 3; i++ {
		select {
		case <-done:
			// Worker completed
		case <-timeout:
			t.Fatal("Thread safety test timed out - possible deadlock")
		}
	}

	t.Log("Thread safety test completed successfully")
}
