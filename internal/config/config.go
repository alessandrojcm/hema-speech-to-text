package config

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/your-org/hema-replay-system/pkg/audio/types"

	commentaryTypes "github.com/your-org/hema-replay-system/pkg/commentary/types"
	llm "github.com/your-org/hema-replay-system/pkg/llm/types"
	"github.com/your-org/hema-replay-system/pkg/pipeline"
	speechTypes "github.com/your-org/hema-replay-system/pkg/speech/types"
)

type Config struct {
	OBS            OBSConfig                        `mapstructure:"obs"`
	Replay         ReplayConfig                     `mapstructure:"replay"`
	Text           TextConfig                       `mapstructure:"text"`
	Scene          SceneConfig                      `mapstructure:"scene"`
	Audio          types.AudioConfig                `mapstructure:"audio"`
	Speech         speechTypes.SpeechConfig         `mapstructure:"speech"`
	LLMConfig      llm.LLMConfig                    `mapstructure:"llm"`
	Commentary     commentaryTypes.CommentaryConfig `mapstructure:"commentary"`
	Pipeline       pipeline.PipelineManagerConfig   `mapstructure:"pipeline"`
	Logging        LoggingConfig                    `mapstructure:"logging"`
	OpenAIEndpoint string                           `mapstructure:"openaiendpoint"`
}

type OBSConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
}

type ReplayConfig struct {
	BufferDuration time.Duration `mapstructure:"buffer_duration"`
	PreRollSeconds int           `mapstructure:"pre_roll_seconds"`
	MinInterval    time.Duration `mapstructure:"min_interval"`
	QueueSize      int           `mapstructure:"queue_size"`
}

type TextConfig struct {
	SourceName      string   `mapstructure:"source_name"`
	DefaultMessages []string `mapstructure:"default_messages"`
	MaxLength       int      `mapstructure:"max_length"`
}

type SceneConfig struct {
	MainScene   string `mapstructure:"main_scene"`
	ReplayScene string `mapstructure:"replay_scene"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Configure viper
	v.SetConfigName("settings")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")
	v.AddConfigPath(".")

	if configPath != "" {
		v.SetConfigFile(configPath)
	}

	// Environment variables
	v.SetEnvPrefix("HEMA_REPLAY")
	v.AutomaticEnv()

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		log.Warn().Msg("No config file found, using defaults")
	}

	var config Config

	// Create decoder with custom hooks for speech types
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			stringToModelSizeHookFunc(),
		),
		WeaklyTypedInput: true,
		Result:           &config,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}

	if err := decoder.Decode(v.AllSettings()); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

func setDefaults(v *viper.Viper) {
	// OBS defaults
	v.SetDefault("obs.host", "localhost")
	v.SetDefault("obs.port", 4455)

	// Replay defaults
	v.SetDefault("replay.buffer_duration", "60s")
	v.SetDefault("replay.pre_roll_seconds", 5)
	v.SetDefault("replay.min_interval", "15s")
	v.SetDefault("replay.queue_size", 10)

	// Text defaults
	v.SetDefault("text.source_name", "ReplayText")
	v.SetDefault("text.default_messages", []string{"Point scored!", "Excellent exchange!"})
	v.SetDefault("text.max_length", 100)

	// Scene defaults
	v.SetDefault("scene.main_scene", "Main")
	v.SetDefault("scene.replay_scene", "Replay")

	// Audio defaults
	v.SetDefault("audio.device.name", "")
	v.SetDefault("audio.device.id", -1)
	v.SetDefault("audio.device.sample_rate", 44100)
	v.SetDefault("audio.device.channels", 2)
	v.SetDefault("audio.device.bit_depth", 16)
	v.SetDefault("audio.device.frames_per_buffer", 1024)
	v.SetDefault("audio.device.fallback_devices", []string{"Built-in Microphone", "Default"})
	v.SetDefault("audio.device.monitor_interval", "5s")

	v.SetDefault("audio.buffer.duration", "60s")
	v.SetDefault("audio.buffer.segment_size", "1s")
	v.SetDefault("audio.buffer.overwrite_policy", "circular")
	v.SetDefault("audio.buffer.preallocate_size", 0)

	v.SetDefault("audio.processing.enable_preprocessing", true)
	v.SetDefault("audio.processing.normalization", true)
	v.SetDefault("audio.processing.highpass_filter", 80.0)
	v.SetDefault("audio.processing.lowpass_filter", 8000.0)
	v.SetDefault("audio.processing.vad_threshold", 0.1)

	// Library selection defaults
	v.SetDefault("audio.processing.resampler_type", "gosamplerate")
	v.SetDefault("audio.processing.vad_type", "webrtc")
	v.SetDefault("audio.processing.wav_exporter_type", "goaudio")
	v.SetDefault("audio.processing.fft_type", "gonum")

	// Library-specific settings
	v.SetDefault("audio.processing.resampler_quality", 0) // Best quality
	v.SetDefault("audio.processing.vad_mode", 3)          // Most aggressive

	v.SetDefault("audio.extraction.default_duration", "10s")
	v.SetDefault("audio.extraction.max_concurrent", 5)
	v.SetDefault("audio.extraction.output_format", "wav")
	v.SetDefault("audio.extraction.output_sample_rate", 16000)
	v.SetDefault("audio.extraction.output_channels", 1)
	v.SetDefault("audio.extraction.timestamp_precision", "10ms")

	// Speech defaults
	v.SetDefault("speech.whisper.model_path", "./models/ggml-base.bin")
	v.SetDefault("speech.whisper.model_size", "base")
	v.SetDefault("speech.whisper.language", "en")
	v.SetDefault("speech.whisper.use_gpu", true)
	v.SetDefault("speech.whisper.thread_count", 4)
	v.SetDefault("speech.whisper.max_tokens", 448)
	v.SetDefault("speech.whisper.temperature", 0.0)
	v.SetDefault("speech.whisper.beam_size", 5)
	v.SetDefault("speech.whisper.word_timestamps", true)

	v.SetDefault("speech.processing.target_sample_rate", 16000)
	v.SetDefault("speech.processing.segment_duration", "10s")
	v.SetDefault("speech.processing.overlap_duration", "1s")
	v.SetDefault("speech.processing.normalization", true)
	v.SetDefault("speech.processing.vad_enabled", true)

	v.SetDefault("speech.performance.max_concurrent", 3)
	v.SetDefault("speech.performance.cache_size", 1000)
	v.SetDefault("speech.performance.cache_ttl", "5m")
	v.SetDefault("speech.performance.timeout_duration", "30s")
	v.SetDefault("speech.performance.memory_limit", 1073741824) // 1GB
	v.SetDefault("speech.performance.metal_optimization", true)

	// Pipeline defaults
	v.SetDefault("pipeline.audio.device.sample_rate", 16000)
	v.SetDefault("pipeline.audio.device.channels", 1)
	v.SetDefault("pipeline.audio.device.bit_depth", 16)
	v.SetDefault("pipeline.audio.buffer.duration", "60s")
	v.SetDefault("pipeline.audio.buffer.segment_size", "1s")
	v.SetDefault("pipeline.audio.processing.vad_type", "webrtc")
	v.SetDefault("pipeline.audio.extraction.default_duration", "5s")
	v.SetDefault("pipeline.audio.extraction.max_concurrent", 2)
	v.SetDefault("pipeline.audio.buffer.frames_per_buffer", 1024)

	v.SetDefault("pipeline.speech.whisper.model_size", "base")
	v.SetDefault("pipeline.speech.whisper.language", "en")
	v.SetDefault("pipeline.speech.whisper.thread_count", 2)
	v.SetDefault("pipeline.speech.performance.max_concurrent", 2)
	v.SetDefault("pipeline.speech.performance.cache_size", 10)
	v.SetDefault("pipeline.speech.performance.timeout_duration", "10s")

	v.SetDefault("pipeline.vad.min_speech_duration_ms", 500)
	v.SetDefault("pipeline.vad.max_silence_duration_ms", 1000)
	v.SetDefault("pipeline.vad.vad_mode", 2)
	v.SetDefault("pipeline.vad.buffer_before_ms", 100)
	v.SetDefault("pipeline.vad.buffer_after_ms", 200)

	v.SetDefault("pipeline.pipeline.max_concurrent_requests", 3)
	v.SetDefault("pipeline.pipeline.processing_timeout", "30s")
	v.SetDefault("pipeline.pipeline.segment_buffer_size", 50)
	v.SetDefault("pipeline.pipeline.max_retries", 3)
	v.SetDefault("pipeline.pipeline.retry_delay", "1s")
	v.SetDefault("pipeline.pipeline.fallback_enabled", true)
	v.SetDefault("pipeline.pipeline.metrics_enabled", true)
	v.SetDefault("pipeline.pipeline.metrics_interval", "10s")

	// LLM defaults
	v.SetDefault("openaiendpoint", "http://localhost:8000")

	// Commentary defaults
	v.SetDefault("commentary.default_template", "point_scored")
	v.SetDefault("commentary.max_latency", "2s")
	v.SetDefault("commentary.default_quality", "balanced")
	v.SetDefault("commentary.concurrent_requests", 3)
	v.SetDefault("commentary.min_confidence", 0.6)
	v.SetDefault("commentary.quality_threshold", 0.7)
	v.SetDefault("commentary.relevance_threshold", 0.75)
	v.SetDefault("commentary.enable_fallback", true)
	v.SetDefault("commentary.fallback_threshold", 0.6)
	v.SetDefault("commentary.max_retries", 2)
	v.SetDefault("commentary.max_output_length", 200)
	v.SetDefault("commentary.min_output_length", 10)
	v.SetDefault("commentary.enable_profanity_filter", true)
	v.SetDefault("commentary.enable_cache", true)
	v.SetDefault("commentary.default_cache_policy", "default")
	v.SetDefault("commentary.enable_metrics", true)
	v.SetDefault("commentary.enable_logging", true)
	v.SetDefault("commentary.log_level", "info")

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
}

func validateConfig(config *Config) error {
	if config.OBS.Host == "" {
		return fmt.Errorf("obs.host cannot be empty")
	}
	if config.OBS.Port <= 0 || config.OBS.Port > 65535 {
		return fmt.Errorf("obs.port must be between 1 and 65535")
	}
	if config.Replay.BufferDuration < time.Second {
		return fmt.Errorf("replay.buffer_duration must be at least 1 second")
	}
	if config.Text.MaxLength <= 0 {
		return fmt.Errorf("text.max_length must be positive")
	}
	if config.Scene.MainScene == "" {
		return fmt.Errorf("scene.main_scene cannot be empty")
	}
	if config.Scene.ReplayScene == "" {
		return fmt.Errorf("scene.replay_scene cannot be empty")
	}

	// Validate audio configuration
	if err := config.Audio.Validate(); err != nil {
		return fmt.Errorf("audio configuration invalid: %w", err)
	}

	// Validate speech configuration
	if config.Speech.Whisper.ModelPath == "" {
		return fmt.Errorf("speech.whisper.model_path cannot be empty")
	}
	if config.Speech.Performance.MaxConcurrent <= 0 {
		return fmt.Errorf("speech.performance.max_concurrent must be positive")
	}
	if config.Speech.Performance.TimeoutDuration <= 0 {
		return fmt.Errorf("speech.performance.timeout_duration must be positive")
	}

	// Validate LLM configuration (optional in pipeline mode)
	if config.OpenAIEndpoint == "" {
		config.OpenAIEndpoint = "http://localhost:8000" // Set default if not provided
	}

	// Validate commentary configuration
	if config.Commentary.MaxLatency <= 0 {
		return fmt.Errorf("commentary.max_latency must be positive")
	}

	if config.Commentary.MinConfidence < 0 || config.Commentary.MinConfidence > 1 {
		return fmt.Errorf("commentary.min_confidence must be between 0 and 1")
	}
	if config.Commentary.ConcurrentRequests <= 0 {
		return fmt.Errorf("commentary.concurrent_requests must be positive")
	}

	// Validate pipeline configuration
	if err := config.Pipeline.Validate(); err != nil {
		return fmt.Errorf("pipeline configuration invalid: %w", err)
	}

	return nil
}

// stringToModelSizeHookFunc returns a DecodeHookFunc that converts strings to ModelSize
func stringToModelSizeHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		// Check if we're converting from string to ModelSize
		if f.Kind() != reflect.String {
			return data, nil
		}

		if t != reflect.TypeOf(speechTypes.ModelSize(0)) {
			return data, nil
		}

		// Convert string to ModelSize
		str := data.(string)
		var modelSize speechTypes.ModelSize
		if err := modelSize.UnmarshalText([]byte(str)); err != nil {
			return nil, err
		}

		return modelSize, nil
	}
}
