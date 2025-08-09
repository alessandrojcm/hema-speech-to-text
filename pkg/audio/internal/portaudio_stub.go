//go:build noaudio
// +build noaudio

package internal

import (
	"fmt"

	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

type PortAudioWrapper struct {
	initialized bool
}

func NewPortAudioWrapper() *PortAudioWrapper {
	return &PortAudioWrapper{}
}

func (pa *PortAudioWrapper) Initialize() error {
	return fmt.Errorf("PortAudio not available in this build")
}

func (pa *PortAudioWrapper) Terminate() error {
	return nil
}

func (pa *PortAudioWrapper) GetDevices() ([]types.DeviceInfo, error) {
	return nil, fmt.Errorf("PortAudio not available in this build")
}

func (pa *PortAudioWrapper) FindDevice(name string, id int) (interface{}, error) {
	return nil, fmt.Errorf("PortAudio not available in this build")
}

func (pa *PortAudioWrapper) OpenStream(device interface{}, config types.DeviceConfig) (*AudioStream, error) {
	return nil, fmt.Errorf("PortAudio not available in this build")
}

type AudioStream struct{}

func (as *AudioStream) Start() error {
	return fmt.Errorf("PortAudio not available in this build")
}

func (as *AudioStream) Stop() error {
	return fmt.Errorf("PortAudio not available in this build")
}

func (as *AudioStream) Close() error {
	return nil
}

func (as *AudioStream) Read(buffer []float32) error {
	return fmt.Errorf("PortAudio not available in this build")
}

func (as *AudioStream) IsActive() bool {
	return false
}
