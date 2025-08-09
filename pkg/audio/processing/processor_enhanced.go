package processing

import (
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

// EnhancedAudioProcessor uses library implementations for audio processing
type EnhancedAudioProcessor struct {
	config     types.ProcessingConfig
	sampleRate int
	channels   int

	// Library-based components
	resampler    ResamplerInterface
	vadDetector  VADInterface
	wavExporter  WAVExporter
	fftProcessor FFTProcessor
	windowFunc   WindowFunction

	// Keep existing components that work
	normalizer   *Normalizer
	noiseReducer *NoiseReducer
	filters      []AudioFilter
	qualityMeter *QualityMeter

	processingBuffer []float32
	historyBuffer    []float32

	logger zerolog.Logger
}

// NewEnhancedAudioProcessor creates a new enhanced audio processor with library implementations
func NewEnhancedAudioProcessor(config types.ProcessingConfig, sampleRate, channels int, logger zerolog.Logger) (*EnhancedAudioProcessor, error) {
	ap := &EnhancedAudioProcessor{
		config:     config,
		sampleRate: sampleRate,
		channels:   channels,
		logger:     logger.With().Str("component", "enhanced_audio_processor").Logger(),
	}

	// Initialize library-based components
	if err := ap.initializeLibraryComponents(); err != nil {
		return nil, fmt.Errorf("failed to initialize library components: %w", err)
	}

	// Initialize existing components
	if config.EnablePreprocessing {
		ap.normalizer = &Normalizer{targetLevel: 0.8}
		ap.noiseReducer = &NoiseReducer{reduction: 0.5}
		ap.qualityMeter = NewQualityMeter(sampleRate, channels)

		if config.HighpassFilter > 0 {
			ap.filters = append(ap.filters, &HighpassFilter{
				cutoffFreq: config.HighpassFilter,
				sampleRate: sampleRate,
			})
		}

		if config.LowpassFilter > 0 {
			ap.filters = append(ap.filters, &LowpassFilter{
				cutoffFreq: config.LowpassFilter,
				sampleRate: sampleRate,
			})
		}
	}

	return ap, nil
}

// initializeLibraryComponents initializes the library-based components based on configuration
func (ap *EnhancedAudioProcessor) initializeLibraryComponents() error {
	// Initialize resampler
	if err := ap.createResampler(); err != nil {
		return fmt.Errorf("failed to create resampler: %w", err)
	}

	// Initialize VAD detector
	if err := ap.createVADDetector(); err != nil {
		return fmt.Errorf("failed to create VAD detector: %w", err)
	}

	// Initialize WAV exporter
	ap.wavExporter = ap.createWAVExporter()

	// Initialize FFT processor
	ap.fftProcessor = ap.createFFTProcessor()

	// Initialize window function
	ap.windowFunc = NewGonumWindowFunction()

	return nil
}

// createResampler creates the appropriate resampler based on configuration
func (ap *EnhancedAudioProcessor) createResampler() error {
	switch ap.config.ResamplerType {
	case "gosamplerate":
		resampler, err := NewGosamplerateResampler(ap.config.ResamplerQuality)
		if err != nil {
			ap.logger.Warn().Err(err).Msg("Failed to create gosamplerate resampler, falling back to custom")
			ap.resampler = NewCustomResampler()
		} else {
			ap.resampler = resampler
		}
	case "custom":
		ap.resampler = NewCustomResampler()
	default:
		ap.logger.Warn().Str("type", ap.config.ResamplerType).Msg("Unknown resampler type, using gosamplerate")
		resampler, err := NewGosamplerateResampler(ap.config.ResamplerQuality)
		if err != nil {
			ap.logger.Warn().Err(err).Msg("Failed to create gosamplerate resampler, falling back to custom")
			ap.resampler = NewCustomResampler()
		} else {
			ap.resampler = resampler
		}
	}
	return nil
}

// createVADDetector creates the appropriate VAD detector based on configuration
func (ap *EnhancedAudioProcessor) createVADDetector() error {
	switch ap.config.VADType {
	case "webrtc":
		vadDetector, err := NewWebRTCVAD(ap.sampleRate, ap.config.VADMode)
		if err != nil {
			ap.logger.Warn().Err(err).Msg("Failed to create WebRTC VAD, falling back to threshold")
			ap.vadDetector = NewThresholdVAD(ap.config.VADThreshold)
		} else {
			ap.vadDetector = vadDetector
		}
	case "threshold":
		ap.vadDetector = NewThresholdVAD(ap.config.VADThreshold)
	default:
		ap.logger.Warn().Str("type", ap.config.VADType).Msg("Unknown VAD type, using WebRTC")
		vadDetector, err := NewWebRTCVAD(ap.sampleRate, ap.config.VADMode)
		if err != nil {
			ap.logger.Warn().Err(err).Msg("Failed to create WebRTC VAD, falling back to threshold")
			ap.vadDetector = NewThresholdVAD(ap.config.VADThreshold)
		} else {
			ap.vadDetector = vadDetector
		}
	}
	return nil
}

// createWAVExporter creates the appropriate WAV exporter based on configuration
func (ap *EnhancedAudioProcessor) createWAVExporter() WAVExporter {
	switch ap.config.WAVExporterType {
	case "goaudio":
		return NewGoAudioWAVExporter()
	case "custom":
		// Return custom implementation (would need to be implemented)
		ap.logger.Warn().Msg("Custom WAV exporter not implemented, using go-audio")
		return NewGoAudioWAVExporter()
	default:
		ap.logger.Warn().Str("type", ap.config.WAVExporterType).Msg("Unknown WAV exporter type, using go-audio")
		return NewGoAudioWAVExporter()
	}
}

// createFFTProcessor creates the appropriate FFT processor based on configuration
func (ap *EnhancedAudioProcessor) createFFTProcessor() FFTProcessor {
	switch ap.config.FFTType {
	case "gonum":
		return NewGonumFFTProcessor()
	case "custom":
		// Return custom implementation (would need to be implemented)
		ap.logger.Warn().Msg("Custom FFT processor not implemented, using Gonum")
		return NewGonumFFTProcessor()
	default:
		ap.logger.Warn().Str("type", ap.config.FFTType).Msg("Unknown FFT type, using Gonum")
		return NewGonumFFTProcessor()
	}
}

// Process applies enhanced audio processing to input samples
func (ap *EnhancedAudioProcessor) Process(samples []float32, timestamp time.Time) ([]float32, error) {
	if len(samples) == 0 {
		return samples, nil
	}

	// Ensure processing buffer is large enough
	if len(ap.processingBuffer) < len(samples) {
		ap.processingBuffer = make([]float32, len(samples))
	}

	// Copy input to processing buffer
	copy(ap.processingBuffer[:len(samples)], samples)
	processed := ap.processingBuffer[:len(samples)]

	// Apply processing pipeline
	if ap.config.EnablePreprocessing {
		// Apply filters
		for _, filter := range ap.filters {
			if err := filter.Apply(processed); err != nil {
				ap.logger.Warn().Err(err).Msg("Filter application failed")
			}
		}

		// Apply noise reduction
		if ap.config.NoiseReduction && ap.noiseReducer != nil {
			if err := ap.noiseReducer.Reduce(processed); err != nil {
				ap.logger.Warn().Err(err).Msg("Noise reduction failed")
			}
		}

		// Apply normalization
		if ap.config.Normalization && ap.normalizer != nil {
			if err := ap.normalizer.Normalize(processed); err != nil {
				ap.logger.Warn().Err(err).Msg("Normalization failed")
			}
		}
	}

	// Update processing history
	ap.updateHistory(processed)

	return processed, nil
}

// Resample resamples audio data using the configured resampler
func (ap *EnhancedAudioProcessor) Resample(samples []float32, inputRate, outputRate int) ([]float32, error) {
	if ap.resampler == nil {
		return nil, fmt.Errorf("resampler not initialized")
	}
	return ap.resampler.Resample(samples, inputRate, outputRate)
}

// DetectVoiceActivity detects voice activity using the configured VAD
func (ap *EnhancedAudioProcessor) DetectVoiceActivity(samples []float32) bool {
	if ap.vadDetector == nil {
		return false
	}
	return ap.vadDetector.DetectVoice(samples)
}

// ExportWAV exports audio segment to WAV file using the configured exporter
func (ap *EnhancedAudioProcessor) ExportWAV(segment *types.AudioSegment, path string) error {
	if ap.wavExporter == nil {
		return fmt.Errorf("WAV exporter not initialized")
	}
	return ap.wavExporter.Export(segment, path)
}

// ExportWAVWithMetadata exports audio segment to WAV file with metadata
func (ap *EnhancedAudioProcessor) ExportWAVWithMetadata(segment *types.AudioSegment, path string, metadata map[string]string) error {
	if ap.wavExporter == nil {
		return fmt.Errorf("WAV exporter not initialized")
	}
	return ap.wavExporter.ExportWithMetadata(segment, path, metadata)
}

// ComputeFFT computes FFT using the configured FFT processor
func (ap *EnhancedAudioProcessor) ComputeFFT(samples []float32) []complex128 {
	if ap.fftProcessor == nil {
		return nil
	}
	return ap.fftProcessor.FFT(samples)
}

// ComputePowerSpectrum computes power spectrum using the configured FFT processor
func (ap *EnhancedAudioProcessor) ComputePowerSpectrum(samples []float32) []float64 {
	if ap.fftProcessor == nil {
		return nil
	}
	return ap.fftProcessor.PowerSpectrum(samples)
}

// ComputeSpectralCentroid computes spectral centroid using the configured FFT processor
func (ap *EnhancedAudioProcessor) ComputeSpectralCentroid(samples []float32) float64 {
	if ap.fftProcessor == nil {
		return 0.0
	}
	return ap.fftProcessor.SpectralCentroid(samples)
}

// ApplyWindow applies a window function to the samples
func (ap *EnhancedAudioProcessor) ApplyWindow(samples []float32, windowType string) []float32 {
	if ap.windowFunc == nil {
		return samples
	}
	return ap.windowFunc.Apply(samples, windowType)
}

// updateHistory updates the processing history buffer
func (ap *EnhancedAudioProcessor) updateHistory(samples []float32) {
	const maxHistorySize = 44100 * 2

	if len(ap.historyBuffer)+len(samples) > maxHistorySize {
		keepSize := maxHistorySize - len(samples)
		if keepSize > 0 {
			copy(ap.historyBuffer, ap.historyBuffer[len(ap.historyBuffer)-keepSize:])
			ap.historyBuffer = ap.historyBuffer[:keepSize]
		} else {
			ap.historyBuffer = ap.historyBuffer[:0]
		}
	}

	ap.historyBuffer = append(ap.historyBuffer, samples...)
}

// Close releases resources used by the processor
func (ap *EnhancedAudioProcessor) Close() error {
	var errors []error

	if ap.resampler != nil {
		if err := ap.resampler.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close resampler: %w", err))
		}
	}

	if ap.vadDetector != nil {
		if err := ap.vadDetector.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close VAD detector: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors closing processor components: %v", errors)
	}

	return nil
}

// GetConfig returns the current processing configuration
func (ap *EnhancedAudioProcessor) GetConfig() types.ProcessingConfig {
	return ap.config
}

// UpdateConfig updates the processing configuration and reinitializes components if needed
func (ap *EnhancedAudioProcessor) UpdateConfig(config types.ProcessingConfig) error {
	// Close existing components if types changed
	if config.ResamplerType != ap.config.ResamplerType {
		if ap.resampler != nil {
			ap.resampler.Close()
		}
		ap.config.ResamplerType = config.ResamplerType
		ap.config.ResamplerQuality = config.ResamplerQuality
		if err := ap.createResampler(); err != nil {
			return fmt.Errorf("failed to recreate resampler: %w", err)
		}
	}

	if config.VADType != ap.config.VADType {
		if ap.vadDetector != nil {
			ap.vadDetector.Close()
		}
		ap.config.VADType = config.VADType
		ap.config.VADMode = config.VADMode
		if err := ap.createVADDetector(); err != nil {
			return fmt.Errorf("failed to recreate VAD detector: %w", err)
		}
	}

	if config.WAVExporterType != ap.config.WAVExporterType {
		ap.config.WAVExporterType = config.WAVExporterType
		ap.wavExporter = ap.createWAVExporter()
	}

	if config.FFTType != ap.config.FFTType {
		ap.config.FFTType = config.FFTType
		ap.fftProcessor = ap.createFFTProcessor()
	}

	// Update other configuration fields
	ap.config = config

	return nil
}
