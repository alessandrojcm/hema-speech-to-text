//go:build !noaudio

package processing

import (
	"fmt"
	"unsafe"

	"github.com/baabaaox/go-webrtcvad"
)

// WebRTCVAD implements VADInterface using WebRTC VAD
type WebRTCVAD struct {
	detector   webrtcvad.VadInst
	mode       int
	sampleRate int
}

// NewWebRTCVAD creates a new WebRTC VAD detector
func NewWebRTCVAD(sampleRate int, mode int) (*WebRTCVAD, error) {
	// Validate sample rate (WebRTC VAD supports 8000, 16000, 32000, 48000 Hz)
	if sampleRate != 8000 && sampleRate != 16000 && sampleRate != 32000 && sampleRate != 48000 {
		return nil, fmt.Errorf("unsupported sample rate: %d (supported: 8000, 16000, 32000, 48000)", sampleRate)
	}

	// Validate mode (0-3, where 3 is most aggressive)
	if mode < 0 || mode > 3 {
		mode = 3 // Default to most aggressive
	}

	// Create VAD detector
	vad := webrtcvad.Create()
	if vad == nil {
		return nil, fmt.Errorf("failed to create WebRTC VAD")
	}

	// Initialize VAD
	if err := webrtcvad.Init(vad); err != nil {
		webrtcvad.Free(vad)
		return nil, fmt.Errorf("failed to initialize WebRTC VAD: %w", err)
	}

	// Set mode
	if err := webrtcvad.SetMode(vad, mode); err != nil {
		webrtcvad.Free(vad)
		return nil, fmt.Errorf("failed to set VAD mode: %w", err)
	}

	return &WebRTCVAD{
		detector:   vad,
		mode:       mode,
		sampleRate: sampleRate,
	}, nil
}

// DetectVoice detects voice activity in the given samples
func (v *WebRTCVAD) DetectVoice(samples []float32) bool {
	if v.detector == nil || len(samples) == 0 {
		return false
	}

	// WebRTC VAD requires specific frame sizes based on sample rate:
	// - 8000 Hz: 80, 160, or 240 samples (10ms, 20ms, 30ms)
	// - 16000 Hz: 160, 320, or 480 samples (10ms, 20ms, 30ms)
	// - 32000 Hz: 320, 640, or 960 samples (10ms, 20ms, 30ms)
	// - 48000 Hz: 480, 960, or 1440 samples (10ms, 20ms, 30ms)

	// Calculate frame size for 20ms (most common)
	frameSize := v.sampleRate / 50 // 20ms frame

	// If input is smaller than frame size, pad with zeros
	if len(samples) < frameSize {
		paddedSamples := make([]float32, frameSize)
		copy(paddedSamples, samples)
		samples = paddedSamples
	}

	// Process in chunks of frameSize
	voiceDetected := false
	for i := 0; i < len(samples); i += frameSize {
		end := i + frameSize
		if end > len(samples) {
			end = len(samples)
		}

		chunk := samples[i:end]
		if len(chunk) < frameSize {
			// Pad the last chunk if necessary
			paddedChunk := make([]float32, frameSize)
			copy(paddedChunk, chunk)
			chunk = paddedChunk
		}

		// Convert float32 to int16 for WebRTC VAD
		int16Samples := make([]int16, len(chunk))
		for j, sample := range chunk {
			// Clamp sample to [-1.0, 1.0] range
			if sample > 1.0 {
				sample = 1.0
			} else if sample < -1.0 {
				sample = -1.0
			}
			// Convert to int16 range
			int16Samples[j] = int16(sample * 32767)
		}

		// Convert int16 samples to bytes
		audioBytes := (*[2]byte)(unsafe.Pointer(&int16Samples[0]))[:len(int16Samples)*2]

		// Process with WebRTC VAD
		active, err := webrtcvad.Process(v.detector, v.sampleRate, audioBytes, len(int16Samples))
		if err != nil {
			// If processing fails, fall back to simple energy-based detection
			return v.fallbackDetection(chunk)
		}

		if active {
			voiceDetected = true
			break // Early exit if voice is detected in any chunk
		}
	}

	return voiceDetected
}

// fallbackDetection provides a simple energy-based fallback when WebRTC VAD fails
func (v *WebRTCVAD) fallbackDetection(samples []float32) bool {
	if len(samples) == 0 {
		return false
	}

	// Calculate RMS energy
	var sum float64
	for _, sample := range samples {
		sum += float64(sample * sample)
	}
	rms := sum / float64(len(samples))

	// Simple threshold-based detection
	const threshold = 0.01
	return rms > threshold
}

// SetSensitivity sets the VAD sensitivity (mode)
func (v *WebRTCVAD) SetSensitivity(level float64) error {
	if v.detector == nil {
		return fmt.Errorf("VAD detector not initialized")
	}

	// Convert level (0.0-1.0) to mode (0-3)
	mode := int(level * 3)
	if mode < 0 {
		mode = 0
	} else if mode > 3 {
		mode = 3
	}

	if err := webrtcvad.SetMode(v.detector, mode); err != nil {
		return fmt.Errorf("failed to set VAD mode: %w", err)
	}

	v.mode = mode
	return nil
}

// GetMode returns the current VAD mode
func (v *WebRTCVAD) GetMode() int {
	return v.mode
}

// GetSampleRate returns the configured sample rate
func (v *WebRTCVAD) GetSampleRate() int {
	return v.sampleRate
}

// Close releases resources used by the VAD detector
func (v *WebRTCVAD) Close() error {
	if v.detector != nil {
		webrtcvad.Free(v.detector)
		v.detector = nil
	}
	return nil
}

// IsValidFrameLength checks if the given frame length is valid for WebRTC VAD
func (v *WebRTCVAD) IsValidFrameLength(frameLength int) bool {
	switch v.sampleRate {
	case 8000:
		return frameLength == 80 || frameLength == 160 || frameLength == 240
	case 16000:
		return frameLength == 160 || frameLength == 320 || frameLength == 480
	case 32000:
		return frameLength == 320 || frameLength == 640 || frameLength == 960
	case 48000:
		return frameLength == 480 || frameLength == 960 || frameLength == 1440
	default:
		return false
	}
}

// GetValidFrameLengths returns valid frame lengths for the current sample rate
func (v *WebRTCVAD) GetValidFrameLengths() []int {
	switch v.sampleRate {
	case 8000:
		return []int{80, 160, 240} // 10ms, 20ms, 30ms
	case 16000:
		return []int{160, 320, 480} // 10ms, 20ms, 30ms
	case 32000:
		return []int{320, 640, 960} // 10ms, 20ms, 30ms
	case 48000:
		return []int{480, 960, 1440} // 10ms, 20ms, 30ms
	default:
		return nil
	}
}

// ProcessFrame processes a single frame of the correct size
func (v *WebRTCVAD) ProcessFrame(samples []float32) (bool, error) {
	if v.detector == nil {
		return false, fmt.Errorf("VAD detector not initialized")
	}

	if !v.IsValidFrameLength(len(samples)) {
		return false, fmt.Errorf("invalid frame length: %d", len(samples))
	}

	// Convert float32 to int16
	int16Samples := make([]int16, len(samples))
	for i, sample := range samples {
		// Clamp sample to [-1.0, 1.0] range
		if sample > 1.0 {
			sample = 1.0
		} else if sample < -1.0 {
			sample = -1.0
		}
		// Convert to int16 range
		int16Samples[i] = int16(sample * 32767)
	}

	// Convert int16 samples to bytes
	audioBytes := (*[2]byte)(unsafe.Pointer(&int16Samples[0]))[:len(int16Samples)*2]

	return webrtcvad.Process(v.detector, v.sampleRate, audioBytes, len(int16Samples))
}

// ThresholdVAD provides a simple threshold-based VAD as fallback
type ThresholdVAD struct {
	threshold float64
}

// NewThresholdVAD creates a new threshold-based VAD detector
func NewThresholdVAD(threshold float64) *ThresholdVAD {
	if threshold <= 0 {
		threshold = 0.01 // Default threshold
	}
	return &ThresholdVAD{
		threshold: threshold,
	}
}

// DetectVoice detects voice activity using simple energy threshold
func (t *ThresholdVAD) DetectVoice(samples []float32) bool {
	if len(samples) == 0 {
		return false
	}

	// Calculate RMS energy
	var sum float64
	for _, sample := range samples {
		sum += float64(sample * sample)
	}
	rms := sum / float64(len(samples))

	return rms > t.threshold
}

// SetSensitivity sets the detection threshold
func (t *ThresholdVAD) SetSensitivity(level float64) error {
	if level < 0 || level > 1 {
		return fmt.Errorf("sensitivity level must be between 0 and 1")
	}
	// Convert level to threshold (inverse relationship)
	t.threshold = (1.0 - level) * 0.1 // Range from 0.1 (low sensitivity) to 0.0 (high sensitivity)
	if t.threshold < 0.001 {
		t.threshold = 0.001 // Minimum threshold
	}
	return nil
}

// Close is a no-op for threshold VAD
func (t *ThresholdVAD) Close() error {
	return nil
}

// GetThreshold returns the current threshold
func (t *ThresholdVAD) GetThreshold() float64 {
	return t.threshold
}
