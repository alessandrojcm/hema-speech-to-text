package types

import (
	"time"
)

// GenerationRequest represents a request for text generation
type GenerationRequest struct {
	Prompt      string        `json:"prompt"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float32       `json:"temperature,omitempty"`
	TopP        float32       `json:"top_p,omitempty"`
	TopK        int           `json:"top_k,omitempty"`
	Timeout     time.Duration `json:"timeout,omitempty"`
}

// GenerationResponse represents a response from text generation
type GenerationResponse struct {
	Text         string             `json:"text"`
	TokenCount   int                `json:"token_count"`
	FinishReason string             `json:"finish_reason"` // "stop", "length", "timeout"
	Latency      time.Duration      `json:"latency"`
	Metadata     GenerationMetadata `json:"metadata"`
	Timestamp    time.Time          `json:"timestamp"`
}

// GenerationMetadata contains metadata about the generation process
type GenerationMetadata struct {
	ModelName        string        `json:"model_name"`
	TokensPerSecond  float64       `json:"tokens_per_second"`
	PromptTokens     int           `json:"prompt_tokens"`
	CompletionTokens int           `json:"completion_tokens"`
	TotalTokens      int           `json:"total_tokens"`
	ProcessingTime   time.Duration `json:"processing_time"`
}

// EngineStatus represents the current status of the LLM engine
type EngineStatus struct {
	Ready          bool          `json:"ready"`
	ModelLoaded    bool          `json:"model_loaded"`
	ModelPath      string        `json:"model_path"`
	MemoryUsage    uint64        `json:"memory_usage_bytes"`
	GPUMemoryUsage uint64        `json:"gpu_memory_usage_bytes"`
	ActiveRequests int           `json:"active_requests"`
	TotalRequests  uint64        `json:"total_requests"`
	SuccessRate    float64       `json:"success_rate"`
	AvgLatency     time.Duration `json:"avg_latency"`
	LastError      string        `json:"last_error,omitempty"`
	Uptime         time.Duration `json:"uptime"`
}

// EngineInterface defines the interface for LLM engines
type EngineInterface interface {
	// Generate generates text based on the given prompt
	Generate(request GenerationRequest) (*GenerationResponse, error)

	// GetStatus returns the current status of the engine
	GetStatus() EngineStatus

	// IsReady returns true if the engine is ready to generate text
	IsReady() bool

	// Close releases resources used by the engine
	Close() error
}
