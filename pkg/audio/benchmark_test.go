//go:build !noaudio
// +build !noaudio

package audio

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/your-org/hema-replay-system/pkg/audio/processing"
	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

// BenchmarkResamplerComparison compares different resampler implementations
func BenchmarkResamplerComparison(b *testing.B) {
	logger := zerolog.Nop()

	// Generate test audio: 10 seconds at 44.1kHz stereo
	inputSamples := generateBenchmarkAudio(44100, 2, 10*time.Second)

	benchmarks := []struct {
		name   string
		config types.ProcessingConfig
	}{
		{
			name: "Gosamplerate_BestQuality",
			config: types.ProcessingConfig{
				ResamplerType:    "gosamplerate",
				ResamplerQuality: 0, // Best quality
			},
		},
		{
			name: "Gosamplerate_MediumQuality",
			config: types.ProcessingConfig{
				ResamplerType:    "gosamplerate",
				ResamplerQuality: 2, // Medium quality
			},
		},
		{
			name: "Gosamplerate_FastestQuality",
			config: types.ProcessingConfig{
				ResamplerType:    "gosamplerate",
				ResamplerQuality: 4, // Fastest
			},
		},
		{
			name: "Custom_LinearInterpolation",
			config: types.ProcessingConfig{
				ResamplerType: "custom",
			},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			processor, err := processing.NewEnhancedAudioProcessor(bm.config, 44100, 2, logger)
			if err != nil {
				b.Fatalf("Failed to create processor: %v", err)
			}
			defer processor.Close()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := processor.Resample(inputSamples, 44100, 16000)
				if err != nil {
					b.Fatalf("Resampling failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkVADComparison compares different VAD implementations
func BenchmarkVADComparison(b *testing.B) {
	logger := zerolog.Nop()

	// Generate test audio: 1 second at 16kHz mono (typical for VAD)
	inputSamples := generateSpeechLikeBenchmarkAudio(16000, 1, 1*time.Second)

	benchmarks := []struct {
		name   string
		config types.ProcessingConfig
	}{
		{
			name: "WebRTC_VAD_Aggressive",
			config: types.ProcessingConfig{
				VADType: "webrtc",
				VADMode: 3, // Most aggressive
			},
		},
		{
			name: "WebRTC_VAD_Normal",
			config: types.ProcessingConfig{
				VADType: "webrtc",
				VADMode: 1, // Normal
			},
		},
		{
			name: "Threshold_VAD",
			config: types.ProcessingConfig{
				VADType:      "threshold",
				VADThreshold: 0.1,
			},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			processor, err := processing.NewEnhancedAudioProcessor(bm.config, 16000, 1, logger)
			if err != nil {
				b.Fatalf("Failed to create processor: %v", err)
			}
			defer processor.Close()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_ = processor.DetectVoiceActivity(inputSamples)
			}
		})
	}
}

// BenchmarkFFTComparison compares different FFT implementations
func BenchmarkFFTComparison(b *testing.B) {
	logger := zerolog.Nop()

	// Test different FFT sizes
	fftSizes := []int{256, 512, 1024, 2048, 4096}

	for _, size := range fftSizes {
		inputSamples := generateBenchmarkAudio(16000, 1, time.Duration(size)*time.Second/16000)[:size]

		b.Run(fmt.Sprintf("Gonum_FFT_%d", size), func(b *testing.B) {
			config := types.ProcessingConfig{FFTType: "gonum"}
			processor, err := processing.NewEnhancedAudioProcessor(config, 16000, 1, logger)
			if err != nil {
				b.Fatalf("Failed to create processor: %v", err)
			}
			defer processor.Close()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_ = processor.ComputeFFT(inputSamples)
			}
		})

		b.Run(fmt.Sprintf("Custom_DFT_%d", size), func(b *testing.B) {
			// Use the simple DFT from quality meter for comparison
			qualityMeter := processing.NewQualityMeter(16000, 1)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// This uses the simple DFT implementation
				_ = qualityMeter.AssessQuality(inputSamples)
			}
		})
	}
}

// BenchmarkAudioProcessingPipeline benchmarks the complete processing pipeline
func BenchmarkAudioProcessingPipeline(b *testing.B) {
	logger := zerolog.Nop()

	// Generate test audio: 5 seconds at 44.1kHz stereo
	inputSamples := generateBenchmarkAudio(44100, 2, 5*time.Second)

	benchmarks := []struct {
		name   string
		config types.ProcessingConfig
	}{
		{
			name: "Full_Library_Pipeline",
			config: types.ProcessingConfig{
				EnablePreprocessing: true,
				NoiseReduction:      true,
				Normalization:       true,
				HighpassFilter:      80.0,
				LowpassFilter:       8000.0,
				ResamplerType:       "gosamplerate",
				VADType:             "webrtc",
				WAVExporterType:     "goaudio",
				FFTType:             "gonum",
				ResamplerQuality:    0,
				VADMode:             3,
			},
		},
		{
			name: "Mixed_Pipeline",
			config: types.ProcessingConfig{
				EnablePreprocessing: true,
				NoiseReduction:      true,
				Normalization:       true,
				HighpassFilter:      80.0,
				LowpassFilter:       8000.0,
				ResamplerType:       "gosamplerate",
				VADType:             "threshold",
				WAVExporterType:     "goaudio",
				FFTType:             "gonum",
				ResamplerQuality:    2,
				VADThreshold:        0.1,
			},
		},
		{
			name: "Minimal_Processing",
			config: types.ProcessingConfig{
				EnablePreprocessing: false,
				ResamplerType:       "custom",
				VADType:             "threshold",
				WAVExporterType:     "goaudio",
				FFTType:             "gonum",
				VADThreshold:        0.1,
			},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			processor, err := processing.NewEnhancedAudioProcessor(bm.config, 44100, 2, logger)
			if err != nil {
				b.Fatalf("Failed to create processor: %v", err)
			}
			defer processor.Close()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := processor.Process(inputSamples, time.Now())
				if err != nil {
					b.Fatalf("Processing failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkQualityAssessment benchmarks quality assessment implementations
func BenchmarkQualityAssessment(b *testing.B) {
	logger := zerolog.Nop()

	// Generate test audio: 1 second at 16kHz mono
	inputSamples := generateBenchmarkAudio(16000, 1, 1*time.Second)

	b.Run("Basic_Quality_Assessment", func(b *testing.B) {
		qualityMeter := processing.NewQualityMeter(16000, 1)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_ = qualityMeter.AssessQuality(inputSamples)
		}
	})

	b.Run("Enhanced_Quality_Assessment", func(b *testing.B) {
		qualityMeter := processing.NewQualityMeterWithLogger(16000, 1, logger)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_ = qualityMeter.AssessEnhancedQuality(inputSamples)
		}
	})
}

// BenchmarkAudioManagerOperations benchmarks audio manager operations
func BenchmarkAudioManagerOperations(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping audio manager benchmark in short mode")
	}

	logger := zerolog.Nop()
	config := types.DefaultAudioConfig()
	config.Device.SampleRate = 16000 // Lower sample rate for testing
	config.Buffer.Duration = 10 * time.Second
	config.Extraction.MaxConcurrent = 3

	manager, err := NewAudioManager(config, logger)
	if err != nil {
		b.Skipf("Cannot create audio manager: %v", err)
		return
	}

	// Note: We can't actually start the manager in benchmarks without audio hardware
	// So we'll benchmark the components that don't require actual audio capture

	b.Run("GetHealth", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = manager.GetHealth()
		}
	})

	b.Run("GetMetrics", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = manager.GetMetrics()
		}
	})

	b.Run("GetPerformanceStats", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = manager.GetPerformanceStats()
		}
	})
}

// BenchmarkMemoryUsage benchmarks memory usage of different configurations
func BenchmarkMemoryUsage(b *testing.B) {
	logger := zerolog.Nop()

	// Large audio buffer to test memory efficiency
	inputSamples := generateBenchmarkAudio(44100, 2, 30*time.Second) // 30 seconds

	b.Run("Memory_Efficient_Config", func(b *testing.B) {
		config := types.ProcessingConfig{
			EnablePreprocessing: false, // Minimal processing
			ResamplerType:       "custom",
			VADType:             "threshold",
			FFTType:             "gonum",
		}

		processor, err := processing.NewEnhancedAudioProcessor(config, 44100, 2, logger)
		if err != nil {
			b.Fatalf("Failed to create processor: %v", err)
		}
		defer processor.Close()

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := processor.Process(inputSamples, time.Now())
			if err != nil {
				b.Fatalf("Processing failed: %v", err)
			}
		}
	})

	b.Run("Full_Featured_Config", func(b *testing.B) {
		config := types.ProcessingConfig{
			EnablePreprocessing: true,
			NoiseReduction:      true,
			Normalization:       true,
			HighpassFilter:      80.0,
			LowpassFilter:       8000.0,
			ResamplerType:       "gosamplerate",
			VADType:             "webrtc",
			FFTType:             "gonum",
			ResamplerQuality:    0,
			VADMode:             3,
		}

		processor, err := processing.NewEnhancedAudioProcessor(config, 44100, 2, logger)
		if err != nil {
			b.Fatalf("Failed to create processor: %v", err)
		}
		defer processor.Close()

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := processor.Process(inputSamples, time.Now())
			if err != nil {
				b.Fatalf("Processing failed: %v", err)
			}
		}
	})
}

// Helper functions for generating benchmark audio

func generateBenchmarkAudio(sampleRate, channels int, duration time.Duration) []float32 {
	samples := int(float64(sampleRate) * duration.Seconds())
	audio := make([]float32, samples*channels)

	// Generate realistic audio with multiple frequency components
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(sampleRate)

		// Mix of frequencies to simulate complex audio
		signal := 0.3*math.Sin(2*math.Pi*440*t) + // A4 note
			0.2*math.Sin(2*math.Pi*880*t) + // A5 note
			0.1*math.Sin(2*math.Pi*1320*t) + // E6 note
			0.05*math.Sin(2*math.Pi*1760*t) // A6 note

		// Add realistic noise
		noise := (rand.Float64() - 0.5) * 0.01
		signal += noise

		// Apply amplitude modulation
		envelope := 0.8 * (1 + 0.2*math.Sin(2*math.Pi*3*t))
		signal *= envelope

		for ch := 0; ch < channels; ch++ {
			audio[i*channels+ch] = float32(signal)
		}
	}

	return audio
}

func generateSpeechLikeBenchmarkAudio(sampleRate, channels int, duration time.Duration) []float32 {
	samples := int(float64(sampleRate) * duration.Seconds())
	audio := make([]float32, samples*channels)

	// Generate speech-like audio with formant structure
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(sampleRate)

		// Formant frequencies typical of speech
		f1 := 0.4 * math.Sin(2*math.Pi*700*t)  // First formant
		f2 := 0.3 * math.Sin(2*math.Pi*1220*t) // Second formant
		f3 := 0.2 * math.Sin(2*math.Pi*2600*t) // Third formant

		// Add pitch variation
		pitch := 150 + 50*math.Sin(2*math.Pi*0.5*t) // Varying fundamental frequency
		fundamental := 0.5 * math.Sin(2*math.Pi*pitch*t)

		// Combine all components
		signal := fundamental + f1 + f2 + f3

		// Add speech-like amplitude modulation
		envelope := 0.7 * (1 + 0.3*math.Sin(2*math.Pi*8*t)) // 8 Hz modulation
		signal *= envelope

		// Add realistic noise
		noise := (rand.Float64() - 0.5) * 0.02
		signal += noise

		for ch := 0; ch < channels; ch++ {
			audio[i*channels+ch] = float32(signal)
		}
	}

	return audio
}

// BenchmarkLatencyMeasurement measures processing latency
func BenchmarkLatencyMeasurement(b *testing.B) {
	logger := zerolog.Nop()

	// Small audio chunks to measure real-time processing latency
	chunkSizes := []int{256, 512, 1024, 2048} // samples

	for _, chunkSize := range chunkSizes {
		inputSamples := generateBenchmarkAudio(16000, 1, time.Duration(chunkSize)*time.Second/16000)[:chunkSize]

		b.Run(fmt.Sprintf("Latency_Chunk_%d", chunkSize), func(b *testing.B) {
			config := types.ProcessingConfig{
				EnablePreprocessing: true,
				ResamplerType:       "gosamplerate",
				VADType:             "webrtc",
				FFTType:             "gonum",
				ResamplerQuality:    2, // Medium quality for real-time
				VADMode:             1, // Less aggressive for real-time
			}

			processor, err := processing.NewEnhancedAudioProcessor(config, 16000, 1, logger)
			if err != nil {
				b.Fatalf("Failed to create processor: %v", err)
			}
			defer processor.Close()

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				start := time.Now()
				_, err := processor.Process(inputSamples, start)
				if err != nil {
					b.Fatalf("Processing failed: %v", err)
				}
				latency := time.Since(start)

				// Report latency as a custom metric
				b.ReportMetric(float64(latency.Nanoseconds()), "ns/op")
			}
		})
	}
}
