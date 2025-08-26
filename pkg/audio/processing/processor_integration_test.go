//go:build !noaudio

package processing

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

// TestAudioProcessor_WithPreprocessing tests the full AudioProcessor pipeline with preprocessing enabled
func TestAudioProcessor_WithPreprocessing(t *testing.T) {
	// Create processor configuration with preprocessing enabled (matching production config)
	config := types.ProcessingConfig{
		EnablePreprocessing: true,
		Normalization:       true,
		HighpassFilter:      80.0,   // From settings.yaml
		LowpassFilter:       8000.0, // From settings.yaml
		VADThreshold:        0.01,
		ResamplerType:       "gosamplerate",
		VADType:             "webrtc",
		WAVExporterType:     "goaudio",
		ResamplerQuality:    0,
		VADMode:             1, // Less aggressive
	}

	logger := zerolog.New(&zerolog.ConsoleWriter{Out: nil}) // Silent logger

	// Test AudioProcessor creation (this might be where freezing occurs)
	processor, err := NewAudioProcessor(config, 16000, 1, logger)
	require.NoError(t, err, "Failed to create AudioProcessor with preprocessing enabled")
	defer processor.Close()

	t.Log("AudioProcessor created successfully with preprocessing enabled")

	// Test 1: Process synthetic audio
	t.Run("Process synthetic audio", func(t *testing.T) {
		// Generate synthetic audio similar to what PortAudio would provide
		chunkSize := 1024 // frames_per_buffer from settings.yaml
		audioChunk := generateWhiteNoise(chunkSize)

		// This is where freezing might occur during preprocessing
		processedAudio, err := processor.Process(audioChunk, time.Now())
		require.NoError(t, err, "Audio processing should not fail")
		require.Len(t, processedAudio, chunkSize, "Processed audio should maintain same length")
		t.Logf("Successfully processed %d samples", len(processedAudio))

		// Test VAD detection
		voiceDetected := processor.DetectVoiceActivity(processedAudio)
		t.Logf("Voice detected: %v", voiceDetected)
	})

	// Test 2: Process multiple chunks (simulating continuous processing)
	t.Run("Process multiple chunks", func(t *testing.T) {
		chunkSize := 1024
		for i := 0; i < 5; i++ {
			chunk := generateWhiteNoise(chunkSize)
			processed, err := processor.Process(chunk, time.Now())
			require.NoError(t, err, "Processing chunk %d should not fail", i+1)
			require.Len(t, processed, chunkSize, "Chunk %d should maintain length", i+1)

			// Test VAD on each chunk
			voice := processor.DetectVoiceActivity(processed)
			t.Logf("Chunk %d: voice_detected=%v", i+1, voice)
		}
	})

	// Test 3: Test resampling functionality
	t.Run("Test resampling", func(t *testing.T) {
		originalSamples := generateWhiteNoise(1000)

		// Test resampling from 44100 to 16000 (common scenario)
		resampled, err := processor.Resample(originalSamples, 44100, 16000)
		require.NoError(t, err, "Resampling should not fail")

		expectedLength := int(float64(len(originalSamples)) * 16000.0 / 44100.0)
		tolerance := expectedLength / 10 // 10% tolerance
		require.InDelta(t, expectedLength, len(resampled), float64(tolerance),
			"Resampled length should be approximately correct")

		t.Logf("Resampled %d samples to %d samples", len(originalSamples), len(resampled))
	})

	// Test 4: Test empty audio handling
	t.Run("Test empty audio", func(t *testing.T) {
		emptyAudio := []float32{}
		processed, err := processor.Process(emptyAudio, time.Now())
		require.NoError(t, err, "Empty audio should not cause error")
		require.Empty(t, processed, "Empty audio should remain empty")
	})
}

// TestAudioProcessor_WithRealAudio tests with the bria.wav file
func TestAudioProcessor_WithRealAudio(t *testing.T) {
	// Load the bria.wav file
	audioSamples, sampleRate, err := loadWAVFileIntegration("../../../bria.wav")
	if err != nil {
		t.Skipf("Skipping real audio test - could not load bria.wav: %v", err)
		return
	}

	t.Logf("Loaded bria.wav: %d samples at %d Hz", len(audioSamples), sampleRate)

	// Create processor with full preprocessing pipeline
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

	logger := zerolog.New(&zerolog.ConsoleWriter{Out: nil})

	processor, err := NewAudioProcessor(config, 16000, 1, logger)
	require.NoError(t, err, "Failed to create AudioProcessor")
	defer processor.Close()

	// Test processing real speech audio in chunks
	chunkSize := 1024
	totalChunks := len(audioSamples) / chunkSize
	voiceDetections := 0

	for i := 0; i < totalChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(audioSamples) {
			end = len(audioSamples)
		}

		chunk := audioSamples[start:end]

		// Pad the last chunk if necessary
		if len(chunk) < chunkSize {
			paddedChunk := make([]float32, chunkSize)
			copy(paddedChunk, chunk)
			chunk = paddedChunk
		}

		// Process the chunk (this is where freezing might occur)
		processed, err := processor.Process(chunk, time.Now())
		require.NoError(t, err, "Processing chunk %d should not fail", i+1)

		// Test VAD detection
		voiceDetected := processor.DetectVoiceActivity(processed)
		if voiceDetected {
			voiceDetections++
		}

		// Log every 100th chunk to avoid spam
		if i%100 == 0 || i == totalChunks-1 {
			t.Logf("Processed chunk %d/%d: voice_detected=%v", i+1, totalChunks, voiceDetected)
		}
	}

	t.Logf("Real audio test completed: %d/%d chunks had voice activity", voiceDetections, totalChunks)

	// We expect some voice activity in a speech file
	require.Greater(t, voiceDetections, 0, "Should detect some voice activity in speech file")
}

// TestAudioProcessor_ComponentIsolation tests each component individually to identify freezing culprit
func TestAudioProcessor_ComponentIsolation(t *testing.T) {
	logger := zerolog.New(&zerolog.ConsoleWriter{Out: nil})
	testSamples := generateWhiteNoise(1024)

	// Test 1: Minimal processor (no preprocessing)
	t.Run("Minimal processor", func(t *testing.T) {
		config := types.ProcessingConfig{
			EnablePreprocessing: false,
			VADType:             "threshold", // Use simple VAD
			ResamplerType:       "custom",    // Use simple resampler
		}

		processor, err := NewAudioProcessor(config, 16000, 1, logger)
		require.NoError(t, err, "Minimal processor should create successfully")
		defer processor.Close()

		processed, err := processor.Process(testSamples, time.Now())
		require.NoError(t, err, "Minimal processing should not fail")
		require.NotEmpty(t, processed, "Should return processed samples")
		t.Log("Minimal processor test passed")
	})

	// Test 2: With WebRTC VAD only
	t.Run("With WebRTC VAD", func(t *testing.T) {
		config := types.ProcessingConfig{
			EnablePreprocessing: false,
			VADType:             "webrtc",
			VADMode:             1,
			ResamplerType:       "custom",
		}

		processor, err := NewAudioProcessor(config, 16000, 1, logger)
		require.NoError(t, err, "WebRTC VAD processor should create successfully")
		defer processor.Close()

		processed, err := processor.Process(testSamples, time.Now())
		require.NoError(t, err, "WebRTC VAD processing should not fail")

		voiceDetected := processor.DetectVoiceActivity(processed)
		t.Logf("WebRTC VAD test passed: voice_detected=%v", voiceDetected)
	})

	// Test 3: With gosamplerate resampler only
	t.Run("With gosamplerate resampler", func(t *testing.T) {
		config := types.ProcessingConfig{
			EnablePreprocessing: false,
			VADType:             "threshold",
			ResamplerType:       "gosamplerate",
			ResamplerQuality:    0,
		}

		processor, err := NewAudioProcessor(config, 16000, 1, logger)
		require.NoError(t, err, "Gosamplerate processor should create successfully")
		defer processor.Close()

		processedResult, err := processor.Process(testSamples, time.Now())
		require.NoError(t, err, "Gosamplerate processing should not fail")
		require.NotEmpty(t, processedResult, "Should process samples")

		// Test resampling
		resampledSamples, err := processor.Resample(testSamples, 44100, 16000)
		require.NoError(t, err, "Resampling should not fail")
		require.NotEmpty(t, resampledSamples, "Should return resampled data")
		t.Logf("Gosamplerate resampler test passed: %d -> %d samples", len(testSamples), len(resampledSamples))
	})

	// Test 4: With preprocessing filters only
	t.Run("With preprocessing filters", func(t *testing.T) {
		config := types.ProcessingConfig{
			EnablePreprocessing: true,
			Normalization:       true,
			HighpassFilter:      80.0,
			LowpassFilter:       8000.0,
			VADType:             "threshold",
			ResamplerType:       "custom",
		}

		processor, err := NewAudioProcessor(config, 16000, 1, logger)
		require.NoError(t, err, "Preprocessing processor should create successfully")
		defer processor.Close()

		processedData, err := processor.Process(testSamples, time.Now())
		require.NoError(t, err, "Preprocessing should not fail")
		require.Len(t, processedData, len(testSamples), "Should maintain sample count")
		t.Log("Preprocessing filters test passed")
	})

	// Test 5: Full production configuration (this might freeze)
	t.Run("Full production config", func(t *testing.T) {
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

		processor, err := NewAudioProcessor(config, 16000, 1, logger)
		require.NoError(t, err, "Full production processor should create successfully")
		defer processor.Close()

		// This is the critical test - if it freezes here, we've found the culprit
		processedSamples, err := processor.Process(testSamples, time.Now())
		require.NoError(t, err, "Full production processing should not fail")

		voiceDetected := processor.DetectVoiceActivity(processedSamples)
		t.Logf("Full production config test passed: voice_detected=%v", voiceDetected)
	})
}

// Helper function to load WAV file for testing
func loadWAVFileIntegration(filepath string) ([]float32, int, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	decoder := wav.NewDecoder(file)
	if !decoder.IsValidFile() {
		return nil, 0, fmt.Errorf("invalid WAV file")
	}

	// Read all audio data
	var samples []float32
	buf := &audio.IntBuffer{
		Format: decoder.Format(),
		Data:   make([]int, 1024),
	}

	for {
		n, err := decoder.PCMBuffer(buf)
		if n == 0 {
			break
		}
		if err != nil {
			return nil, 0, err
		}

		// Convert int samples to float32
		for i := 0; i < n; i++ {
			sample := float32(buf.Data[i]) / 32768.0 // Convert from int16 range to [-1, 1]
			samples = append(samples, sample)
		}
	}

	return samples, decoder.Format().SampleRate, nil
}
