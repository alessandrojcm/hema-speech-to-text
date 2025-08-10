package preprocessing

import (
	"fmt"
	"time"

	"github.com/rs/zerolog"
	audioProcessing "github.com/your-org/hema-replay-system/pkg/audio/processing"
	audioTypes "github.com/your-org/hema-replay-system/pkg/audio/types"
	"github.com/your-org/hema-replay-system/pkg/speech/types"
)

// SpeechAudioPreprocessor handles audio preprocessing specifically for speech recognition
type SpeechAudioPreprocessor struct {
	config            types.ProcessingConfig
	enhancedProcessor *audioProcessing.EnhancedAudioProcessor
	logger            zerolog.Logger
}

// NewSpeechAudioPreprocessor creates a new speech-specific audio preprocessor
func NewSpeechAudioPreprocessor(config types.ProcessingConfig, logger zerolog.Logger) (*SpeechAudioPreprocessor, error) {
	// Convert speech processing config to audio processing config
	audioConfig := audioTypes.ProcessingConfig{
		EnablePreprocessing: true,
		Normalization:       config.Normalization,
		NoiseReduction:      config.NoiseReduction,
		HighpassFilter:      80.0,   // Remove low-frequency noise for speech
		LowpassFilter:       8000.0, // Remove high-frequency noise above speech range
		VADThreshold:        0.01,
		VADType:             "webrtc", // Use WebRTC VAD for better speech detection
		VADMode:             2,        // Aggressive mode for noisy environments
		ResamplerType:       "gosamplerate",
		ResamplerQuality:    4, // High quality resampling for speech
		WAVExporterType:     "goaudio",
		FFTType:             "gonum",
	}

	// Create enhanced audio processor with speech-optimized settings
	enhancedProcessor, err := audioProcessing.NewEnhancedAudioProcessor(
		audioConfig,
		config.TargetSampleRate,
		1, // Mono for speech recognition
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create enhanced audio processor: %w", err)
	}

	return &SpeechAudioPreprocessor{
		config:            config,
		enhancedProcessor: enhancedProcessor,
		logger:            logger.With().Str("component", "speech_audio_preprocessor").Logger(),
	}, nil
}

// PrepareAudioForSpeech prepares audio segment for speech recognition
func (sap *SpeechAudioPreprocessor) PrepareAudioForSpeech(segment *audioTypes.AudioSegment) ([]float32, error) {
	if segment == nil || len(segment.Data) == 0 {
		return nil, fmt.Errorf("empty audio segment")
	}

	startTime := time.Now()

	// Step 1: Convert raw audio data to float32 samples
	samples, err := sap.convertToFloat32Samples(segment)
	if err != nil {
		return nil, fmt.Errorf("failed to convert audio data: %w", err)
	}

	// Step 2: Resample to target sample rate if needed
	if segment.Metadata.SampleRate != sap.config.TargetSampleRate {
		resampled, err := sap.enhancedProcessor.Resample(
			samples,
			segment.Metadata.SampleRate,
			sap.config.TargetSampleRate,
		)
		if err != nil {
			sap.logger.Warn().
				Err(err).
				Int("from_rate", segment.Metadata.SampleRate).
				Int("to_rate", sap.config.TargetSampleRate).
				Msg("Resampling failed, using original samples")
		} else {
			samples = resampled
		}
	}

	// Step 3: Convert to mono if needed
	if segment.Metadata.Channels > 1 {
		samples = sap.convertToMono(samples, segment.Metadata.Channels)
	}

	// Step 4: Apply voice activity detection if enabled
	if sap.config.VADEnabled {
		hasVoice := sap.enhancedProcessor.DetectVoiceActivity(samples)
		if !hasVoice {
			sap.logger.Debug().
				Str("segment_id", segment.ID).
				Msg("No voice activity detected in audio segment")
			// Return empty samples or very quiet samples to indicate no speech
			return make([]float32, len(samples)), nil
		}
	}

	// Step 5: Apply enhanced audio processing
	processed, err := sap.enhancedProcessor.Process(samples, segment.StartTime)
	if err != nil {
		sap.logger.Warn().
			Err(err).
			Msg("Enhanced processing failed, using basic processing")
		processed = samples
	}

	// Step 6: Apply speech-specific optimizations
	processed = sap.applySpeechOptimizations(processed)

	processingTime := time.Since(startTime)
	sap.logger.Debug().
		Str("segment_id", segment.ID).
		Dur("processing_time", processingTime).
		Int("input_samples", len(samples)).
		Int("output_samples", len(processed)).
		Msg("Audio preprocessing completed")

	return processed, nil
}

// convertToFloat32Samples converts raw audio data to float32 samples
func (sap *SpeechAudioPreprocessor) convertToFloat32Samples(segment *audioTypes.AudioSegment) ([]float32, error) {
	data := segment.Data

	// Determine sample format based on metadata or data length
	var samples []float32

	// Assume 16-bit PCM for now (most common format)
	if len(data)%2 != 0 {
		return nil, fmt.Errorf("invalid audio data length for 16-bit samples")
	}

	samples = make([]float32, len(data)/2)

	for i := 0; i < len(samples); i++ {
		if i*2+1 < len(data) {
			// Convert 16-bit PCM to float32 (-1.0 to 1.0)
			sample := int16(data[i*2]) | int16(data[i*2+1])<<8
			samples[i] = float32(sample) / 32768.0
		}
	}

	return samples, nil
}

// convertToMono converts multi-channel audio to mono
func (sap *SpeechAudioPreprocessor) convertToMono(samples []float32, channels int) []float32 {
	if channels <= 1 {
		return samples
	}

	monoSamples := make([]float32, len(samples)/channels)

	for i := 0; i < len(monoSamples); i++ {
		var sum float32
		for ch := 0; ch < channels; ch++ {
			if i*channels+ch < len(samples) {
				sum += samples[i*channels+ch]
			}
		}
		monoSamples[i] = sum / float32(channels)
	}

	return monoSamples
}

// applySpeechOptimizations applies speech-specific audio optimizations
func (sap *SpeechAudioPreprocessor) applySpeechOptimizations(samples []float32) []float32 {
	// Apply pre-emphasis filter to enhance high frequencies (common in speech processing)
	if len(samples) > 1 {
		preEmphasized := make([]float32, len(samples))
		preEmphasized[0] = samples[0]

		const preEmphasisCoeff = 0.97
		for i := 1; i < len(samples); i++ {
			preEmphasized[i] = samples[i] - preEmphasisCoeff*samples[i-1]
		}

		samples = preEmphasized
	}

	// Apply windowing to reduce spectral leakage
	samples = sap.applyHammingWindow(samples)

	return samples
}

// applyHammingWindow applies a Hamming window to the samples
func (sap *SpeechAudioPreprocessor) applyHammingWindow(samples []float32) []float32 {
	if len(samples) == 0 {
		return samples
	}

	windowed := make([]float32, len(samples))
	n := len(samples)

	for i := 0; i < n; i++ {
		// Hamming window formula: 0.54 - 0.46 * cos(2π * i / (N-1))
		window := 0.54 - 0.46*float32(cosApprox(2.0*3.14159*float64(i)/float64(n-1)))
		windowed[i] = samples[i] * window
	}

	return windowed
}

// cosApprox provides a fast cosine approximation
func cosApprox(x float64) float64 {
	// Simple cosine approximation using Taylor series (first few terms)
	x2 := x * x
	return 1.0 - x2/2.0 + x2*x2/24.0 - x2*x2*x2/720.0
}

// GetProcessingStats returns preprocessing statistics
func (sap *SpeechAudioPreprocessor) GetProcessingStats() map[string]interface{} {
	return map[string]interface{}{
		"target_sample_rate": sap.config.TargetSampleRate,
		"normalization":      sap.config.Normalization,
		"noise_reduction":    sap.config.NoiseReduction,
		"vad_enabled":        sap.config.VADEnabled,
		"processor_config":   sap.enhancedProcessor.GetConfig(),
	}
}

// Close releases resources used by the preprocessor
func (sap *SpeechAudioPreprocessor) Close() error {
	if sap.enhancedProcessor != nil {
		return sap.enhancedProcessor.Close()
	}
	return nil
}

// ValidateAudioSegment validates that an audio segment is suitable for speech recognition
func (sap *SpeechAudioPreprocessor) ValidateAudioSegment(segment *audioTypes.AudioSegment) error {
	if segment == nil {
		return fmt.Errorf("audio segment is nil")
	}

	if len(segment.Data) == 0 {
		return fmt.Errorf("audio segment data is empty")
	}

	if segment.Duration < sap.config.SegmentDuration/10 {
		return fmt.Errorf("audio segment too short: %v (minimum: %v)",
			segment.Duration, sap.config.SegmentDuration/10)
	}

	if segment.Duration > sap.config.SegmentDuration*5 {
		sap.logger.Warn().
			Dur("duration", segment.Duration).
			Dur("max_recommended", sap.config.SegmentDuration*5).
			Msg("Audio segment longer than recommended for speech recognition")
	}

	if segment.Metadata.SampleRate < 8000 {
		return fmt.Errorf("sample rate too low for speech recognition: %d Hz (minimum: 8000 Hz)",
			segment.Metadata.SampleRate)
	}

	return nil
}

// EstimateProcessingTime estimates how long preprocessing will take
func (sap *SpeechAudioPreprocessor) EstimateProcessingTime(segment *audioTypes.AudioSegment) time.Duration {
	if segment == nil {
		return 0
	}

	// Base processing time estimate: ~1ms per second of audio
	baseTime := segment.Duration / 1000

	// Add overhead for resampling if needed
	if segment.Metadata.SampleRate != sap.config.TargetSampleRate {
		baseTime += segment.Duration / 500 // Resampling overhead
	}

	// Add overhead for multi-channel conversion
	if segment.Metadata.Channels > 1 {
		baseTime += segment.Duration / 2000 // Channel conversion overhead
	}

	// Add overhead for enhanced processing
	if sap.config.NoiseReduction {
		baseTime += segment.Duration / 200 // Noise reduction overhead
	}

	return baseTime
}
