package processing

import (
	"fmt"
	"os"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

// GoAudioWAVExporter implements WAVExporter using go-audio/wav
type GoAudioWAVExporter struct{}

// NewGoAudioWAVExporter creates a new go-audio WAV exporter
func NewGoAudioWAVExporter() *GoAudioWAVExporter {
	return &GoAudioWAVExporter{}
}

// Export exports an audio segment to a WAV file
func (e *GoAudioWAVExporter) Export(segment *types.AudioSegment, path string) error {
	return e.ExportWithMetadata(segment, path, nil)
}

// ExportWithMetadata exports an audio segment to a WAV file with metadata
func (e *GoAudioWAVExporter) ExportWithMetadata(segment *types.AudioSegment, path string, metadata map[string]string) error {
	if segment == nil {
		return fmt.Errorf("segment cannot be nil")
	}

	if len(segment.Data) == 0 {
		return fmt.Errorf("segment data cannot be empty")
	}

	// Create output file
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create WAV file: %w", err)
	}
	defer file.Close()

	// Convert float32 samples to int format for go-audio
	// go-audio expects int samples, so we convert from float32 [-1.0, 1.0] to int16 range
	intSamples := make([]int, len(segment.Data))
	for i, sample := range segment.Data {
		// Clamp sample to [-1.0, 1.0] range
		if sample > 1.0 {
			sample = 1.0
		} else if sample < -1.0 {
			sample = -1.0
		}
		// Convert to int16 range
		intSamples[i] = int(sample * 32767)
	}

	// Get bit depth
	bitDepth := segment.Metadata.BitDepth
	if bitDepth == 0 {
		bitDepth = 16 // Default to 16-bit
	}

	// Create audio buffer with proper format
	audioFormat := &audio.Format{
		NumChannels: segment.Metadata.Channels,
		SampleRate:  segment.Metadata.SampleRate,
	}

	audioBuffer := &audio.IntBuffer{
		Data:           intSamples,
		Format:         audioFormat,
		SourceBitDepth: bitDepth,
	}
	// Create WAV encoder (sampleRate, bitDepth, numChans, audioFormat)
	encoder := wav.NewEncoder(file, audioBuffer.Format.SampleRate, bitDepth, audioBuffer.Format.NumChannels, 1)
	if encoder == nil {
		return fmt.Errorf("failed to create WAV encoder")
	}
	defer encoder.Close()

	// Write audio data
	if err := encoder.Write(audioBuffer); err != nil {
		return fmt.Errorf("failed to write audio data: %w", err)
	}

	return nil
}
