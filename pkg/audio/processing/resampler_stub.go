//go:build noaudio

package processing

import "fmt"

// GosamplerateResampler stub implementation for noaudio builds
type GosamplerateResampler struct{}

// NewGosamplerateResampler creates a stub resampler
func NewGosamplerateResampler(quality int) (*GosamplerateResampler, error) {
	return nil, fmt.Errorf("gosamplerate not available in noaudio build")
}

// Resample stub implementation
func (r *GosamplerateResampler) Resample(input []float32, inputRate, outputRate int) ([]float32, error) {
	return nil, fmt.Errorf("gosamplerate not available in noaudio build")
}

// ResampleMultiChannel stub implementation
func (r *GosamplerateResampler) ResampleMultiChannel(input []float32, inputRate, outputRate, channels int) ([]float32, error) {
	return nil, fmt.Errorf("gosamplerate not available in noaudio build")
}

// Close stub implementation
func (r *GosamplerateResampler) Close() error {
	return nil
}

// Reset stub implementation
func (r *GosamplerateResampler) Reset() error {
	return fmt.Errorf("gosamplerate not available in noaudio build")
}

// GetQuality stub implementation
func (r *GosamplerateResampler) GetQuality() int {
	return 0
}

// SetQuality stub implementation
func (r *GosamplerateResampler) SetQuality(quality int) error {
	return fmt.Errorf("gosamplerate not available in noaudio build")
}

// GetQualityName stub implementation
func (r *GosamplerateResampler) GetQualityName() string {
	return "Not Available"
}

// CustomResampler is available in both builds
type CustomResampler struct{}

// NewCustomResampler creates a new custom resampler
func NewCustomResampler() *CustomResampler {
	return &CustomResampler{}
}

// Resample performs linear interpolation resampling
func (r *CustomResampler) Resample(input []float32, inputRate, outputRate int) ([]float32, error) {
	if len(input) == 0 {
		return input, nil
	}

	if inputRate <= 0 || outputRate <= 0 {
		return nil, fmt.Errorf("invalid sample rates: input=%d, output=%d", inputRate, outputRate)
	}

	// If rates are the same, return input unchanged
	if inputRate == outputRate {
		result := make([]float32, len(input))
		copy(result, input)
		return result, nil
	}

	// Calculate output length
	ratio := float64(outputRate) / float64(inputRate)
	outputLength := int(float64(len(input)) * ratio)
	output := make([]float32, outputLength)

	// Linear interpolation
	for i := 0; i < outputLength; i++ {
		// Calculate corresponding position in input
		pos := float64(i) / ratio

		// Get integer and fractional parts
		intPos := int(pos)
		fracPos := pos - float64(intPos)

		if intPos >= len(input)-1 {
			// At or beyond the end, use last sample
			output[i] = input[len(input)-1]
		} else {
			// Linear interpolation between two samples
			sample1 := input[intPos]
			sample2 := input[intPos+1]
			output[i] = sample1 + float32(fracPos)*(sample2-sample1)
		}
	}

	return output, nil
}

// Close is a no-op for the custom resampler
func (r *CustomResampler) Close() error {
	return nil
}
