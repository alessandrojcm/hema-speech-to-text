package processing

import (
	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

// ResamplerInterface interface for sample rate conversion
type ResamplerInterface interface {
	Resample(input []float32, inputRate, outputRate int) ([]float32, error)
	Close() error
}

// VADInterface interface for voice activity detection
type VADInterface interface {
	DetectVoice(samples []float32) bool
	SetSensitivity(level float64) error
	Close() error
}

// WAVExporter interface for WAV file export
type WAVExporter interface {
	Export(segment *types.AudioSegment, path string) error
	ExportWithMetadata(segment *types.AudioSegment, path string, metadata map[string]string) error
}

// FFTProcessor interface for frequency domain analysis
type FFTProcessor interface {
	FFT(samples []float32) []complex128
	PowerSpectrum(samples []float32) []float64
	SpectralCentroid(samples []float32) float64
}

// WindowFunction interface for windowing functions
type WindowFunction interface {
	Apply(samples []float32, windowType string) []float32
	GetWindow(size int, windowType string) []float32
}
