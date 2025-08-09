package engine

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	audioTypes "github.com/your-org/hema-replay-system/pkg/audio/types"
	"github.com/your-org/hema-replay-system/pkg/speech/types"
	"github.com/your-org/hema-replay-system/pkg/speech/vocabulary"
	"github.com/your-org/hema-replay-system/pkg/speech/whisper"
)

// ProcessingPipeline orchestrates the speech recognition processing
type ProcessingPipeline struct {
	config       types.SpeechConfig
	modelManager *whisper.ModelManager
	vocabulary   *vocabulary.HEMAVocabulary
	logger       zerolog.Logger
}

// NewProcessingPipeline creates a new processing pipeline
func NewProcessingPipeline(config types.SpeechConfig, logger zerolog.Logger) *ProcessingPipeline {
	return &ProcessingPipeline{
		config: config,
		logger: logger.With().Str("component", "processing_pipeline").Logger(),
	}
}

// SetModelManager sets the model manager for the pipeline
func (pp *ProcessingPipeline) SetModelManager(modelManager *whisper.ModelManager) {
	pp.modelManager = modelManager
}

// SetVocabulary sets the vocabulary for the pipeline
func (pp *ProcessingPipeline) SetVocabulary(vocabulary *vocabulary.HEMAVocabulary) {
	pp.vocabulary = vocabulary
}

// Process processes a transcription request through the complete pipeline
func (pp *ProcessingPipeline) Process(ctx context.Context, request types.TranscriptionRequest) (*types.TranscriptionResult, error) {
	// Step 1: Get the appropriate model
	wrapper, err := pp.modelManager.GetModel(request.ModelSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}

	// Step 2: Prepare audio data
	audioData, err := pp.prepareAudioData(request.AudioSegment)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare audio data: %w", err)
	}

	// Step 3: Create whisper parameters
	whisperParams := types.WhisperParams{
		Language:       request.Language,
		ThreadCount:    pp.config.Whisper.ThreadCount,
		Temperature:    pp.config.Whisper.Temperature,
		WordTimestamps: pp.config.Whisper.WordTimestamps,
	}

	// Step 4: Perform transcription
	result, err := wrapper.Transcribe(audioData, whisperParams)
	if err != nil {
		return nil, fmt.Errorf("transcription failed: %w", err)
	}

	// Step 5: Apply vocabulary boosting if enabled
	if request.UseVocabulary && pp.vocabulary != nil {
		pp.applyVocabularyBoosting(result)
	}

	// Step 6: Apply confidence filtering
	if result.Confidence < request.ConfidenceThreshold {
		pp.logger.Debug().
			Float64("confidence", result.Confidence).
			Float64("threshold", request.ConfidenceThreshold).
			Msg("Result below confidence threshold")
	}

	return result, nil
}

// prepareAudioData converts audio segment to format expected by whisper
func (pp *ProcessingPipeline) prepareAudioData(segment *audioTypes.AudioSegment) ([]float32, error) {
	// For now, assume the audio data is already in the correct format
	// In a full implementation, this would handle:
	// - Sample rate conversion to 16kHz
	// - Channel conversion to mono
	// - Format conversion to float32
	// - Normalization

	if segment == nil || len(segment.Data) == 0 {
		return nil, fmt.Errorf("empty audio segment")
	}

	// Convert byte data to float32 samples
	// This is a simplified conversion - real implementation would depend on the audio format
	samples := make([]float32, len(segment.Data)/2) // Assuming 16-bit samples

	for i := 0; i < len(samples); i++ {
		if i*2+1 < len(segment.Data) {
			// Convert 16-bit PCM to float32 (-1.0 to 1.0)
			sample := int16(segment.Data[i*2]) | int16(segment.Data[i*2+1])<<8
			samples[i] = float32(sample) / 32768.0
		}
	}

	return samples, nil
}

// applyVocabularyBoosting applies HEMA vocabulary boosting to the result
func (pp *ProcessingPipeline) applyVocabularyBoosting(result *types.TranscriptionResult) {
	hemaTermsFound := make([]string, 0)

	// Process each segment
	for i := range result.Segments {
		segment := &result.Segments[i]

		// Process each token in the segment
		for j := range segment.Tokens {
			token := &segment.Tokens[j]

			// Check if this token is a HEMA term
			if pp.vocabulary.IsHEMATerm(token.Text) {
				token.IsHEMA = true
				hemaTermsFound = append(hemaTermsFound, token.Text)

				// Apply boost to confidence
				boost := pp.vocabulary.GetBoost(token.Text)
				token.Confidence = pp.applyBoost(token.Confidence, boost)
			}
		}

		// Recalculate segment confidence based on boosted tokens
		pp.recalculateSegmentConfidence(segment)
	}

	// Update metadata
	result.Metadata.HEMATermsFound = hemaTermsFound
	result.Metadata.VocabularyBoost = len(hemaTermsFound) > 0

	// Recalculate overall confidence
	pp.recalculateOverallConfidence(result)
}

// applyBoost applies a boost factor to a confidence score
func (pp *ProcessingPipeline) applyBoost(confidence float64, boost float64) float64 {
	// Apply boost while keeping confidence in valid range [0, 1]
	boosted := confidence * boost
	if boosted > 1.0 {
		boosted = 1.0
	}
	return boosted
}

// recalculateSegmentConfidence recalculates segment confidence based on token confidences
func (pp *ProcessingPipeline) recalculateSegmentConfidence(segment *types.TranscriptionSegment) {
	if len(segment.Tokens) == 0 {
		return
	}

	var totalConfidence float64
	for _, token := range segment.Tokens {
		totalConfidence += token.Confidence
	}

	segment.Confidence = totalConfidence / float64(len(segment.Tokens))
}

// recalculateOverallConfidence recalculates overall result confidence
func (pp *ProcessingPipeline) recalculateOverallConfidence(result *types.TranscriptionResult) {
	if len(result.Segments) == 0 {
		return
	}

	var totalConfidence float64
	for _, segment := range result.Segments {
		totalConfidence += segment.Confidence
	}

	result.Confidence = totalConfidence / float64(len(result.Segments))
}
