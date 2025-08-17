package processing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFormatConverter(t *testing.T) {
	converter, err := NewFormatConverter(44100, 16000, 2, 1)
	require.NoError(t, err)

	assert.NotNil(t, converter)
	assert.Equal(t, 44100, converter.inputSampleRate)
	assert.Equal(t, 16000, converter.outputSampleRate)
	assert.Equal(t, 2, converter.inputChannels)
	assert.Equal(t, 1, converter.outputChannels)
}

func TestFormatConverterConvertChannels(t *testing.T) {
	converter, err := NewFormatConverter(44100, 44100, 2, 1)
	require.NoError(t, err)

	input := []float32{0.5, -0.5, 0.3, -0.3, 0.1, -0.1}
	output, err := converter.Convert(input)

	require.NoError(t, err)
	assert.Len(t, output, 3)
	assert.InDelta(t, 0.0, output[0], 0.001)
	assert.InDelta(t, 0.0, output[1], 0.001)
	assert.InDelta(t, 0.0, output[2], 0.001)
}

func TestFormatConverterMonoToStereo(t *testing.T) {
	converter, err := NewFormatConverter(44100, 44100, 1, 2)
	require.NoError(t, err)

	input := []float32{0.5, 0.3, 0.1}
	output, err := converter.Convert(input)

	require.NoError(t, err)
	assert.Len(t, output, 6)
	assert.Equal(t, input[0], output[0])
	assert.Equal(t, input[0], output[1])
	assert.Equal(t, input[1], output[2])
	assert.Equal(t, input[1], output[3])
}

func TestFormatConverterResample(t *testing.T) {
	converter, err := NewFormatConverter(44100, 22050, 1, 1)
	require.NoError(t, err)

	input := make([]float32, 44100)
	for i := range input {
		input[i] = float32(i) / 44100.0
	}

	output, err := converter.Convert(input)

	require.NoError(t, err)
	assert.Len(t, output, 22050)
}

func TestApplyHighpassFilter(t *testing.T) {
	samples := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
	original := make([]float32, len(samples))
	copy(original, samples)

	ApplyHighpassFilter(samples, 1000.0, 44100)

	assert.NotEqual(t, original, samples)
}

func TestApplyLowpassFilter(t *testing.T) {
	samples := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
	original := make([]float32, len(samples))
	copy(original, samples)

	ApplyLowpassFilter(samples, 1000.0, 44100)

	assert.NotEqual(t, original, samples)
}

func TestApplyFiltersWithZeroCutoff(t *testing.T) {
	samples := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
	original := make([]float32, len(samples))
	copy(original, samples)

	ApplyHighpassFilter(samples, 0.0, 44100)
	assert.Equal(t, original, samples)

	ApplyLowpassFilter(samples, 0.0, 44100)
	assert.Equal(t, original, samples)
}

func TestApplyFiltersWithEmptyInput(t *testing.T) {
	samples := []float32{}

	ApplyHighpassFilter(samples, 1000.0, 44100)
	ApplyLowpassFilter(samples, 1000.0, 44100)

	assert.Len(t, samples, 0)
}
