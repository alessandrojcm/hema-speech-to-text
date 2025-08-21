package context

import (
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// MatchState represents the current state of a HEMA match
type MatchState struct {
	mu             sync.RWMutex
	ScoreRed       int           `json:"score_red"`
	ScoreBlue      int           `json:"score_blue"`
	Period         int           `json:"period"`
	TimeRemaining  time.Duration `json:"time_remaining"`
	LastAction     string        `json:"last_action"`
	LastScorer     string        `json:"last_scorer"` // "red", "blue", or ""
	CurrentScore   string        `json:"current_score"`
	MatchIntensity string        `json:"match_intensity"` // "low", "medium", "high"
	TotalDuration  time.Duration `json:"total_duration"`
	StartTime      time.Time     `json:"start_time"`
	LastUpdateTime time.Time     `json:"last_update_time"`
	MatchPhase     string        `json:"match_phase"` // "opening", "middle", "closing"
	Momentum       string        `json:"momentum"`    // "red", "blue", "neutral"
}

// NewMatchState creates a new match state with default values
func NewMatchState() *MatchState {
	now := time.Now()
	return &MatchState{
		Period:         1,
		TimeRemaining:  time.Minute * 3, // Standard HEMA bout time
		CurrentScore:   "0-0",
		MatchIntensity: "low",
		StartTime:      now,
		LastUpdateTime: now,
		MatchPhase:     "opening",
		Momentum:       "neutral",
	}
}

// UpdateScore updates the match score and related state
func (ms *MatchState) UpdateScore(scorer string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.LastScorer = scorer
	ms.LastAction = fmt.Sprintf("Point scored by %s", scorer)
	ms.LastUpdateTime = time.Now()

	switch scorer {
	case "red":
		ms.ScoreRed++
	case "blue":
		ms.ScoreBlue++
	}

	ms.CurrentScore = fmt.Sprintf("%d-%d", ms.ScoreRed, ms.ScoreBlue)
	ms.updateMatchIntensity()
	ms.updateMatchPhase()
	ms.updateMomentum()
}

// UpdateAction updates the last action without scoring
func (ms *MatchState) UpdateAction(action string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.LastAction = action
	ms.LastUpdateTime = time.Now()
	ms.updateMatchIntensity()
}

// UpdateTime updates the remaining time in the match
func (ms *MatchState) UpdateTime(remaining time.Duration) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.TimeRemaining = remaining
	ms.LastUpdateTime = time.Now()
	ms.updateMatchPhase()
}

// GetState returns a thread-safe copy of the current match state
func (ms *MatchState) GetState() MatchState {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return *ms
}

// IsMatchEnd returns true if the match should end
func (ms *MatchState) IsMatchEnd() bool {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	// Standard HEMA rules: first to 5 points or time runs out
	return ms.ScoreRed >= 5 || ms.ScoreBlue >= 5 || ms.TimeRemaining <= 0
}

// GetScoreDifference returns the absolute score difference
func (ms *MatchState) GetScoreDifference() int {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if ms.ScoreRed > ms.ScoreBlue {
		return ms.ScoreRed - ms.ScoreBlue
	}
	return ms.ScoreBlue - ms.ScoreRed
}

// GetLeader returns who is currently leading
func (ms *MatchState) GetLeader() string {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if ms.ScoreRed > ms.ScoreBlue {
		return "red"
	} else if ms.ScoreBlue > ms.ScoreRed {
		return "blue"
	}
	return "tied"
}

// updateMatchIntensity updates the match intensity based on score and time
// Note: This method assumes the caller already holds a lock
func (ms *MatchState) updateMatchIntensity() {
	scoreDiff := ms.getScoreDifferenceUnsafe()
	timeElapsed := time.Since(ms.StartTime)
	totalTime := ms.TotalDuration
	if totalTime == 0 {
		totalTime = time.Minute * 3 // Default match time
	}

	// Calculate intensity based on multiple factors
	if ms.TimeRemaining < time.Minute && scoreDiff <= 1 {
		ms.MatchIntensity = "high"
	} else if scoreDiff >= 3 || timeElapsed < time.Second*30 {
		ms.MatchIntensity = "low"
	} else {
		ms.MatchIntensity = "medium"
	}
}

// getScoreDifferenceUnsafe returns the absolute score difference without locking
// Note: This method assumes the caller already holds a lock
func (ms *MatchState) getScoreDifferenceUnsafe() int {
	if ms.ScoreRed > ms.ScoreBlue {
		return ms.ScoreRed - ms.ScoreBlue
	}
	return ms.ScoreBlue - ms.ScoreRed
}

// updateMatchPhase updates the current phase of the match
func (ms *MatchState) updateMatchPhase() {
	totalTime := ms.TotalDuration
	if totalTime == 0 {
		totalTime = time.Minute * 3 // Default match time
	}

	elapsed := time.Since(ms.StartTime)

	if elapsed < totalTime/3 {
		ms.MatchPhase = "opening"
	} else if elapsed < totalTime*2/3 {
		ms.MatchPhase = "middle"
	} else {
		ms.MatchPhase = "closing"
	}
}

// updateMomentum updates the momentum based on recent scoring
func (ms *MatchState) updateMomentum() {
	// Simple momentum calculation based on last scorer
	// In a more sophisticated system, this would track scoring patterns
	if ms.LastScorer == "red" {
		ms.Momentum = "red"
	} else if ms.LastScorer == "blue" {
		ms.Momentum = "blue"
	} else {
		ms.Momentum = "neutral"
	}
}

// RingBuffer represents a circular buffer for recent calls
type RingBuffer struct {
	mu     sync.RWMutex
	buffer []string
	size   int
	index  int
	full   bool
}

// NewRingBuffer creates a new ring buffer with the specified size
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		buffer: make([]string, size),
		size:   size,
	}
}

// Add adds a new item to the ring buffer
func (rb *RingBuffer) Add(item string) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.buffer[rb.index] = item
	rb.index = (rb.index + 1) % rb.size
	if rb.index == 0 {
		rb.full = true
	}
}

// GetRecent returns the most recent items (up to count)
func (rb *RingBuffer) GetRecent(count int) []string {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if count <= 0 {
		return nil
	}

	actualSize := rb.size
	if !rb.full {
		actualSize = rb.index
	}

	if count > actualSize {
		count = actualSize
	}

	result := make([]string, count)

	for i := 0; i < count; i++ {
		pos := (rb.index - 1 - i + rb.size) % rb.size
		if pos >= 0 && pos < len(rb.buffer) {
			result[i] = rb.buffer[pos]
		}
	}

	return result
}

// GetAll returns all items in the buffer in chronological order
func (rb *RingBuffer) GetAll() []string {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if !rb.full && rb.index == 0 {
		return nil
	}

	size := rb.size
	if !rb.full {
		size = rb.index
	}

	result := make([]string, size)

	if rb.full {
		// Copy from current index to end
		copy(result, rb.buffer[rb.index:])
		// Copy from beginning to current index
		copy(result[rb.size-rb.index:], rb.buffer[:rb.index])
	} else {
		// Copy from beginning to current index
		copy(result, rb.buffer[:rb.index])
	}

	return result
}

// ContextManager manages match context and call history
type ContextManager struct {
	mu          sync.RWMutex
	matchState  *MatchState
	recentCalls *RingBuffer
	logger      zerolog.Logger
	contextData map[string]string
}

// NewContextManager creates a new context manager
func NewContextManager(logger zerolog.Logger) *ContextManager {
	return &ContextManager{
		matchState:  NewMatchState(),
		recentCalls: NewRingBuffer(10), // Keep last 10 calls
		logger:      logger.With().Str("component", "context").Logger(),
		contextData: make(map[string]string),
	}
}

// UpdateMatchState updates the current match state
func (cm *ContextManager) UpdateMatchState(state *MatchState) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.matchState = state
}

// GetMatchState returns a thread-safe copy of the match state
func (cm *ContextManager) GetMatchState() *MatchState {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.matchState == nil {
		return NewMatchState()
	}

	state := cm.matchState.GetState()
	return &state
}

// AddCall adds a new call to the recent calls buffer
func (cm *ContextManager) AddCall(call string) {
	cm.recentCalls.Add(call)
	cm.logger.Debug().Str("call", call).Msg("Added call to context")
}

// GetRecentCalls returns recent calls from the buffer
func (cm *ContextManager) GetRecentCalls(count int) []string {
	return cm.recentCalls.GetRecent(count)
}

// SetContext sets a context value
func (cm *ContextManager) SetContext(key, value string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.contextData[key] = value
}

// GetContext gets a context value
func (cm *ContextManager) GetContext(key string) (string, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	value, exists := cm.contextData[key]
	return value, exists
}

// GetAllContext returns all context data
func (cm *ContextManager) GetAllContext() map[string]string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	result := make(map[string]string)
	for k, v := range cm.contextData {
		result[k] = v
	}
	return result
}

// EnrichContext enriches context based on current match state and call history
func (cm *ContextManager) EnrichContext(transcription string) map[string]string {
	context := cm.GetAllContext()
	state := cm.GetMatchState()

	// Add match phase information
	context["match_phase"] = state.MatchPhase
	context["match_intensity"] = state.MatchIntensity
	context["momentum"] = state.Momentum

	// Add score situation
	context["score_situation"] = state.CurrentScore
	if state.IsMatchEnd() {
		context["match_status"] = "ending"
	}

	// Add time context
	if state.TimeRemaining < time.Minute {
		context["time_pressure"] = "high"
	} else if state.TimeRemaining < time.Minute*2 {
		context["time_pressure"] = "medium"
	} else {
		context["time_pressure"] = "low"
	}

	// Analyze recent call patterns
	recentCalls := cm.GetRecentCalls(3)
	if len(recentCalls) > 1 {
		// Look for patterns in recent calls
		if containsPattern(recentCalls, "double") {
			context["recent_pattern"] = "double_hits"
		} else if containsPattern(recentCalls, "point") {
			context["recent_pattern"] = "active_scoring"
		}
	}

	return context
}

// containsPattern checks if recent calls contain a specific pattern
func containsPattern(calls []string, pattern string) bool {
	count := 0
	for _, call := range calls {
		if contains(call, pattern) {
			count++
		}
	}
	return count >= 2 // Pattern if appears in 2+ recent calls
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstring(s, substr))))
}

// containsSubstring performs a simple substring search
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
