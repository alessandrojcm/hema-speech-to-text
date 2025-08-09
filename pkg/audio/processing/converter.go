package processing

import (
	"fmt"
	"math"

	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

type FormatConverter struct {
	inputSampleRate  int
	outputSampleRate int
	inputChannels    int
	outputChannels   int
}

func NewFormatConverter(inputSampleRate, outputSampleRate, inputChannels, outputChannels int) *FormatConverter {
	return &FormatConverter{
		inputSampleRate:  inputSampleRate,
		outputSampleRate: outputSampleRate,
		inputChannels:    inputChannels,
		outputChannels:   outputChannels,
	}
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

	inputFrames := len(input) / fc.inputChannels
	output := make([]float32, inputFrames*fc.outputChannels)

	if fc.inputChannels == 2 && fc.outputChannels == 1 {
		for i := 0; i < inputFrames; i++ {
			left := input[i*2]
			right := input[i*2+1]
			output[i] = (left + right) / 2.0
		}
	} else if fc.inputChannels == 1 && fc.outputChannels == 2 {
		for i := 0; i < inputFrames; i++ {
			mono := input[i]
			output[i*2] = mono
			output[i*2+1] = mono
		}
	} else {
		return nil, fmt.Errorf("unsupported channel conversion: %d -> %d", fc.inputChannels, fc.outputChannels)
	}

	return output, nil
}

func (fc *FormatConverter) resample(input []float32) ([]float32, error) {
	if fc.inputSampleRate == fc.outputSampleRate {
		return input, nil
	}

	ratio := float64(fc.outputSampleRate) / float64(fc.inputSampleRate)
	inputFrames := len(input) / fc.outputChannels
	outputFrames := int(float64(inputFrames) * ratio)
	output := make([]float32, outputFrames*fc.outputChannels)

	for i := 0; i < outputFrames; i++ {
		srcIndex := float64(i) / ratio
		srcIndexInt := int(srcIndex)
		srcIndexFrac := srcIndex - float64(srcIndexInt)

		if srcIndexInt >= inputFrames-1 {
			srcIndexInt = inputFrames - 1
			srcIndexFrac = 0.0
		}

		for ch := 0; ch < fc.outputChannels; ch++ {
			sample1 := input[srcIndexInt*fc.outputChannels+ch]
			sample2 := sample1
			if srcIndexInt < inputFrames-1 {
				sample2 = input[(srcIndexInt+1)*fc.outputChannels+ch]
			}

			interpolated := sample1 + float32(srcIndexFrac)*float32(sample2-sample1)
			output[i*fc.outputChannels+ch] = interpolated
		}
	}

	return output, nil
}

func ConvertToWAV(segment *types.AudioSegment, outputSampleRate, outputChannels int) ([]byte, error) {
	converter := NewFormatConverter(
		segment.Metadata.SampleRate,
		outputSampleRate,
		segment.Metadata.Channels,
		outputChannels,
	)

	convertedData, err := converter.Convert(segment.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert audio format: %w", err)
	}

	return createWAVHeader(convertedData, outputSampleRate, outputChannels), nil
}

func createWAVHeader(data []float32, sampleRate, channels int) []byte {
	const bitsPerSample = 16
	byteRate := sampleRate * channels * bitsPerSample / 8
	blockAlign := channels * bitsPerSample / 8
	dataSize := len(data) * 2
	fileSize := 36 + dataSize

	header := make([]byte, 44)

	copy(header[0:4], "RIFF")
	writeUint32(header[4:8], uint32(fileSize))
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	writeUint32(header[16:20], 16)
	writeUint16(header[20:22], 1)
	writeUint16(header[22:24], uint16(channels))
	writeUint32(header[24:28], uint32(sampleRate))
	writeUint32(header[28:32], uint32(byteRate))
	writeUint16(header[32:34], uint16(blockAlign))
	writeUint16(header[34:36], uint16(bitsPerSample))
	copy(header[36:40], "data")
	writeUint32(header[40:44], uint32(dataSize))

	audioData := make([]byte, dataSize)
	for i, sample := range data {
		value := int16(sample * 32767.0)
		if value > 32767 {
			value = 32767
		} else if value < -32768 {
			value = -32768
		}
		audioData[i*2] = byte(value)
		audioData[i*2+1] = byte(value >> 8)
	}

	return append(header, audioData...)
}

func writeUint32(b []byte, v uint32) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
}

func writeUint16(b []byte, v uint16) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
}

func NormalizeAudio(samples []float32) {
	if len(samples) == 0 {
		return
	}

	var maxAbs float32
	for _, sample := range samples {
		abs := sample
		if abs < 0 {
			abs = -abs
		}
		if abs > maxAbs {
			maxAbs = abs
		}
	}

	if maxAbs > 0 && maxAbs != 1.0 {
		scale := 1.0 / maxAbs
		for i := range samples {
			samples[i] *= scale
		}
	}
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
