package types

import (
	"time"
)

// Commentary represents generated commentary for a HEMA match event
type Commentary struct {
	ID          string             `json:"id"`
	Text        string             `json:"text"`
	DisplayText string             `json:"display_text"` // Formatted for OBS overlay
	Source      string             `json:"source"`       // "llm", "cache", "fallback"
	Confidence  float32            `json:"confidence"`
	Timestamp   time.Time          `json:"timestamp"`
	Metadata    CommentaryMetadata `json:"metadata"`

	// Source information
	InputText       string  `json:"input_text"`
	InputConfidence float32 `json:"input_confidence"`
	TemplateID      string  `json:"template_id"`
	PromptUsed      string  `json:"prompt_used,omitempty"`

	// Performance metrics
	GenerationLatency time.Duration `json:"generation_latency"`
	CacheHit          bool          `json:"cache_hit"`

	// Quality metrics
	QualityScore     float32 `json:"quality_score"`
	RelevanceScore   float32 `json:"relevance_score"`
	ValidationPassed bool    `json:"validation_passed"`
}

// CommentaryMetadata contains additional metadata about the commentary
type CommentaryMetadata struct {
	Category        string            `json:"category"`    // "scoring", "rules", "technique", etc.
	Priority        string            `json:"priority"`    // "high", "medium", "low"
	TargetTone      string            `json:"target_tone"` // "exciting", "educational", "neutral"
	MatchContext    *MatchContext     `json:"match_context,omitempty"`
	AudioQuality    *AudioQuality     `json:"audio_quality,omitempty"`
	ProcessingSteps []string          `json:"processing_steps"` // Track what steps were used
	Fallbacks       []string          `json:"fallbacks"`        // Track fallback strategies used
	ExtraData       map[string]string `json:"extra_data,omitempty"`
}

// MatchContext provides context about the current match state
type MatchContext struct {
	ScoreRed      int           `json:"score_red"`
	ScoreBlue     int           `json:"score_blue"`
	Period        int           `json:"period"`
	TimeRemaining time.Duration `json:"time_remaining"`
	LastScorer    string        `json:"last_scorer"` // "red", "blue", or ""
	LastAction    string        `json:"last_action"`
	RecentActions []string      `json:"recent_actions"`
	MatchPhase    string        `json:"match_phase"` // "opening", "middle", "closing"
	Intensity     string        `json:"intensity"`   // "low", "medium", "high"
}

// AudioQuality contains information about the audio input quality
type AudioQuality struct {
	SignalToNoise   float32 `json:"signal_to_noise"`
	Clarity         float32 `json:"clarity"`
	VoiceDetection  bool    `json:"voice_detection"`
	BackgroundNoise float32 `json:"background_noise"`
	Distortion      float32 `json:"distortion"`
}

// TranscriptionInput represents the input for commentary generation
type TranscriptionInput struct {
	Text         string            `json:"text"`
	Confidence   float32           `json:"confidence"`
	Timestamp    time.Time         `json:"timestamp"`
	AudioMetrics *AudioQuality     `json:"audio_metrics,omitempty"`
	Context      *MatchContext     `json:"context,omitempty"`
	ExtraData    map[string]string `json:"extra_data,omitempty"`
}

// CommentaryRequest represents a request for commentary generation
type CommentaryRequest struct {
	Input       TranscriptionInput `json:"input"`
	TemplateID  string             `json:"template_id,omitempty"`  // Override template selection
	MaxLatency  time.Duration      `json:"max_latency,omitempty"`  // Maximum acceptable latency
	Quality     QualityLevel       `json:"quality,omitempty"`      // Quality vs speed preference
	CachePolicy CachePolicy        `json:"cache_policy,omitempty"` // Cache behavior preference
}

// QualityLevel defines the quality preference for generation
type QualityLevel string

const (
	QualityLevelFast     QualityLevel = "fast"     // Prefer speed, allow fallbacks
	QualityLevelBalanced QualityLevel = "balanced" // Balance speed and quality
	QualityLevelHigh     QualityLevel = "high"     // Prefer quality, longer latency OK
)

// CachePolicy defines cache behavior preference
type CachePolicy string

const (
	CachePolicyDefault  CachePolicy = "default"  // Use normal cache behavior
	CachePolicyForce    CachePolicy = "force"    // Prefer cached results
	CachePolicyBypass   CachePolicy = "bypass"   // Force generation, update cache
	CachePolicyDisabled CachePolicy = "disabled" // No cache usage
)

// CommentaryResponse represents the response from commentary generation
type CommentaryResponse struct {
	Commentary *Commentary   `json:"commentary"`
	Success    bool          `json:"success"`
	Error      string        `json:"error,omitempty"`
	Latency    time.Duration `json:"latency"`
	Timestamp  time.Time     `json:"timestamp"`

	// Debug information
	CacheStats    *CacheStats `json:"cache_stats,omitempty"`
	ProcessingLog []LogEntry  `json:"processing_log,omitempty"`
}

// CacheStats provides information about cache usage
type CacheStats struct {
	L1Hit           bool          `json:"l1_hit"`
	L2Hit           bool          `json:"l2_hit"`
	L3Hit           bool          `json:"l3_hit"`
	CacheKey        string        `json:"cache_key"`
	LookupTime      time.Duration `json:"lookup_time"`
	SimilarityScore float32       `json:"similarity_score,omitempty"`
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
	FallbackUsed uint64 `json:"fallback_used"`

	// Quality metrics
	AvgQualityScore    float32 `json:"avg_quality_score"`
	AvgRelevanceScore  float32 `json:"avg_relevance_score"`
	ValidationFailures uint64  `json:"validation_failures"`

	// Cache metrics
	CacheHitRate  float32 `json:"cache_hit_rate"`
	CacheMissRate float32 `json:"cache_miss_rate"`

	// Performance metrics
	TokensPerSecond float32 `json:"tokens_per_second"`
	ActiveRequests  int     `json:"active_requests"`

	// Error metrics
	TimeoutErrors    uint64 `json:"timeout_errors"`
	ValidationErrors uint64 `json:"validation_errors"`
	LLMErrors        uint64 `json:"llm_errors"`
	CacheErrors      uint64 `json:"cache_errors"`

	// Resource metrics
	MemoryUsage uint64  `json:"memory_usage_bytes"`
	CPUUsage    float32 `json:"cpu_usage_percent"`

	// Timestamp
	LastUpdated     time.Time `json:"last_updated"`
	CollectionStart time.Time `json:"collection_start"`
}

// CommentaryConfig represents configuration for the commentary system
type CommentaryConfig struct {
	// Generation settings
	DefaultTemplate    string        `mapstructure:"default_template"`
	MaxLatency         time.Duration `mapstructure:"max_latency"`
	DefaultQuality     QualityLevel  `mapstructure:"default_quality"`
	ConcurrentRequests int           `mapstructure:"concurrent_requests"`

	// Quality settings
	MinConfidence      float32 `mapstructure:"min_confidence"`
	QualityThreshold   float32 `mapstructure:"quality_threshold"`
	RelevanceThreshold float32 `mapstructure:"relevance_threshold"`

	// Fallback settings
	EnableFallback    bool    `mapstructure:"enable_fallback"`
	FallbackThreshold float32 `mapstructure:"fallback_threshold"`
	MaxRetries        int     `mapstructure:"max_retries"`

	// Output settings
	MaxOutputLength       int  `mapstructure:"max_output_length"`
	MinOutputLength       int  `mapstructure:"min_output_length"`
	EnableProfanityFilter bool `mapstructure:"enable_profanity_filter"`

	// Cache settings
	EnableCache        bool        `mapstructure:"enable_cache"`
	DefaultCachePolicy CachePolicy `mapstructure:"default_cache_policy"`

	// Monitoring settings
	EnableMetrics bool   `mapstructure:"enable_metrics"`
	EnableLogging bool   `mapstructure:"enable_logging"`
	LogLevel      string `mapstructure:"log_level"`
}

// DefaultCommentaryConfig returns default configuration for the commentary system
func DefaultCommentaryConfig() *CommentaryConfig {
	return &CommentaryConfig{
		DefaultTemplate:       "point_scored",
		MaxLatency:            2 * time.Second,
		DefaultQuality:        QualityLevelBalanced,
		ConcurrentRequests:    3,
		MinConfidence:         0.6,
		QualityThreshold:      0.3,
		RelevanceThreshold:    0.25,
		EnableFallback:        true,
		FallbackThreshold:     0.6,
		MaxRetries:            2,
		MaxOutputLength:       200,
		MinOutputLength:       10,
		EnableProfanityFilter: true,
		EnableCache:           true,
		DefaultCachePolicy:    CachePolicyDefault,
		EnableMetrics:         true,
		EnableLogging:         true,
		LogLevel:              "info",
	}
}
