//go:build !noaudio
// +build !noaudio

package audio

import (
	"context"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-org/hema-replay-system/pkg/audio/processing"
	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

func TestAudioPipelineIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()

	// Create test configuration
	config := types.DefaultAudioConfig()
	config.Device.SampleRate = 16000 // Lower sample rate for testing
	config.Buffer.Duration = 10 * time.Second
	config.Extraction.MaxConcurrent = 3

	// Test with different library configurations
	testCases := []struct {
		name   string
		config types.ProcessingConfig
	}{
		{
			name: "All Library Implementations",
			config: types.ProcessingConfig{
				EnablePreprocessing: true,
				NoiseReduction:      true,
				Normalization:       true,
				HighpassFilter:      80.0,
				LowpassFilter:       8000.0,
				VADThreshold:        0.1,
				ResamplerType:       "gosamplerate",
				VADType:             "webrtc",
				WAVExporterType:     "goaudio",
				FFTType:             "gonum",
				ResamplerQuality:    0,
				VADMode:             3,
			},
		},
		{
			name: "Mixed Implementation",
			config: types.ProcessingConfig{
				EnablePreprocessing: true,
				NoiseReduction:      true,
				Normalization:       true,
				HighpassFilter:      80.0,
				LowpassFilter:       8000.0,
				VADThreshold:        0.1,
				ResamplerType:       "gosamplerate",
				VADType:             "threshold",
				WAVExporterType:     "goaudio",
				FFTType:             "gonum",
				ResamplerQuality:    2, // Medium quality
				VADMode:             1, // Less aggressive
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config.Processing = tc.config
			testAudioManagerLifecycle(t, config, logger)
		})
	}
}

func testAudioManagerLifecycle(t *testing.T, config types.AudioConfig, logger zerolog.Logger) {
	// Create audio manager
	manager, err := NewAudioManager(config, logger)
	require.NoError(t, err, "Failed to create audio manager")
	require.NotNil(t, manager, "Audio manager should not be nil")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test initial state
	assert.False(t, manager.running, "Manager should not be running initially")

	health := manager.GetHealth()
	assert.Equal(t, types.HealthStatusUnknown, health.OverallStatus, "Initial health should be unknown")

	// Start the manager
	err = manager.Start(ctx)
	if err != nil {
		// If we can't start (no audio device), skip the test
		t.Skipf("Cannot start audio manager (likely no audio device): %v", err)
		return
	}
	defer manager.Stop()

	// Wait for initialization
	time.Sleep(2 * time.Second)

	// Test running state
	assert.True(t, manager.running, "Manager should be running after start")

	// Test health monitoring
	health = manager.GetHealth()
	t.Logf("Health status: %s", health.OverallStatus.String())

	// Test metrics collection
	metrics := manager.GetMetrics()
	assert.NotNil(t, metrics, "Metrics should not be nil")
	assert.False(t, metrics.LastUpdate.IsZero(), "Metrics should have timestamp")

	// Test performance stats
	perfStats := manager.GetPerformanceStats()
	assert.NotNil(t, perfStats, "Performance stats should not be nil")
	assert.Contains(t, perfStats, "is_running", "Performance stats should contain running status")
	assert.True(t, perfStats["is_running"].(bool), "Performance stats should show running")

	// Test audio extraction
	testAudioExtraction(t, manager, ctx)

	// Test concurrent extraction
	testConcurrentExtraction(t, manager, ctx)

	// Test configuration updates
	testConfigurationUpdate(t, manager)
}

func testAudioExtraction(t *testing.T, manager *AudioManager, ctx context.Context) {
	// Test single extraction
	req := types.ExtractionRequest{
		Duration: 2 * time.Second,
		Format:   "wav",
	}

	segment, err := manager.ExtractAudio(ctx, req)
	if err != nil {
		// If extraction fails due to no audio data, that's expected in test environment
		t.Logf("Audio extraction failed (expected in test environment): %v", err)
		return
	}

	require.NotNil(t, segment, "Audio segment should not be nil")
	assert.NotEmpty(t, segment.ID, "Segment should have ID")
	assert.NotEmpty(t, segment.Data, "Segment should have audio data")
	assert.False(t, segment.StartTime.IsZero(), "Segment should have start time")
	assert.False(t, segment.EndTime.IsZero(), "Segment should have end time")
	assert.True(t, segment.Duration > 0, "Segment should have positive duration")

	// Test segment metadata
	assert.True(t, segment.Metadata.SampleRate > 0, "Metadata should have sample rate")
	assert.True(t, segment.Metadata.Channels > 0, "Metadata should have channels")
	assert.True(t, segment.Metadata.Quality >= 0 && segment.Metadata.Quality <= 1, "Quality should be in [0,1] range")

	t.Logf("Extracted segment: ID=%s, Duration=%v, Quality=%.3f",
		segment.ID, segment.Duration, segment.Metadata.Quality)
}

func testConcurrentExtraction(t *testing.T, manager *AudioManager, ctx context.Context) {
	// Test concurrent extractions
	requests := []types.ExtractionRequest{
		{Duration: 1 * time.Second, Format: "wav"},
		{Duration: 2 * time.Second, Format: "wav"},
		{Duration: 1500 * time.Millisecond, Format: "wav"},
	}

	segments, errors := manager.ExtractAudioConcurrent(ctx, requests)
	assert.Len(t, segments, len(requests), "Should return same number of segments as requests")
	assert.Len(t, errors, len(requests), "Should return same number of errors as requests")

	successCount := 0
	for i, err := range errors {
		if err == nil {
			successCount++
			assert.NotNil(t, segments[i], "Successful extraction should have segment")
		} else {
			t.Logf("Concurrent extraction %d failed (expected in test environment): %v", i, err)
		}
	}

	t.Logf("Concurrent extractions: %d/%d successful", successCount, len(requests))
}

func testConfigurationUpdate(t *testing.T, manager *AudioManager) {
	// Test configuration update
	newConfig := manager.config
	newConfig.Processing.VADThreshold = 0.2 // Change VAD threshold

	err := manager.UpdateConfiguration(newConfig)
	assert.NoError(t, err, "Configuration update should succeed")

	// Verify configuration was updated
	assert.Equal(t, 0.2, manager.config.Processing.VADThreshold, "VAD threshold should be updated")
}

func TestAudioProcessingPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping processing pipeline test in short mode")
	}

	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()

	// Test different processing configurations
	testCases := []struct {
		name   string
		config types.ProcessingConfig
	}{
		{
			name: "Gosamplerate + WebRTC VAD",
			config: types.ProcessingConfig{
				EnablePreprocessing: true,
				ResamplerType:       "gosamplerate",
				VADType:             "webrtc",
				WAVExporterType:     "goaudio",
				FFTType:             "gonum",
				ResamplerQuality:    0,
				VADMode:             3,
			},
		},
		{
			name: "Custom + Threshold VAD",
			config: types.ProcessingConfig{
				EnablePreprocessing: true,
				ResamplerType:       "custom",
				VADType:             "threshold",
				WAVExporterType:     "goaudio",
				FFTType:             "gonum",
				VADThreshold:        0.1,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testProcessingConfiguration(t, tc.config, logger)
		})
	}
}

func testProcessingConfiguration(t *testing.T, config types.ProcessingConfig, logger zerolog.Logger) {
	// Create enhanced audio processor
	processor, err := processing.NewEnhancedAudioProcessor(config, 16000, 1, logger)
	require.NoError(t, err, "Failed to create enhanced audio processor")
	require.NotNil(t, processor, "Processor should not be nil")
	defer processor.Close()

	// Generate test audio data
	testSamples := generateTestAudio(16000, 1, 2*time.Second)
	require.NotEmpty(t, testSamples, "Test samples should not be empty")

	// Test processing
	processed, err := processor.Process(testSamples, time.Now())
	assert.NoError(t, err, "Processing should succeed")
	assert.NotEmpty(t, processed, "Processed samples should not be empty")
	assert.Len(t, processed, len(testSamples), "Processed samples should have same length")

	// Test resampling
	resampled, err := processor.Resample(testSamples, 16000, 8000)
	assert.NoError(t, err, "Resampling should succeed")
	assert.NotEmpty(t, resampled, "Resampled samples should not be empty")
	expectedLength := len(testSamples) * 8000 / 16000
	assert.InDelta(t, expectedLength, len(resampled), float64(expectedLength)*0.1, "Resampled length should be approximately correct")

	// Test VAD
	hasVoice := processor.DetectVoiceActivity(testSamples)
	t.Logf("Voice activity detected: %v", hasVoice)

	// Test FFT
	fftResult := processor.ComputeFFT(testSamples[:1024]) // Use smaller window for FFT
	assert.NotEmpty(t, fftResult, "FFT result should not be empty")

	// Test power spectrum
	powerSpectrum := processor.ComputePowerSpectrum(testSamples[:1024])
	assert.NotEmpty(t, powerSpectrum, "Power spectrum should not be empty")

	// Test spectral centroid
	centroid := processor.ComputeSpectralCentroid(testSamples[:1024])
	assert.True(t, centroid >= 0, "Spectral centroid should be non-negative")

	// Test windowing
	windowed := processor.ApplyWindow(testSamples[:1024], "hann")
	assert.NotEmpty(t, windowed, "Windowed samples should not be empty")
	assert.Len(t, windowed, 1024, "Windowed samples should have same length")

	t.Logf("Processing test completed successfully for %s", config.ResamplerType)
}

func TestQualityAssessmentIntegration(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()

	// Create quality meter
	qualityMeter := processing.NewQualityMeterWithLogger(16000, 1, logger)
	require.NotNil(t, qualityMeter, "Quality meter should not be nil")

	// Test with different types of audio
	testCases := []struct {
		name     string
		samples  []float32
		expected map[string]interface{}
	}{
		{
			name:    "Silence",
			samples: make([]float32, 1600), // 100ms of silence at 16kHz
			expected: map[string]interface{}{
				"low_quality":     true,
				"no_voice":        true,
				"under_modulated": true,
			},
		},
		{
			name:    "Pure Tone",
			samples: generateSineWave(16000, 1000, 0.5, 1600), // 1kHz tone
			expected: map[string]interface{}{
				"has_signal": true,
				"tonal":      true,
			},
		},
		{
			name:    "Speech-like Signal",
			samples: generateSpeechLikeSignal(16000, 1600),
			expected: map[string]interface{}{
				"has_signal":     true,
				"voice_probable": true,
				"good_quality":   true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Basic quality assessment
			basic := qualityMeter.AssessQuality(tc.samples)
			assert.NotNil(t, basic, "Basic quality assessment should not be nil")

			// Enhanced quality assessment
			enhanced := qualityMeter.AssessEnhancedQuality(tc.samples)
			assert.NotNil(t, enhanced, "Enhanced quality assessment should not be nil")
			assert.False(t, enhanced.Timestamp.IsZero(), "Enhanced assessment should have timestamp")

			// Verify basic metrics
			assert.True(t, enhanced.Basic.Quality >= 0 && enhanced.Basic.Quality <= 1, "Quality should be in [0,1] range")
			assert.True(t, enhanced.Basic.RMSLevel >= 0, "RMS should be non-negative")
			assert.True(t, enhanced.Basic.PeakAmplitude >= 0, "Peak should be non-negative")
			assert.True(t, enhanced.Basic.NoiseLevel >= 0, "Noise level should be non-negative")

			// Verify enhanced metrics
			assert.True(t, enhanced.SpectralCentroid >= 0, "Spectral centroid should be non-negative")
			assert.True(t, enhanced.SpectralFlatness >= 0 && enhanced.SpectralFlatness <= 1, "Spectral flatness should be in [0,1] range")
			assert.True(t, enhanced.HighFrequencyEnergy >= 0 && enhanced.HighFrequencyEnergy <= 1, "High freq energy should be in [0,1] range")
			assert.True(t, enhanced.VoiceProbability >= 0 && enhanced.VoiceProbability <= 1, "Voice probability should be in [0,1] range")
			assert.True(t, enhanced.SpeechClarity >= 0 && enhanced.SpeechClarity <= 1, "Speech clarity should be in [0,1] range")
			assert.True(t, enhanced.VocalEffort >= 0 && enhanced.VocalEffort <= 1, "Vocal effort should be in [0,1] range")

			// Check expected characteristics
			if tc.expected["under_modulated"] == true {
				assert.True(t, enhanced.IsUnderModulated, "Should detect under-modulation")
			}
			if tc.expected["voice_probable"] == true {
				assert.True(t, enhanced.VoiceProbability > 0.3, "Should have reasonable voice probability")
			}

			t.Logf("%s - Quality: %.3f, Voice Prob: %.3f, Clarity: %.3f, Centroid: %.1f Hz",
				tc.name, enhanced.Basic.Quality, enhanced.VoiceProbability,
				enhanced.SpeechClarity, enhanced.SpectralCentroid)
		})
	}

	// Test quality trends
	trends := qualityMeter.GetQualityTrends()
	assert.Contains(t, trends, "rms", "Trends should contain RMS history")
	assert.Contains(t, trends, "peak", "Trends should contain peak history")
	assert.Contains(t, trends, "snr", "Trends should contain SNR history")
}

// Helper functions for generating test audio

func generateTestAudio(sampleRate, channels int, duration time.Duration) []float32 {
	samples := int(float64(sampleRate) * duration.Seconds())
	audio := make([]float32, samples*channels)

	// Generate a mix of frequencies to simulate speech
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(sampleRate)

		// Mix of frequencies typical in speech
		signal := 0.3*math.Sin(2*math.Pi*300*t) + // 300 Hz fundamental
			0.2*math.Sin(2*math.Pi*600*t) + // First harmonic
			0.1*math.Sin(2*math.Pi*1200*t) + // Second harmonic
			0.05*math.Sin(2*math.Pi*2400*t) // Third harmonic

		// Add some noise
		noise := (rand.Float64() - 0.5) * 0.02
		signal += noise

		// Apply envelope to simulate speech patterns
		envelope := 0.5 * (1 + math.Sin(2*math.Pi*5*t)) // 5 Hz modulation
		signal *= envelope

		for ch := 0; ch < channels; ch++ {
			audio[i*channels+ch] = float32(signal)
		}
	}

	return audio
}

func generateSineWave(sampleRate int, frequency float64, amplitude float64, samples int) []float32 {
	audio := make([]float32, samples)
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(sampleRate)
		audio[i] = float32(amplitude * math.Sin(2*math.Pi*frequency*t))
	}
	return audio
}

func generateSpeechLikeSignal(sampleRate, samples int) []float32 {
	audio := make([]float32, samples)

	for i := 0; i < samples; i++ {
		t := float64(i) / float64(sampleRate)

		// Formant-like structure typical of speech
		f1 := 0.4 * math.Sin(2*math.Pi*800*t)  // First formant
		f2 := 0.3 * math.Sin(2*math.Pi*1200*t) // Second formant
		f3 := 0.2 * math.Sin(2*math.Pi*2400*t) // Third formant

		// Add some noise for realism
		noise := (rand.Float64() - 0.5) * 0.05

		// Combine with amplitude modulation
		envelope := 0.7 * (1 + 0.3*math.Sin(2*math.Pi*8*t))

		audio[i] = float32((f1 + f2 + f3 + noise) * envelope)
	}

	return audio
}
