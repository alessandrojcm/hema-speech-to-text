package pipeline

import (
	"fmt"
	"sync"
)

// State represents the current pipeline state
type State int

const (
	StateIdle       State = iota
	StateListening        // Monitoring audio for VAD
	StateProcessing       // Processing detected audio
	StateGenerating       // Generating content
	StateUpdating         // Updating outputs
	StateError
	StateRecovering
)

// String returns string representation of state
func (s State) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateListening:
		return "listening"
	case StateProcessing:
		return "processing"
	case StateGenerating:
		return "generating"
	case StateUpdating:
		return "updating"
	case StateError:
		return "error"
	case StateRecovering:
		return "recovering"
	default:
		return fmt.Sprintf("unknown_state_%d", int(s))
	}
}

// Event represents state transition events
type Event int

const (
	EventStart Event = iota
	EventSpeechDetected
	EventProcessingStart
	EventProcessingComplete
	EventCommentaryReady
	EventOverlayUpdated
	EventError
	EventRecover
)

// String returns string representation of event
func (e Event) String() string {
	switch e {
	case EventStart:
		return "start"
	case EventSpeechDetected:
		return "speech_detected"
	case EventProcessingStart:
		return "processing_start"
	case EventProcessingComplete:
		return "processing_complete"
	case EventCommentaryReady:
		return "commentary_ready"
	case EventOverlayUpdated:
		return "overlay_updated"
	case EventError:
		return "error"
	case EventRecover:
		return "recover"
	default:
		return fmt.Sprintf("unknown_event_%d", int(e))
	}
}

// StateTransition represents a state transition
type StateTransition struct {
	From  State
	Event Event
}

// TransitionFunc represents a state transition function
type TransitionFunc func() (State, error)

// StateManager manages pipeline states
type StateManager struct {
	current     State
	previous    State
	transitions map[StateTransition]TransitionFunc
	mu          sync.RWMutex
}

// NewStateManager creates a new state manager
func NewStateManager() *StateManager {
	sm := &StateManager{
		current:     StateIdle,
		previous:    StateIdle,
		transitions: make(map[StateTransition]TransitionFunc),
	}

	// Define valid state transitions
	sm.defineTransitions()
	return sm
}

// defineTransitions defines the valid state transitions
func (sm *StateManager) defineTransitions() {
	// From Idle
	sm.transitions[StateTransition{StateIdle, EventStart}] = func() (State, error) {
		return StateListening, nil
	}

	// From Listening
	sm.transitions[StateTransition{StateListening, EventSpeechDetected}] = func() (State, error) {
		return StateProcessing, nil
	}

	// From Processing
	sm.transitions[StateTransition{StateProcessing, EventProcessingStart}] = func() (State, error) {
		return StateProcessing, nil // Stay in processing
	}
	sm.transitions[StateTransition{StateProcessing, EventProcessingComplete}] = func() (State, error) {
		return StateListening, nil // Return to listening for continuous processing
	}

	// Error handling from any state
	for _, state := range []State{StateIdle, StateListening, StateProcessing, StateGenerating, StateUpdating} {
		sm.transitions[StateTransition{state, EventError}] = func() (State, error) {
			return StateError, nil
		}
		sm.transitions[StateTransition{state, EventRecover}] = func() (State, error) {
			return StateListening, nil // Recovery goes back to listening
		}
	}

	// From Error
	sm.transitions[StateTransition{StateError, EventRecover}] = func() (State, error) {
		return StateListening, nil
	}
}

// Transition transitions to a new state based on an event
func (sm *StateManager) Transition(event Event) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	transition := StateTransition{
		From:  sm.current,
		Event: event,
	}

	handler, ok := sm.transitions[transition]
	if !ok {
		return fmt.Errorf("invalid transition from %s with event %s", sm.current, event)
	}

	newState, err := handler()
	if err != nil {
		return fmt.Errorf("transition failed: %w", err)
	}

	sm.previous = sm.current
	sm.current = newState
	return nil
}

// Current returns the current state
func (sm *StateManager) Current() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.current
}

// Previous returns the previous state
func (sm *StateManager) Previous() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.previous
}
