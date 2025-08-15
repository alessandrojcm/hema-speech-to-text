package processing

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"

	audio "github.com/go-audio/audio"
	tranforms "github.com/go-audio/transforms"
	wav "github.com/go-audio/wav"
	"github.com/orcaman/writerseeker"
	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

type FormatConverter struct {
	inputSampleRate  int
	outputSampleRate int
	inputChannels    int
	outputChannels   int
	resampler        *GosamplerateResampler
}

func NewFormatConverter(inputSampleRate, outputSampleRate, inputChannels, outputChannels int) (*FormatConverter, error) {
	resampler, error := NewGosamplerateResampler(0)
	if error != nil {
		return nil, error
	}
	return &FormatConverter{
		inputSampleRate:  inputSampleRate,
		outputSampleRate: outputSampleRate,
		inputChannels:    inputChannels,
		outputChannels:   outputChannels,
		resampler:        resampler,
	}, nil
}

func (fc *FormatConverter) Convert(input []float32) ([]float32, error) {
	if len(input) == 0 {
		return input, nil
	}

	output := input

	if fc.inputChannels != fc.outputChannels {
		var err error
		output, err = fc.convertChannels(output)
		if err != nil {
			return nil, err
		}
	}

	if fc.inputSampleRate != fc.outputSampleRate {
		var err error
		output, err = fc.resample(output)
		if err != nil {
			return nil, err
		}
	}

	return output, nil
}

func (fc *FormatConverter) convertChannels(input []float32) ([]float32, error) {
	if fc.inputChannels == fc.outputChannels {
		return input, nil
	}
	if fc.inputChannels == 2 && fc.outputChannels == 1 {
		floatBuffer := &audio.Float32Buffer{Format: &audio.Format{
			SampleRate:  fc.inputSampleRate,
			NumChannels: fc.inputChannels,
		}, Data: input}
		output := tranforms.MonoDownmix(floatBuffer.AsFloatBuffer())
		if output == nil {
			return nil, errors.New("failed to convert channels")
		}
		return floatBuffer.AsFloat32Buffer().Data, nil
	}
	if fc.inputChannels == 1 && fc.outputChannels == 2 {
		floatBuffer := &audio.Float32Buffer{Format: &audio.Format{
			SampleRate:  fc.inputSampleRate,
			NumChannels: fc.inputChannels,
		}, Data: input}
		output := tranforms.MonoToStereoF32(floatBuffer.AsFloat32Buffer())
		if output == nil {
			return nil, errors.New("failed to convert channels")
		}
		return floatBuffer.AsFloat32Buffer().Data, nil
	}
	return nil, errors.ErrUnsupported
}

func (fc *FormatConverter) resample(input []float32) ([]float32, error) {
	output, err := fc.resampler.Resample(input, fc.inputSampleRate, fc.outputSampleRate)
	if err != nil {
		return nil, fmt.Errorf("failed to resample: %w", err)
	}
	return output, nil
}

func ConvertToWAV(segment *types.AudioSegment, outputSampleRate, outputChannels int) ([]byte, error) {
	converter, error := NewFormatConverter(
		segment.Metadata.SampleRate,
		outputSampleRate,
		segment.Metadata.Channels,
		outputChannels,
	)
	if error != nil {
		return nil, fmt.Errorf("failed to create format converter: %w", error)
	}

	convertedData, err := converter.Convert(segment.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert audio format: %w", err)
	}
	// Convert float32 samples to int16 for WAV encoding
	intSamples := make([]int, len(convertedData))
	for i, sample := range convertedData {
		// Clamp to [-1, 1] range
		if sample > 1.0 {
			sample = 1.0
		} else if sample < -1.0 {
			sample = -1.0
		}
		// Convert to 16-bit integer
		intSamples[i] = int(sample * 32767.0)
	}

	seeker := &writerseeker.WriterSeeker{}
	encoder := wav.NewEncoder(seeker, outputSampleRate, 16, outputChannels, 1)

	intBuffer := &audio.IntBuffer{
		Format: &audio.Format{
			SampleRate:  outputSampleRate,
			NumChannels: outputChannels,
		},
		Data:           intSamples,
		SourceBitDepth: 16,
	}

	if err := encoder.Write(intBuffer); err != nil {
		return nil, fmt.Errorf("failed to write WAV data: %w", err)
	}

	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("failed to close WAV encoder: %w", err)
	}

	buf := bytes.NewBuffer([]byte{})
	_, err = io.Copy(buf, seeker.Reader())
	if err != nil {
		return nil, fmt.Errorf("failed to copy data to buffer: %w", err)
	}

	return buf.Bytes(), nil
}

func ApplyHighpassFilter(samples []float32, cutoffFreq float64, sampleRate int) {
	if len(samples) == 0 || cutoffFreq <= 0 {
		return
	}

	rc := 1.0 / (2.0 * math.Pi * cutoffFreq)
	dt := 1.0 / float64(sampleRate)
	alpha := rc / (rc + dt)

	var prevInput, prevOutput float32
	for i, input := range samples {
		output := float32(alpha) * (prevOutput + input - prevInput)
		samples[i] = output
		prevInput = input
		prevOutput = output
	}
}

func ApplyLowpassFilter(samples []float32, cutoffFreq float64, sampleRate int) {
	if len(samples) == 0 || cutoffFreq <= 0 {
		return
	}

	rc := 1.0 / (2.0 * math.Pi * cutoffFreq)
	dt := 1.0 / float64(sampleRate)
	alpha := dt / (rc + dt)

	var prevOutput float32
	for i, input := range samples {
		output := prevOutput + float32(alpha)*(input-prevOutput)
		samples[i] = output
		prevOutput = output
	}
}
