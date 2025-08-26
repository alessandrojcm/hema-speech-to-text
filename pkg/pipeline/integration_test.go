//go:build noaudio
// +build noaudio

package pipeline

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	audioTypes "github.com/your-org/hema-replay-system/pkg/audio/types"
	"github.com/your-org/hema-replay-system/pkg/pipeline/vad"
	speechTypes "github.com/your-org/hema-replay-system/pkg/speech/types"
)

// TestPipelineComponents_Standalone tests pipeline components that don't need audio system
func TestPipelineComponents_Standalone(t *testing.T) {
	// Test EventBus
	t.Run("EventBus", func(t *testing.T) {
		eventBus := NewEventBus()
		require.NotNil(t, eventBus)

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
	})

	// Test SegmentBuffer
	t.Run("SegmentBuffer", func(t *testing.T) {
		buffer := NewSegmentBuffer(5)
		require.NotNil(t, buffer)

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

		// Test metadata propagation
		assert.Equal(t, vad.EventSpeechSegment, pending[0].VADEvent.Type)
		assert.Equal(t, time.Second, pending[0].VADEvent.Duration)

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
	})

	// Test MetricsCollector
	t.Run("MetricsCollector", func(t *testing.T) {
		metrics := NewMetricsCollector()
		require.NotNil(t, metrics)

		// Test recording processing time
		metrics.RecordProcessingTime("audio_extraction", 100*time.Millisecond)
		metrics.RecordProcessingTime("speech_processing", 200*time.Millisecond)

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
		assert.Contains(t, stats, "audio_extraction_avg_ms")
		assert.Contains(t, stats, "speech_processing_avg_ms")
		assert.Contains(t, stats, "segments_processed")
		assert.Contains(t, stats, "vad_detections")

		// Verify calculated values
		assert.Equal(t, 0.5, stats["success_rate"])
		assert.Equal(t, int64(100), stats["audio_extraction_avg_ms"])
		assert.Equal(t, int64(200), stats["speech_processing_avg_ms"])
		assert.Equal(t, int64(1), stats["segments_processed"])
		assert.Equal(t, int64(2), stats["vad_detections"])
	})

	// Test StateManager
	t.Run("StateManager", func(t *testing.T) {
		sm := NewStateManager()
		require.NotNil(t, sm)

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
	})

	// Test Configuration
	t.Run("Configuration", func(t *testing.T) {
		config := &PipelineManagerConfig{
			Speech: speechTypes.SpeechConfig{
				Whisper: speechTypes.WhisperConfig{
					ModelSize: speechTypes.ModelBase,
					Language:  "en",
				},
			},
		}
		config.SetDefaults()

		// Test that defaults were set
		assert.Greater(t, config.Pipeline.MaxConcurrentRequests, 0)
		assert.Greater(t, config.Pipeline.ProcessingTimeout, time.Duration(0))
		assert.Greater(t, config.Pipeline.SegmentBufferSize, 0)
		assert.True(t, config.Pipeline.MetricsEnabled)
		assert.NotNil(t, config.VAD)
		assert.Greater(t, config.VAD.MinSpeechDurationMs, 0)

		// Test validation with valid config
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
	})
}

// TestMetadataPropagation tests that VAD metadata flows through the pipeline correctly
func TestMetadataPropagation(t *testing.T) {
	// Test AudioSegmentData structure for metadata propagation
	vadEvent := vad.VADEvent{
		Type:        vad.EventSpeechSegment,
		StartTime:   time.Now(),
		Duration:    2 * time.Second,
		Confidence:  0.85,
		BufferStart: time.Now().Add(-5 * time.Second),
		BufferEnd:   time.Now(),
	}

	segment := &audioTypes.AudioSegment{
		ID:        "segment_test_1",
		Data:      make([]float32, 1000),
		StartTime: time.Now(),
		Duration:  2 * time.Second,
	}

	// Test that AudioSegmentData preserves VAD metadata
	segmentData := &AudioSegmentData{
		SegmentID: "test_123",
		Segment:   segment,
		VADEvent:  vadEvent,
	}

	assert.Equal(t, "test_123", segmentData.SegmentID)
	assert.Equal(t, vad.EventSpeechSegment, segmentData.VADEvent.Type)
	assert.Equal(t, 2*time.Second, segmentData.VADEvent.Duration)
	assert.Equal(t, float32(0.85), segmentData.VADEvent.Confidence)

	// Test that TranscriptData preserves VAD metadata
	transcriptResult := &speechTypes.TranscriptionResult{
		ID:         "result_123",
		Text:       "Hello world",
		Confidence: 0.92,
	}

	transcriptData := TranscriptData{
		SegmentID: "test_123",
		Result:    transcriptResult,
		VADEvent:  vadEvent,
	}

	assert.Equal(t, "test_123", transcriptData.SegmentID)
	assert.Equal(t, "Hello world", transcriptData.Result.Text)
	assert.Equal(t, vad.EventSpeechSegment, transcriptData.VADEvent.Type)
	assert.Equal(t, 2*time.Second, transcriptData.VADEvent.Duration)
}

// TestEventTypesAndFlow tests the complete event flow without external dependencies
func TestEventTypesAndFlow(t *testing.T) {
	// Test that all expected event types are defined
	expectedEvents := []EventType{
		EventTypeVADSpeechStart,
		EventTypeVADSpeechEnd,
		EventTypeVADSpeechSegment,
		EventTypeAudioSegmentReady,
		EventTypeTranscriptReady,
		EventTypeCommentaryReady,
		EventTypeOverlayUpdated,
		EventTypeError,
	}

	for _, eventType := range expectedEvents {
		assert.NotEmpty(t, string(eventType), "Event type should have non-empty string value")
	}

	// Test event publishing and subscription
	eventBus := NewEventBus()

	// Track received events with mutex for concurrent safety
	var mu sync.Mutex
	receivedEvents := make(map[EventType]int)

	// Subscribe to specific event types to test
	testEvents := []EventType{
		EventTypeVADSpeechStart,
		EventTypeVADSpeechEnd,
		EventTypeVADSpeechSegment,
	}

	for _, eventType := range testEvents {
		eventBus.Subscribe(eventType, func(event PipelineEvent) {
			mu.Lock()
			receivedEvents[event.Type]++
			mu.Unlock()
		})
	}

	// Publish each test event
	for _, eventType := range testEvents {
		event := PipelineEvent{
			Type:      eventType,
			Timestamp: time.Now(),
			Data:      "test_data",
		}
		eventBus.Publish(event)
	}

	// Wait for all events to be processed
	time.Sleep(50 * time.Millisecond)

	// Verify events were received
	mu.Lock()
	for _, eventType := range testEvents {
		assert.Equal(t, 1, receivedEvents[eventType], "Each event type should be received exactly once")
	}
	mu.Unlock()

	// Test subscriber count
	for _, eventType := range testEvents {
		assert.Equal(t, 1, eventBus.GetSubscriberCount(eventType))
	}
}
