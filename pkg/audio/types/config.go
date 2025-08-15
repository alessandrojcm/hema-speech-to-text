package types

import "time"

type AudioConfig struct {
	Device     DeviceConfig     `mapstructure:"device"`
	Buffer     BufferConfig     `mapstructure:"buffer"`
	Processing ProcessingConfig `mapstructure:"processing"`
	Extraction ExtractionConfig `mapstructure:"extraction"`
}

type DeviceConfig struct {
	Name            string        `mapstructure:"name"`
	ID              int           `mapstructure:"id"`
	SampleRate      int           `mapstructure:"sample_rate"`
	Channels        int           `mapstructure:"channels"`
	BitDepth        int           `mapstructure:"bit_depth"`
	FramesPerBuffer int           `mapstructure:"frames_per_buffer"`
	FallbackDevices []string      `mapstructure:"fallback_devices"`
	MonitorInterval time.Duration `mapstructure:"monitor_interval"`
}

type BufferConfig struct {
	Duration        time.Duration `mapstructure:"duration"`
	SegmentSize     time.Duration `mapstructure:"segment_size"`
	OverwritePolicy string        `mapstructure:"overwrite_policy"`
	PreallocateSize int           `mapstructure:"preallocate_size"`
}

type ProcessingConfig struct {
	EnablePreprocessing bool    `mapstructure:"enable_preprocessing"`
	Normalization       bool    `mapstructure:"normalization"`
	HighpassFilter      float64 `mapstructure:"highpass_filter"`
	LowpassFilter       float64 `mapstructure:"lowpass_filter"`
	VADThreshold        float64 `mapstructure:"vad_threshold"`

	// Library selection
	ResamplerType   string `mapstructure:"resampler_type"`    // "gosamplerate", "custom"
	VADType         string `mapstructure:"vad_type"`          // "webrtc", "threshold"
	WAVExporterType string `mapstructure:"wav_exporter_type"` // "goaudio", "custom"

	// Library-specific settings
	ResamplerQuality int `mapstructure:"resampler_quality"` // gosamplerate quality level (0-4)
	VADMode          int `mapstructure:"vad_mode"`          // WebRTC VAD aggressiveness (0-3)
}

type ExtractionConfig struct {
	DefaultDuration    time.Duration `mapstructure:"default_duration"`
	MaxConcurrent      int           `mapstructure:"max_concurrent"`
	OutputFormat       string        `mapstructure:"output_format"`
	OutputSampleRate   int           `mapstructure:"output_sample_rate"`
	OutputChannels     int           `mapstructure:"output_channels"`
	TimestampPrecision time.Duration `mapstructure:"timestamp_precision"`
}

func DefaultAudioConfig() AudioConfig {
	return AudioConfig{
		Device: DeviceConfig{
			Name:            "",
			ID:              -1,
			SampleRate:      44100,
			Channels:        2,
			BitDepth:        16,
			FramesPerBuffer: 1024,
			FallbackDevices: []string{"Built-in Microphone", "Default"},
			MonitorInterval: 5 * time.Second,
		},
		Buffer: BufferConfig{
			Duration:        60 * time.Second,
			SegmentSize:     1 * time.Second,
			OverwritePolicy: "circular",
			PreallocateSize: 0,
		},
		Processing: ProcessingConfig{
			EnablePreprocessing: true, Normalization: true,
			HighpassFilter: 80.0,
			LowpassFilter:  8000.0,
			VADThreshold:   0.1,

			// Library selection defaults
			ResamplerType:   "gosamplerate",
			VADType:         "webrtc",
			WAVExporterType: "goaudio",

			// Library-specific settings
			ResamplerQuality: 0, // Best quality
			VADMode:          3, // Most aggressive
		},
		Extraction: ExtractionConfig{
			DefaultDuration:    10 * time.Second,
			MaxConcurrent:      5,
			OutputFormat:       "wav",
			OutputSampleRate:   16000,
			OutputChannels:     1,
			TimestampPrecision: 10 * time.Millisecond,
		},
	}
}

func (c *AudioConfig) Validate() error {
	if c.Device.SampleRate <= 0 {
		return ErrInvalidConfig
	}
	if c.Device.Channels <= 0 {
		return ErrInvalidConfig
	}
	if c.Device.BitDepth != 16 && c.Device.BitDepth != 24 && c.Device.BitDepth != 32 {
		return ErrInvalidConfig
	}
	if c.Device.FramesPerBuffer <= 0 {
		return ErrInvalidConfig
	}
	if c.Buffer.Duration <= 0 {
		return ErrInvalidConfig
	}
	if c.Buffer.SegmentSize <= 0 {
		return ErrInvalidConfig
	}
	if c.Extraction.MaxConcurrent <= 0 {
		return ErrInvalidConfig
	}
	return nil
}
