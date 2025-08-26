//go:build !noaudio
// +build !noaudio

package internal

import (
	"fmt"

	"github.com/gordonklaus/portaudio"
	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

type PortAudioWrapper struct {
	initialized bool
}

func NewPortAudioWrapper() *PortAudioWrapper {
	return &PortAudioWrapper{}
}

func (pa *PortAudioWrapper) Initialize() error {
	if pa.initialized {
		return nil
	}

	if err := portaudio.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize PortAudio: %w", err)
	}

	pa.initialized = true
	return nil
}

func (pa *PortAudioWrapper) Terminate() error {
	if !pa.initialized {
		return nil
	}

	if err := portaudio.Terminate(); err != nil {
		return fmt.Errorf("failed to terminate PortAudio: %w", err)
	}

	pa.initialized = false
	return nil
}

func (pa *PortAudioWrapper) GetDevices() ([]types.DeviceInfo, error) {
	if !pa.initialized {
		return nil, types.ErrInitializationFailed
	}

	devices, err := portaudio.Devices()
	if err != nil {
		return nil, fmt.Errorf("failed to get devices: %w", err)
	}

	var deviceInfos []types.DeviceInfo
	for i, device := range devices {
		deviceInfo := types.DeviceInfo{
			ID:                i,
			Name:              device.Name,
			MaxInputChannels:  device.MaxInputChannels,
			MaxOutputChannels: device.MaxOutputChannels,
			DefaultSampleRate: device.DefaultSampleRate,
			IsDefault:         false,
			IsAvailable:       true,
		}
		deviceInfos = append(deviceInfos, deviceInfo)
	}

	defaultInput, err := portaudio.DefaultInputDevice()
	if err == nil {
		for i := range deviceInfos {
			if deviceInfos[i].Name == defaultInput.Name {
				deviceInfos[i].IsDefault = true
				break
			}
		}
	}

	return deviceInfos, nil
}

func (pa *PortAudioWrapper) FindDevice(name string, id int) (*portaudio.DeviceInfo, error) {
	if !pa.initialized {
		return nil, types.ErrInitializationFailed
	}

	devices, err := portaudio.Devices()
	if err != nil {
		return nil, fmt.Errorf("failed to get devices: %w", err)
	}

	if id >= 0 && id < len(devices) {
		device := devices[id]
		if device.MaxInputChannels > 0 {
			return device, nil
		}
	}

	if name != "" {
		for _, device := range devices {
			if device.Name == name && device.MaxInputChannels > 0 {
				return device, nil
			}
		}
		return nil, types.ErrDeviceNotFound
	}

	defaultDevice, err := portaudio.DefaultInputDevice()
	if err != nil {
		return nil, fmt.Errorf("failed to get default device: %w", err)
	}

	return defaultDevice, nil
}

type AudioStream struct {
	stream *portaudio.Stream
	buffer []float32
}

func (pa *PortAudioWrapper) OpenStream(device *portaudio.DeviceInfo, config types.DeviceConfig) (*AudioStream, error) {
	if !pa.initialized {
		return nil, types.ErrInitializationFailed
	}

	// Validate channel count against device capabilities
	if config.Channels > device.MaxInputChannels {
		return nil, fmt.Errorf("requested %d channels but device '%s' only supports %d input channels",
			config.Channels, device.Name, device.MaxInputChannels)
	}

	parameters := portaudio.StreamParameters{
		Input: portaudio.StreamDeviceParameters{
			Device:   device,
			Channels: config.Channels,
			Latency:  device.DefaultLowInputLatency,
		},
		SampleRate:      float64(config.SampleRate),
		FramesPerBuffer: config.FramesPerBuffer,
	}

	buffer := make([]float32, config.FramesPerBuffer*config.Channels)

	// Create the stream with the input buffer for blocking I/O
	stream, err := portaudio.OpenStream(parameters, buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to open stream: %w", err)
	}

	return &AudioStream{
		stream: stream,
		buffer: buffer,
	}, nil
}

func (as *AudioStream) Start() error {
	if as.stream == nil {
		return types.ErrStreamClosed
	}
	return as.stream.Start()
}

func (as *AudioStream) Stop() error {
	if as.stream == nil {
		return types.ErrStreamClosed
	}
	return as.stream.Stop()
}

func (as *AudioStream) Close() error {
	if as.stream == nil {
		return nil
	}
	err := as.stream.Close()
	as.stream = nil
	return err
}

func (as *AudioStream) Read(buffer []float32) error {
	if as.stream == nil {
		return types.ErrStreamClosed
	}

	err := as.stream.Read()
	if err != nil {
		return fmt.Errorf("failed to read from stream: %w", err)
	}

	copy(buffer, as.buffer)

	// Enhanced audio analysis for debugging speech issues
	var nonZeroCount int
	var maxSample float32
	var rmsSum float32

	for _, sample := range as.buffer {
		rmsSum += sample * sample
		if sample != 0.0 {
			nonZeroCount++
			abs := sample
			if abs < 0 {
				abs = -abs
			}
			if abs > maxSample {
				maxSample = abs
			}
		}
	}

	return nil
}

func (as *AudioStream) IsActive() bool {
	if as.stream == nil {
		return false
	}
	return true
}
