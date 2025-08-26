package integration

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	llmtypes "github.com/your-org/hema-replay-system/pkg/llm/types"

	commentarycontext "github.com/your-org/hema-replay-system/pkg/commentary/context"
	"github.com/your-org/hema-replay-system/pkg/commentary/engine"
	"github.com/your-org/hema-replay-system/pkg/commentary/prompt"
	"github.com/your-org/hema-replay-system/pkg/commentary/templates"
	"github.com/your-org/hema-replay-system/pkg/commentary/types"
	llmengine "github.com/your-org/hema-replay-system/pkg/llm/engine"
)

func TestCommentaryFallbackBehavior(t *testing.T) {
	// Create logger
	logger := zerolog.New(zerolog.NewConsoleWriter()).Level(zerolog.InfoLevel)

	commentaryConfig := types.DefaultCommentaryConfig()
	commentaryConfig.EnableFallback = true
	commentaryConfig.MaxRetries = 1
	commentaryConfig.EnableCache = false // Disable cache for testing

	// Create required dependencies - use default template manager with registered templates
	contextManager := commentarycontext.NewContextManager(logger)
	promptConfig := prompt.DefaultBuilderConfig()
	promptBuilder := prompt.NewBuilder(templates.DefaultTemplateManager, contextManager, promptConfig, logger)

	// Create commentary generator (with nil LLM engine to force fallback)
	generator, err := engine.NewCommentaryGenerator(nil, promptBuilder, commentaryConfig, logger)
	require.NoError(t, err)
	defer generator.Stop()

	// Start generator
	err = generator.Start()
	require.NoError(t, err, "Generator should start even without LLM engine")

	// Try to generate commentary - should fall back to template
	input := types.TranscriptionInput{
		Text:       "Point scored to red",
		Confidence: 0.8,
		Timestamp:  time.Now(),
	}

	request := types.CommentaryRequest{
		Input:       input,
		MaxLatency:  3 * time.Second,
		Quality:     types.QualityLevelFast, // Fast mode should prefer fallback
		CachePolicy: types.CachePolicyDisabled,
	}

	response, err := generator.Generate(context.Background(), &request)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.True(t, response.Success)
	require.NotNil(t, response.Commentary)

	// Should use fallback
	assert.Equal(t, "fallback", response.Commentary.Source)
	assert.NotEmpty(t, response.Commentary.Text)
	assert.True(t, response.Commentary.Confidence > 0)

	t.Logf("Fallback commentary: %s", response.Commentary.Text)
}

func TestCommentaryGeneratorLifecycle(t *testing.T) {
	// Create logger
	logger := zerolog.New(zerolog.NewConsoleWriter()).Level(zerolog.InfoLevel)

	commentaryConfig := types.DefaultCommentaryConfig()
	commentaryConfig.EnableFallback = true
	commentaryConfig.MaxLatency = 5 * time.Second
	commentaryConfig.EnableCache = false // Disable cache for testing

	// Create required dependencies
	contextManager := commentarycontext.NewContextManager(logger)
	promptConfig := prompt.DefaultBuilderConfig()
	promptBuilder := prompt.NewBuilder(templates.DefaultTemplateManager, contextManager, promptConfig, logger)

	// Create commentary generator
	generator, err := engine.NewCommentaryGenerator(nil, promptBuilder, commentaryConfig, logger)
	require.NoError(t, err)

	// Test start/stop cycle
	err = generator.Start()
	require.NoError(t, err)

	// Get metrics to verify it's running
	metrics := generator.GetMetrics()
	assert.NotNil(t, metrics)

	// Test graceful shutdown
	err = generator.Stop()
	assert.NoError(t, err, "Generator should stop gracefully")
}

func TestCommentaryTemplateSelection(t *testing.T) {
	// Create logger
	logger := zerolog.New(zerolog.NewConsoleWriter()).Level(zerolog.InfoLevel)

	commentaryConfig := types.DefaultCommentaryConfig()
	commentaryConfig.EnableFallback = true

	// Create required dependencies
	contextManager := commentarycontext.NewContextManager(logger)
	promptConfig := prompt.DefaultBuilderConfig()
	promptBuilder := prompt.NewBuilder(templates.DefaultTemplateManager, contextManager, promptConfig, logger)

	// Create commentary generator
	generator, err := engine.NewCommentaryGenerator(nil, promptBuilder, commentaryConfig, logger)
	require.NoError(t, err)
	defer generator.Stop()

	err = generator.Start()
	require.NoError(t, err)

	// Test different types of HEMA calls to see template selection
	testCases := []struct {
		name string
		text string
	}{
		{"Point Scored", "Point scored to red fencer"},
		{"Technical Action", "Beautiful riposte from blue"},
		{"Rules Clarification", "No point awarded, simultaneous attack"},
		{"Generic Action", "Good exchange between both fencers"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := types.TranscriptionInput{
				Text:       tc.text,
				Confidence: 0.8,
				Timestamp:  time.Now(),
			}

			request := types.CommentaryRequest{
				Input:       input,
				MaxLatency:  2 * time.Second,
				Quality:     types.QualityLevelBalanced,
				CachePolicy: types.CachePolicyDisabled,
			}

			response, err := generator.Generate(context.Background(), &request)
			require.NoError(t, err)
			require.NotNil(t, response)
			require.True(t, response.Success)

			commentary := response.Commentary
			assert.NotEmpty(t, commentary.Text)
			assert.Equal(t, "fallback", commentary.Source) // Should use fallback since no LLM

			t.Logf("%s -> %s", tc.text, commentary.Text)
		})
	}
}

func TestCommentaryWithActualLLM(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping LLM test in short mode")
	}

	// Create logger
	logger := zerolog.New(zerolog.NewConsoleWriter()).Level(zerolog.InfoLevel)

	// Commentary configuration
	commentaryConfig := types.DefaultCommentaryConfig()
	commentaryConfig.EnableFallback = true
	commentaryConfig.MaxLatency = 15 * time.Second // Generous timeout for LLM
	commentaryConfig.EnableCache = false           // Disable cache for testing
	config := llmtypes.DefaultLLMConfig()
	// Create LLM engine
	llmEngine, err := llmengine.NewLlmEngine(config, context.Background(), logger)
	require.NoError(t, err, "Failed to create LLM engine")
	defer llmEngine.Close()

	// Create required dependencies
	contextManager := commentarycontext.NewContextManager(logger)
	promptConfig := prompt.DefaultBuilderConfig()
	promptBuilder := prompt.NewBuilder(templates.DefaultTemplateManager, contextManager, promptConfig, logger)

	// Create commentary generator with actual LLM engine
	generator, err := engine.NewCommentaryGenerator(llmEngine, promptBuilder, commentaryConfig, logger)
	require.NoError(t, err, "Failed to create commentary generator")
	defer generator.Stop()

	// Start generator
	err = generator.Start()
	require.NoError(t, err, "Failed to start commentary generator")

	// Test cases with different HEMA scenarios
	testCases := []struct {
		name           string
		transcription  string
		confidence     float32
		expectedSource string
		expectedMinLen int
	}{
		{
			name:           "Point Scored",
			transcription:  "Red deep target blue",
			confidence:     0.9,
			expectedSource: "llm", // Should use LLM with high confidence
			expectedMinLen: 20,
		},
		{
			name:           "Technical Action",
			transcription:  "Beautiful thrust and riposte from blue",
			confidence:     0.8,
			expectedSource: "llm",
			expectedMinLen: 15,
		},
		{
			name:           "Double Hit",
			transcription:  "Double hit, both fencers hit simultaneously",
			confidence:     0.85,
			expectedSource: "llm",
			expectedMinLen: 25,
		},
		{
			name:           "Point scored",
			transcription:  "Deep target blue",
			confidence:     0.85,
			expectedSource: "llm",
			expectedMinLen: 15,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Allow generous timeout for this subtest

			input := types.TranscriptionInput{
				Text:       tc.transcription,
				Confidence: tc.confidence,
				Timestamp:  time.Now(),
			}

			request := types.CommentaryRequest{
				Input:       input,
				MaxLatency:  15 * time.Second,
				Quality:     types.QualityLevelBalanced,
				CachePolicy: types.CachePolicyDisabled,
			}

			t.Logf("Generating commentary for: %s", tc.transcription)
			startTime := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			response, err := generator.Generate(ctx, &request)

			duration := time.Since(startTime)
			t.Logf("Generation took: %v", duration)

			require.NoError(t, err, "Commentary generation should not fail")
			require.NotNil(t, response, "Response should not be nil")
			require.True(t, response.Success, "Response should be successful: %s", response.Error)
			require.NotNil(t, response.Commentary, "Commentary should not be nil")

			commentary := response.Commentary

			// Validate basic properties
			assert.NotEmpty(t, commentary.Text, "Commentary text should not be empty")
			assert.NotEmpty(t, commentary.DisplayText, "Display text should not be empty")
			assert.True(t, commentary.Confidence > 0, "Confidence should be positive")
			assert.WithinDuration(t, time.Now(), commentary.Timestamp, 5*time.Second, "Timestamp should be recent")

			// Validate length
			assert.True(t, len(commentary.Text) >= tc.expectedMinLen,
				"Commentary too short: got %d chars, expected >= %d", len(commentary.Text), tc.expectedMinLen)
			assert.True(t, len(commentary.Text) <= commentaryConfig.MaxOutputLength,
				"Commentary too long: got %d chars, expected <= %d", len(commentary.Text), commentaryConfig.MaxOutputLength)

			// Validate generation performance
			assert.True(t, commentary.GenerationLatency > 0, "Generation latency should be positive")
			assert.True(t, commentary.GenerationLatency < 20*time.Second, "Generation should complete in reasonable time")

			// Log results for manual inspection
			t.Logf("Input: %s", tc.transcription)
			t.Logf("Generated Commentary: %s", commentary.Text)
			t.Logf("Source: %s", commentary.Source)
			t.Logf("Confidence: %.2f", commentary.Confidence)
			t.Logf("Generation Latency: %v", commentary.GenerationLatency)
			t.Logf("Quality Score: %.2f", commentary.QualityScore)

			// Verify it's not empty/generic fallback text
			assert.NotEqual(t, "The action continues.", commentary.Text, "Should not use generic fallback")
			assert.True(t, len(commentary.Text) > 10, "Commentary should be substantive")
		})
	}
}
