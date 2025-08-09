//go:build !noaudio

package whisper

import (
	"fmt"
	"time"

	"github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
	"github.com/rs/zerolog"
	"github.com/your-org/hema-replay-system/pkg/speech/types"
)

// WhisperWrapper wraps the official whisper.cpp Go bindings
type WhisperWrapper struct {
	model  whisper.Model
	logger zerolog.Logger
}

// NewWhisperWrapper creates a new whisper wrapper
func NewWhisperWrapper(modelPath string, logger zerolog.Logger) (*WhisperWrapper, error) {
	model, err := whisper.New(modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load whisper model: %w", err)
	}

	return &WhisperWrapper{
		model:  model,
		logger: logger.With().Str("component", "whisper_wrapper").Logger(),
	}, nil
}

// Close releases the whisper model
func (ww *WhisperWrapper) Close() error {
	if ww.model != nil {
		return ww.model.Close()
	}
	return nil
}

// Transcribe performs transcription on audio samples
func (ww *WhisperWrapper) Transcribe(samples []float32, params types.WhisperParams) (*types.TranscriptionResult, error) {
	if ww.model == nil {
		return nil, fmt.Errorf("whisper model is nil")
	}

	startTime := time.Now()

	// Create whisper context for this transcription
	context, err := ww.model.NewContext()
	if err != nil {
		return nil, fmt.Errorf("failed to create whisper context: %w", err)
	}

	// Configure context parameters
	if err := ww.configureContext(context, params); err != nil {
		return nil, fmt.Errorf("failed to configure context: %w", err)
	}

	// Process the audio
	if err := context.Process(samples, nil, nil, nil); err != nil {
		return nil, fmt.Errorf("failed to process audio: %w", err)
	}

	// Extract results
	result, err := ww.extractResult(context, time.Since(startTime))
	if err != nil {
		return nil, fmt.Errorf("failed to extract result: %w", err)
	}

	return result, nil
}

// configureContext configures the whisper context with parameters
func (ww *WhisperWrapper) configureContext(context whisper.Context, params types.WhisperParams) error {
	// Set language
	if params.Language != "" {
		if err := context.SetLanguage(params.Language); err != nil {
			return fmt.Errorf("failed to set language: %w", err)
		}
	}

	// Set thread count
	if params.ThreadCount > 0 {
		context.SetThreads(uint(params.ThreadCount))
	}

	// Set other parameters
	context.SetTranslate(false) // We want transcription, not translation
	context.SetTokenTimestamps(params.WordTimestamps)

	// Set sampling parameters
	if params.Temperature > 0 {
		context.SetTemperature(params.Temperature)
	}

	return nil
}

// extractResult extracts transcription result from whisper context
func (ww *WhisperWrapper) extractResult(context whisper.Context, processingTime time.Duration) (*types.TranscriptionResult, error) {
	segments := make([]types.TranscriptionSegment, 0)
	var fullText string
	var totalConfidence float64
	segmentCount := 0

	// Iterate through segments
	for {
		segment, err := context.NextSegment()
		if err != nil {
			break // No more segments
		}

		segmentText := segment.Text
		if segmentText == "" {
			continue
		}

		// Extract tokens for this segment
		tokens := ww.extractTokensFromSegment(segment)

		// Calculate segment confidence (average of token confidences)
		var segmentConfidence float64
		if len(tokens) > 0 {
			for _, token := range tokens {
				segmentConfidence += token.Confidence
			}
			segmentConfidence /= float64(len(tokens))
		}

		transcriptionSegment := types.TranscriptionSegment{
			Text:       segmentText,
			StartTime:  time.Duration(segment.Start) * time.Millisecond,
			EndTime:    time.Duration(segment.End) * time.Millisecond,
			Confidence: segmentConfidence,
			Tokens:     tokens,
		}

		segments = append(segments, transcriptionSegment)
		fullText += segmentText
		totalConfidence += segmentConfidence
		segmentCount++
	}

	// Calculate overall confidence
	var overallConfidence float64
	if segmentCount > 0 {
		overallConfidence = totalConfidence / float64(segmentCount)
	}

	// Create metadata
	metadata := types.TranscriptionMetadata{
		ModelUsed:        "whisper.cpp",
		ProcessingTime:   processingTime,
		MetalAccelerated: ww.isMetalAccelerated(),
		TokenCount:       ww.countTokens(segments),
		HEMATermsFound:   ww.extractHEMATerms(segments),
	}

	result := &types.TranscriptionResult{
		ID:          generateTranscriptionID(),
		Text:        fullText,
		Confidence:  overallConfidence,
		Language:    context.Language(),
		Duration:    processingTime,
		Segments:    segments,
		Metadata:    metadata,
		ProcessedAt: time.Now(),
	}

	return result, nil
}

// extractTokensFromSegment extracts tokens from a whisper segment
func (ww *WhisperWrapper) extractTokensFromSegment(segment whisper.Segment) []types.Token {
	tokens := make([]types.Token, 0)

	// Extract tokens from the segment
	for _, token := range segment.Tokens {
		transcriptionToken := types.Token{
			Text:       token.Text,
			Confidence: float64(token.P), // Probability as confidence
			StartTime:  time.Duration(token.Start) * time.Millisecond,
			EndTime:    time.Duration(token.End) * time.Millisecond,
			IsHEMA:     false, // Will be set by vocabulary system
		}

		tokens = append(tokens, transcriptionToken)
	}

	return tokens
}

// isMetalAccelerated checks if Metal acceleration is being used
func (ww *WhisperWrapper) isMetalAccelerated() bool {
	// This would need to be implemented based on whisper.cpp capabilities
	// For now, return true on macOS if GPU is available
	return true // Placeholder
}

// countTokens counts total tokens in all segments
func (ww *WhisperWrapper) countTokens(segments []types.TranscriptionSegment) int {
	count := 0
	for _, segment := range segments {
		count += len(segment.Tokens)
	}
	return count
}

// extractHEMATerms extracts HEMA-specific terms found in transcription
func (ww *WhisperWrapper) extractHEMATerms(segments []types.TranscriptionSegment) []string {
	hemaTerms := make([]string, 0)

	for _, segment := range segments {
		for _, token := range segment.Tokens {
			if token.IsHEMA {
				hemaTerms = append(hemaTerms, token.Text)
			}
		}
	}

	return hemaTerms
}

// generateTranscriptionID generates a unique ID for transcription results
func generateTranscriptionID() string {
	return fmt.Sprintf("transcription_%d", time.Now().UnixNano())
}
