//go:build noaudio
// +build noaudio

package audio

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
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
