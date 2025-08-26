//go:build !noaudio
// +build !noaudio

package pipeline

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	audioTypes "github.com/your-org/hema-replay-system/pkg/audio/types"
	"github.com/your-org/hema-replay-system/pkg/pipeline/vad"
	speechTypes "github.com/your-org/hema-replay-system/pkg/speech/types"
)

func TestPipelineManagerIntegration(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()

	// Create test audio configuration
	audioConfig := audioTypes.AudioConfig{
			Device: audioTypes.DeviceConfig{
				SampleRate: 16000,
				Channels:   1,
				BitDepth:   16,
			},
			Buffer: audioTypes.BufferConfig{
				Duration:    60 * time.Second,
				SegmentSize: time.Second,
			},
			Processing: audioTypes.ProcessingConfig{
				VADType: "webrtc",
			},
			Extraction: audioTypes.ExtractionConfig{
				DefaultDuration: 5 * time.Second,
				MaxConcurrent:   2,
			},
		},
		Speech: speechTypes.SpeechConfig{
			Whisper: speechTypes.WhisperConfig{
				ModelSize:   speechTypes.ModelBase,
				Language:    "en",
				ThreadCount: 2,
			},
			Performance: speechTypes.PerformanceConfig{
				MaxConcurrent:   2,
				CacheSize:       10,
				CacheTTL:        time.Minute,
				TimeoutDuration: 10 * time.Second,
			},
		},
		VAD: &vad.Config{
			MinSpeechDurationMs:  300, // 300ms minimum
			MaxSilenceDurationMs: 800, // 800ms max silence
			VADMode:              1,   // Less aggressive
			BufferBeforeMs:       50,  // 50ms before
			BufferAfterMs:        100, // 100ms after
		},
		Pipeline: PipelineConfig{
			MaxConcurrentRequests: 2,
			ProcessingTimeout:     10 * time.Second,
			SegmentBufferSize:     20,
			MetricsEnabled:        true,
			MetricsInterval:       time.Second,
		},
	}

	// Set defaults
	config.SetDefaults()

	// Validate configuration
	err := config.Validate()
	require.NoError(t, err)

	// Create pipeline manager
	manager, err := NewManager(config, logger)
	require.NoError(t, err)
	require.NotNil(t, manager)

	// Test configuration
	assert.Equal(t, config, manager.config)
	assert.NotNil(t, manager.audioManager)
	assert.NotNil(t, manager.speechManager)
	assert.NotNil(t, manager.vadDetector)
	assert.NotNil(t, manager.state)
	assert.NotNil(t, manager.eventBus)
	assert.NotNil(t, manager.metrics)
	assert.NotNil(t, manager.segmentBuffer)
}

func TestPipelineManagerLifecycle(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()

	config := &PipelineManagerConfig{}
	config.SetDefaults()

	manager, err := NewManager(config, logger)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test start
	err = manager.Start(ctx)
	assert.NoError(t, err)
	assert.Equal(t, StateListening, manager.GetState())

	// Test state management
	assert.Equal(t, StateListening, manager.state.Current())

	// Test metrics
	metrics := manager.GetMetrics()
	assert.NotNil(t, metrics)
	assert.Contains(t, metrics, "pipeline_running")
	assert.True(t, metrics["pipeline_running"].(bool))

	// Test stop
	err = manager.Stop()
	assert.NoError(t, err)
}

func TestEventBusIntegration(t *testing.T) {
	eventBus := NewEventBus()

	// Test event subscription and publishing
	var receivedEvent PipelineEvent
	received := false

	eventBus.Subscribe(EventTypeVADSpeechStart, func(event PipelineEvent) {
		receivedEvent = event
		received = true
	})

	// Publish test event
	testEvent := PipelineEvent{
		Type:      EventTypeVADSpeechStart,
		Timestamp: time.Now(),
		Data:      "test_data",
	}

	eventBus.Publish(testEvent)

	// Wait briefly for goroutine
	time.Sleep(10 * time.Millisecond)

	assert.True(t, received)
	assert.Equal(t, EventTypeVADSpeechStart, receivedEvent.Type)
	assert.Equal(t, "test_data", receivedEvent.Data)
}

func TestSegmentBufferManagement(t *testing.T) {
	buffer := NewSegmentBuffer(5)

	// Create test audio segment
	segment := &audioTypes.AudioSegment{
		ID:        "test_1",
		Data:      make([]float32, 1000),
		StartTime: time.Now(),
		Duration:  time.Second,
	}

	// Create test VAD event
	vadEvent := vad.VADEvent{
		Type:      vad.EventSpeechSegment,
		StartTime: time.Now(),
		Duration:  time.Second,
	}

	// Test adding segment
	segmentID := buffer.Add(segment, vadEvent)
	assert.NotEmpty(t, segmentID)
	assert.Equal(t, 1, buffer.GetPendingCount())
	assert.Equal(t, 0, buffer.GetProcessedCount())

	// Test getting pending segments
	pending := buffer.GetPending()
	assert.Len(t, pending, 1)
	assert.Equal(t, segmentID, pending[0].SegmentID)

	// Test processing segment
	err := buffer.MarkProcessing(segmentID)
	assert.NoError(t, err)

	// Create mock transcription result
	result := &speechTypes.TranscriptionResult{
		ID:         "result_1",
		Text:       "Test transcription",
		Confidence: 0.9,
	}

	// Test marking as processed
	err = buffer.MarkProcessed(segmentID, result)
	assert.NoError(t, err)
	assert.Equal(t, 0, buffer.GetPendingCount())
	assert.Equal(t, 1, buffer.GetProcessedCount())

	// Test getting processed results
	results := buffer.GetProcessedResults()
	assert.Len(t, results, 1)
	assert.Equal(t, "Test transcription", results[0].Text)
}

func TestMetricsCollection(t *testing.T) {
	metrics := NewMetricsCollector()

	// Test recording processing time
	metrics.RecordProcessingTime("test_stage", 100*time.Millisecond)
	metrics.RecordProcessingTime("test_stage", 200*time.Millisecond)

	// Test recording success/failure
	metrics.RecordSuccess()
	metrics.RecordError(nil)

	// Test segment metrics
	metrics.RecordSegmentProcessed(true)
	metrics.RecordSegmentProcessed(false)

	// Test VAD metrics
	metrics.RecordVADDetection(true, false)
	metrics.RecordVADDetection(true, true)

	// Test confidence score
	metrics.UpdateConfidenceScore(0.8)

	// Get stats and verify
	stats := metrics.GetStats()
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "success_rate")
	assert.Contains(t, stats, "test_stage_avg_ms")
	assert.Contains(t, stats, "segments_processed")
	assert.Contains(t, stats, "vad_detections")

	// Verify calculated values
	assert.Equal(t, 0.5, stats["success_rate"])
	assert.Equal(t, int64(150), stats["test_stage_avg_ms"])
	assert.Equal(t, int64(1), stats["segments_processed"])
	assert.Equal(t, int64(2), stats["vad_detections"])
}

func TestStateTransitions(t *testing.T) {
	sm := NewStateManager()

	// Test initial state
	assert.Equal(t, StateIdle, sm.Current())

	// Test valid transition
	err := sm.Transition(EventStart)
	assert.NoError(t, err)
	assert.Equal(t, StateListening, sm.Current())

	// Test another transition
	err = sm.Transition(EventSpeechDetected)
	assert.NoError(t, err)
	assert.Equal(t, StateProcessing, sm.Current())

	// Test invalid transition
	err = sm.Transition(EventStart)
	assert.Error(t, err)

	// Test error handling
	err = sm.Transition(EventError)
	assert.NoError(t, err)
	assert.Equal(t, StateError, sm.Current())

	// Test recovery
	err = sm.Transition(EventRecover)
	assert.NoError(t, err)
	assert.Equal(t, StateListening, sm.Current())
}

func TestConfigurationValidation(t *testing.T) {
	// Test valid configuration
	config := &PipelineManagerConfig{}
	config.SetDefaults()
	err := config.Validate()
	assert.NoError(t, err)

	// Test invalid configuration - invalid model size
	config.Speech.Whisper.ModelSize = speechTypes.ModelSize(999) // Invalid value
	err = config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "speech model size is invalid")

	// Reset to valid
	config.Speech.Whisper.ModelSize = speechTypes.ModelBase

	// Test invalid VAD config
	config.VAD.MinSpeechDurationMs = -1
	err = config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "VAD min speech duration must be positive")

	// Test invalid pipeline config
	config.VAD.MinSpeechDurationMs = 500
	config.Pipeline.MaxConcurrentRequests = 0
	err = config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max concurrent requests must be positive")
}
