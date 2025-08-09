package types

import "time"

// SpeechConfig represents speech recognition configuration
type SpeechConfig struct {
	Whisper     WhisperConfig     `mapstructure:"whisper"`
	Vocabulary  VocabularyConfig  `mapstructure:"vocabulary"`
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
}

// VocabularyConfig contains HEMA vocabulary configuration
type VocabularyConfig struct {
	HEMAVocabPath    string             `mapstructure:"hema_vocab_path"`
	BoostWeights     map[string]float64 `mapstructure:"boost_weights"`
	ContextSwitching bool               `mapstructure:"context_switching"`
	ValidationRules  []string           `mapstructure:"validation_rules"`
	CustomTerms      []string           `mapstructure:"custom_terms"`
}

// ProcessingConfig contains audio processing configuration for speech
type ProcessingConfig struct {
	TargetSampleRate int           `mapstructure:"target_sample_rate"`
	SegmentDuration  time.Duration `mapstructure:"segment_duration"`
	OverlapDuration  time.Duration `mapstructure:"overlap_duration"`
	NoiseReduction   bool          `mapstructure:"noise_reduction"`
	Normalization    bool          `mapstructure:"normalization"`
	VADEnabled       bool          `mapstructure:"vad_enabled"`
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
