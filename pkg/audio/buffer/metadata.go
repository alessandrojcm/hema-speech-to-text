package buffer

import (
	"time"

	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

type SegmentInfo struct {
	StartPos   int64
	EndPos     int64
	Timestamp  time.Time
	Quality    float64
	HasVoice   bool
	NoiseLevel float64
}

func (si *SegmentInfo) ToMetadata(sampleRate, channels, bitDepth int) types.SegmentMetadata {
	return types.SegmentMetadata{
		SampleRate:    sampleRate,
		Channels:      channels,
		BitDepth:      bitDepth,
		Quality:       si.Quality,
		HasVoice:      si.HasVoice,
		NoiseLevel:    si.NoiseLevel,
		PeakAmplitude: 0.0,
		RMSLevel:      0.0,
	}
}

func calculateRMS(samples []float32) float64 {
	if len(samples) == 0 {
		return 0.0
	}

	var sum float64
	for _, sample := range samples {
		sum += float64(sample * sample)
	}
	return sum / float64(len(samples))
}

func calculatePeak(samples []float32) float64 {
	if len(samples) == 0 {
		return 0.0
	}

	var peak float32
	for _, sample := range samples {
		if sample < 0 {
			sample = -sample
		}
		if sample > peak {
			peak = sample
		}
	}
	return float64(peak)
}

func estimateNoise(samples []float32) float64 {
	if len(samples) == 0 {
		return 0.0
	}

	rms := calculateRMS(samples)
	return rms * 0.1
}

func calculateQuality(rms, peak, noise float64, hasVoice bool) float64 {
	if rms == 0 {
		return 0.0
	}

	snr := rms / (noise + 1e-10)
	quality := snr / 10.0

	if hasVoice {
		quality *= 1.2
	}

	if quality > 1.0 {
		quality = 1.0
	}

	return quality
}
