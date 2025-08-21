package prompt

import (
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/your-org/hema-replay-system/pkg/commentary/context"
	"github.com/your-org/hema-replay-system/pkg/commentary/templates"
)

func TestNewBuilder(t *testing.T) {
	logger := zerolog.Nop()
	templateManager := templates.NewTemplateManager()
	contextManager := context.NewContextManager(logger)

	builder := NewBuilder(templateManager, contextManager, nil, logger)

	assert.NotNil(t, builder)
	assert.NotNil(t, builder.config)
	assert.Equal(t, DefaultBuilderConfig(), builder.config)
}

func TestBuilder_Build_BasicFunctionality(t *testing.T) {
	logger := zerolog.Nop()
	templateManager := templates.NewTemplateManager()
	contextManager := context.NewContextManager(logger)

	// Register HEMA templates
	err := templates.RegisterHEMATemplates()
	require.NoError(t, err)

	builder := NewBuilder(templateManager, contextManager, nil, logger)

	tests := []struct {
		name    string
		request *BuildRequest
		wantErr bool
	}{
		{
			name: "valid_request",
			request: &BuildRequest{
				Transcription: "Point left",
				Confidence:    0.95,
				Timestamp:     time.Now(),
			},
			wantErr: false,
		},
		{
			name: "empty_transcription",
			request: &BuildRequest{
				Transcription: "",
				Confidence:    0.95,
			},
			wantErr: true,
		},
		{
			name: "invalid_confidence",
			request: &BuildRequest{
				Transcription: "Point left",
				Confidence:    1.5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := builder.Build(tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotEmpty(t, result.Prompt)
				assert.NotEmpty(t, result.TemplateID)
				assert.Greater(t, result.BuildTime, time.Duration(0))
			}
		})
	}
}

func TestBuilder_TemplateSelection(t *testing.T) {
	logger := zerolog.Nop()
	templateManager := templates.NewTemplateManager()
	contextManager := context.NewContextManager(logger)

	err := templates.RegisterHEMATemplates()
	require.NoError(t, err)

	builder := NewBuilder(templateManager, contextManager, nil, logger)

	tests := []struct {
		name               string
		transcription      string
		expectedTemplateID string
		matchState         *context.MatchState
	}{
		{
			name:               "point_detection",
			transcription:      "Point left clean hit",
			expectedTemplateID: "point_scored",
		},
		{
			name:               "double_hit_detection",
			transcription:      "Double hit both fencers",
			expectedTemplateID: "double_hit",
		},
		{
			name:               "explicit_template",
			transcription:      "Any text",
			expectedTemplateID: "halt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.matchState != nil {
				contextManager.UpdateMatchState(tt.matchState)
			}

			request := &BuildRequest{
				Transcription: tt.transcription,
				Confidence:    0.95,
				Timestamp:     time.Now(),
			}

			// For explicit template test, set the template ID
			if tt.name == "explicit_template" {
				request.TemplateID = "halt"
			}

			result, err := builder.Build(request)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedTemplateID, result.TemplateID)
		})
	}
}

func TestBuilder_ContextEnrichment(t *testing.T) {
	logger := zerolog.Nop()
	templateManager := templates.NewTemplateManager()
	contextManager := context.NewContextManager(logger)

	err := templates.RegisterHEMATemplates()
	require.NoError(t, err)

	// Set up rich context
	matchState := context.NewMatchState()
	matchState.ScoreRed = 3
	matchState.ScoreBlue = 2
	matchState.TimeRemaining = time.Second * 30
	contextManager.UpdateMatchState(matchState)

	contextManager.AddCall("Point left")
	contextManager.AddCall("Point right")

	builder := NewBuilder(templateManager, contextManager, nil, logger)

	request := &BuildRequest{
		Transcription: "Point left excellent attack",
		Confidence:    0.95,
		Timestamp:     time.Now(),
		ExtraContext: map[string]string{
			"technique": "thrust",
		},
	}

	result, err := builder.Build(request)
	require.NoError(t, err)

	// Verify context enrichment worked
	assert.NotEmpty(t, result.UsedContext)
	assert.Contains(t, result.UsedContext, "technique")
	assert.Equal(t, "thrust", result.UsedContext["technique"])
}

func TestBuilder_AudioMetrics(t *testing.T) {
	logger := zerolog.Nop()
	templateManager := templates.NewTemplateManager()
	contextManager := context.NewContextManager(logger)

	err := templates.RegisterHEMATemplates()
	require.NoError(t, err)

	config := &BuilderConfig{
		EnableContextEnrichment: true,
		IncludeAudioMetrics:     true,
		IncludeConfidence:       true,
		IncludeTimestamp:        true,
	}

	builder := NewBuilder(templateManager, contextManager, config, logger)

	audioMetrics := &templates.AudioMetrics{
		Volume:       0.8,
		Clarity:      0.9,
		NoiseLevel:   0.1,
		VoicePresent: true,
	}

	request := &BuildRequest{
		Transcription: "Point left",
		Confidence:    0.95,
		Timestamp:     time.Now(),
		AudioMetrics:  audioMetrics,
	}

	result, err := builder.Build(request)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Prompt)
}

func TestBuilder_PromptTrimming(t *testing.T) {
	logger := zerolog.Nop()
	templateManager := templates.NewTemplateManager()
	contextManager := context.NewContextManager(logger)

	// Create a template that generates very long prompts
	longTemplate := &templates.PromptTemplate{
		ID:       "long_template",
		Template: strings.Repeat("This is a very long prompt that will exceed the maximum length. ", 20),
	}
	err := templateManager.RegisterTemplate(longTemplate)
	require.NoError(t, err)

	config := &BuilderConfig{
		MaxPromptLength: 100, // Short limit
	}

	builder := NewBuilder(templateManager, contextManager, config, logger)

	request := &BuildRequest{
		Transcription: "Point left",
		Confidence:    0.95,
		Timestamp:     time.Now(),
		TemplateID:    "long_template", // Force use of long template
	}

	result, err := builder.Build(request)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(result.Prompt), 105) // Allow some margin for ellipsis
}

func TestBuilder_FallbackBehavior(t *testing.T) {
	logger := zerolog.Nop()
	templateManager := templates.NewTemplateManager()
	contextManager := context.NewContextManager(logger)

	tests := []struct {
		name             string
		fallbackBehavior string
		templateID       string
		expectError      bool
	}{
		{
			name:             "fallback_to_simple",
			fallbackBehavior: "simple",
			templateID:       "nonexistent_template",
			expectError:      false,
		},
		{
			name:             "fallback_none",
			fallbackBehavior: "none",
			templateID:       "nonexistent_template",
			expectError:      true,
		},
		{
			name:             "fallback_default",
			fallbackBehavior: "default",
			templateID:       "nonexistent_template",
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &BuilderConfig{
				FallbackBehavior: tt.fallbackBehavior,
				DefaultTemplate:  "point_scored",
			}

			// Register at least the default template
			err := templates.RegisterHEMATemplates()
			require.NoError(t, err)

			builder := NewBuilder(templateManager, contextManager, config, logger)

			request := &BuildRequest{
				Transcription: "Point left",
				Confidence:    0.95,
				Timestamp:     time.Now(),
				TemplateID:    tt.templateID,
			}

			result, err := builder.Build(request)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotEmpty(t, result.Prompt)
			}
		})
	}
}

func TestBuilder_BatchBuild(t *testing.T) {
	logger := zerolog.Nop()
	templateManager := templates.NewTemplateManager()
	contextManager := context.NewContextManager(logger)

	err := templates.RegisterHEMATemplates()
	require.NoError(t, err)

	builder := NewBuilder(templateManager, contextManager, nil, logger)

	requests := []*BuildRequest{
		{
			Transcription: "Point left",
			Confidence:    0.95,
			Timestamp:     time.Now(),
		},
		{
			Transcription: "Double hit",
			Confidence:    0.90,
			Timestamp:     time.Now(),
		},
		{
			Transcription: "", // Invalid request
			Confidence:    0.85,
		},
	}

	results, errors := builder.BatchBuild(requests)

	assert.Len(t, results, 3)
	assert.Len(t, errors, 3)

	// First two should succeed
	assert.NoError(t, errors[0])
	assert.NoError(t, errors[1])
	assert.NotNil(t, results[0])
	assert.NotNil(t, results[1])

	// Third should fail
	assert.Error(t, errors[2])
	assert.Nil(t, results[2])
}

func TestBuilderConfig_DefaultValues(t *testing.T) {
	config := DefaultBuilderConfig()

	assert.True(t, config.EnableContextEnrichment)
	assert.Equal(t, 500, config.MaxPromptLength)
	assert.Equal(t, "point_scored", config.DefaultTemplate)
	assert.Equal(t, float32(1.0), config.ContextWeight)
	assert.False(t, config.IncludeTimestamp)
	assert.True(t, config.IncludeConfidence)
	assert.False(t, config.IncludeAudioMetrics)
	assert.Equal(t, "default", config.FallbackBehavior)
}

func TestBuilder_GetStats(t *testing.T) {
	logger := zerolog.Nop()
	templateManager := templates.NewTemplateManager()
	contextManager := context.NewContextManager(logger)

	err := templates.RegisterHEMATemplates()
	require.NoError(t, err)

	builder := NewBuilder(templateManager, contextManager, nil, logger)

	stats := builder.GetStats()
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "template_count")
	assert.Contains(t, stats, "config")

	templateCount, ok := stats["template_count"].(int)
	assert.True(t, ok)
	assert.Greater(t, templateCount, 0) // Should have HEMA templates
}

// Integration test combining multiple components
func TestFullPromptGenerationPipeline(t *testing.T) {
	logger := zerolog.Nop()

	// Set up all components
	templateManager := templates.NewTemplateManager()
	contextManager := context.NewContextManager(logger)

	// Register templates
	err := templates.RegisterHEMATemplates()
	require.NoError(t, err)

	// Set up match context
	matchState := context.NewMatchState()
	matchState.UpdateScore("left")
	matchState.UpdateScore("right")
	matchState.UpdateScore("left") // 2-1
	contextManager.UpdateMatchState(matchState)

	// Add call history
	contextManager.AddCall("Fence")
	contextManager.AddCall("Point left")
	contextManager.AddCall("Point right")

	// Build prompt
	builder := NewBuilder(templateManager, contextManager, nil, logger)

	request := &BuildRequest{
		Transcription: "Point left excellent thrust",
		Confidence:    0.95,
		Timestamp:     time.Now(),
		ExtraContext: map[string]string{
			"technique": "thrust",
			"quality":   "excellent",
		},
	}

	result, err := builder.Build(request)
	require.NoError(t, err)

	// Verify comprehensive result
	assert.NotEmpty(t, result.Prompt)
	assert.Equal(t, "point_scored", result.TemplateID)
	assert.Greater(t, result.BuildTime, time.Duration(0))
	assert.NotEmpty(t, result.UsedContext)

	// Check that the prompt contains expected elements
	assert.Contains(t, result.Prompt, "Point left excellent thrust")

	// Verify metadata
	assert.NotNil(t, result.Metadata)
	assert.Contains(t, result.Metadata, "template_category")

	t.Logf("Generated prompt: %s", result.Prompt)
	t.Logf("Build time: %v", result.BuildTime)
	t.Logf("Used context: %+v", result.UsedContext)
}

// Benchmark tests
func BenchmarkBuilder_Build(b *testing.B) {
	logger := zerolog.Nop()
	templateManager := templates.NewTemplateManager()
	contextManager := context.NewContextManager(logger)

	err := templates.RegisterHEMATemplates()
	require.NoError(b, err)

	builder := NewBuilder(templateManager, contextManager, nil, logger)

	request := &BuildRequest{
		Transcription: "Point left clean hit",
		Confidence:    0.95,
		Timestamp:     time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := builder.Build(request)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuilder_BatchBuild(b *testing.B) {
	logger := zerolog.Nop()
	templateManager := templates.NewTemplateManager()
	contextManager := context.NewContextManager(logger)

	err := templates.RegisterHEMATemplates()
	require.NoError(b, err)

	builder := NewBuilder(templateManager, contextManager, nil, logger)

	requests := []*BuildRequest{
		{Transcription: "Point left", Confidence: 0.95, Timestamp: time.Now()},
		{Transcription: "Double hit", Confidence: 0.90, Timestamp: time.Now()},
		{Transcription: "Halt", Confidence: 0.85, Timestamp: time.Now()},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = builder.BatchBuild(requests)
	}
}
