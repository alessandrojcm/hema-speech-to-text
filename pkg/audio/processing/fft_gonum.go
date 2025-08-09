package processing

import (
	"math"

	"gonum.org/v1/gonum/dsp/fourier"
	"gonum.org/v1/gonum/dsp/window"
)

// GonumFFTProcessor implements FFTProcessor using Gonum DSP
type GonumFFTProcessor struct {
	fft *fourier.FFT
}

// NewGonumFFTProcessor creates a new Gonum FFT processor
func NewGonumFFTProcessor() *GonumFFTProcessor {
	return &GonumFFTProcessor{}
}

// FFT computes the Fast Fourier Transform of the input samples
func (f *GonumFFTProcessor) FFT(samples []float32) []complex128 {
	if len(samples) == 0 {
		return nil
	}

	// Convert float32 to float64
	float64Samples := make([]float64, len(samples))
	for i, sample := range samples {
		float64Samples[i] = float64(sample)
	}

	// Create FFT if not exists or size changed
	if f.fft == nil || f.fft.Len() != len(samples) {
		f.fft = fourier.NewFFT(len(samples))
	}

	// Compute FFT
	result := make([]complex128, len(samples)/2+1)
	f.fft.Coefficients(result, float64Samples)

	// Expand to full spectrum for compatibility
	fullResult := make([]complex128, len(samples))
	copy(fullResult, result)

	// Mirror the negative frequencies (conjugate symmetry for real input)
	for i := 1; i < len(result)-1; i++ {
		fullResult[len(samples)-i] = complex(real(result[i]), -imag(result[i]))
	}

	return fullResult
}

// PowerSpectrum computes the power spectrum of the input samples
func (f *GonumFFTProcessor) PowerSpectrum(samples []float32) []float64 {
	if len(samples) == 0 {
		return nil
	}

	// Get FFT coefficients
	fftResult := f.FFT(samples)
	if fftResult == nil {
		return nil
	}

	// Compute power spectrum (magnitude squared)
	powerSpectrum := make([]float64, len(fftResult)/2+1) // Only positive frequencies
	for i := 0; i < len(powerSpectrum); i++ {
		magnitude := real(fftResult[i])*real(fftResult[i]) + imag(fftResult[i])*imag(fftResult[i])
		powerSpectrum[i] = magnitude
	}

	return powerSpectrum
}

// SpectralCentroid computes the spectral centroid of the input samples
func (f *GonumFFTProcessor) SpectralCentroid(samples []float32) float64 {
	if len(samples) == 0 {
		return 0.0
	}

	powerSpectrum := f.PowerSpectrum(samples)
	if powerSpectrum == nil || len(powerSpectrum) == 0 {
		return 0.0
	}

	// Calculate spectral centroid
	var weightedSum, totalPower float64
	for i, power := range powerSpectrum {
		frequency := float64(i) // Frequency bin index
		weightedSum += frequency * power
		totalPower += power
	}

	if totalPower == 0 {
		return 0.0
	}

	return weightedSum / totalPower
}

// GonumWindowFunction implements WindowFunction using Gonum DSP
type GonumWindowFunction struct{}

// NewGonumWindowFunction creates a new Gonum window function processor
func NewGonumWindowFunction() *GonumWindowFunction {
	return &GonumWindowFunction{}
}

// Apply applies a window function to the input samples
func (w *GonumWindowFunction) Apply(samples []float32, windowType string) []float32 {
	if len(samples) == 0 {
		return samples
	}

	windowCoeffs := w.GetWindow(len(samples), windowType)
	if windowCoeffs == nil {
		return samples
	}

	result := make([]float32, len(samples))
	for i, sample := range samples {
		result[i] = sample * windowCoeffs[i]
	}

	return result
}

// GetWindow generates window coefficients for the specified size and type
func (w *GonumWindowFunction) GetWindow(size int, windowType string) []float32 {
	if size <= 0 {
		return nil
	}

	// Create a slice filled with ones (Gonum window functions modify in-place)
	coeffs := make([]float64, size)
	for i := range coeffs {
		coeffs[i] = 1.0
	}

	switch windowType {
	case "hann", "hanning":
		coeffs = window.Hann(coeffs)
	case "hamming":
		coeffs = window.Hamming(coeffs)
	case "blackman":
		coeffs = window.Blackman(coeffs)
	case "bartlett", "triangular":
		coeffs = window.Triangular(coeffs)
	case "rectangular", "rect":
		coeffs = window.Rectangular(coeffs)
	case "blackman-harris":
		coeffs = window.BlackmanHarris(coeffs)
	case "blackman-nuttall":
		coeffs = window.BlackmanNuttall(coeffs)
	case "flattop":
		coeffs = window.FlatTop(coeffs)
	case "lanczos":
		coeffs = window.Lanczos(coeffs)
	case "nuttall":
		coeffs = window.Nuttall(coeffs)
	case "sine":
		coeffs = window.Sine(coeffs)
	default:
		// Default to Hann window
		coeffs = window.Hann(coeffs)
	}

	// Convert to float32
	result := make([]float32, size)
	for i, coeff := range coeffs {
		result[i] = float32(coeff)
	}

	return result
}

// Additional utility functions for spectral analysis

// SpectralRolloff computes the spectral rolloff frequency (frequency below which 85% of energy is contained)
func (f *GonumFFTProcessor) SpectralRolloff(samples []float32, sampleRate int) float64 {
	powerSpectrum := f.PowerSpectrum(samples)
	if powerSpectrum == nil || len(powerSpectrum) == 0 {
		return 0.0
	}

	// Calculate total energy
	var totalEnergy float64
	for _, power := range powerSpectrum {
		totalEnergy += power
	}

	if totalEnergy == 0 {
		return 0.0
	}

	// Find frequency where 85% of energy is contained
	threshold := 0.85 * totalEnergy
	var cumulativeEnergy float64

	for i, power := range powerSpectrum {
		cumulativeEnergy += power
		if cumulativeEnergy >= threshold {
			// Convert bin index to frequency
			frequency := float64(i) * float64(sampleRate) / float64(len(powerSpectrum)*2-2)
			return frequency
		}
	}

	// If we reach here, return Nyquist frequency
	return float64(sampleRate) / 2.0
}

// SpectralFlux computes the spectral flux (measure of how quickly the power spectrum changes)
func (f *GonumFFTProcessor) SpectralFlux(currentSamples, previousSamples []float32) float64 {
	if len(currentSamples) == 0 || len(previousSamples) == 0 {
		return 0.0
	}

	currentSpectrum := f.PowerSpectrum(currentSamples)
	previousSpectrum := f.PowerSpectrum(previousSamples)

	if currentSpectrum == nil || previousSpectrum == nil {
		return 0.0
	}

	minLen := len(currentSpectrum)
	if len(previousSpectrum) < minLen {
		minLen = len(previousSpectrum)
	}

	var flux float64
	for i := 0; i < minLen; i++ {
		diff := currentSpectrum[i] - previousSpectrum[i]
		if diff > 0 {
			flux += diff
		}
	}

	return flux
}

// ZeroCrossingRate computes the zero crossing rate of the signal
func ZeroCrossingRate(samples []float32) float64 {
	if len(samples) <= 1 {
		return 0.0
	}

	crossings := 0
	for i := 1; i < len(samples); i++ {
		if (samples[i-1] >= 0 && samples[i] < 0) || (samples[i-1] < 0 && samples[i] >= 0) {
			crossings++
		}
	}

	return float64(crossings) / float64(len(samples)-1)
}

// MelFilterBank applies a mel-scale filter bank to the power spectrum
func (f *GonumFFTProcessor) MelFilterBank(powerSpectrum []float64, sampleRate int, numFilters int) []float64 {
	if len(powerSpectrum) == 0 || numFilters <= 0 {
		return nil
	}

	// Convert Hz to Mel scale
	hzToMel := func(hz float64) float64 {
		return 2595.0 * math.Log10(1.0+hz/700.0)
	}

	// Convert Mel to Hz scale
	melToHz := func(mel float64) float64 {
		return 700.0 * (math.Pow(10.0, mel/2595.0) - 1.0)
	}

	// Define frequency range
	lowFreq := 0.0
	highFreq := float64(sampleRate) / 2.0

	// Convert to mel scale
	lowMel := hzToMel(lowFreq)
	highMel := hzToMel(highFreq)

	// Create mel points
	melPoints := make([]float64, numFilters+2)
	melStep := (highMel - lowMel) / float64(numFilters+1)
	for i := range melPoints {
		melPoints[i] = lowMel + float64(i)*melStep
	}

	// Convert back to Hz
	hzPoints := make([]float64, len(melPoints))
	for i, mel := range melPoints {
		hzPoints[i] = melToHz(mel)
	}

	// Convert Hz to FFT bin indices
	binPoints := make([]int, len(hzPoints))
	for i, hz := range hzPoints {
		binPoints[i] = int(math.Floor(hz * float64(len(powerSpectrum)*2-2) / float64(sampleRate)))
		if binPoints[i] >= len(powerSpectrum) {
			binPoints[i] = len(powerSpectrum) - 1
		}
	}

	// Apply triangular filters
	melFeatures := make([]float64, numFilters)
	for i := 0; i < numFilters; i++ {
		leftBin := binPoints[i]
		centerBin := binPoints[i+1]
		rightBin := binPoints[i+2]

		var sum float64
		// Left slope
		for j := leftBin; j < centerBin; j++ {
			if j < len(powerSpectrum) {
				weight := float64(j-leftBin) / float64(centerBin-leftBin)
				sum += powerSpectrum[j] * weight
			}
		}
		// Right slope
		for j := centerBin; j < rightBin; j++ {
			if j < len(powerSpectrum) {
				weight := float64(rightBin-j) / float64(rightBin-centerBin)
				sum += powerSpectrum[j] * weight
			}
		}

		melFeatures[i] = sum
	}

	return melFeatures
}
