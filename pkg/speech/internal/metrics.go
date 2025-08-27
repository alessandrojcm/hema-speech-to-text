package internal

import (
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// SpeechMetrics collects speech recognition performance metrics
type SpeechMetrics struct {
	mu sync.RWMutex

	// Processing metrics
	totalTranscriptions      int64
	successfulTranscriptions int64
	failedTranscriptions     int64

	// Performance metrics
	processingTimes  []time.Duration
	confidenceScores []float64

	// HEMA-specific metrics
	hemaTermsDetected int64
	hemaTermsTotal    int64

	// Model metrics
	modelLoadTimes        map[string]time.Duration
	metalAccelerationUsed int64

	// Memory metrics
	peakMemoryUsage    int64
	averageMemoryUsage int64

	// Audio preprocessing metrics
	preprocessingTimes    []time.Duration
	vadDetections         int64
	vadFalsePositives     int64
	resamplingOperations  int64
	noiseReductionApplied int64

	logger zerolog.Logger
}

// NewSpeechMetrics creates a new speech metrics collector
func NewSpeechMetrics(logger zerolog.Logger) *SpeechMetrics {
	return &SpeechMetrics{
		processingTimes:    make([]time.Duration, 0, 1000),
		confidenceScores:   make([]float64, 0, 1000),
		preprocessingTimes: make([]time.Duration, 0, 1000),
		modelLoadTimes:     make(map[string]time.Duration),
		logger:             logger.With().Str("component", "speech_metrics").Logger(),
	}
}

// RecordTranscription records a transcription attempt
func (sm *SpeechMetrics) RecordTranscription(
	success bool,
	processingTime time.Duration,
	confidence float64,
	hemaTermsFound int,
	metalUsed bool,
) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.totalTranscriptions++
	if success {
		sm.successfulTranscriptions++
	} else {
		sm.failedTranscriptions++
	}

	// Record processing time (keep last 1000)
	if len(sm.processingTimes) >= 1000 {
		sm.processingTimes = sm.processingTimes[1:]
	}
	sm.processingTimes = append(sm.processingTimes, processingTime)

	// Record confidence score
	if len(sm.confidenceScores) >= 1000 {
		sm.confidenceScores = sm.confidenceScores[1:]
	}
	sm.confidenceScores = append(sm.confidenceScores, confidence)

	// Record HEMA terms
	sm.hemaTermsDetected += int64(hemaTermsFound)
	sm.hemaTermsTotal++

	// Record Metal usage
	if metalUsed {
		sm.metalAccelerationUsed++
	}
}

// RecordModelLoad records model loading time
func (sm *SpeechMetrics) RecordModelLoad(modelName string, loadTime time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.modelLoadTimes[modelName] = loadTime
}

// RecordPreprocessing records audio preprocessing metrics
func (sm *SpeechMetrics) RecordPreprocessing(
	preprocessingTime time.Duration,
	vadDetected bool,
	vadFalsePositive bool,
	resamplingApplied bool,
	noiseReductionApplied bool,
) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Record preprocessing time (keep last 1000)
	if len(sm.preprocessingTimes) >= 1000 {
		sm.preprocessingTimes = sm.preprocessingTimes[1:]
	}
	sm.preprocessingTimes = append(sm.preprocessingTimes, preprocessingTime)

	// Record VAD metrics
	if vadDetected {
		sm.vadDetections++
	}
	if vadFalsePositive {
		sm.vadFalsePositives++
	}

	// Record processing operations
	if resamplingApplied {
		sm.resamplingOperations++
	}
	if noiseReductionApplied {
		sm.noiseReductionApplied++
	}
}

// RecordMemoryUsage records memory usage statistics
func (sm *SpeechMetrics) RecordMemoryUsage(current, peak int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if peak > sm.peakMemoryUsage {
		sm.peakMemoryUsage = peak
	}

	// Simple moving average for memory usage
	if sm.averageMemoryUsage == 0 {
		sm.averageMemoryUsage = current
	} else {
		sm.averageMemoryUsage = (sm.averageMemoryUsage + current) / 2
	}
}

// GetMetrics returns current metrics snapshot
func (sm *SpeechMetrics) GetMetrics() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var avgProcessingTime time.Duration
	if len(sm.processingTimes) > 0 {
		sum := time.Duration(0)
		for _, t := range sm.processingTimes {
			sum += t
		}
		avgProcessingTime = sum / time.Duration(len(sm.processingTimes))
	}

	var avgPreprocessingTime time.Duration
	if len(sm.preprocessingTimes) > 0 {
		sum := time.Duration(0)
		for _, t := range sm.preprocessingTimes {
			sum += t
		}
		avgPreprocessingTime = sum / time.Duration(len(sm.preprocessingTimes))
	}

	var avgConfidence float64
	if len(sm.confidenceScores) > 0 {
		sum := 0.0
		for _, c := range sm.confidenceScores {
			sum += c
		}
		avgConfidence = sum / float64(len(sm.confidenceScores))
	}

	var successRate float64
	if sm.totalTranscriptions > 0 {
		successRate = float64(sm.successfulTranscriptions) / float64(sm.totalTranscriptions) * 100
	}

	var hemaDetectionRate float64
	if sm.hemaTermsTotal > 0 {
		hemaDetectionRate = float64(sm.hemaTermsDetected) / float64(sm.hemaTermsTotal)
	}

	var metalUsageRate float64
	if sm.totalTranscriptions > 0 {
		metalUsageRate = float64(sm.metalAccelerationUsed) / float64(sm.totalTranscriptions) * 100
	}

	var vadAccuracy float64
	if sm.vadDetections > 0 {
		vadAccuracy = float64(sm.vadDetections-sm.vadFalsePositives) / float64(sm.vadDetections) * 100
	}

	return map[string]interface{}{
		"total_transcriptions":      sm.totalTranscriptions,
		"successful_transcriptions": sm.successfulTranscriptions,
		"failed_transcriptions":     sm.failedTranscriptions,
		"success_rate":              successRate,
		"avg_processing_time":       avgProcessingTime,
		"avg_preprocessing_time":    avgPreprocessingTime,
		"avg_confidence":            avgConfidence,
		"hema_terms_detected":       sm.hemaTermsDetected,
		"hema_detection_rate":       hemaDetectionRate,
		"metal_usage_rate":          metalUsageRate,
		"peak_memory_usage":         sm.peakMemoryUsage,
		"avg_memory_usage":          sm.averageMemoryUsage,
		"model_load_times":          sm.modelLoadTimes,
		"vad_detections":            sm.vadDetections,
		"vad_false_positives":       sm.vadFalsePositives,
		"vad_accuracy":              vadAccuracy,
		"resampling_operations":     sm.resamplingOperations,
		"noise_reduction_applied":   sm.noiseReductionApplied,
	}
}

// Reset clears all metrics
func (sm *SpeechMetrics) Reset() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.totalTranscriptions = 0
	sm.successfulTranscriptions = 0
	sm.failedTranscriptions = 0
	sm.processingTimes = make([]time.Duration, 0, 1000)
	sm.confidenceScores = make([]float64, 0, 1000)
	sm.preprocessingTimes = make([]time.Duration, 0, 1000)
	sm.hemaTermsDetected = 0
	sm.hemaTermsTotal = 0
	sm.metalAccelerationUsed = 0
	sm.peakMemoryUsage = 0
	sm.averageMemoryUsage = 0
	sm.modelLoadTimes = make(map[string]time.Duration)
	sm.vadDetections = 0
	sm.vadFalsePositives = 0
	sm.resamplingOperations = 0
	sm.noiseReductionApplied = 0

	sm.logger.Info().Msg("Speech metrics reset")
}

// LogSummary logs a summary of current metrics
func (sm *SpeechMetrics) LogSummary() {
	metrics := sm.GetMetrics()

	sm.logger.Info().
		Int64("total_transcriptions", metrics["total_transcriptions"].(int64)).
		Float64("success_rate", metrics["success_rate"].(float64)).
		Interface("avg_processing_time", metrics["avg_processing_time"]).
		Float64("avg_confidence", metrics["avg_confidence"].(float64)).
		Float64("hema_detection_rate", metrics["hema_detection_rate"].(float64)).
		Float64("metal_usage_rate", metrics["metal_usage_rate"].(float64)).
		Msg("Speech recognition metrics summary")
}
