//go:build noaudio

package processing

import "fmt"

// WebRTCVAD stub implementation for noaudio builds
type WebRTCVAD struct{}

// NewWebRTCVAD creates a stub WebRTC VAD detector
func NewWebRTCVAD(sampleRate int, mode int) (*WebRTCVAD, error) {
	return nil, fmt.Errorf("WebRTC VAD not available in noaudio build")
}

// DetectVoice stub implementation
func (v *WebRTCVAD) DetectVoice(samples []float32) bool {
	return false
}

// SetSensitivity stub implementation
func (v *WebRTCVAD) SetSensitivity(level float64) error {
	return fmt.Errorf("WebRTC VAD not available in noaudio build")
}

// GetMode stub implementation
func (v *WebRTCVAD) GetMode() int {
	return 0
}

// GetSampleRate stub implementation
func (v *WebRTCVAD) GetSampleRate() int {
	return 0
}

// Close stub implementation
func (v *WebRTCVAD) Close() error {
	return nil
}

// IsValidFrameLength stub implementation
func (v *WebRTCVAD) IsValidFrameLength(frameLength int) bool {
	return false
}

// GetValidFrameLengths stub implementation
func (v *WebRTCVAD) GetValidFrameLengths() []int {
	return nil
}

// ProcessFrame stub implementation
func (v *WebRTCVAD) ProcessFrame(samples []float32) (bool, error) {
	return false, fmt.Errorf("WebRTC VAD not available in noaudio build")
}

// ThresholdVAD is available in both builds
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
