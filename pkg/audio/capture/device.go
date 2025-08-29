//go:build !noaudio
// +build !noaudio

package capture

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gordonklaus/portaudio"
	"github.com/rs/zerolog"
	"github.com/your-org/hema-replay-system/pkg/audio/internal"
	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

type DeviceManager struct {
	mu            sync.RWMutex
	config        types.DeviceConfig
	portAudio     *internal.PortAudioWrapper
	currentDevice *portaudio.DeviceInfo
	devices       []types.DeviceInfo
	health        types.DeviceHealth
	logger        zerolog.Logger
	stopChan      chan struct{}
	running       bool
}

func NewDeviceManager(config types.DeviceConfig, logger zerolog.Logger) *DeviceManager {
	return &DeviceManager{
		config:    config,
		portAudio: internal.NewPortAudioWrapper(),
		logger:    logger.With().Str("component", "device_manager").Logger(),
		stopChan:  make(chan struct{}),
		health: types.DeviceHealth{
			IsConnected:   false,
			SignalLevel:   0.0,
			NoiseFloor:    0.0,
			LastHeartbeat: time.Now(),
			ErrorCount:    0,
			WarningCount:  0,
		},
	}
}

func (dm *DeviceManager) Start(ctx context.Context) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dm.running {
		return types.ErrAlreadyRunning
	}

	if err := dm.portAudio.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize PortAudio: %w", err)
	}

	if err := dm.RefreshDevices(); err != nil {
		return fmt.Errorf("failed to refresh devices: %w", err)
	}

	device, err := dm.findBestDevice()
	if err != nil {
		return fmt.Errorf("failed to find suitable device: %w", err)
	}

	dm.currentDevice = device
	dm.health.IsConnected = true
	dm.health.LastHeartbeat = time.Now()
	dm.running = true

	go dm.monitoringLoop(ctx)

	dm.logger.Info().
		Str("device_name", device.Name).
		Int("max_input_channels", device.MaxInputChannels).
		Float64("default_sample_rate", device.DefaultSampleRate).
		Msg("Device manager started")

	return nil
}

func (dm *DeviceManager) Stop() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if !dm.running {
		return nil
	}

	close(dm.stopChan)
	dm.running = false

	if err := dm.portAudio.Terminate(); err != nil {
		dm.logger.Error().Err(err).Msg("Failed to terminate PortAudio")
		return err
	}

	dm.logger.Info().Msg("Device manager stopped")
	return nil
}

func (dm *DeviceManager) GetCurrentDevice() *portaudio.DeviceInfo {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.currentDevice
}

func (dm *DeviceManager) GetHealth() types.DeviceHealth {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.health
}

func (dm *DeviceManager) GetDevices() []types.DeviceInfo {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	dm.RefreshDevices()
	return dm.devices
}

// RefreshDevices updates the list of available audio devices
func (dm *DeviceManager) RefreshDevices() error {
	devices, err := dm.portAudio.GetDevices()
	if err != nil {
		return err
	}
	dm.devices = devices
	return nil
}

func (dm *DeviceManager) findBestDevice() (*portaudio.DeviceInfo, error) {
	device, err := dm.portAudio.FindDevice(dm.config.Name, dm.config.ID)
	if err == nil {
		return device, nil
	}

	dm.logger.Warn().
		Str("preferred_device", dm.config.Name).
		Int("preferred_id", dm.config.ID).
		Err(err).
		Msg("Preferred device not found, trying fallbacks")

	for _, fallbackName := range dm.config.FallbackDevices {
		device, err := dm.portAudio.FindDevice(fallbackName, -1)
		if err == nil {
			dm.logger.Info().
				Str("fallback_device", fallbackName).
				Msg("Using fallback device")
			return device, nil
		}
	}

	return nil, types.ErrDeviceNotFound
}

func (dm *DeviceManager) monitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(dm.config.MonitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-dm.stopChan:
			return
		case <-ticker.C:
			dm.performHealthCheck()
		}
	}
}

func (dm *DeviceManager) performHealthCheck() {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dm.currentDevice == nil {
		dm.health.IsConnected = false
		dm.health.ErrorCount++
		return
	}

	if err := dm.RefreshDevices(); err != nil {
		dm.logger.Warn().Err(err).Msg("Failed to refresh devices during health check")
		dm.health.WarningCount++
		return
	}

	deviceFound := false
	for _, device := range dm.devices {
		if device.Name == dm.currentDevice.Name {
			deviceFound = true
			dm.health.IsConnected = device.IsAvailable
			break
		}
	}

	if !deviceFound {
		dm.health.IsConnected = false
		dm.health.ErrorCount++
		dm.logger.Error().
			Str("device_name", dm.currentDevice.Name).
			Msg("Current device no longer available")
	} else {
		dm.health.LastHeartbeat = time.Now()
	}
}
