//go:build !noaudio

package processing

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-audio/wav"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

// TestAudioProcessor_VADFreeze tests if WebRTC VAD causes freezing issues
func TestAudioProcessor_VADFreeze(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	tests := []struct {
		name        string
		config      types.ProcessingConfig
		sampleRate  int
		expectError bool
		timeout     time.Duration
	}{
		{
			name: "WebRTC VAD with 16000 Hz (supported)",
			config: types.ProcessingConfig{
				EnablePreprocessing: true,
				VADType:             "webrtc",
				VADMode:             3,
				ResamplerType:       "gosamplerate",
				ResamplerQuality:    0,
				WAVExporterType:     "goaudio",
			},
			sampleRate:  16000,
			expectError: false,
			timeout:     5 * time.Second,
		},
		{
			name: "WebRTC VAD with 8000 Hz (supported)",
			config: types.ProcessingConfig{
				EnablePreprocessing: true,
				VADType:             "webrtc",
				VADMode:             3,
				ResamplerType:       "gosamplerate",
				ResamplerQuality:    0,
				WAVExporterType:     "goaudio",
			},
			sampleRate:  8000,
			expectError: false,
			timeout:     5 * time.Second,
		},
		{
			name: "WebRTC VAD with 32000 Hz (supported)",
			config: types.ProcessingConfig{
				EnablePreprocessing: true,
				VADType:             "webrtc",
				VADMode:             3,
				ResamplerType:       "gosamplerate",
				ResamplerQuality:    0,
				WAVExporterType:     "goaudio",
			},
			sampleRate:  32000,
			expectError: false,
			timeout:     5 * time.Second,
		},
		{
			name: "WebRTC VAD with 44100 Hz (should fall back to threshold)",
			config: types.ProcessingConfig{
				EnablePreprocessing: true,
				VADType:             "webrtc",
				VADMode:             3,
				ResamplerType:       "gosamplerate",
				ResamplerQuality:    0,
				WAVExporterType:     "goaudio",
			},
			sampleRate:  44100,
			expectError: false,
			timeout:     5 * time.Second,
		},
		{
			name: "Threshold VAD with 44100 Hz (control test)",
			config: types.ProcessingConfig{
				EnablePreprocessing: true,
				VADType:             "threshold",
				VADThreshold:        0.01,
				ResamplerType:       "gosamplerate",
				ResamplerQuality:    0,
				WAVExporterType:     "goaudio",
			},
			sampleRate:  44100,
			expectError: false,
			timeout:     5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create processor
			processor, err := NewAudioProcessor(tt.config, tt.sampleRate, 1, logger)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err, "Failed to create AudioProcessor")
			defer processor.Close()

			// Generate test audio (white noise simulating speech)
			testSamples := generateWhiteNoise(tt.sampleRate / 10) // 100ms of audio

			// Run VAD test with timeout to catch freezes
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			// Channel to signal completion
			done := make(chan bool, 1)
			var vadResult bool

			go func() {
				// This should not freeze if WebRTC VAD is working correctly
				vadResult = processor.DetectVoiceActivity(testSamples)
				done <- true
			}()

			select {
			case <-done:
				t.Logf("VAD completed successfully for %s: voice_detected=%v", tt.name, vadResult)
			case <-ctx.Done():
				t.Errorf("VAD froze/timed out for %s after %v", tt.name, tt.timeout)
				return
			}

			// Test multiple calls to ensure no cumulative freezing
			for i := 0; i < 5; i++ {
				ctx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Second)
				done2 := make(chan bool, 1)

				go func() {
					processor.DetectVoiceActivity(testSamples)
					done2 <- true
				}()

				select {
				case <-done2:
					// Success
				case <-ctx2.Done():
					cancel2()
					t.Errorf("VAD froze on call %d for %s", i+1, tt.name)
					return
				}
				cancel2()
			}
		})
	}
}

// TestAudioProcessor_Resampling tests audio resampling functionality
func TestAudioProcessor_Resampling(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	config := types.ProcessingConfig{
		EnablePreprocessing: true,
		ResamplerType:       "gosamplerate",
		ResamplerQuality:    0,
		VADType:             "threshold", // Use threshold to avoid WebRTC VAD issues
		VADThreshold:        0.01,
		WAVExporterType:     "goaudio",
	}

	processor, err := NewAudioProcessor(config, 44100, 1, logger)
	require.NoError(t, err, "Failed to create AudioProcessor")
	defer processor.Close()

	tests := []struct {
		name       string
		inputRate  int
		outputRate int
		inputSize  int
	}{
		{
			name:       "44100 Hz to 16000 Hz",
			inputRate:  44100,
			outputRate: 16000,
			inputSize:  4410, // 100ms at 44100 Hz
		},
		{
			name:       "16000 Hz to 44100 Hz",
			inputRate:  16000,
			outputRate: 44100,
			inputSize:  1600, // 100ms at 16000 Hz
		},
		{
			name:       "44100 Hz to 8000 Hz",
			inputRate:  44100,
			outputRate: 8000,
			inputSize:  4410, // 100ms at 44100 Hz
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate test samples (using simple noise instead of sine wave to avoid dependencies)
			inputSamples := generateWhiteNoise(tt.inputSize)

			// Resample
			outputSamples, err := processor.Resample(inputSamples, tt.inputRate, tt.outputRate)
			require.NoError(t, err, "Resampling failed")

			// Verify output
			assert.NotEmpty(t, outputSamples, "Resampled output should not be empty")

			// Expected output size (approximate due to resampling)
			expectedSize := int(float64(len(inputSamples)) * float64(tt.outputRate) / float64(tt.inputRate))
			tolerance := expectedSize / 10 // 10% tolerance

			assert.InDelta(t, expectedSize, len(outputSamples), float64(tolerance),
				"Resampled size should be approximately correct")

			t.Logf("Resampled from %d samples to %d samples (expected ~%d)",
				len(inputSamples), len(outputSamples), expectedSize)
		})
	}
}

// TestAudioProcessor_EmptyAudioRejection tests rejection of empty/insufficient audio
func TestAudioProcessor_EmptyAudioRejection(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	config := types.ProcessingConfig{
		EnablePreprocessing: true,
		VADType:             "threshold", // Use threshold to avoid WebRTC VAD issues
		VADThreshold:        0.01,
		ResamplerType:       "gosamplerate",
		ResamplerQuality:    0,
		WAVExporterType:     "goaudio",
	}

	processor, err := NewAudioProcessor(config, 16000, 1, logger)
	require.NoError(t, err, "Failed to create AudioProcessor")
	defer processor.Close()

	tests := []struct {
		name        string
		samples     []float32
		expectVoice bool
		description string
	}{
		{
			name:        "Empty samples",
			samples:     []float32{},
			expectVoice: false,
			description: "Empty slice should not detect voice",
		},
		{
			name:        "Nil samples",
			samples:     nil,
			expectVoice: false,
			description: "Nil slice should not detect voice",
		},
		{
			name:        "Silent samples",
			samples:     make([]float32, 1600, 1600), // 100ms of silence at 16kHz
			expectVoice: false,
			description: "Silent audio should not detect voice",
		},
		{
			name:        "Very quiet samples",
			samples:     generateQuietNoise(1600, 0.001), // Very quiet noise
			expectVoice: false,
			description: "Very quiet audio should not detect voice",
		},
		{
			name:        "Loud samples",
			samples:     generateWhiteNoise(1600), // Normal level noise
			expectVoice: true,
			description: "Loud audio should detect voice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			voiceDetected := processor.DetectVoiceActivity(tt.samples)
			assert.Equal(t, tt.expectVoice, voiceDetected, tt.description)
			t.Logf("Test '%s': voice_detected=%v (expected=%v)", tt.name, voiceDetected, tt.expectVoice)
		})
	}
}

// TestAudioProcessor_RealAudioFile tests voice detection with real audio file
func TestAudioProcessor_RealAudioFile(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Find the bria.wav file
	briaPath := filepath.Join("..", "..", "..", "bria.wav")
	if _, err := os.Stat(briaPath); os.IsNotExist(err) {
		t.Skip("bria.wav file not found, skipping real audio test")
		return
	}

	config := types.ProcessingConfig{
		EnablePreprocessing: true,
		VADType:             "threshold", // Start with threshold to avoid freeze
		VADThreshold:        0.01,
		ResamplerType:       "gosamplerate",
		ResamplerQuality:    0,
		WAVExporterType:     "goaudio",
	}

	processor, err := NewAudioProcessor(config, 16000, 1, logger)
	require.NoError(t, err, "Failed to create AudioProcessor")
	defer processor.Close()

	// Load audio file
	samples, sampleRate, err := loadWAVFile(briaPath)
	require.NoError(t, err, "Failed to load bria.wav")
	require.NotEmpty(t, samples, "Audio file should not be empty")

	t.Logf("Loaded audio file: %d samples at %d Hz", len(samples), sampleRate)

	// Resample to 16kHz if needed
	if sampleRate != 16000 {
		samples, err = processor.Resample(samples, sampleRate, 16000)
		require.NoError(t, err, "Failed to resample audio")
		t.Logf("Resampled to 16kHz: %d samples", len(samples))
	}

	// Test voice activity detection on chunks
	chunkSize := 1600 // 100ms at 16kHz
	voiceChunks := 0
	totalChunks := 0

	for i := 0; i < len(samples); i += chunkSize {
		end := i + chunkSize
		if end > len(samples) {
			end = len(samples)
		}
		chunk := samples[i:end]

		if len(chunk) < chunkSize/2 { // Skip very small chunks
			continue
		}

		voiceDetected := processor.DetectVoiceActivity(chunk)
		if voiceDetected {
			voiceChunks++
		}
		totalChunks++
	}

	t.Logf("Voice detection results: %d/%d chunks detected voice (%.1f%%)",
		voiceChunks, totalChunks, float64(voiceChunks)*100.0/float64(totalChunks))

	// Real speech file should have some voice activity
	assert.Greater(t, voiceChunks, 0, "Real speech file should have some voice activity detected")
}

// Helper functions

// generateWhiteNoise generates white noise samples
func generateWhiteNoise(size int) []float32 {
	samples := make([]float32, size)
	for i := range samples {
		// Generate random values between -0.1 and 0.1 (moderate level)
		samples[i] = (float32(i%100) - 50) * 0.002
	}
	return samples
}

// generateQuietNoise generates very quiet noise samples
func generateQuietNoise(size int, amplitude float32) []float32 {
	samples := make([]float32, size)
	for i := range samples {
		samples[i] = (float32(i%100) - 50) * amplitude / 50
	}
	return samples
}

// loadWAVFile loads a WAV file and returns samples and sample rate
func loadWAVFile(filename string) ([]float32, int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	decoder := wav.NewDecoder(file)
	if !decoder.IsValidFile() {
		return nil, 0, err
	}

	format := decoder.Format()
	audioBuffer, err := decoder.FullPCMBuffer()
	if err != nil {
		return nil, 0, err
	}

	// Convert to float32
	samples := make([]float32, len(audioBuffer.Data))
	for i, sample := range audioBuffer.Data {
		samples[i] = float32(sample) / 32768.0 // Convert int to [-1, 1] float range
	}

	return samples, int(format.SampleRate), nil
}
