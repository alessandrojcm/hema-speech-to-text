package engine

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	speechTypes "github.com/your-org/hema-replay-system/pkg/speech/types"
)

func createTestProcessingPipeline(t *testing.T) *ProcessingPipeline {
	config := speechTypes.SpeechConfig{
		Processing: speechTypes.ProcessingConfig{
			TargetSampleRate: 16000,
			SegmentDuration:  2 * time.Second,
			OverlapDuration:  200 * time.Millisecond,
			NoiseReduction:   true,
			Normalization:    true,
			VADEnabled:       true,
		},
	}

	logger := zerolog.New(zerolog.NewTestWriter(t))
	pipeline, err := NewProcessingPipeline(config, logger)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	return pipeline
}

func TestNewProcessingPipeline(t *testing.T) {
	config := speechTypes.SpeechConfig{
		Processing: speechTypes.ProcessingConfig{
			TargetSampleRate: 16000,
			SegmentDuration:  2 * time.Second,
		},
	}

	logger := zerolog.New(zerolog.NewTestWriter(t))
	pipeline, err := NewProcessingPipeline(config, logger)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	assert.NotNil(t, pipeline)
	assert.Equal(t, config, pipeline.config)
}

func TestProcessingPipeline_SetDependencies(t *testing.T) {
	pipeline := createTestProcessingPipeline(t)

	// Test setting model manager (would be nil without actual whisper setup)
	pipeline.SetModelManager(nil)

	// Test setting vocabulary (would be nil without actual vocabulary setup)
	pipeline.SetVocabulary(nil)

	// Pipeline should handle nil dependencies gracefully
	assert.NotNil(t, pipeline)
}

func TestProcessingPipeline_Process_NilRequest(t *testing.T) {
	pipeline := createTestProcessingPipeline(t)

	ctx := context.Background()
	result, err := pipeline.Process(ctx, speechTypes.TranscriptionRequest{})

	// Should handle empty request gracefully
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestProcessingPipeline_Close(t *testing.T) {
	pipeline := createTestProcessingPipeline(t)

	err := pipeline.Close()

	assert.NoError(t, err)
}

func TestProcessingPipeline_ApplyBoost(t *testing.T) {
	pipeline := createTestProcessingPipeline(t)

	tests := []struct {
		name       string
		confidence float64
		boost      float64
		expected   float64
	}{
		{
			name:       "normal boost",
			confidence: 0.8,
			boost:      1.2,
			expected:   0.96,
		},
		{
			name:       "boost over 1.0 clamped",
			confidence: 0.9,
			boost:      1.5,
			expected:   1.0,
		},
		{
			name:       "no boost",
			confidence: 0.7,
			boost:      1.0,
			expected:   0.7,
		},
		{
			name:       "reduce confidence",
			confidence: 0.8,
			boost:      0.5,
			expected:   0.4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pipeline.applyBoost(tt.confidence, tt.boost)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestProcessingPipeline_RecalculateSegmentConfidence(t *testing.T) {
	pipeline := createTestProcessingPipeline(t)

	segment := &speechTypes.TranscriptionSegment{
		Text:       "hello world",
		Confidence: 0.5, // Will be recalculated
		Tokens: []speechTypes.Token{
			{Text: "hello", Confidence: 0.8},
			{Text: "world", Confidence: 0.9},
		},
	}

	pipeline.recalculateSegmentConfidence(segment)

	expectedConfidence := (0.8 + 0.9) / 2.0
	assert.InDelta(t, expectedConfidence, segment.Confidence, 0.001)
}

func TestProcessingPipeline_RecalculateSegmentConfidence_EmptyTokens(t *testing.T) {
	pipeline := createTestProcessingPipeline(t)

	segment := &speechTypes.TranscriptionSegment{
		Text:       "hello world",
		Confidence: 0.5,
		Tokens:     []speechTypes.Token{},
	}

	originalConfidence := segment.Confidence
	pipeline.recalculateSegmentConfidence(segment)

	// Should not change confidence if no tokens
	assert.Equal(t, originalConfidence, segment.Confidence)
}

func TestProcessingPipeline_RecalculateOverallConfidence(t *testing.T) {
	pipeline := createTestProcessingPipeline(t)

	result := &speechTypes.TranscriptionResult{
		Text:       "hello world test",
		Confidence: 0.5, // Will be recalculated
		Segments: []speechTypes.TranscriptionSegment{
			{Text: "hello", Confidence: 0.8},
			{Text: "world", Confidence: 0.9},
			{Text: "test", Confidence: 0.7},
		},
	}

	pipeline.recalculateOverallConfidence(result)

	expectedConfidence := (0.8 + 0.9 + 0.7) / 3.0
	assert.InDelta(t, expectedConfidence, result.Confidence, 0.001)
}

func TestProcessingPipeline_ContextCancellation(t *testing.T) {
	pipeline := createTestProcessingPipeline(t)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	request := speechTypes.TranscriptionRequest{
		ID:       "test-1",
		Language: "en",
	}

	result, err := pipeline.Process(ctx, request)

	assert.Error(t, err)
	assert.Nil(t, result)
	// Note: The actual error will depend on where context cancellation is checked
}

func TestProcessingPipeline_Timeout(t *testing.T) {
	pipeline := createTestProcessingPipeline(t)

	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	request := speechTypes.TranscriptionRequest{
		ID:       "test-1",
		Language: "en",
	}

	// Wait for timeout
	time.Sleep(10 * time.Millisecond)

	result, err := pipeline.Process(ctx, request)

	assert.Error(t, err)
	assert.Nil(t, result)
}
