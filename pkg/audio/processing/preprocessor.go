package processing

import (
	"time"

	"github.com/rs/zerolog"
	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

type AudioProcessor struct {
	config     types.ProcessingConfig
	sampleRate int
	channels   int

	normalizer   *Normalizer
	noiseReducer *NoiseReducer
	filters      []AudioFilter
	vadDetector  *VADDetector
	qualityMeter *QualityMeter

	processingBuffer []float32
	historyBuffer    []float32

	logger zerolog.Logger
}

type AudioFilter interface {
	Apply(samples []float32) error
}

type Normalizer struct {
	targetLevel float32
}

type NoiseReducer struct {
	noiseProfile []float32
	reduction    float32
}

type VADDetector struct {
	threshold float64
}

type HighpassFilter struct {
	cutoffFreq float64
	sampleRate int
	prevInput  float32
	prevOutput float32
}

type LowpassFilter struct {
	cutoffFreq float64
	sampleRate int
	prevOutput float32
}

func NewAudioProcessor(config types.ProcessingConfig, sampleRate, channels int, logger zerolog.Logger) *AudioProcessor {
	ap := &AudioProcessor{
		config:     config,
		sampleRate: sampleRate,
		channels:   channels,
		logger:     logger.With().Str("component", "audio_processor").Logger(),
	}

	if config.EnablePreprocessing {
		ap.normalizer = &Normalizer{targetLevel: 0.8}
		ap.noiseReducer = &NoiseReducer{reduction: 0.5}
		ap.vadDetector = &VADDetector{threshold: config.VADThreshold}
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

	return ap
}

func (ap *AudioProcessor) Process(samples []float32, timestamp time.Time) ([]float32, error) {
	if len(samples) == 0 {
		return samples, nil
	}

	if len(ap.processingBuffer) < len(samples) {
		ap.processingBuffer = make([]float32, len(samples))
	}

	copy(ap.processingBuffer[:len(samples)], samples)
	processed := ap.processingBuffer[:len(samples)]

	if ap.config.EnablePreprocessing {
		for _, filter := range ap.filters {
			if err := filter.Apply(processed); err != nil {
				ap.logger.Warn().Err(err).Msg("Filter application failed")
			}
		}

		if ap.config.NoiseReduction && ap.noiseReducer != nil {
			if err := ap.noiseReducer.Reduce(processed); err != nil {
				ap.logger.Warn().Err(err).Msg("Noise reduction failed")
			}
		}

		if ap.config.Normalization && ap.normalizer != nil {
			if err := ap.normalizer.Normalize(processed); err != nil {
				ap.logger.Warn().Err(err).Msg("Normalization failed")
			}
		}
	}

	ap.updateHistory(processed)

	return processed, nil
}

func (ap *AudioProcessor) updateHistory(samples []float32) {
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

func (n *Normalizer) Normalize(samples []float32) error {
	NormalizeAudio(samples)
	return nil
}

func (nr *NoiseReducer) Reduce(samples []float32) error {
	if len(samples) == 0 {
		return nil
	}

	for i := range samples {
		samples[i] *= (1.0 - nr.reduction)
	}

	return nil
}

func (hf *HighpassFilter) Apply(samples []float32) error {
	ApplyHighpassFilter(samples, hf.cutoffFreq, hf.sampleRate)
	return nil
}

func (lf *LowpassFilter) Apply(samples []float32) error {
	ApplyLowpassFilter(samples, lf.cutoffFreq, lf.sampleRate)
	return nil
}

func (vad *VADDetector) DetectVoice(samples []float32) bool {
	if len(samples) == 0 {
		return false
	}

	var sum float64
	for _, sample := range samples {
		sum += float64(sample * sample)
	}
	rms := sum / float64(len(samples))

	return rms > vad.threshold
}
