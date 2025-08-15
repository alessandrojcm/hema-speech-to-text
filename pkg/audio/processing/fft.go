package processing

import (
	"math/cmplx"

	"gonum.org/v1/gonum/dsp/fourier"
	window "gonum.org/v1/gonum/dsp/window"
)

type FFT struct {
	FFTProcessor
	sampleRate int
	window     func([]float64) []float64
}

type FFTWindowFunction struct{}

// NewGonumWindowFunction creates a new Gonum window function processor
func NewFFTWindowFunction() *FFTWindowFunction {
	return &FFTWindowFunction{}
}

// Apply applies a window function to the input samples
func (w *FFTWindowFunction) Apply(samples []float32, windowType string) []float32 {
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
func (w *FFTWindowFunction) GetWindow(size int, windowType string) []float32 {
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

func NewFFT(samplerate int) *FFT {
	return &FFT{
		sampleRate: samplerate,
		window:     window.BartlettHann,
	}
}

func (g *FFT) FFT(samples []float32) []complex128 {
	if len(samples) == 0 {
		return nil
	}
	nfft := g.chooseNFFT(len(samples))

	// copy & cast to float64, zero-padded to nfft
	buf := make([]float64, nfft)
	for i := 0; i < len(samples); i++ {
		buf[i] = float64(samples[i])
	}

	// window in-place
	g.applyWindow(buf)

	// real FFT
	fft := fourier.NewFFT(nfft)
	spec := fft.Coefficients(nil, buf)

	// one-sided
	half := nfft/2 + 1
	out := make([]complex128, half)
	copy(out, spec[:half])
	return out
}

// PowerSpectrum returns a one-sided power spectrum (periodogram-style) with
// window & one-sided scaling applied: |X[k]|^2 / (nfft^2 * U),
// where U = mean(w[n]^2). Interior bins are doubled to conserve power.
func (g *FFT) PowerSpectrum(samples []float32) []float64 {
	if len(samples) == 0 {
		return nil
	}
	nfft := g.chooseNFFT(len(samples))

	// copy & cast
	buf := make([]float64, nfft)
	for i := 0; i < len(samples); i++ {
		buf[i] = float64(samples[i])
	}

	// window, track window power U
	U := g.applyWindow(buf) // mean square of window

	fft := fourier.NewFFT(nfft)
	spec := fft.Coefficients(nil, buf)

	half := nfft/2 + 1
	ps := make([]float64, half)
	for k := 0; k < half; k++ {
		ps[k] = cmplx.Abs(spec[k])
		ps[k] *= ps[k] // |X[k]|^2
	}

	// normalize by nfft^2 and window power
	scale := 1.0 / (float64(nfft) * float64(nfft) * U)
	for k := 0; k < half; k++ {
		ps[k] *= scale
	}

	// one-sided doubling (except DC and Nyquist when present)
	last := half - 1
	nyqExists := (nfft%2 == 0)
	stop := last
	if nyqExists {
		stop = last - 1
	}
	for k := 1; k <= stop; k++ {
		ps[k] *= 2.0
	}

	return ps
}

// SpectralCentroid returns the power-weighted mean frequency in Hz.
func (g *FFT) SpectralCentroid(samples []float32) float64 {
	ps := g.PowerSpectrum(samples)
	if len(ps) == 0 || g.sampleRate <= 0 {
		return 0
	}
	nfft := g.chooseNFFT(len(samples))
	df := float64(g.sampleRate) / float64(nfft)

	var num, den float64
	for k, p := range ps {
		f := float64(k) * df
		num += f * p
		den += p
	}
	if den == 0 {
		return 0
	}
	return num / den
}

// ---- helpers ----

func (g *FFT) chooseNFFT(n int) int {
	// next power of two >= n
	p := 1
	for p < n {
		p <<= 1
	}
	return p
}

// applyWindow applies g.WindowFn (or Hann if nil) and returns U = mean(w^2).
func (g *FFT) applyWindow(buf []float64) float64 {
	newBuf := g.window(buf)
	var sumsq float64
	for _, v := range buf {
		sumsq += v * v
	}
	return sumsq / float64(len(newBuf))
}

// ZeroCrossingRate returns the zero-crossing rate in crossings per second.
// If sampleRate <= 0, it returns the per-sample rate instead.
func zeroCrossingRate(samples []float32, sampleRate float64) float64 {
	n := len(samples)
	if n < 2 {
		return 0
	}

	var crossings int
	prevSign := signNZ(samples[0]) // treat zeros as previous nonzero sign

	for i := 1; i < n; i++ {
		s := signNZ(samples[i])
		if s != 0 && prevSign != 0 && s != prevSign {
			crossings++
		}
		if s != 0 {
			prevSign = s
		}
	}

	perSample := float64(crossings) / float64(n-1)
	if sampleRate > 0 {
		return perSample * sampleRate
	}
	return perSample
}

// ZeroCrossingRateFrame returns ZCR for a single frame (crossings per sample).
func ZeroCrossingRate(frame []float32) float64 {
	return zeroCrossingRate(frame, -1) // per-sample
}

// ZeroCrossingRateOverTime computes ZCR per frame over a long signal,
// using hopLen to step between frames. Returns per-frame ZCR in crossings/second
// if sampleRate > 0, otherwise crossings/sample.
func ZeroCrossingRateOverTime(samples []float32, frameLen, hopLen int, sampleRate float64) []float64 {
	if frameLen <= 1 || hopLen <= 0 || len(samples) < frameLen {
		return nil
	}
	var out []float64
	for start := 0; start+frameLen <= len(samples); start += hopLen {
		z := zeroCrossingRate(samples[start:start+frameLen], sampleRate)
		out = append(out, z)
	}
	return out
}

// signNZ returns -1 for x<0, +1 for x>0, and 0 for x==0.
// ZCR uses a "sticky" sign: zeros inherit the last nonzero sign.
func signNZ(x float32) int8 {
	if x > 0 {
		return 1
	}
	if x < 0 {
		return -1
	}
	return 0
}
