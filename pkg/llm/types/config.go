package types

import "time"

type LLMConfig struct {
	ModelPath     string        `mapstructure:"model_path"`      // Path to GGUF model file
	ContextSize   int           `mapstructure:"context_size"`    // Default: 2048 for small models
	BatchSize     int           `mapstructure:"batch_size"`      // Batch size for processing
	Threads       int           `mapstructure:"threads"`         // CPU threads
	Temperature   float32       `mapstructure:"temperature"`     // Default: 0.7
	TopP          float32       `mapstructure:"top_p"`           // Default: 0.9
	TopK          int           `mapstructure:"top_k"`           // Default: 40
	RepeatPenalty float32       `mapstructure:"repeat_penalty"`  // Default: 1.1
	MaxTokens     int           `mapstructure:"max_tokens"`      // Default: 150
	Seed          int           `mapstructure:"seed"`            // For reproducibility
	UseGPU        bool          `mapstructure:"use_gpu"`         // GPU acceleration
	GPULayers     int           `mapstructure:"gpu_layers"`      // Layers to offload
	UseMMap       bool          `mapstructure:"use_mmap"`        // Memory-mapped I/O
	UseMlock      bool          `mapstructure:"use_mlock"`       // Lock model in memory
	EnableLowVRAM bool          `mapstructure:"enable_low_vram"` // Use more CPU/less GPU
	Timeout       time.Duration `mapstructure:"timeout"`         // Generation timeout
}

func DefaultLLMConfig() *LLMConfig {
	return &LLMConfig{
		ContextSize:   2048,
		BatchSize:     512,
		Threads:       4,
		Temperature:   0.7,
		TopP:          0.9,
		TopK:          40,
		RepeatPenalty: 1.1,
		MaxTokens:     150,
		Seed:          -1, // Random seed
		UseGPU:        false,
		GPULayers:     0,
		UseMMap:       true,
		UseMlock:      false,
		EnableLowVRAM: false,
		Timeout:       2 * time.Second,
	}
}

func (c *LLMConfig) Validate() error {
	if c.ModelPath == "" {
		return ErrInvalidModelPath
	}
	if c.ContextSize <= 0 {
		return ErrInvalidContextSize
	}
	if c.Threads <= 0 {
		return ErrInvalidThreadCount
	}
	if c.Temperature < 0 || c.Temperature > 2.0 {
		return ErrInvalidTemperature
	}
	if c.TopP <= 0 || c.TopP > 1.0 {
		return ErrInvalidTopP
	}
	if c.MaxTokens <= 0 {
		return ErrInvalidMaxTokens
	}
	if c.Timeout <= 0 {
		return ErrInvalidTimeout
	}
	return nil
}
