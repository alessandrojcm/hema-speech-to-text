package context

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestNewMatchState(t *testing.T) {
	ms := NewMatchState()

	assert.Equal(t, 1, ms.Period)
	assert.Equal(t, 0, ms.ScoreRed)
	assert.Equal(t, 0, ms.ScoreBlue)
	assert.Equal(t, "0-0", ms.CurrentScore)
	assert.Equal(t, "low", ms.MatchIntensity)
	assert.Equal(t, "opening", ms.MatchPhase)
	assert.Equal(t, "neutral", ms.Momentum)
	assert.Equal(t, time.Minute*3, ms.TimeRemaining)
}

func TestMatchState_UpdateScore(t *testing.T) {
	ms := NewMatchState()

	// Test red scorer
	ms.UpdateScore("red")
	assert.Equal(t, 1, ms.ScoreRed)
	assert.Equal(t, 0, ms.ScoreBlue)
	assert.Equal(t, "1-0", ms.CurrentScore)
	assert.Equal(t, "red", ms.LastScorer)
	assert.Equal(t, "red", ms.Momentum)
	assert.Contains(t, ms.LastAction, "Point scored by red")

	// Test blue scorer
	ms.UpdateScore("blue")
	assert.Equal(t, 1, ms.ScoreRed)
	assert.Equal(t, 1, ms.ScoreBlue)
	assert.Equal(t, "1-1", ms.CurrentScore)
	assert.Equal(t, "blue", ms.LastScorer)
	assert.Equal(t, "blue", ms.Momentum)
}

func TestMatchState_UpdateAction(t *testing.T) {
	ms := NewMatchState()

	ms.UpdateAction("Halt called")
	assert.Equal(t, "Halt called", ms.LastAction)
}

func TestMatchState_UpdateTime(t *testing.T) {
	ms := NewMatchState()

	newTime := time.Minute * 2
	ms.UpdateTime(newTime)
	assert.Equal(t, newTime, ms.TimeRemaining)
}

func TestMatchState_IsMatchEnd(t *testing.T) {
	ms := NewMatchState()

	// Not ended initially
	assert.False(t, ms.IsMatchEnd())

	// Ended by left score
	ms.ScoreRed = 5
	assert.True(t, ms.IsMatchEnd())

	// Reset and test right score
	ms.ScoreRed = 0
	ms.ScoreBlue = 5
	assert.True(t, ms.IsMatchEnd())

	// Reset and test time
	ms.ScoreBlue = 0
	ms.TimeRemaining = 0
	assert.True(t, ms.IsMatchEnd())
}

func TestMatchState_GetScoreDifference(t *testing.T) {
	ms := NewMatchState()

	assert.Equal(t, 0, ms.GetScoreDifference())

	ms.ScoreRed = 3
	ms.ScoreBlue = 1
	assert.Equal(t, 2, ms.GetScoreDifference())

	ms.ScoreRed = 1
	ms.ScoreBlue = 4
	assert.Equal(t, 3, ms.GetScoreDifference())
}

func TestMatchState_GetLeader(t *testing.T) {
	ms := NewMatchState()

	assert.Equal(t, "tied", ms.GetLeader())

	ms.ScoreRed = 3
	assert.Equal(t, "red", ms.GetLeader())

	ms.ScoreBlue = 4
	assert.Equal(t, "blue", ms.GetLeader())

	ms.ScoreBlue = 3
	assert.Equal(t, "tied", ms.GetLeader())
}

func TestMatchState_ThreadSafety(t *testing.T) {
	ms := NewMatchState()

	// Test concurrent access
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 10; i++ {
			ms.UpdateScore("red")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			state := ms.GetState()
			assert.NotNil(t, state)
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Verify final state
	assert.Equal(t, 10, ms.ScoreRed)
}

func TestRingBuffer(t *testing.T) {
	rb := NewRingBuffer(3)

	// Test empty buffer
	assert.Empty(t, rb.GetRecent(5))
	assert.Empty(t, rb.GetAll())

	// Add items
	rb.Add("call1")
	rb.Add("call2")
	rb.Add("call3")

	recent := rb.GetRecent(2)
	assert.Len(t, recent, 2)
	assert.Equal(t, "call3", recent[0]) // Most recent first
	assert.Equal(t, "call2", recent[1])

	all := rb.GetAll()
	assert.Len(t, all, 3)
	assert.Equal(t, "call1", all[0]) // Chronological order
	assert.Equal(t, "call2", all[1])
	assert.Equal(t, "call3", all[2])

	// Test buffer overflow
	rb.Add("call4")
	all = rb.GetAll()
	assert.Len(t, all, 3)
	assert.Equal(t, "call2", all[0]) // call1 should be overwritten
	assert.Equal(t, "call3", all[1])
	assert.Equal(t, "call4", all[2])
}

func TestRingBuffer_EdgeCases(t *testing.T) {
	rb := NewRingBuffer(2)

	// Test getting more items than exist
	rb.Add("item1")
	recent := rb.GetRecent(5)
	assert.Len(t, recent, 1)
	assert.Equal(t, "item1", recent[0])

	// Test zero/negative requests
	assert.Empty(t, rb.GetRecent(0))
	assert.Empty(t, rb.GetRecent(-1))
}

func TestNewContextManager(t *testing.T) {
	logger := zerolog.Nop()
	cm := NewContextManager(logger)

	assert.NotNil(t, cm.matchState)
	assert.NotNil(t, cm.recentCalls)
	assert.NotNil(t, cm.contextData)
}

func TestContextManager_MatchState(t *testing.T) {
	logger := zerolog.Nop()
	cm := NewContextManager(logger)

	// Get initial state
	state := cm.GetMatchState()
	assert.NotNil(t, state)
	assert.Equal(t, 0, state.ScoreRed)

	// Update state
	newState := NewMatchState()
	newState.ScoreRed = 3
	newState.ScoreBlue = 2
	cm.UpdateMatchState(newState)

	retrievedState := cm.GetMatchState()
	assert.Equal(t, 3, retrievedState.ScoreRed)
	assert.Equal(t, 2, retrievedState.ScoreBlue)
}

func TestContextManager_Calls(t *testing.T) {
	logger := zerolog.Nop()
	cm := NewContextManager(logger)

	// Add calls
	cm.AddCall("Point red")
	cm.AddCall("Halt")
	cm.AddCall("Point blue")

	recent := cm.GetRecentCalls(2)
	assert.Len(t, recent, 2)
	assert.Equal(t, "Point blue", recent[0])
	assert.Equal(t, "Halt", recent[1])
}

func TestContextManager_Context(t *testing.T) {
	logger := zerolog.Nop()
	cm := NewContextManager(logger)

	// Set context values
	cm.SetContext("test_key", "test_value")
	cm.SetContext("another_key", "another_value")

	// Get individual values
	value, exists := cm.GetContext("test_key")
	assert.True(t, exists)
	assert.Equal(t, "test_value", value)

	_, exists = cm.GetContext("nonexistent_key")
	assert.False(t, exists)

	// Get all context
	allContext := cm.GetAllContext()
	assert.Len(t, allContext, 2)
	assert.Equal(t, "test_value", allContext["test_key"])
	assert.Equal(t, "another_value", allContext["another_key"])
}

func TestContextManager_EnrichContext(t *testing.T) {
	logger := zerolog.Nop()
	cm := NewContextManager(logger)

	// Set up match state
	matchState := NewMatchState()
	matchState.ScoreRed = 3
	matchState.ScoreBlue = 1
	matchState.CurrentScore = "3-1" // Update the current score to match the individual scores
	matchState.TimeRemaining = time.Second * 30
	cm.UpdateMatchState(matchState)

	// Add some recent calls with patterns
	cm.AddCall("double hit")
	cm.AddCall("double hit")
	cm.AddCall("point red")

	enriched := cm.EnrichContext("Point scored")

	// Check enriched context
	assert.Contains(t, enriched, "match_phase")
	assert.Contains(t, enriched, "match_intensity")
	assert.Contains(t, enriched, "momentum")
	assert.Contains(t, enriched, "score_situation")
	assert.Contains(t, enriched, "time_pressure")

	assert.Equal(t, "3-1", enriched["score_situation"])
	assert.Equal(t, "high", enriched["time_pressure"]) // < 1 minute
	assert.Contains(t, enriched, "recent_pattern")
}

func TestContextManager_ThreadSafety(t *testing.T) {
	logger := zerolog.Nop()
	cm := NewContextManager(logger)

	// Test concurrent access to different methods
	done := make(chan bool, 3)

	go func() {
		for i := 0; i < 10; i++ {
			cm.SetContext("key1", "value1")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			cm.AddCall("test call")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			_ = cm.GetAllContext()
			_ = cm.GetRecentCalls(5)
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// Verify no race conditions occurred
	context := cm.GetAllContext()
	assert.Contains(t, context, "key1")

	calls := cm.GetRecentCalls(5)
	assert.NotEmpty(t, calls)
}

// Benchmark tests
func BenchmarkMatchState_UpdateScore(b *testing.B) {
	ms := NewMatchState()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scorer := "red"
		if i%2 == 1 {
			scorer = "blue"
		}
		ms.UpdateScore(scorer)
	}
}

func BenchmarkRingBuffer_Add(b *testing.B) {
	rb := NewRingBuffer(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.Add("test call")
	}
}

func BenchmarkContextManager_EnrichContext(b *testing.B) {
	logger := zerolog.Nop()
	cm := NewContextManager(logger)

	// Setup
	matchState := NewMatchState()
	matchState.ScoreRed = 2
	matchState.ScoreBlue = 1
	cm.UpdateMatchState(matchState)

	for i := 0; i < 5; i++ {
		cm.AddCall("test call")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cm.EnrichContext("Point scored")
	}
}
