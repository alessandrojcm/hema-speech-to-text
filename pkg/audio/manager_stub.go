//go:build noaudio
// +build noaudio

package audio

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/your-org/hema-replay-system/pkg/audio/processing"
	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

type AudioManager struct{}

func NewAudioManager(config types.AudioConfig, logger zerolog.Logger) (*AudioManager, error) {
	return nil, fmt.Errorf("audio support not available in this build")
}

func (am *AudioManager) Start(ctx context.Context) error {
	return fmt.Errorf("audio support not available in this build")
}

func (am *AudioManager) Stop() error {
	return fmt.Errorf("audio support not available in this build")
}

func (am *AudioManager) ExtractAudio(ctx context.Context, req types.ExtractionRequest) (*types.AudioSegment, error) {
	return nil, fmt.Errorf("audio support not available in this build")
}

func (am *AudioManager) GetHealth() types.SystemHealth {
	return types.SystemHealth{}
}

func (am *AudioManager) GetStats() types.CaptureStats {
	return types.CaptureStats{}
}

func (am *AudioManager) ListDevices() ([]types.DeviceInfo, error) {
	return nil, fmt.Errorf("audio support not available in this build")
}

func (am *AudioManager) GetMetrics() types.SystemMetrics {
	return types.SystemMetrics{}
}

func (am *AudioManager) GetPerformanceStats() map[string]interface{} {
	return map[string]interface{}{}
}

func (am *AudioManager) ExportSegmentToWAV(segment *types.AudioSegment) ([]byte, error) {
	return nil, fmt.Errorf("audio support not available in this build")
}

func (am *AudioManager) ExtractAudioConcurrent(ctx context.Context, requests []types.ExtractionRequest) ([]*types.AudioSegment, []error) {
	errors := make([]error, len(requests))
	for i := range errors {
		errors[i] = fmt.Errorf("audio support not available in this build")
	}
	return nil, errors
}

func (am *AudioManager) ProcessAudioSegment(segment *types.AudioSegment) error {
	return fmt.Errorf("audio support not available in this build")
}

func (am *AudioManager) UpdateConfiguration(config types.AudioConfig) error {
	return fmt.Errorf("audio support not available in this build")
}

// GetProcessor returns the audio processor for direct access to VAD and other processing features
func (am *AudioManager) GetProcessor() *processing.AudioProcessor {
	return nil
}

// GetRecentAudioSamples extracts recent audio samples for real-time analysis
func (am *AudioManager) GetRecentAudioSamples(duration time.Duration) ([]float32, error) {
	return nil, fmt.Errorf("audio support not available in this build")
}
