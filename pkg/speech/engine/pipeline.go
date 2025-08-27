package engine

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/your-org/hema-replay-system/pkg/audio/debug"
	"github.com/your-org/hema-replay-system/pkg/speech/preprocessing"
	"github.com/your-org/hema-replay-system/pkg/speech/types"
	"github.com/your-org/hema-replay-system/pkg/speech/vocabulary"
	"github.com/your-org/hema-replay-system/pkg/speech/whisper"
)

// ProcessingPipeline orchestrates the speech recognition processing
type ProcessingPipeline struct {
	config        types.SpeechConfig
	modelManager  *whisper.ModelManager
	vocabulary    *vocabulary.HEMAVocabulary
	preprocessor  *preprocessing.SpeechAudioPreprocessor
	qualityFilter *preprocessing.QualityFilter
	debugSaver    *debug.SegmentSaver
	logger        zerolog.Logger
}

// NewProcessingPipeline creates a new processing pipeline
func NewProcessingPipeline(config types.SpeechConfig, logger zerolog.Logger) (*ProcessingPipeline, error) {
	// Create audio preprocessor for speech recognition
	preprocessor, err := preprocessing.NewSpeechAudioPreprocessor(config.Processing, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio preprocessor: %w", err)
	}

	// Create quality filter - always enabled with default params
	qualityFilter := preprocessing.NewQualityFilterWithParams(config.Processing.MinEnergy, config.Processing.MinSNR, config.Processing.MinVoiceRatio, logger)

	return &ProcessingPipeline{
		config:        config,
		preprocessor:  preprocessor,
		qualityFilter: qualityFilter,
		logger:        logger.With().Str("component", "processing_pipeline").Logger(),
	}, nil
}

// SetModelManager sets the model manager for the pipeline
func (pp *ProcessingPipeline) SetModelManager(modelManager *whisper.ModelManager) {
	pp.modelManager = modelManager
}

// SetVocabulary sets the vocabulary for the pipeline
func (pp *ProcessingPipeline) SetVocabulary(vocabulary *vocabulary.HEMAVocabulary) {
	pp.vocabulary = vocabulary
}

// SetDebugSaver sets the debug saver for the pipeline
func (pp *ProcessingPipeline) SetDebugSaver(debugSaver *debug.SegmentSaver) {
	pp.debugSaver = debugSaver
}

// Process processes a transcription request through the complete pipeline
func (pp *ProcessingPipeline) Process(ctx context.Context, request types.TranscriptionRequest) (*types.TranscriptionResult, error) {
	// Step 1: Get the appropriate model
	wrapper, err := pp.modelManager.GetModel(request.ModelSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}

	// Step 2: Prepare audio data using enhanced preprocessing
	audioData, err := pp.preprocessor.PrepareAudioForSpeech(request.AudioSegment)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare audio data: %w", err)
	}

	// Step 3: Quality filtering (early exit for low-quality audio)
	if pp.qualityFilter != nil {
		shouldProcess, qualityMetrics := pp.qualityFilter.ShouldProcessSegment(audioData)
		if !shouldProcess {
			// Save debug audio for rejected segments
			if pp.debugSaver != nil {
				metadata := map[string]interface{}{
					"type":            "quality_rejected",
					"quality_metrics": qualityMetrics,
					"vad_confidence":  request.AudioSegment.Metadata.Quality,
				}
				if err := pp.debugSaver.SaveSegment(audioData, metadata); err != nil {
					pp.logger.Warn().Err(err).Msg("Failed to save debug audio for quality rejected segment")
				}
			}

			pp.logger.Debug().
				Interface("quality_metrics", qualityMetrics).
				Msg("Segment filtered out due to low quality")
			return nil, fmt.Errorf("audio quality too low: rms_energy=%.4f, voice_ratio=%.4f, snr_db=%.2f",
				qualityMetrics["rms_energy"], qualityMetrics["voice_ratio"], qualityMetrics["snr_db"])
		}

		// Log quality metrics for monitoring
		pp.logger.Debug().
			Interface("quality_metrics", qualityMetrics).
			Msg("Audio quality check passed")
	}

	// Step 4: Create whisper parameters with noise suppression
	whisperParams := types.WhisperParams{
		Language:           request.Language,
		ThreadCount:        pp.config.Whisper.ThreadCount,
		Temperature:        pp.config.Whisper.Temperature,
		WordTimestamps:     pp.config.Whisper.WordTimestamps,
		SuppressBlank:      pp.config.Whisper.SuppressBlank,
		SuppressNonSpeech:  pp.config.Whisper.SuppressNonSpeech,
		NoSpeechThreshold:  pp.config.Whisper.NoSpeechThreshold,
		LogProbThreshold:   pp.config.Whisper.LogProbThreshold,
		MinTokenConfidence: pp.config.Whisper.MinTokenConfidence,
	}

	// Build initial prompt from vocabulary if available
	if request.UseVocabulary && pp.vocabulary != nil {
		// Note: BuildInitialPrompt method needs to be implemented in HEMAVocabulary
		// For now, just enable noise suppression without initial prompt
		whisperParams.InitialPrompt = ""
	}

	// Step 5: Save debug audio for segments that will be transcribed
	if pp.debugSaver != nil {
		metadata := map[string]interface{}{
			"type":           "pre_whisper",
			"vad_confidence": request.AudioSegment.Metadata.Quality,
			"language":       whisperParams.Language,
			"model_size":     request.ModelSize,
		}
		if err := pp.debugSaver.SaveSegment(audioData, metadata); err != nil {
			pp.logger.Warn().Err(err).Msg("Failed to save debug audio for pre-whisper segment")
		}
	}

	// Step 6: Perform transcription
	result, err := wrapper.Transcribe(audioData, whisperParams)
	if err != nil {
		return nil, fmt.Errorf("transcription failed: %w", err)
	}

	// Step 7: Apply vocabulary boosting if enabled (post-processing fallback)
	if request.UseVocabulary && pp.vocabulary != nil {
		pp.applyVocabularyBoosting(result)
	}

	// Step 8: Apply confidence filtering
	if result.Confidence < request.ConfidenceThreshold {
		pp.logger.Debug().
			Float64("confidence", result.Confidence).
			Float64("threshold", request.ConfidenceThreshold).
			Msg("Result below confidence threshold")
	}

	return result, nil
}

// Close releases resources used by the pipeline
func (pp *ProcessingPipeline) Close() error {
	if pp.preprocessor != nil {
		return pp.preprocessor.Close()
	}
	return nil
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
