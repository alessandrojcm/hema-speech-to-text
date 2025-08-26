package vad

import (
	"context"
	"fmt"
	"time"

	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

// Start starts the VAD monitoring
func (v *VADDetector) Start(ctx context.Context) error {
	v.logger.Info().Msg("Starting VAD monitoring")

	// Get audio processor from existing manager
	v.processor = v.audioManager.GetProcessor()
	if v.processor == nil {
		return fmt.Errorf("audio processor not available")
	}

	// Start monitoring goroutine
	go v.monitorAudioStream(ctx)

	return nil
}

// monitorAudioStream monitors the audio stream for voice activity
func (v *VADDetector) monitorAudioStream(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond) // Check every 100ms
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-v.stopChan:
			return

		case <-ticker.C:
			// Check current VAD state from audio manager
			vadActive := v.checkVADState()
			v.handleVADState(vadActive)
		}
	}
}

// checkVADState checks the current VAD state using real audio data
func (v *VADDetector) checkVADState() bool {
	if v.processor == nil {
		return false
	}

	// Extract recent audio samples for VAD analysis (100ms window)
	samples, err := v.audioManager.GetRecentAudioSamples(100 * time.Millisecond)
	if err != nil {
		v.logger.Debug().Err(err).Msg("Failed to get recent audio samples")
		return false
	}

	if len(samples) == 0 {
		return false
	}

	// Use the processor's VAD to detect voice activity
	vadActive := v.processor.DetectVoiceActivity(samples)

	return vadActive
}

// handleVADState handles changes in VAD state and manages speech segments
func (v *VADDetector) handleVADState(vadActive bool) {
	v.mu.Lock()
	defer v.mu.Unlock()

	now := time.Now()

	if vadActive && !v.isActive {
		// Speech started
		v.isActive = true
		v.activityStart = now
		v.logger.Debug().Time("start", now).Msg("Speech detected")

		// Send start event
		v.sendEvent(VADEvent{
			Type:      EventSpeechStart,
			StartTime: now,
		})

	} else if !vadActive && v.isActive {
		// Potential speech end - check silence duration
		if v.silenceStart.IsZero() {
			v.silenceStart = now
		}

		silenceDuration := now.Sub(v.silenceStart)
		if silenceDuration > time.Duration(v.config.MaxSilenceDurationMs)*time.Millisecond {
			// Speech ended
			v.handleSpeechEnd(now)
		}

	} else if vadActive && v.isActive {
		// Reset silence timer if speech resumes
		v.silenceStart = time.Time{}
	}
}

// handleSpeechEnd handles the end of a speech segment with duration filtering
func (v *VADDetector) handleSpeechEnd(endTime time.Time) {
	speechDuration := endTime.Sub(v.activityStart)

	// Check minimum duration
	if speechDuration < time.Duration(v.config.MinSpeechDurationMs)*time.Millisecond {
		v.logger.Debug().
			Dur("duration", speechDuration).
			Msg("Speech too short, ignoring")
		v.resetState()
		return
	}

	// Calculate buffer boundaries
	bufferStart := v.activityStart.Add(-time.Duration(v.config.BufferBeforeMs) * time.Millisecond)
	bufferEnd := endTime.Add(time.Duration(v.config.BufferAfterMs) * time.Millisecond)

	// Send complete segment event
	event := VADEvent{
		Type:        EventSpeechSegment,
		StartTime:   v.activityStart,
		EndTime:     endTime,
		Duration:    speechDuration,
		BufferStart: bufferStart,
		BufferEnd:   bufferEnd,
		Confidence:  v.calculateConfidence(speechDuration),
	}

	v.sendEvent(event)
	v.logger.Info().
		Dur("duration", speechDuration).
		Time("start", v.activityStart).
		Time("end", endTime).
		Msg("Speech segment detected")

	v.resetState()
}

// ExtractTimeRange is a method we need to add to AudioManager to extract audio segments by time
// This is a placeholder to show the interface we need
func (v *VADDetector) ExtractAudioSegment(startTime, endTime time.Time) (*types.AudioSegment, error) {
	// Calculate duration
	duration := endTime.Sub(startTime)

	// Use the existing ExtractAudio method from AudioManager
	// We need to adapt this to work with time ranges
	// For now, extract from the end time backwards
	req := types.ExtractionRequest{
		Duration: duration,
		EndTime:  endTime,
		Format:   "raw", // Raw float32 samples
	}

	segment, err := v.audioManager.ExtractAudio(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to extract audio segment: %w", err)
	}

	return segment, nil
}
