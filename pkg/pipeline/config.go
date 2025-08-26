package pipeline

import (
	"fmt"
	"time"

	"github.com/your-org/hema-replay-system/pkg/pipeline/vad"
	speechTypes "github.com/your-org/hema-replay-system/pkg/speech/types"
)

// PipelineManagerConfig holds complete pipeline configuration
type PipelineManagerConfig struct {
	// Component configurations
	Speech speechTypes.SpeechConfig `mapstructure:"speech"`
	VAD    *vad.Config              `mapstructure:"vad"`

	// Pipeline orchestration settings
	Pipeline PipelineConfig `mapstructure:"pipeline"`
}

// PipelineConfig holds pipeline-specific configuration
type PipelineConfig struct {
	// Processing settings
	MaxConcurrentRequests int           `mapstructure:"max_concurrent_requests"`
	ProcessingTimeout     time.Duration `mapstructure:"processing_timeout"`
	SegmentBufferSize     int           `mapstructure:"segment_buffer_size"`

	// Error handling
	MaxRetries      int           `mapstructure:"max_retries"`
	RetryDelay      time.Duration `mapstructure:"retry_delay"`
	FallbackEnabled bool          `mapstructure:"fallback_enabled"`

	// Metadata and monitoring
	ShowMetadata    bool          `mapstructure:"show_metadata"`
	MetricsEnabled  bool          `mapstructure:"metrics_enabled"`
	MetricsInterval time.Duration `mapstructure:"metrics_interval"`

	// State management
	StatePersistence bool   `mapstructure:"state_persistence"`
	StateFile        string `mapstructure:"state_file"`

	// Performance tuning
	HealthCheckInterval time.Duration `mapstructure:"health_check_interval"`
	CleanupInterval     time.Duration `mapstructure:"cleanup_interval"`
}

// Validate validates the pipeline configuration
func (c *PipelineManagerConfig) Validate() error {
	// Validate speech config (basic validation)
	if c.Speech.Whisper.ModelSize.String() == "unknown" {
		return fmt.Errorf("speech model size is invalid")
	}

	// Validate VAD config
	if c.VAD != nil {
		if c.VAD.MinSpeechDurationMs <= 0 {
			return fmt.Errorf("VAD min speech duration must be positive")
		}
		if c.VAD.MaxSilenceDurationMs <= 0 {
			return fmt.Errorf("VAD max silence duration must be positive")
		}
	}

	// Validate pipeline config
	return c.Pipeline.Validate()
}

// Validate validates pipeline-specific configuration
func (pc *PipelineConfig) Validate() error {
	if pc.MaxConcurrentRequests <= 0 {
		return fmt.Errorf("max concurrent requests must be positive")
	}

	if pc.ProcessingTimeout <= 0 {
		return fmt.Errorf("processing timeout must be positive")
	}

	if pc.SegmentBufferSize <= 0 {
		return fmt.Errorf("segment buffer size must be positive")
	}

	if pc.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}

	if pc.RetryDelay < 0 {
		return fmt.Errorf("retry delay cannot be negative")
	}

	if pc.MetricsInterval <= 0 && pc.MetricsEnabled {
		return fmt.Errorf("metrics interval must be positive when metrics are enabled")
	}

	return nil
}

// SetDefaults sets default values for the configuration
func (c *PipelineManagerConfig) SetDefaults() {
	c.Pipeline.SetDefaults()

	// Set default VAD config if not provided
	if c.VAD == nil {
		c.VAD = &vad.Config{
			MinSpeechDurationMs:  500,  // 500ms minimum speech
			MaxSilenceDurationMs: 1000, // 1s max silence
			VADMode:              2,    // Balanced WebRTC VAD mode
			BufferBeforeMs:       100,  // 100ms before speech
			BufferAfterMs:        200,  // 200ms after speech
		}
	}
}

// SetDefaults sets default values for pipeline configuration
func (pc *PipelineConfig) SetDefaults() {
	if pc.MaxConcurrentRequests == 0 {
		pc.MaxConcurrentRequests = 3
	}

	if pc.ProcessingTimeout == 0 {
		pc.ProcessingTimeout = 30 * time.Second
	}

	if pc.SegmentBufferSize == 0 {
		pc.SegmentBufferSize = 50
	}

	if pc.MaxRetries == 0 {
		pc.MaxRetries = 3
	}

	if pc.RetryDelay == 0 {
		pc.RetryDelay = time.Second
	}

	if pc.MetricsInterval == 0 {
		pc.MetricsInterval = 10 * time.Second
	}

	if pc.HealthCheckInterval == 0 {
		pc.HealthCheckInterval = 5 * time.Second
	}

	if pc.CleanupInterval == 0 {
		pc.CleanupInterval = time.Minute
	}

	if pc.StateFile == "" {
		pc.StateFile = "pipeline_state.json"
	}

	// Default to enabled metrics and fallbacks
	if !pc.MetricsEnabled {
		pc.MetricsEnabled = true
	}

	if !pc.FallbackEnabled {
		pc.FallbackEnabled = true
	}
}

// GetPipelineConfig returns pipeline-specific configuration from a general config
func GetPipelineConfig() *PipelineConfig {
	config := &PipelineConfig{}
	config.SetDefaults()
	return config
}
