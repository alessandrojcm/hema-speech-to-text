package processing

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

func TestGonumFFTProcessor(t *testing.T) {
	processor := NewFFT(16000)
	require.NotNil(t, processor)

	t.Run("FFTBasic", func(t *testing.T) {
		// Generate a simple sine wave
		sampleRate := 1000
		frequency := 100.0 // 100 Hz
		samples := generateSineWave(sampleRate, frequency, 1.0, 1024)

		// Compute FFT
		fftResult := processor.FFT(samples)
		require.NotNil(t, fftResult)
		assert.Equal(t, len(samples), len(fftResult))

		// The peak should be around the frequency bin corresponding to 100 Hz
		// For 1000 Hz sample rate and 1024 samples, bin 102.4 ≈ bin 102
		expectedBin := int(frequency * float64(len(samples)) / float64(sampleRate))

		// Find the bin with maximum magnitude
		maxMagnitude := 0.0
		maxBin := 0
		for i := 1; i < len(fftResult)/2; i++ { // Skip DC component
			magnitude := real(fftResult[i])*real(fftResult[i]) + imag(fftResult[i])*imag(fftResult[i])
			if magnitude > maxMagnitude {
				maxMagnitude = magnitude
				maxBin = i
			}
		}

		// The maximum should be close to the expected bin
		assert.InDelta(t, expectedBin, maxBin, 2.0) // Allow ±2 bins tolerance
	})

	t.Run("PowerSpectrum", func(t *testing.T) {
		// Generate test signal with two frequencies
		sampleRate := 2000
		samples1 := generateSineWave(sampleRate, 200.0, 0.5, 512) // 200 Hz
		samples2 := generateSineWave(sampleRate, 400.0, 0.3, 512) // 400 Hz

		// Combine signals
		samples := make([]float32, len(samples1))
		for i := range samples {
			samples[i] = samples1[i] + samples2[i]
		}

		// Compute power spectrum
		powerSpectrum := processor.PowerSpectrum(samples)
		require.NotNil(t, powerSpectrum)
		assert.Equal(t, len(samples)/2+1, len(powerSpectrum))

		// Find peaks
		peaks := findPeaks(powerSpectrum, 0.01) // Threshold for peak detection
		assert.GreaterOrEqual(t, len(peaks), 2) // Should find at least 2 peaks

		// Convert peak bins to frequencies
		peakFreqs := make([]float64, len(peaks))
		for i, bin := range peaks {
			peakFreqs[i] = float64(bin) * float64(sampleRate) / float64(len(samples))
		}

		// Check if we found frequencies close to 200 and 400 Hz
		found200 := false
		found400 := false
		for _, freq := range peakFreqs {
			if math.Abs(freq-200.0) < 20.0 {
				found200 = true
			}
			if math.Abs(freq-400.0) < 20.0 {
				found400 = true
			}
		}
		assert.True(t, found200, "Should find peak near 200 Hz")
		assert.True(t, found400, "Should find peak near 400 Hz")
	})

	t.Run("SpectralCentroid", func(t *testing.T) {
		// Test with low frequency signal
		lowFreqSamples := generateSineWave(1000, 100.0, 1.0, 512)
		lowCentroid := processor.SpectralCentroid(lowFreqSamples)

		// Test with high frequency signal
		highFreqSamples := generateSineWave(1000, 400.0, 1.0, 512)
		highCentroid := processor.SpectralCentroid(highFreqSamples)

		// High frequency signal should have higher spectral centroid
		assert.Greater(t, highCentroid, lowCentroid)
	})

	t.Run("EmptyInput", func(t *testing.T) {
		// Test with empty input
		fftResult := processor.FFT([]float32{})
		assert.Nil(t, fftResult)

		powerSpectrum := processor.PowerSpectrum([]float32{})
		assert.Nil(t, powerSpectrum)

		centroid := processor.SpectralCentroid([]float32{})
		assert.Equal(t, 0.0, centroid)
	})
}

func TestZeroCrossingRate(t *testing.T) {
	t.Run("SineWave", func(t *testing.T) {
		// Generate sine wave
		samples := generateSineWave(1000, 100.0, 1.0, 1000) // 100 Hz for 1 second
		zcr := ZeroCrossingRate(samples)

		// For a 100 Hz sine wave, we expect about 200 zero crossings per second
		// (2 crossings per cycle)
		expectedZCR := 200.0 / 1000.0             // 200 crossings in 1000 samples
		assert.InDelta(t, expectedZCR, zcr, 0.05) // Allow 5% tolerance
	})

	t.Run("ConstantSignal", func(t *testing.T) {
		// Constant positive signal
		samples := make([]float32, 100)
		for i := range samples {
			samples[i] = 0.5
		}
		zcr := ZeroCrossingRate(samples)
		assert.Equal(t, 0.0, zcr) // No zero crossings
	})

	t.Run("AlternatingSignal", func(t *testing.T) {
		// Alternating +1, -1 signal
		samples := make([]float32, 100)
		for i := range samples {
			if i%2 == 0 {
				samples[i] = 1.0
			} else {
				samples[i] = -1.0
			}
		}
		zcr := ZeroCrossingRate(samples)
		assert.InDelta(t, 1.0, zcr, 0.01) // Maximum possible ZCR
	})

	t.Run("EmptyInput", func(t *testing.T) {
		zcr := ZeroCrossingRate([]float32{})
		assert.Equal(t, 0.0, zcr)
	})

	t.Run("SingleSample", func(t *testing.T) {
		zcr := ZeroCrossingRate([]float32{1.0})
		assert.Equal(t, 0.0, zcr)
	})
}

// Helper functions

func generateSineWave(sampleRate int, frequency, amplitude float64, numSamples int) []float32 {
	samples := make([]float32, numSamples)
	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)
		samples[i] = float32(amplitude * math.Sin(2*math.Pi*frequency*t))
	}
	return samples
}

func findPeaks(spectrum []float64, threshold float64) []int {
	var peaks []int
	for i := 1; i < len(spectrum)-1; i++ {
		if spectrum[i] > threshold && spectrum[i] > spectrum[i-1] && spectrum[i] > spectrum[i+1] {
			peaks = append(peaks, i)
		}
	}
	return peaks
}

func BenchmarkGonumFFT(b *testing.B) {
	processor := NewFFT(16000)
	samples := generateSineWave(44100, 1000.0, 1.0, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.FFT(samples)
	}
}

func BenchmarkGonumPowerSpectrum(b *testing.B) {
	// Create test configuration
	config := types.DefaultAudioConfig()
	config.Device.SampleRate = 16000 // Lower sample rate for testing
	config.Buffer.Duration = 10 * time.Second
	config.Extraction.MaxConcurrent = 3
	processor := NewFFT(16000)
	samples := generateSineWave(44100, 1000.0, 1.0, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.PowerSpectrum(samples)
	}
}

func BenchmarkWindowFunction(b *testing.B) {
	processor := NewFFTWindowFunction()
	sine := generateSineWave(44100, 1000.0, 1.0, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.Apply(sine, "hann")
	}
}
