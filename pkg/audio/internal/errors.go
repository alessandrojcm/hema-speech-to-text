package internal

import (
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

type ErrorHandler struct {
	logger        zerolog.Logger
	errorCounts   map[string]int
	lastErrors    map[string]time.Time
	recoveryFuncs map[string]func() error
	mu            sync.RWMutex
}

func NewErrorHandler(logger zerolog.Logger) *ErrorHandler {
	return &ErrorHandler{
		logger:        logger.With().Str("component", "error_handler").Logger(),
		errorCounts:   make(map[string]int),
		lastErrors:    make(map[string]time.Time),
		recoveryFuncs: make(map[string]func() error),
	}
}

func (eh *ErrorHandler) HandleError(err error, context string) error {
	eh.mu.Lock()
	defer eh.mu.Unlock()

	errorType := eh.classifyError(err)
	eh.errorCounts[errorType]++
	eh.lastErrors[errorType] = time.Now()

	eh.logger.Error().
		Err(err).
		Str("context", context).
		Str("error_type", errorType).
		Int("count", eh.errorCounts[errorType]).
		Msg("Audio system error")

	if recoveryFunc, exists := eh.recoveryFuncs[errorType]; exists {
		if recoveryErr := recoveryFunc(); recoveryErr != nil {
			eh.logger.Error().
				Err(recoveryErr).
				Str("error_type", errorType).
				Msg("Error recovery failed")
			return fmt.Errorf("recovery failed: %w", recoveryErr)
		}

		eh.logger.Info().
			Str("error_type", errorType).
			Msg("Error recovery successful")
	}

	return err
}

func (eh *ErrorHandler) classifyError(err error) string {
	switch err {
	case types.ErrDeviceNotFound:
		return "device_not_found"
	case types.ErrDeviceUnavailable:
		return "device_unavailable"
	case types.ErrBufferOverrun:
		return "buffer_overrun"
	case types.ErrBufferUnderrun:
		return "buffer_underrun"
	case types.ErrStreamClosed:
		return "stream_closed"
	case types.ErrInsufficientData:
		return "insufficient_data"
	default:
		return "unknown"
	}
}

func (eh *ErrorHandler) RegisterRecoveryFunc(errorType string, recoveryFunc func() error) {
	eh.mu.Lock()
	defer eh.mu.Unlock()
	eh.recoveryFuncs[errorType] = recoveryFunc
}

func (eh *ErrorHandler) GetErrorStats() map[string]ErrorStats {
	eh.mu.RLock()
	defer eh.mu.RUnlock()

	stats := make(map[string]ErrorStats)
	for errorType, count := range eh.errorCounts {
		stats[errorType] = ErrorStats{
			Count:     count,
			LastError: eh.lastErrors[errorType],
		}
	}

	return stats
}

func (eh *ErrorHandler) Reset() {
	eh.mu.Lock()
	defer eh.mu.Unlock()

	eh.errorCounts = make(map[string]int)
	eh.lastErrors = make(map[string]time.Time)
}

type ErrorStats struct {
	Count     int
	LastError time.Time
}

type CircuitBreaker struct {
	mu           sync.RWMutex
	failureCount int
	lastFailure  time.Time
	state        CircuitState
	timeout      time.Duration
	threshold    int
	logger       zerolog.Logger
}

type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

func NewCircuitBreaker(threshold int, timeout time.Duration, logger zerolog.Logger) *CircuitBreaker {
	return &CircuitBreaker{
		threshold: threshold,
		timeout:   timeout,
		state:     CircuitClosed,
		logger:    logger.With().Str("component", "circuit_breaker").Logger(),
	}
}

func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == CircuitOpen {
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.state = CircuitHalfOpen
			cb.logger.Info().Msg("Circuit breaker transitioning to half-open")
		} else {
			return fmt.Errorf("circuit breaker is open")
		}
	}

	err := fn()
	if err != nil {
		cb.failureCount++
		cb.lastFailure = time.Now()

		if cb.failureCount >= cb.threshold {
			cb.state = CircuitOpen
			cb.logger.Warn().
				Int("failure_count", cb.failureCount).
				Msg("Circuit breaker opened due to failures")
		}

		return err
	}

	if cb.state == CircuitHalfOpen {
		cb.state = CircuitClosed
		cb.failureCount = 0
		cb.logger.Info().Msg("Circuit breaker closed after successful call")
	}

	return nil
}

func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = CircuitClosed
	cb.failureCount = 0
	cb.lastFailure = time.Time{}
	cb.logger.Info().Msg("Circuit breaker reset")
}

type RetryPolicy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Multiplier  float64
}

func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    5 * time.Second,
		Multiplier:  2.0,
	}
}

func (rp RetryPolicy) Execute(fn func() error) error {
	var lastErr error
	delay := rp.BaseDelay

	for attempt := 0; attempt < rp.MaxAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(delay)
			delay = time.Duration(float64(delay) * rp.Multiplier)
			if delay > rp.MaxDelay {
				delay = rp.MaxDelay
			}
		}

		if err := fn(); err != nil {
			lastErr = err
			continue
		}

		return nil
	}

	return fmt.Errorf("failed after %d attempts: %w", rp.MaxAttempts, lastErr)
}
