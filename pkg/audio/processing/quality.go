package processing

import (
	"math"

	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

type QualityMeter struct {
	sampleRate int
	channels   int

	rmsHistory    []float64
	peakHistory   []float64
	noiseEstimate float64
	snrHistory    []float64
	historySize   int
}

func NewQualityMeter(sampleRate, channels int) *QualityMeter {
	return &QualityMeter{
		sampleRate:  sampleRate,
		channels:    channels,
		historySize: 10,
		rmsHistory:  make([]float64, 0, 10),
		peakHistory: make([]float64, 0, 10),
		snrHistory:  make([]float64, 0, 10),
	}
}

func (qm *QualityMeter) AssessQuality(samples []float32) types.SegmentMetadata {
	if len(samples) == 0 {
		return types.SegmentMetadata{
			SampleRate: qm.sampleRate,
			Channels:   qm.channels,
		}
	}

	rms := qm.calculateRMS(samples)
	peak := qm.calculatePeak(samples)
	noise := qm.estimateNoise(samples)
	snr := qm.calculateSNR(rms, noise)
	hasVoice := qm.detectVoiceActivity(samples, rms, snr)
	quality := qm.calculateQualityScore(rms, peak, snr, hasVoice)

	qm.updateHistory(rms, peak, snr)

	return types.SegmentMetadata{
		SampleRate:    qm.sampleRate,
		Channels:      qm.channels,
		Quality:       quality,
		HasVoice:      hasVoice,
		NoiseLevel:    noise,
		PeakAmplitude: peak,
		RMSLevel:      rms,
	}
}

func (qm *QualityMeter) calculateRMS(samples []float32) float64 {
	if len(samples) == 0 {
		return 0.0
	}

	var sum float64
	for _, sample := range samples {
		sum += float64(sample * sample)
	}
	return math.Sqrt(sum / float64(len(samples)))
}

func (qm *QualityMeter) calculatePeak(samples []float32) float64 {
	if len(samples) == 0 {
		return 0.0
	}

	var peak float32
	for _, sample := range samples {
		abs := sample
		if abs < 0 {
			abs = -abs
		}
		if abs > peak {
			peak = abs
		}
	}
	return float64(peak)
}

func (qm *QualityMeter) estimateNoise(samples []float32) float64 {
	if len(samples) == 0 {
		return 0.0
	}

	rms := qm.calculateRMS(samples)

	if len(qm.rmsHistory) > 0 {
		minRMS := qm.rmsHistory[0]
		for _, histRMS := range qm.rmsHistory {
			if histRMS < minRMS {
				minRMS = histRMS
			}
		}
		qm.noiseEstimate = minRMS * 0.8
	} else {
		qm.noiseEstimate = rms * 0.1
	}

	return qm.noiseEstimate
}

func (qm *QualityMeter) calculateSNR(rms, noise float64) float64 {
	if noise <= 0 {
		noise = 1e-10
	}
	return 20.0 * math.Log10(rms/noise)
}

func (qm *QualityMeter) detectVoiceActivity(samples []float32, rms, snr float64) bool {
	const (
		rmsThreshold = 0.01
		snrThreshold = 10.0
	)

	if rms < rmsThreshold {
		return false
	}

	if snr < snrThreshold {
		return false
	}

	spectralCentroid := qm.calculateSpectralCentroid(samples)
	return spectralCentroid > 200.0 && spectralCentroid < 4000.0
}

func (qm *QualityMeter) calculateSpectralCentroid(samples []float32) float64 {
	if len(samples) < 2 {
		return 0.0
	}

	fft := qm.simpleFFT(samples)

	var weightedSum, magnitudeSum float64
	for i, magnitude := range fft {
		freq := float64(i) * float64(qm.sampleRate) / float64(len(fft))
		weightedSum += freq * magnitude
		magnitudeSum += magnitude
	}

	if magnitudeSum == 0 {
		return 0.0
	}

	return weightedSum / magnitudeSum
}

func (qm *QualityMeter) simpleFFT(samples []float32) []float64 {
	n := len(samples)
	if n < 2 {
		return []float64{}
	}

	magnitudes := make([]float64, n/2)

	for k := 0; k < n/2; k++ {
		var real, imag float64
		for i := 0; i < n; i++ {
			angle := -2.0 * math.Pi * float64(k) * float64(i) / float64(n)
			real += float64(samples[i]) * math.Cos(angle)
			imag += float64(samples[i]) * math.Sin(angle)
		}
		magnitudes[k] = math.Sqrt(real*real + imag*imag)
	}

	return magnitudes
}

func (qm *QualityMeter) calculateQualityScore(rms, peak, snr float64, hasVoice bool) float64 {
	if rms == 0 {
		return 0.0
	}

	quality := 0.0

	snrScore := math.Min(snr/30.0, 1.0)
	quality += snrScore * 0.4

	dynamicRange := peak / rms
	dynamicScore := math.Min(dynamicRange/4.0, 1.0)
	quality += dynamicScore * 0.3

	levelScore := math.Min(rms/0.5, 1.0)
	quality += levelScore * 0.2

	if hasVoice {
		quality += 0.1
	}

	if quality > 1.0 {
		quality = 1.0
	}

	return quality
}

func (qm *QualityMeter) updateHistory(rms, peak, snr float64) {
	if len(qm.rmsHistory) >= qm.historySize {
		qm.rmsHistory = qm.rmsHistory[1:]
		qm.peakHistory = qm.peakHistory[1:]
		qm.snrHistory = qm.snrHistory[1:]
	}

	qm.rmsHistory = append(qm.rmsHistory, rms)
	qm.peakHistory = append(qm.peakHistory, peak)
	qm.snrHistory = append(qm.snrHistory, snr)
}
