package testing

import (
	"math"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/your-org/hema-replay-system/pkg/speech/preprocessing"
)

// TestQualityFiltering tests the quality filtering functionality
func TestQualityFiltering(t *testing.T) {
	logger := zerolog.Nop()

	tests := []struct {
		name          string
		samples       []float32
		shouldProcess bool
		description   string
	}{
		{
			name:          "silence",
			samples:       generateSilence(16000), // 1 second of silence
			shouldProcess: false,
			description:   "Silent audio should be filtered",
		},
		{
			name:          "white_noise",
			samples:       generateWhiteNoise(16000, 0.1), // 1 second of low-level white noise
			shouldProcess: false,
			description:   "Pure noise should be filtered",
		},
		{
			name:          "clear_speech",
			samples:       generateSpeechLike(16000, 0.2, 200), // Clear speech simulation
			shouldProcess: true,
			description:   "Clear speech should pass",
		},
		{
			name:          "noisy_speech",
			samples:       generateNoisySpeech(16000, 0.15, 150, 0.05), // Speech with background noise
			shouldProcess: true,
			description:   "Speech with background noise should pass if SNR is acceptable",
		},
		{
			name:          "very_low_energy",
			samples:       generateLowEnergy(16000, 0.001), // Very low energy signal
			shouldProcess: false,
			description:   "Very low energy signals should be filtered",
		},
		{
			name:          "high_frequency_noise",
			samples:       generateHighFreqNoise(16000, 0.1, 4000), // High frequency noise
			shouldProcess: false,
			description:   "High frequency noise should be filtered based on voice ratio",
		},
	}

	qualityFilter := preprocessing.NewQualityFilter(logger)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldProcess, metrics := qualityFilter.ShouldProcessSegment(tt.samples)

			assert.Equal(t, tt.shouldProcess, shouldProcess, tt.description)

			// Verify metrics are calculated
			assert.Contains(t, metrics, "rms_energy")
			assert.Contains(t, metrics, "voice_ratio")
			assert.Contains(t, metrics, "snr_db")
			assert.Contains(t, metrics, "zcr")

			// Log metrics for debugging
			t.Logf("Test: %s", tt.name)
			t.Logf("  RMS Energy: %.6f", metrics["rms_energy"])
			t.Logf("  Voice Ratio: %.3f", metrics["voice_ratio"])
			t.Logf("  SNR (dB): %.2f", metrics["snr_db"])
			t.Logf("  ZCR: %.4f", metrics["zcr"])
			t.Logf("  Should Process: %v", shouldProcess)
		})
	}
}

// TestQualityFilterThresholds tests threshold adjustment
func TestQualityFilterThresholds(t *testing.T) {
	logger := zerolog.Nop()

	// Create filter with very permissive thresholds
	qualityFilter := preprocessing.NewQualityFilterWithParams(0.001, 0.5, 0.05, logger)

	// Generate borderline audio that should pass with permissive thresholds
	samples := generateLowEnergy(16000, 0.005)

	shouldProcess, metrics := qualityFilter.ShouldProcessSegment(samples)
	assert.True(t, shouldProcess, "Borderline audio should pass with permissive thresholds")

	// Create a stricter filter for comparison
	strictFilter := preprocessing.NewQualityFilterWithParams(0.02, 5.0, 0.3, logger)

	shouldProcessStrict, _ := strictFilter.ShouldProcessSegment(samples)
	assert.False(t, shouldProcessStrict, "Same audio should fail with strict filter")

	t.Logf("Metrics for borderline audio: %+v", metrics)
}

// TestSpeechPipelineIntegration tests the complete pipeline integration
func TestSpeechPipelineIntegration(t *testing.T) {
	t.Run("quality_filter_integration", func(t *testing.T) {
		// This test would verify:
		// 1. Quality filter is properly integrated in pipeline
		// 2. Low-quality segments are rejected before Whisper processing
		// 3. Debug information is properly saved when enabled
		// 4. Pipeline handles quality filter errors gracefully

		t.Skip("Requires full pipeline setup")
	})

}

// Helper functions for generating test audio data

// generateSilence creates a silent audio segment
func generateSilence(samples int) []float32 {
	return make([]float32, samples)
}

// generateWhiteNoise creates white noise audio
func generateWhiteNoise(samples int, amplitude float32) []float32 {
	audio := make([]float32, samples)
	for i := range audio {
		// Simple pseudo-random noise
		audio[i] = amplitude * (float32(i%1000)/500.0 - 1.0)
	}
	return audio
}

// generateSpeechLike creates speech-like audio with periodic energy patterns
func generateSpeechLike(samples int, amplitude float32, fundamentalFreq float32) []float32 {
	audio := make([]float32, samples)
	sampleRate := 16000.0

	for i := range audio {
		t := float32(i) / float32(sampleRate)

		// Create speech-like pattern with fundamental frequency and harmonics
		signal := float32(math.Sin(2 * math.Pi * float64(fundamentalFreq) * float64(t)))
		signal += 0.5 * float32(math.Sin(2*math.Pi*float64(fundamentalFreq*2)*float64(t)))
		signal += 0.25 * float32(math.Sin(2*math.Pi*float64(fundamentalFreq*3)*float64(t)))

		// Add amplitude modulation to simulate speech patterns
		envelope := 0.5 + 0.5*float32(math.Sin(2*math.Pi*5*float64(t))) // 5 Hz modulation

		audio[i] = amplitude * signal * envelope
	}
	return audio
}

// generateNoisySpeech creates speech with added background noise
func generateNoisySpeech(samples int, speechAmp float32, speechFreq float32, noiseAmp float32) []float32 {
	speech := generateSpeechLike(samples, speechAmp, speechFreq)
	noise := generateWhiteNoise(samples, noiseAmp)

	for i := range speech {
		speech[i] += noise[i]
	}
	return speech
}

// generateLowEnergy creates very low energy audio
func generateLowEnergy(samples int, amplitude float32) []float32 {
	audio := make([]float32, samples)
	for i := range audio {
		// Very low amplitude sine wave
		audio[i] = amplitude * float32(math.Sin(2*math.Pi*100*float64(i)/16000))
	}
	return audio
}

// generateHighFreqNoise creates high frequency noise (not speech-like)
func generateHighFreqNoise(samples int, amplitude float32, frequency float32) []float32 {
	audio := make([]float32, samples)
	sampleRate := 16000.0

	for i := range audio {
		t := float32(i) / float32(sampleRate)
		audio[i] = amplitude * float32(math.Sin(2*math.Pi*float64(frequency)*float64(t)))
	}
	return audio
}

// Benchmark tests for quality filtering performance
func BenchmarkQualityFilter(b *testing.B) {
	logger := zerolog.Nop()
	qualityFilter := preprocessing.NewQualityFilter(logger)

	// Generate test audio (1 second at 16kHz)
	samples := generateSpeechLike(16000, 0.2, 200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qualityFilter.ShouldProcessSegment(samples)
	}
}

func BenchmarkQualityFilterLongSegment(b *testing.B) {
	logger := zerolog.Nop()
	qualityFilter := preprocessing.NewQualityFilter(logger)

	// Generate longer test audio (10 seconds at 16kHz)
	samples := generateSpeechLike(160000, 0.2, 200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qualityFilter.ShouldProcessSegment(samples)
	}
}
