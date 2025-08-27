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

	// Vocabulary system removed - using whisper's initial prompt directly

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
