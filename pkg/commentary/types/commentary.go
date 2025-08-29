package types

import (
	"time"
)

// Commentary represents generated commentary for a HEMA match event (simplified)
type Commentary struct {
	ID          string    `json:"id"`
	Text        string    `json:"text"`
	DisplayText string    `json:"display_text"` // Formatted for OBS overlay
	Source      string    `json:"source"`       // "llm"
	Confidence  float32   `json:"confidence"`
	Timestamp   time.Time `json:"timestamp"`

	// Basic source information
	InputText       string  `json:"input_text"`
	InputConfidence float32 `json:"input_confidence"`

	// Performance metrics
	GenerationLatency time.Duration `json:"generation_latency"`
	CacheHit          bool          `json:"cache_hit"`

	// Quality metrics
	QualityScore     float32 `json:"quality_score"`
	RelevanceScore   float32 `json:"relevance_score"`
	ValidationPassed bool    `json:"validation_passed"`

	// Simplified metadata
	Metadata CommentaryMetadata `json:"metadata"`
}

// CommentaryMetadata contains minimal metadata about the commentary
type CommentaryMetadata struct {
	ProcessingSteps []string          `json:"processing_steps"` // Track what steps were used
	ExtraData       map[string]string `json:"extra_data,omitempty"`
}

// AudioQuality contains information about the audio input quality
type AudioQuality struct {
	SignalToNoise   float32 `json:"signal_to_noise"`
	Clarity         float32 `json:"clarity"`
	VoiceDetection  bool    `json:"voice_detection"`
	BackgroundNoise float32 `json:"background_noise"`
	Distortion      float32 `json:"distortion"`
}

// TranscriptionInput represents the input for commentary generation (simplified)
type TranscriptionInput struct {
	Text         string            `json:"text"`
	Confidence   float32           `json:"confidence"`
	Timestamp    time.Time         `json:"timestamp"`
	AudioMetrics *AudioQuality     `json:"audio_metrics,omitempty"`
	ExtraData    map[string]string `json:"extra_data,omitempty"`
}

// CommentaryRequest represents a request for commentary generation (simplified)
type CommentaryRequest struct {
	Input      TranscriptionInput `json:"input"`
	MaxLatency time.Duration      `json:"max_latency,omitempty"` // Maximum acceptable latency
}

// CommentaryResponse represents the response from commentary generation
type CommentaryResponse struct {
	Commentary *Commentary   `json:"commentary"`
	Success    bool          `json:"success"`
	Error      string        `json:"error,omitempty"`
	Latency    time.Duration `json:"latency"`
	Timestamp  time.Time     `json:"timestamp"`

	// Debug information
	ProcessingLog []LogEntry `json:"processing_log,omitempty"`
}

// LogEntry represents a processing step in the commentary generation
type LogEntry struct {
	Step      string        `json:"step"`
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration,omitempty"`
	Success   bool          `json:"success"`
	Message   string        `json:"message,omitempty"`
	Data      interface{}   `json:"data,omitempty"`
}

// GenerationMetrics contains metrics about the commentary generation process
type GenerationMetrics struct {
	// Request metrics
	TotalRequests  uint64 `json:"total_requests"`
	SuccessfulReqs uint64 `json:"successful_requests"`
	FailedReqs     uint64 `json:"failed_requests"`

	// Latency metrics
	AvgLatency time.Duration `json:"avg_latency"`
	P50Latency time.Duration `json:"p50_latency"`
	P95Latency time.Duration `json:"p95_latency"`
	P99Latency time.Duration `json:"p99_latency"`
	MaxLatency time.Duration `json:"max_latency"`

	// Source distribution
	LLMGenerated uint64 `json:"llm_generated"`
	CacheHits    uint64 `json:"cache_hits"`

	// Quality metrics
	AvgQualityScore    float32 `json:"avg_quality_score"`
	AvgRelevanceScore  float32 `json:"avg_relevance_score"`
	ValidationFailures uint64  `json:"validation_failures"`

	// Performance metrics
	TokensPerSecond float32 `json:"tokens_per_second"`
	ActiveRequests  int     `json:"active_requests"`

	// Error metrics
	TimeoutErrors    uint64 `json:"timeout_errors"`
	ValidationErrors uint64 `json:"validation_errors"`
	LLMErrors        uint64 `json:"llm_errors"`

	// Resource metrics
	MemoryUsage uint64  `json:"memory_usage_bytes"`
	CPUUsage    float32 `json:"cpu_usage_percent"`

	// Timestamp
	LastUpdated     time.Time `json:"last_updated"`
	CollectionStart time.Time `json:"collection_start"`
}

// CommentaryConfig represents configuration for the simplified commentary system
type CommentaryConfig struct {
	// Generation settings
	MaxLatency         time.Duration `mapstructure:"max_latency"`
	ConcurrentRequests int           `mapstructure:"concurrent_requests"`

	// Quality settings
	MinConfidence      float32 `mapstructure:"min_confidence"`
	QualityThreshold   float32 `mapstructure:"quality_threshold"`
	RelevanceThreshold float32 `mapstructure:"relevance_threshold"`

	// Output settings
	MaxOutputLength int `mapstructure:"max_output_length"`
	MinOutputLength int `mapstructure:"min_output_length"`

	// Monitoring settings
	EnableMetrics bool   `mapstructure:"enable_metrics"`
	EnableLogging bool   `mapstructure:"enable_logging"`
	LogLevel      string `mapstructure:"log_level"`
}

// DefaultCommentaryConfig returns default configuration for the simplified commentary system
func DefaultCommentaryConfig() *CommentaryConfig {
	return &CommentaryConfig{
		MaxLatency:         2 * time.Second,
		ConcurrentRequests: 3,
		MinConfidence:      0.6,
		QualityThreshold:   0.3,
		RelevanceThreshold: 0.25,
		MaxOutputLength:    200,
		MinOutputLength:    10,
		EnableMetrics:      true,
		EnableLogging:      true,
		LogLevel:           "info",
	}
}
