package engine

import (
	"fmt"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-org/hema-replay-system/pkg/llm/types"
)

func TestNewLlamaEngine(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))

	tests := []struct {
		name        string
		config      *types.LLMConfig
		expectError bool
		errorType   error
	}{
		{
			name:        "nil config should fail",
			config:      nil,
			expectError: true,
			errorType:   types.ErrInvalidModelPath,
		},
		{
			name: "invalid model path should fail",
			config: &types.LLMConfig{
				ModelPath:   "",
				ContextSize: 2048,
				Threads:     4,
				Temperature: 0.7,
				TopP:        0.9,
				MaxTokens:   150,
				Timeout:     2 * time.Second,
			},
			expectError: true,
			errorType:   types.ErrInvalidModelPath,
		},
		{
			name: "invalid context size should fail",
			config: &types.LLMConfig{
				ModelPath:   "/path/to/model.gguf",
				ContextSize: -1,
				Threads:     4,
				Temperature: 0.7,
				TopP:        0.9,
				MaxTokens:   150,
				Timeout:     2 * time.Second,
			},
			expectError: true,
			errorType:   types.ErrInvalidContextSize,
		},
		{
			name: "invalid temperature should fail",
			config: &types.LLMConfig{
				ModelPath:   "/path/to/model.gguf",
				ContextSize: 2048,
				Threads:     4,
				Temperature: -0.1,
				TopP:        0.9,
				MaxTokens:   150,
				Timeout:     2 * time.Second,
			},
			expectError: true,
			errorType:   types.ErrInvalidTemperature,
		},
		{
			name: "valid config should succeed",
			config: &types.LLMConfig{
				ModelPath:   "/path/to/model.gguf",
				ContextSize: 2048,
				Threads:     4,
				Temperature: 0.7,
				TopP:        0.9,
				MaxTokens:   150,
				Timeout:     2 * time.Second,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewLlamaEngine(tt.config, logger)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, engine)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, engine)
				assert.True(t, engine.IsReady())

				// Clean up
				err = engine.Close()
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultLLMConfig(t *testing.T) {
	config := types.DefaultLLMConfig()

	assert.Equal(t, 2048, config.ContextSize)
	assert.Equal(t, 512, config.BatchSize)
	assert.Equal(t, 4, config.Threads)
	assert.Equal(t, float32(0.7), config.Temperature)
	assert.Equal(t, float32(0.9), config.TopP)
	assert.Equal(t, 40, config.TopK)
	assert.Equal(t, float32(1.1), config.RepeatPenalty)
	assert.Equal(t, 150, config.MaxTokens)
	assert.Equal(t, -1, config.Seed)
	assert.False(t, config.UseGPU)
	assert.Equal(t, 0, config.GPULayers)
	assert.True(t, config.UseMMap)
	assert.False(t, config.UseMlock)
	assert.False(t, config.EnableLowVRAM)
	assert.Equal(t, 2*time.Second, config.Timeout)
}

func TestLLMConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *types.LLMConfig
		expectError bool
		errorType   error
	}{
		{
			name: "valid config",
			config: &types.LLMConfig{
				ModelPath:   "/path/to/model.gguf",
				ContextSize: 2048,
				Threads:     4,
				Temperature: 0.7,
				TopP:        0.9,
				MaxTokens:   150,
				Timeout:     2 * time.Second,
			},
			expectError: false,
		},
		{
			name: "empty model path",
			config: &types.LLMConfig{
				ContextSize: 2048,
				Threads:     4,
				Temperature: 0.7,
				TopP:        0.9,
				MaxTokens:   150,
				Timeout:     2 * time.Second,
			},
			expectError: true,
			errorType:   types.ErrInvalidModelPath,
		},
		{
			name: "invalid context size",
			config: &types.LLMConfig{
				ModelPath:   "/path/to/model.gguf",
				ContextSize: 0,
				Threads:     4,
				Temperature: 0.7,
				TopP:        0.9,
				MaxTokens:   150,
				Timeout:     2 * time.Second,
			},
			expectError: true,
			errorType:   types.ErrInvalidContextSize,
		},
		{
			name: "invalid thread count",
			config: &types.LLMConfig{
				ModelPath:   "/path/to/model.gguf",
				ContextSize: 2048,
				Threads:     0,
				Temperature: 0.7,
				TopP:        0.9,
				MaxTokens:   150,
				Timeout:     2 * time.Second,
			},
			expectError: true,
			errorType:   types.ErrInvalidThreadCount,
		},
		{
			name: "temperature too high",
			config: &types.LLMConfig{
				ModelPath:   "/path/to/model.gguf",
				ContextSize: 2048,
				Threads:     4,
				Temperature: 2.1,
				TopP:        0.9,
				MaxTokens:   150,
				Timeout:     2 * time.Second,
			},
			expectError: true,
			errorType:   types.ErrInvalidTemperature,
		},
		{
			name: "invalid top_p",
			config: &types.LLMConfig{
				ModelPath:   "/path/to/model.gguf",
				ContextSize: 2048,
				Threads:     4,
				Temperature: 0.7,
				TopP:        1.1,
				MaxTokens:   150,
				Timeout:     2 * time.Second,
			},
			expectError: true,
			errorType:   types.ErrInvalidTopP,
		},
		{
			name: "invalid max tokens",
			config: &types.LLMConfig{
				ModelPath:   "/path/to/model.gguf",
				ContextSize: 2048,
				Threads:     4,
				Temperature: 0.7,
				TopP:        0.9,
				MaxTokens:   0,
				Timeout:     2 * time.Second,
			},
			expectError: true,
			errorType:   types.ErrInvalidMaxTokens,
		},
		{
			name: "invalid timeout",
			config: &types.LLMConfig{
				ModelPath:   "/path/to/model.gguf",
				ContextSize: 2048,
				Threads:     4,
				Temperature: 0.7,
				TopP:        0.9,
				MaxTokens:   150,
				Timeout:     0,
			},
			expectError: true,
			errorType:   types.ErrInvalidTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLlamaEngine_Generate(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	config := types.DefaultLLMConfig()
	config.ModelPath = "/path/to/model.gguf"

	engine, err := NewLlamaEngine(config, logger)
	require.NoError(t, err)
	defer engine.Close()

	tests := []struct {
		name        string
		request     types.GenerationRequest
		expectError bool
		errorType   error
	}{
		{
			name: "valid request",
			request: types.GenerationRequest{
				Prompt:      "Generate commentary for: Point left",
				MaxTokens:   50,
				Temperature: 0.7,
			},
			expectError: false,
		},
		{
			name: "empty prompt should fail",
			request: types.GenerationRequest{
				Prompt:    "",
				MaxTokens: 50,
			},
			expectError: true,
			errorType:   types.ErrInvalidPrompt,
		},
		{
			name: "request with timeout",
			request: types.GenerationRequest{
				Prompt:  "Generate commentary",
				Timeout: 1 * time.Second,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := engine.Generate(tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, response)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.NotEmpty(t, response.Text)
				assert.Greater(t, response.TokenCount, 0)
				assert.Equal(t, "stop", response.FinishReason)
				assert.GreaterOrEqual(t, response.Latency, time.Duration(0))
				assert.NotZero(t, response.Timestamp)
			}
		})
	}
}

func TestLlamaEngine_GetStatus(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	config := types.DefaultLLMConfig()
	config.ModelPath = "/path/to/model.gguf"

	engine, err := NewLlamaEngine(config, logger)
	require.NoError(t, err)
	defer engine.Close()

	status := engine.GetStatus()

	assert.True(t, status.Ready)
	assert.True(t, status.ModelLoaded)
	assert.Equal(t, config.ModelPath, status.ModelPath)
	assert.Equal(t, 0, status.ActiveRequests)
	assert.Equal(t, uint64(0), status.TotalRequests)
	assert.Greater(t, status.Uptime, time.Duration(0))
}

func TestLlamaEngine_IsReady(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	config := types.DefaultLLMConfig()
	config.ModelPath = "/path/to/model.gguf"

	engine, err := NewLlamaEngine(config, logger)
	require.NoError(t, err)

	assert.True(t, engine.IsReady())

	err = engine.Close()
	require.NoError(t, err)

	assert.False(t, engine.IsReady())
}

func TestLlamaEngine_Close(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	config := types.DefaultLLMConfig()
	config.ModelPath = "/path/to/model.gguf"

	engine, err := NewLlamaEngine(config, logger)
	require.NoError(t, err)

	assert.True(t, engine.IsReady())

	err = engine.Close()
	assert.NoError(t, err)
	assert.False(t, engine.IsReady())

	// Closing again should not error
	err = engine.Close()
	assert.NoError(t, err)
}

func TestLlamaEngine_ConcurrentRequests(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	config := types.DefaultLLMConfig()
	config.ModelPath = "/path/to/model.gguf"

	engine, err := NewLlamaEngine(config, logger)
	require.NoError(t, err)
	defer engine.Close()

	// Test concurrent generation requests
	const numRequests = 5
	results := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			request := types.GenerationRequest{
				Prompt: fmt.Sprintf("Generate commentary %d", id),
			}
			_, err := engine.Generate(request)
			results <- err
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		err := <-results
		assert.NoError(t, err)
	}

	// Check that metrics are updated
	status := engine.GetStatus()
	assert.Equal(t, uint64(numRequests), status.TotalRequests)
	assert.Equal(t, 0, status.ActiveRequests) // All should be completed
}
