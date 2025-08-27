package preprocessing

import (
	"github.com/rs/zerolog"
	"math"
)

type QualityFilter struct {
	minEnergy     float32
	minSNR        float32
	minVoiceRatio float32
	logger        zerolog.Logger
}

func NewQualityFilter(logger zerolog.Logger) *QualityFilter {
	return &QualityFilter{
		minEnergy:     0.01, // Minimum RMS energy
		minSNR:        3.0,  // Minimum signal-to-noise ratio in dB
		minVoiceRatio: 0.05, // Minimum ratio of voiced frames (lowered from 0.2 to allow real speech)
		logger:        logger.With().Str("component", "quality_filter").Logger(),
	}
}

// NewQualityFilterWithParams creates a quality filter with custom parameters
func NewQualityFilterWithParams(minEnergy, minSNR, minVoiceRatio float32, logger zerolog.Logger) *QualityFilter {
	return &QualityFilter{
		minEnergy:     minEnergy,
		minSNR:        minSNR,
		minVoiceRatio: minVoiceRatio,
		logger:        logger.With().Str("component", "quality_filter").Logger(),
	}
}

// ShouldProcessSegment determines if an audio segment is worth processing
func (qf *QualityFilter) ShouldProcessSegment(samples []float32) (bool, map[string]float32) {
	metrics := qf.calculateMetrics(samples)

	// Check minimum energy (avoid silent segments)
	if metrics["rms_energy"] < qf.minEnergy {
		qf.logger.Debug().
			Float32("rms_energy", metrics["rms_energy"]).
			Float32("threshold", qf.minEnergy).
			Msg("Segment rejected: energy too low")
		return false, metrics
	}

	// Check for sufficient voice content
	if metrics["voice_ratio"] < qf.minVoiceRatio {
		qf.logger.Debug().
			Float32("voice_ratio", metrics["voice_ratio"]).
			Float32("threshold", qf.minVoiceRatio).
			Msg("Segment rejected: insufficient voice content")
		return false, metrics
	}

	// Check signal-to-noise ratio
	if metrics["snr_db"] < qf.minSNR {
		qf.logger.Debug().
			Float32("snr_db", metrics["snr_db"]).
			Float32("threshold", qf.minSNR).
			Msg("Segment rejected: SNR too low")
		return false, metrics
	}

	qf.logger.Debug().
		Interface("metrics", metrics).
		Msg("Segment passed quality filter")

	return true, metrics
}

func (qf *QualityFilter) calculateMetrics(samples []float32) map[string]float32 {
	metrics := make(map[string]float32)

	// Calculate RMS energy
	var sumSquares float64
	for _, sample := range samples {
		sumSquares += float64(sample * sample)
	}
	metrics["rms_energy"] = float32(math.Sqrt(sumSquares / float64(len(samples))))

	// Calculate zero-crossing rate (indicator of voice vs noise)
	var zeroCrossings int
	for i := 1; i < len(samples); i++ {
		if (samples[i-1] >= 0 && samples[i] < 0) || (samples[i-1] < 0 && samples[i] >= 0) {
			zeroCrossings++
		}
	}
	metrics["zcr"] = float32(zeroCrossings) / float32(len(samples))

	// Estimate voice ratio using simple energy-based VAD
	frameSize := 160 // 10ms at 16kHz
	voicedFrames := 0
	totalFrames := len(samples) / frameSize

	for i := 0; i < len(samples)-frameSize; i += frameSize {
		frameEnergy := qf.calculateFrameEnergy(samples[i : i+frameSize])
		if frameEnergy > qf.minEnergy*2 {
			voicedFrames++
		}
	}

	metrics["voice_ratio"] = float32(voicedFrames) / float32(totalFrames)

	// Estimate SNR (simplified)
	metrics["snr_db"] = 20 * float32(math.Log10(float64(metrics["rms_energy"]/0.001)))

	return metrics
}

func (qf *QualityFilter) calculateFrameEnergy(frame []float32) float32 {
	var energy float32
	for _, sample := range frame {
		energy += sample * sample
	}
	return energy / float32(len(frame))
}
