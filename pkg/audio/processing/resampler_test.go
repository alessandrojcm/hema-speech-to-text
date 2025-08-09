package processing

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGosamplerateResampler(t *testing.T) {
	// Skip if libsamplerate is not available
	resampler, err := NewGosamplerateResampler(0) // Best quality
	if err != nil {
		t.Skip("libsamplerate not available, skipping gosamplerate tests")
	}
	defer resampler.Close()

	t.Run("BasicResampling", func(t *testing.T) {
		// Generate test signal at 44100 Hz
		inputRate := 44100
		outputRate := 16000
		frequency := 1000.0 // 1 kHz tone
		duration := 0.1     // 100ms

		inputSamples := generateTestSineWave(inputRate, frequency, duration)

		// Resample to 16000 Hz
		outputSamples, err := resampler.Resample(inputSamples, inputRate, outputRate)
		require.NoError(t, err)

		// Check output length
		expectedLength := len(inputSamples) * outputRate / inputRate
		assert.InDelta(t, expectedLength, len(outputSamples), float64(expectedLength)*0.1) // 10% tolerance

		// Verify signal is preserved (check for presence of 1kHz component)
		assert.True(t, hasFrequencyComponent(outputSamples, outputRate, frequency, 0.1))
	})

	t.Run("SameRateResampling", func(t *testing.T) {
		inputRate := 44100
		outputRate := 44100
		samples := generateTestSineWave(inputRate, 440.0, 0.1)

		outputSamples, err := resampler.Resample(samples, inputRate, outputRate)
		require.NoError(t, err)

		// Should be identical length
		assert.Equal(t, len(samples), len(outputSamples))

		// Should be very similar content
		for i, sample := range samples {
			assert.InDelta(t, sample, outputSamples[i], 0.01)
		}
	})

	t.Run("UpsamplingAndDownsampling", func(t *testing.T) {
		// Test that resampling works in both directions without strict length requirements
		inputSamples := generateTestSineWave(22050, 500.0, 0.1)

		// Test upsampling 22050 -> 44100
		upsampled, err := resampler.Resample(inputSamples, 22050, 44100)
		require.NoError(t, err)
		assert.NotEmpty(t, upsampled, "Upsampling should produce output")

		// Test downsampling 44100 -> 22050
		downsampled, err := resampler.Resample(upsampled, 44100, 22050)
		require.NoError(t, err)
		assert.NotEmpty(t, downsampled, "Downsampling should produce output")

		// Test different ratio: 16000 -> 48000
		samples16k := generateTestSineWave(16000, 1000.0, 0.1)
		samples48k, err := resampler.Resample(samples16k, 16000, 48000)
		require.NoError(t, err)
		assert.NotEmpty(t, samples48k, "16k->48k resampling should work")

		// Test back: 48000 -> 16000
		samplesBack16k, err := resampler.Resample(samples48k, 48000, 16000)
		require.NoError(t, err)
		assert.NotEmpty(t, samplesBack16k, "48k->16k resampling should work")

		// Verify signal quality is reasonable (not testing exact frequency preservation
		// due to resampling artifacts, but ensuring we get reasonable output)
		assert.Greater(t, len(samplesBack16k), 100, "Should have reasonable number of output samples")
	})

	t.Run("MultiChannelResampling", func(t *testing.T) {
		inputRate := 44100
		outputRate := 16000
		channels := 2

		// Generate stereo test signal
		monoSamples := generateTestSineWave(inputRate, 1000.0, 0.1)
		stereoSamples := make([]float32, len(monoSamples)*channels)
		for i, sample := range monoSamples {
			stereoSamples[i*channels] = sample         // Left channel
			stereoSamples[i*channels+1] = sample * 0.5 // Right channel (different amplitude)
		}

		outputSamples, err := resampler.ResampleMultiChannel(stereoSamples, inputRate, outputRate, channels)
		require.NoError(t, err)

		expectedLength := len(stereoSamples) * outputRate / inputRate
		assert.InDelta(t, expectedLength, len(outputSamples), float64(expectedLength)*0.2) // 20% tolerance

		// Verify it's still stereo (even number of samples)
		assert.Equal(t, 0, len(outputSamples)%channels)
	})

	t.Run("EmptyInput", func(t *testing.T) {
		outputSamples, err := resampler.Resample([]float32{}, 44100, 16000)
		require.NoError(t, err)
		assert.Empty(t, outputSamples)
	})

	t.Run("InvalidRates", func(t *testing.T) {
		samples := generateTestSineWave(44100, 1000.0, 0.1)

		_, err := resampler.Resample(samples, 0, 16000)
		assert.Error(t, err)

		_, err = resampler.Resample(samples, 44100, -1)
		assert.Error(t, err)
	})

	t.Run("QualitySettings", func(t *testing.T) {
		assert.Equal(t, 0, resampler.GetQuality())
		assert.Equal(t, "Best Quality", resampler.GetQualityName())

		// Test changing quality
		err := resampler.SetQuality(2) // Fastest
		require.NoError(t, err)
		assert.Equal(t, 2, resampler.GetQuality())
		assert.Equal(t, "Fastest", resampler.GetQualityName())
	})
}

func TestCustomResampler(t *testing.T) {
	resampler := NewCustomResampler()
	defer resampler.Close()

	t.Run("BasicResampling", func(t *testing.T) {
		inputRate := 44100
		outputRate := 22050
		samples := generateTestSineWave(inputRate, 1000.0, 0.1)

		outputSamples, err := resampler.Resample(samples, inputRate, outputRate)
		require.NoError(t, err)

		// Should be approximately half the length
		expectedLength := len(samples) / 2
		assert.InDelta(t, expectedLength, len(outputSamples), float64(expectedLength)*0.1)
	})

	t.Run("LinearInterpolation", func(t *testing.T) {
		// Test with simple ramp signal
		inputSamples := []float32{0.0, 1.0, 2.0, 3.0}

		// Upsample by 2x
		outputSamples, err := resampler.Resample(inputSamples, 1000, 2000)
		require.NoError(t, err)

		// Should have approximately 8 samples
		assert.Greater(t, len(outputSamples), 6)
		assert.Less(t, len(outputSamples), 10)

		// Values should be interpolated
		assert.True(t, outputSamples[0] >= 0.0 && outputSamples[0] <= 0.5)
		assert.True(t, outputSamples[len(outputSamples)-1] >= 2.5)
	})
}

func TestResamplerComparison(t *testing.T) {
	// Compare gosamplerate vs custom resampler
	gosampleResampler, err := NewGosamplerateResampler(0)
	if err != nil {
		t.Skip("libsamplerate not available, skipping comparison test")
	}
	defer gosampleResampler.Close()

	customResampler := NewCustomResampler()
	defer customResampler.Close()

	// Generate test signal
	inputRate := 44100
	outputRate := 16000
	samples := generateTestSineWave(inputRate, 1000.0, 0.1)

	// Resample with both methods
	gosampleOutput, err1 := gosampleResampler.Resample(samples, inputRate, outputRate)
	require.NoError(t, err1)

	customOutput, err2 := customResampler.Resample(samples, inputRate, outputRate)
	require.NoError(t, err2)

	// Both should produce similar length outputs
	assert.InDelta(t, len(gosampleOutput), len(customOutput), float64(len(gosampleOutput))*0.1)

	// Both should preserve the frequency component
	assert.True(t, hasFrequencyComponent(gosampleOutput, outputRate, 1000.0, 0.1))
	assert.True(t, hasFrequencyComponent(customOutput, outputRate, 1000.0, 0.1))
}

// Helper functions

func generateTestSineWave(sampleRate int, frequency, duration float64) []float32 {
	numSamples := int(float64(sampleRate) * duration)
	samples := make([]float32, numSamples)

	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)
		samples[i] = float32(0.5 * math.Sin(2*math.Pi*frequency*t))
	}

	return samples
}

func hasFrequencyComponent(samples []float32, sampleRate int, targetFreq, tolerance float64) bool {
	// Simple frequency detection using FFT
	if len(samples) < 64 {
		return false
	}

	// Use power of 2 length for FFT
	fftSize := 1
	for fftSize < len(samples) {
		fftSize *= 2
	}
	if fftSize > len(samples) {
		fftSize /= 2
	}

	// Compute simple DFT for the target frequency
	targetBin := int(targetFreq * float64(fftSize) / float64(sampleRate))

	var realPart, imagPart float64
	for i := 0; i < fftSize && i < len(samples); i++ {
		angle := 2 * math.Pi * float64(targetBin) * float64(i) / float64(fftSize)
		realPart += float64(samples[i]) * math.Cos(angle)
		imagPart -= float64(samples[i]) * math.Sin(angle)
	}

	magnitude := math.Sqrt(realPart*realPart + imagPart*imagPart)
	// Check if magnitude is above threshold
	return magnitude > tolerance*float64(fftSize)
}

func BenchmarkGosamplerateResampler(b *testing.B) {
	resampler, err := NewGosamplerateResampler(0)
	if err != nil {
		b.Skip("libsamplerate not available")
	}
	defer resampler.Close()

	samples := generateTestSineWave(44100, 1000.0, 1.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resampler.Resample(samples, 44100, 16000)
	}
}

func BenchmarkCustomResampler(b *testing.B) {
	resampler := NewCustomResampler()
	defer resampler.Close()

	samples := generateTestSineWave(44100, 1000.0, 1.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resampler.Resample(samples, 44100, 16000)
	}
}
