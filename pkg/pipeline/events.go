package pipeline

import (
	"sync"
	"time"

	"github.com/your-org/hema-replay-system/pkg/audio/types"
	"github.com/your-org/hema-replay-system/pkg/pipeline/vad"
	speechTypes "github.com/your-org/hema-replay-system/pkg/speech/types"
)

// EventType represents different types of pipeline events
type EventType string

const (
	EventTypeVADSpeechStart    EventType = "vad.speech.start"
	EventTypeVADSpeechEnd      EventType = "vad.speech.end"
	EventTypeVADSpeechSegment  EventType = "vad.speech.segment"
	EventTypeAudioSegmentReady EventType = "audio.segment.ready"
	EventTypeTranscriptReady   EventType = "transcript.ready"
	EventTypeCommentaryReady   EventType = "commentary.ready"
	EventTypeOverlayUpdated    EventType = "overlay.updated"
	EventTypeError             EventType = "error"
)

// PipelineEvent represents an event in the pipeline
type PipelineEvent struct {
	Type      EventType
	Timestamp time.Time
	Data      interface{}
	Error     error
}

// AudioSegmentData holds data for audio segment events
type AudioSegmentData struct {
	SegmentID string
	Segment   *types.AudioSegment
	VADEvent  vad.VADEvent
}

// TranscriptData holds data for transcript events
type TranscriptData struct {
	SegmentID string
	Result    *speechTypes.TranscriptionResult
	VADEvent  vad.VADEvent
}

// EventHandler represents an event handler function
type EventHandler func(event PipelineEvent)

// EventBus manages event publishing and subscription
type EventBus struct {
	subscribers map[EventType][]EventHandler
	mu          sync.RWMutex
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[EventType][]EventHandler),
	}
}

// Subscribe subscribes a handler to an event type
func (eb *EventBus) Subscribe(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)
}

// Publish publishes an event to all subscribers
func (eb *EventBus) Publish(event PipelineEvent) {
	eb.mu.RLock()
	handlers := eb.subscribers[event.Type]
	eb.mu.RUnlock()

	// Run handlers concurrently to avoid blocking
	for _, handler := range handlers {
		go handler(event)
	}
}

// Unsubscribe removes all handlers for an event type
func (eb *EventBus) Unsubscribe(eventType EventType) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	delete(eb.subscribers, eventType)
}

// GetSubscriberCount returns the number of subscribers for an event type
func (eb *EventBus) GetSubscriberCount(eventType EventType) int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return len(eb.subscribers[eventType])
}
