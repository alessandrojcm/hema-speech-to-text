package processing

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

func TestGoAudioWAVExporter(t *testing.T) {
	exporter := NewGoAudioWAVExporter()
	require.NotNil(t, exporter)

	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "wav_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("ExportBasicWAV", func(t *testing.T) {
		// Create test audio segment
		sampleRate := 44100
		channels := 2
		duration := 1 * time.Second
		samples := generateTestTone(sampleRate, channels, 1000.0, duration) // 1kHz tone

		segment := &types.AudioSegment{
			ID:        "test_segment",
			Data:      samples,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(duration),
			Duration:  duration,
			Metadata: types.SegmentMetadata{
				SampleRate: sampleRate,
				Channels:   channels,
				BitDepth:   16,
				Quality:    0.8,
			},
		}

		// Export to WAV file
		outputPath := filepath.Join(tempDir, "test_basic.wav")
		err := exporter.Export(segment, outputPath)
		require.NoError(t, err)

		// Verify file exists and has reasonable size
		info, err := os.Stat(outputPath)
		require.NoError(t, err)
		assert.Greater(t, info.Size(), int64(1000)) // Should be larger than 1KB
	})

	t.Run("ExportWithMetadata", func(t *testing.T) {
		// Create test audio segment
		sampleRate := 16000
		channels := 1
		duration := 500 * time.Millisecond
		samples := generateTestTone(sampleRate, channels, 800.0, duration) // 800Hz tone

		segment := &types.AudioSegment{
			ID:        "test_segment_meta",
			Data:      samples,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(duration),
			Duration:  duration,
			Metadata: types.SegmentMetadata{
				SampleRate: sampleRate,
				Channels:   channels,
				BitDepth:   16,
				Quality:    0.9,
			},
		}

		metadata := map[string]string{
			"title":  "Test Audio",
			"artist": "Test Suite",
		}

		// Export to WAV file with metadata
		outputPath := filepath.Join(tempDir, "test_metadata.wav")
		err := exporter.ExportWithMetadata(segment, outputPath, metadata)
		require.NoError(t, err)

		// Verify file exists
		info, err := os.Stat(outputPath)
		require.NoError(t, err)
		assert.Greater(t, info.Size(), int64(500))
	})

	t.Run("ExportEmptySegment", func(t *testing.T) {
		// Test with empty segment
		segment := &types.AudioSegment{
			ID:   "empty_segment",
			Data: []float32{},
			Metadata: types.SegmentMetadata{
				SampleRate: 44100,
				Channels:   2,
				BitDepth:   16,
			},
		}

		outputPath := filepath.Join(tempDir, "test_empty.wav")
		err := exporter.Export(segment, outputPath)
		assert.Error(t, err) // Should fail with empty data
	})

	t.Run("ExportNilSegment", func(t *testing.T) {
		outputPath := filepath.Join(tempDir, "test_nil.wav")
		err := exporter.Export(nil, outputPath)
		assert.Error(t, err) // Should fail with nil segment
	})
}

// generateTestTone generates a sine wave tone for testing
func generateTestTone(sampleRate, channels int, frequency float64, duration time.Duration) []float32 {
	samplesPerChannel := int(float64(sampleRate) * duration.Seconds())
	totalSamples := samplesPerChannel * channels
	samples := make([]float32, totalSamples)

	for i := 0; i < samplesPerChannel; i++ {
		// Generate sine wave
		t := float64(i) / float64(sampleRate)
		sample := float32(0.5 * math.Sin(2*math.Pi*frequency*t))

		// Duplicate for all channels
		for ch := 0; ch < channels; ch++ {
			samples[i*channels+ch] = sample
		}
	}

	return samples
}

func BenchmarkGoAudioWAVExport(b *testing.B) {
	exporter := NewGoAudioWAVExporter()

	// Create test data
	sampleRate := 44100
	channels := 2
	duration := 1 * time.Second
	samples := generateTestTone(sampleRate, channels, 1000.0, duration)

	segment := &types.AudioSegment{
		ID:        "benchmark_segment",
		Data:      samples,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(duration),
		Duration:  duration,
		Metadata: types.SegmentMetadata{
			SampleRate: sampleRate,
			Channels:   channels,
			BitDepth:   16,
		},
	}

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "wav_benchmark")
	require.NoError(b, err)
	defer os.RemoveAll(tempDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		outputPath := filepath.Join(tempDir, fmt.Sprintf("benchmark_%d.wav", i))
		err := exporter.Export(segment, outputPath)
		require.NoError(b, err)
	}
}
