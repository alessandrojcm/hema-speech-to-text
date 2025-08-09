//go:build !noaudio

package processing

import (
	"fmt"

	"github.com/dh1tw/gosamplerate"
)

// GosamplerateResampler implements ResamplerInterface using gosamplerate
type GosamplerateResampler struct {
	converter *gosamplerate.Src
	quality   int
}

// NewGosamplerateResampler creates a new gosamplerate resampler
func NewGosamplerateResampler(quality int) (*GosamplerateResampler, error) {
	// Validate quality parameter
	if quality < 0 || quality > 4 {
		quality = 0 // SRC_SINC_BEST_QUALITY
	}

	// Create converter with single channel (we'll handle multi-channel separately)
	// Use a larger buffer size to handle longer audio segments
	converter, err := gosamplerate.New(quality, 1, 65536)
	if err != nil {
		return nil, fmt.Errorf("failed to create gosamplerate converter: %w", err)
	}

	return &GosamplerateResampler{
		converter: &converter,
		quality:   quality,
	}, nil
}

// Resample converts the input samples from inputRate to outputRate
func (r *GosamplerateResampler) Resample(input []float32, inputRate, outputRate int) ([]float32, error) {
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

	// Calculate conversion ratio
	ratio := float64(outputRate) / float64(inputRate)

	// Process the audio data
	output, err := r.converter.Process(input, ratio, false)
	if err != nil {
		return nil, fmt.Errorf("resampling failed: %w", err)
	}

	return output, nil
}

// ResampleMultiChannel resamples multi-channel audio data
func (r *GosamplerateResampler) ResampleMultiChannel(input []float32, inputRate, outputRate, channels int) ([]float32, error) {
	if len(input) == 0 {
		return input, nil
	}

	if channels <= 0 {
		return nil, fmt.Errorf("invalid channel count: %d", channels)
	}

	if channels == 1 {
		return r.Resample(input, inputRate, outputRate)
	}

	// For multi-channel audio, we need to deinterleave, resample each channel, then interleave
	samplesPerChannel := len(input) / channels
	if len(input)%channels != 0 {
		return nil, fmt.Errorf("input length %d is not divisible by channel count %d", len(input), channels)
	}

	// Deinterleave channels
	channelData := make([][]float32, channels)
	for ch := 0; ch < channels; ch++ {
		channelData[ch] = make([]float32, samplesPerChannel)
		for i := 0; i < samplesPerChannel; i++ {
			channelData[ch][i] = input[i*channels+ch]
		}
	}

	// Resample each channel
	resampledChannels := make([][]float32, channels)
	var outputSamplesPerChannel int
	for ch := 0; ch < channels; ch++ {
		resampled, err := r.Resample(channelData[ch], inputRate, outputRate)
		if err != nil {
			return nil, fmt.Errorf("failed to resample channel %d: %w", ch, err)
		}
		resampledChannels[ch] = resampled
		if ch == 0 {
			outputSamplesPerChannel = len(resampled)
		} else if len(resampled) != outputSamplesPerChannel {
			// Handle slight differences in output length by truncating to the minimum
			minLength := outputSamplesPerChannel
			if len(resampled) < minLength {
				minLength = len(resampled)
			}
			// Truncate all channels to the minimum length
			for i := 0; i <= ch; i++ {
				if len(resampledChannels[i]) > minLength {
					resampledChannels[i] = resampledChannels[i][:minLength]
				}
			}
			outputSamplesPerChannel = minLength
		}
	}

	// Interleave channels back together
	output := make([]float32, outputSamplesPerChannel*channels)
	for i := 0; i < outputSamplesPerChannel; i++ {
		for ch := 0; ch < channels; ch++ {
			output[i*channels+ch] = resampledChannels[ch][i]
		}
	}

	return output, nil
}

// Close releases resources used by the resampler
func (r *GosamplerateResampler) Close() error {
	if r.converter != nil {
		gosamplerate.Delete(*r.converter)
		r.converter = nil
	}
	return nil
}

// Reset resets the internal state of the resampler
func (r *GosamplerateResampler) Reset() error {
	if r.converter != nil {
		return r.converter.Reset()
	}
	return nil
}

// GetQuality returns the quality setting of the resampler
func (r *GosamplerateResampler) GetQuality() int {
	return r.quality
}

// SetQuality changes the quality setting (requires recreating the converter)
func (r *GosamplerateResampler) SetQuality(quality int) error {
	if quality < gosamplerate.SRC_SINC_BEST_QUALITY || quality > gosamplerate.SRC_LINEAR {
		return fmt.Errorf("invalid quality setting: %d", quality)
	}

	// Close existing converter
	if r.converter != nil {
		gosamplerate.Delete(*r.converter)
	}

	// Create new converter with new quality
	converter, err := gosamplerate.New(quality, 1, 65536)
	if err != nil {
		return fmt.Errorf("failed to create new converter: %w", err)
	}

	r.converter = &converter
	r.quality = quality
	return nil
}

// GetQualityName returns a human-readable name for the quality setting
func (r *GosamplerateResampler) GetQualityName() string {
	switch r.quality {
	case 0: // SRC_SINC_BEST_QUALITY
		return "Best Quality"
	case 1: // SRC_SINC_MEDIUM_QUALITY
		return "Medium Quality"
	case 2: // SRC_SINC_FASTEST
		return "Fastest"
	case 3: // SRC_ZERO_ORDER_HOLD
		return "Zero Order Hold"
	case 4: // SRC_LINEAR
		return "Linear"
	default:
		return "Unknown"
	}
}

// CustomResampler provides a fallback implementation using linear interpolation
type CustomResampler struct{}

// NewCustomResampler creates a new custom resampler (fallback implementation)
func NewCustomResampler() *CustomResampler {
	return &CustomResampler{}
}

// Resample performs linear interpolation resampling (basic implementation)
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
