package processing

import (
	"math"
	"time"

	"github.com/rs/zerolog"
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

	// Enhanced quality assessment
	fftProcessor FFTProcessor
	windowFunc   WindowFunction
	logger       zerolog.Logger
}

// EnhancedQualityMetrics provides detailed quality assessment
type EnhancedQualityMetrics struct {
	// Basic metrics (from existing SegmentMetadata)
	Basic types.SegmentMetadata

	// Enhanced spectral analysis
	SpectralCentroid    float64 `json:"spectral_centroid"`
	SpectralRolloff     float64 `json:"spectral_rolloff"`
	SpectralFlatness    float64 `json:"spectral_flatness"`
	HighFrequencyEnergy float64 `json:"high_frequency_energy"`

	// Distortion and artifacts
	THDEstimate  float64 `json:"thd_estimate"`
	ClippingRate float64 `json:"clipping_rate"`
	CrestFactor  float64 `json:"crest_factor"`

	// Voice characteristics
	VoiceProbability float64 `json:"voice_probability"`
	SpeechClarity    float64 `json:"speech_clarity"`
	VocalEffort      float64 `json:"vocal_effort"`

	// Quality indicators
	IsClipped         bool `json:"is_clipped"`
	IsSaturated       bool `json:"is_saturated"`
	HasExcessiveNoise bool `json:"has_excessive_noise"`
	IsUnderModulated  bool `json:"is_under_modulated"`

	// Timestamp
	Timestamp time.Time `json:"timestamp"`
}

func NewQualityMeter(sampleRate, channels int) *QualityMeter {
	return &QualityMeter{
		sampleRate:   sampleRate,
		channels:     channels,
		historySize:  10,
		rmsHistory:   make([]float64, 0, 10),
		peakHistory:  make([]float64, 0, 10),
		snrHistory:   make([]float64, 0, 10),
		fftProcessor: NewGonumFFTProcessor(),
		windowFunc:   NewGonumWindowFunction(),
		logger:       zerolog.Nop(), // Default no-op logger
	}
}

func NewQualityMeterWithLogger(sampleRate, channels int, logger zerolog.Logger) *QualityMeter {
	qm := NewQualityMeter(sampleRate, channels)
	qm.logger = logger.With().Str("component", "quality_meter").Logger()
	return qm
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

// AssessEnhancedQuality performs comprehensive quality assessment
func (qm *QualityMeter) AssessEnhancedQuality(samples []float32) EnhancedQualityMetrics {
	// Get basic quality metrics
	basic := qm.AssessQuality(samples)

	enhanced := EnhancedQualityMetrics{
		Basic:     basic,
		Timestamp: time.Now(),
	}

	if len(samples) == 0 {
		return enhanced
	}

	// Enhanced spectral analysis
	qm.analyzeSpectralCharacteristics(samples, &enhanced)

	// Distortion analysis
	qm.analyzeDistortionCharacteristics(samples, &enhanced)

	// Voice characteristics
	qm.analyzeVoiceCharacteristics(&enhanced)

	// Set quality indicators
	qm.setQualityIndicators(&enhanced)

	return enhanced
}

// analyzeSpectralCharacteristics performs detailed spectral analysis
func (qm *QualityMeter) analyzeSpectralCharacteristics(samples []float32, metrics *EnhancedQualityMetrics) {
	if qm.fftProcessor == nil || len(samples) < 64 {
		return
	}

	// Apply window function for better spectral analysis
	windowed := samples
	if qm.windowFunc != nil {
		windowed = qm.windowFunc.Apply(samples, "hann")
	}

	// Compute power spectrum
	powerSpectrum := qm.fftProcessor.PowerSpectrum(windowed)
	if len(powerSpectrum) == 0 {
		return
	}

	// Calculate spectral centroid
	metrics.SpectralCentroid = qm.calculateEnhancedSpectralCentroid(powerSpectrum)

	// Calculate spectral rolloff (85% energy point)
	metrics.SpectralRolloff = qm.calculateSpectralRolloff(powerSpectrum, 0.85)

	// Calculate spectral flatness
	metrics.SpectralFlatness = qm.calculateSpectralFlatness(powerSpectrum)

	// Calculate high frequency energy (above 4kHz)
	metrics.HighFrequencyEnergy = qm.calculateHighFrequencyEnergy(powerSpectrum)
}

// analyzeDistortionCharacteristics analyzes distortion and artifacts
func (qm *QualityMeter) analyzeDistortionCharacteristics(samples []float32, metrics *EnhancedQualityMetrics) {
	// Calculate crest factor
	if metrics.Basic.RMSLevel > 0 {
		metrics.CrestFactor = metrics.Basic.PeakAmplitude / metrics.Basic.RMSLevel
	}

	// Estimate THD (simplified)
	metrics.THDEstimate = qm.estimateTHD(samples, metrics.CrestFactor)

	// Calculate clipping rate
	metrics.ClippingRate = qm.calculateClippingRate(samples)
}

// analyzeVoiceCharacteristics analyzes voice-specific characteristics
func (qm *QualityMeter) analyzeVoiceCharacteristics(metrics *EnhancedQualityMetrics) {
	// Voice probability based on spectral characteristics
	metrics.VoiceProbability = qm.calculateVoiceProbability(metrics)

	// Speech clarity based on SNR and spectral characteristics
	snr := 20.0 * math.Log10(metrics.Basic.RMSLevel/metrics.Basic.NoiseLevel)
	metrics.SpeechClarity = qm.calculateSpeechClarity(snr, metrics.HighFrequencyEnergy, metrics.THDEstimate)

	// Vocal effort based on RMS and spectral centroid
	metrics.VocalEffort = qm.calculateVocalEffort(metrics.Basic.RMSLevel, metrics.SpectralCentroid)
}

// setQualityIndicators sets boolean quality indicators
func (qm *QualityMeter) setQualityIndicators(metrics *EnhancedQualityMetrics) {
	// Clipping detection
	metrics.IsClipped = metrics.ClippingRate > 0.01 // More than 1% clipped

	// Saturation detection
	metrics.IsSaturated = metrics.Basic.PeakAmplitude > 0.95

	// Excessive noise detection
	snr := 20.0 * math.Log10(metrics.Basic.RMSLevel/metrics.Basic.NoiseLevel)
	metrics.HasExcessiveNoise = snr < 10.0 // SNR below 10dB

	// Under-modulation detection
	metrics.IsUnderModulated = metrics.Basic.RMSLevel < 0.01
}

// Enhanced spectral analysis methods

func (qm *QualityMeter) calculateEnhancedSpectralCentroid(powerSpectrum []float64) float64 {
	if len(powerSpectrum) == 0 {
		return 0.0
	}

	var weightedSum, totalPower float64
	freqBinSize := float64(qm.sampleRate) / float64(2*len(powerSpectrum))

	for i, power := range powerSpectrum {
		freq := float64(i) * freqBinSize
		weightedSum += freq * power
		totalPower += power
	}

	if totalPower > 0 {
		return weightedSum / totalPower
	}
	return 0.0
}

func (qm *QualityMeter) calculateSpectralRolloff(powerSpectrum []float64, threshold float64) float64 {
	if len(powerSpectrum) == 0 {
		return 0.0
	}

	var totalPower float64
	for _, power := range powerSpectrum {
		totalPower += power
	}

	targetPower := totalPower * threshold
	var cumulativePower float64
	freqBinSize := float64(qm.sampleRate) / float64(2*len(powerSpectrum))

	for i, power := range powerSpectrum {
		cumulativePower += power
		if cumulativePower >= targetPower {
			return float64(i) * freqBinSize
		}
	}

	return float64(len(powerSpectrum)-1) * freqBinSize
}

func (qm *QualityMeter) calculateSpectralFlatness(powerSpectrum []float64) float64 {
	if len(powerSpectrum) == 0 {
		return 0.0
	}

	var geometricMean, arithmeticMean float64
	validBins := 0

	for _, power := range powerSpectrum {
		if power > 0 {
			geometricMean += math.Log(power)
			arithmeticMean += power
			validBins++
		}
	}

	if validBins == 0 {
		return 0.0
	}

	geometricMean = math.Exp(geometricMean / float64(validBins))
	arithmeticMean = arithmeticMean / float64(validBins)

	if arithmeticMean > 0 {
		return geometricMean / arithmeticMean
	}
	return 0.0
}

func (qm *QualityMeter) calculateHighFrequencyEnergy(powerSpectrum []float64) float64 {
	if len(powerSpectrum) == 0 {
		return 0.0
	}

	freqBinSize := float64(qm.sampleRate) / float64(2*len(powerSpectrum))
	highFreqThreshold := 4000.0 // 4kHz
	startBin := int(highFreqThreshold / freqBinSize)

	if startBin >= len(powerSpectrum) {
		return 0.0
	}

	var highFreqEnergy, totalEnergy float64
	for i, power := range powerSpectrum {
		totalEnergy += power
		if i >= startBin {
			highFreqEnergy += power
		}
	}

	if totalEnergy > 0 {
		return highFreqEnergy / totalEnergy
	}
	return 0.0
}

// Distortion analysis methods

func (qm *QualityMeter) estimateTHD(_ []float32, crestFactor float64) float64 {
	// Simplified THD estimation based on crest factor deviation
	// For clean sine waves, crest factor should be ~1.414 (sqrt(2))
	idealCrestFactor := math.Sqrt(2)
	distortion := math.Abs(crestFactor-idealCrestFactor) / idealCrestFactor

	// Clamp to reasonable range
	if distortion > 1.0 {
		distortion = 1.0
	}

	return distortion
}

func (qm *QualityMeter) calculateClippingRate(samples []float32) float64 {
	if len(samples) == 0 {
		return 0.0
	}

	clippingThreshold := float32(0.99)
	clippedSamples := 0

	for _, sample := range samples {
		if math.Abs(float64(sample)) > float64(clippingThreshold) {
			clippedSamples++
		}
	}

	return float64(clippedSamples) / float64(len(samples))
}

// Voice analysis methods

func (qm *QualityMeter) calculateVoiceProbability(metrics *EnhancedQualityMetrics) float64 {
	score := 0.0

	// Voice typically has spectral centroid in 500-2000 Hz range
	if metrics.SpectralCentroid >= 500 && metrics.SpectralCentroid <= 2000 {
		score += 0.3
	} else if metrics.SpectralCentroid >= 300 && metrics.SpectralCentroid <= 3000 {
		score += 0.2
	}

	// Voice has moderate spectral flatness
	if metrics.SpectralFlatness >= 0.1 && metrics.SpectralFlatness <= 0.6 {
		score += 0.3
	}

	// Voice has some high frequency content but not excessive
	if metrics.HighFrequencyEnergy >= 0.1 && metrics.HighFrequencyEnergy <= 0.4 {
		score += 0.2
	}

	// Voice has reasonable RMS level
	if metrics.Basic.RMSLevel >= 0.01 && metrics.Basic.RMSLevel <= 0.8 {
		score += 0.2
	}

	return score
}

func (qm *QualityMeter) calculateSpeechClarity(snr, highFreqEnergy, thd float64) float64 {
	clarity := 0.0

	// Good SNR contributes to clarity
	if snr > 20 {
		clarity += 0.4
	} else if snr > 15 {
		clarity += 0.3
	} else if snr > 10 {
		clarity += 0.2
	}

	// Adequate high frequency content
	if highFreqEnergy >= 0.15 && highFreqEnergy <= 0.35 {
		clarity += 0.3
	}

	// Low distortion
	if thd < 0.1 {
		clarity += 0.3
	} else if thd < 0.2 {
		clarity += 0.2
	}

	return clarity
}

func (qm *QualityMeter) calculateVocalEffort(rmsLevel, spectralCentroid float64) float64 {
	effort := 0.0

	// Higher RMS indicates more effort
	if rmsLevel > 0.3 {
		effort = 1.0
	} else if rmsLevel > 0.2 {
		effort = 0.8
	} else if rmsLevel > 0.1 {
		effort = 0.6
	} else if rmsLevel > 0.05 {
		effort = 0.4
	} else {
		effort = 0.2
	}

	// Higher spectral centroid can indicate more effort
	if spectralCentroid > 2000 {
		effort *= 1.2
	}

	// Clamp to [0, 1] range
	if effort > 1.0 {
		effort = 1.0
	}

	return effort
}

// GetQualityTrends returns historical quality trends
func (qm *QualityMeter) GetQualityTrends() map[string][]float64 {
	return map[string][]float64{
		"rms":  append([]float64(nil), qm.rmsHistory...),
		"peak": append([]float64(nil), qm.peakHistory...),
		"snr":  append([]float64(nil), qm.snrHistory...),
	}
}
