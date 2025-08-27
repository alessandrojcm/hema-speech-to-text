package types

import "time"

// SpeechConfig represents speech recognition configuration
type SpeechConfig struct {
	Whisper     WhisperConfig     `mapstructure:"whisper"`
	Processing  ProcessingConfig  `mapstructure:"processing"`
	Performance PerformanceConfig `mapstructure:"performance"`
}

// WhisperConfig contains whisper.cpp specific configuration
type WhisperConfig struct {
	ModelPath      string    `mapstructure:"model_path"`
	ModelSize      ModelSize `mapstructure:"model_size"`
	Language       string    `mapstructure:"language"`
	UseGPU         bool      `mapstructure:"use_gpu"`
	ThreadCount    int       `mapstructure:"thread_count"`
	MaxTokens      int       `mapstructure:"max_tokens"`
	Temperature    float32   `mapstructure:"temperature"`
	BeamSize       int       `mapstructure:"beam_size"`
	WordTimestamps bool      `mapstructure:"word_timestamps"`

	// Noise suppression
	SuppressBlank     bool   `mapstructure:"suppress_blank"`
	SuppressNonSpeech bool   `mapstructure:"suppress_non_speech"`
	SuppressRegex     string `mapstructure:"suppress_regex"`
	InitialPrompt     string `mapstructure:"initial_prompt"`

	// Quality thresholds
	NoSpeechThreshold float32 `mapstructure:"no_speech_threshold"`
	LogProbThreshold  float32 `mapstructure:"logprob_threshold"`

	// Token filtering
	MinTokenConfidence float32 `mapstructure:"min_token_confidence"`
}

// ProcessingConfig contains audio processing configuration for speech
type ProcessingConfig struct {
	TargetSampleRate int           `mapstructure:"target_sample_rate"`
	SegmentDuration  time.Duration `mapstructure:"segment_duration"`
	OverlapDuration  time.Duration `mapstructure:"overlap_duration"`
	NoiseReduction   bool          `mapstructure:"noise_reduction"`
	Normalization    bool          `mapstructure:"normalization"`
	VADEnabled       bool          `mapstructure:"vad_enabled"`

	// Quality filtering parameters
	QualityEnabled bool    `mapstructure:"quality_enabled"`
	MinEnergy      float32 `mapstructure:"min_energy"`
	MinSNR         float32 `mapstructure:"min_snr"`
	MinVoiceRatio  float32 `mapstructure:"min_voice_ratio"`
}

// PerformanceConfig contains performance tuning configuration
type PerformanceConfig struct {
	MaxConcurrent     int           `mapstructure:"max_concurrent"`
	CacheSize         int           `mapstructure:"cache_size"`
	CacheTTL          time.Duration `mapstructure:"cache_ttl"`
	TimeoutDuration   time.Duration `mapstructure:"timeout_duration"`
	MemoryLimit       int64         `mapstructure:"memory_limit"`
	MetalOptimization bool          `mapstructure:"metal_optimization"`
}
