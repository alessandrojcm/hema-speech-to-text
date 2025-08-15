//go:build !noaudio
// +build !noaudio

package audio

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"github.com/your-org/hema-replay-system/pkg/audio/buffer"
	"github.com/your-org/hema-replay-system/pkg/audio/capture"
	"github.com/your-org/hema-replay-system/pkg/audio/processing"
	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

// MetricsCollector collects and aggregates audio system metrics
type MetricsCollector struct {
	mu sync.RWMutex

	// Processing metrics
	totalSamplesProcessed int64
	totalProcessingTime   time.Duration
	qualityScores         []float64
	vadDetections         int64
	vadFalsePositives     int64

	// Extraction metrics
	totalExtractions  int64
	failedExtractions int64
	extractionTimes   []time.Duration

	// System metrics
	memoryUsage int64
	cpuUsage    float64

	logger zerolog.Logger
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(logger zerolog.Logger) *MetricsCollector {
	return &MetricsCollector{
		qualityScores:   make([]float64, 0, 1000),
		extractionTimes: make([]time.Duration, 0, 1000),
		logger:          logger.With().Str("component", "metrics_collector").Logger(),
	}
}

// RecordProcessing records processing metrics
func (mc *MetricsCollector) RecordProcessing(samplesProcessed int64, processingTime time.Duration, qualityScore float64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	atomic.AddInt64(&mc.totalSamplesProcessed, samplesProcessed)
	mc.totalProcessingTime += processingTime

	if len(mc.qualityScores) >= 1000 {
		mc.qualityScores = mc.qualityScores[1:]
	}
	mc.qualityScores = append(mc.qualityScores, qualityScore)
}

// RecordVADDetection records VAD detection metrics
func (mc *MetricsCollector) RecordVADDetection(detected bool, falsePositive bool) {
	if detected {
		atomic.AddInt64(&mc.vadDetections, 1)
	}
	if falsePositive {
		atomic.AddInt64(&mc.vadFalsePositives, 1)
	}
}

// RecordExtraction records extraction metrics
func (mc *MetricsCollector) RecordExtraction(duration time.Duration, success bool) {
	atomic.AddInt64(&mc.totalExtractions, 1)
	if !success {
		atomic.AddInt64(&mc.failedExtractions, 1)
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	if len(mc.extractionTimes) >= 1000 {
		mc.extractionTimes = mc.extractionTimes[1:]
	}
	mc.extractionTimes = append(mc.extractionTimes, duration)
}

// GetMetrics returns current metrics snapshot
func (mc *MetricsCollector) GetMetrics() types.SystemMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	var avgQuality float64
	if len(mc.qualityScores) > 0 {
		sum := 0.0
		for _, score := range mc.qualityScores {
			sum += score
		}
		avgQuality = sum / float64(len(mc.qualityScores))
	}

	var avgExtractionTime time.Duration
	if len(mc.extractionTimes) > 0 {
		sum := time.Duration(0)
		for _, duration := range mc.extractionTimes {
			sum += duration
		}
		avgExtractionTime = sum / time.Duration(len(mc.extractionTimes))
	}

	totalExtractions := atomic.LoadInt64(&mc.totalExtractions)
	failedExtractions := atomic.LoadInt64(&mc.failedExtractions)

	var failureRate float64
	if totalExtractions > 0 {
		failureRate = float64(failedExtractions) / float64(totalExtractions)
	}

	return types.SystemMetrics{
		TotalSamplesProcessed: atomic.LoadInt64(&mc.totalSamplesProcessed),
		TotalProcessingTime:   mc.totalProcessingTime,
		AverageQualityScore:   avgQuality,
		VADDetections:         atomic.LoadInt64(&mc.vadDetections),
		VADFalsePositives:     atomic.LoadInt64(&mc.vadFalsePositives),
		TotalExtractions:      totalExtractions,
		FailedExtractions:     failedExtractions,
		ExtractionFailureRate: failureRate,
		AverageExtractionTime: avgExtractionTime,
		MemoryUsage:           atomic.LoadInt64(&mc.memoryUsage),
		CPUUsage:              mc.cpuUsage,
	}
}

type AudioManager struct {
	config        types.AudioConfig
	captureEngine *capture.CaptureEngine
	ringBuffer    *buffer.RingBuffer
	processor     *processing.AudioProcessor
	deviceManager *capture.DeviceManager
	extractor     *AudioExtractor

	mu      sync.RWMutex
	running bool
	health  types.SystemHealth

	// Enhanced monitoring
	statsChan        chan types.CaptureStats
	healthChan       chan types.SystemHealth
	errorChan        chan error
	metricsCollector *MetricsCollector

	// Performance tracking
	totalExtractions  int64
	failedExtractions int64
	avgExtractionTime time.Duration

	logger zerolog.Logger
}

type AudioExtractor struct {
	config      types.ExtractionConfig
	ringBuffer  *buffer.RingBuffer
	activeTasks map[string]*ExtractionTask
	taskMutex   sync.RWMutex
	semaphore   chan struct{}
	logger      zerolog.Logger
}

type ExtractionTask struct {
	ID        string
	Request   types.ExtractionRequest
	StartTime time.Time
	Done      chan *types.AudioSegment
	Error     chan error
}

func NewAudioManager(config types.AudioConfig, logger zerolog.Logger) (*AudioManager, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	ringBuffer := buffer.NewRingBuffer(
		config.Buffer,
		config.Device.SampleRate,
		config.Device.Channels,
		config.Device.BitDepth,
	)

	processor, err := processing.NewAudioProcessor(
		config.Processing,
		config.Device.SampleRate,
		config.Device.Channels,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create enhanced audio processor: %w", err)
	}

	captureEngine, err := capture.NewCaptureEngine(
		config.Device,
		config.Processing,
		ringBuffer,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create capture engine: %w", err)
	}

	deviceManager := capture.NewDeviceManager(config.Device, logger)

	extractor := NewAudioExtractor(config.Extraction, ringBuffer, logger)
	metricsCollector := NewMetricsCollector(logger)

	return &AudioManager{
		config:           config,
		captureEngine:    captureEngine,
		ringBuffer:       ringBuffer,
		processor:        processor,
		deviceManager:    deviceManager,
		extractor:        extractor,
		metricsCollector: metricsCollector,
		statsChan:        make(chan types.CaptureStats, 1),
		healthChan:       make(chan types.SystemHealth, 1),
		errorChan:        make(chan error, 10),
		logger:           logger.With().Str("component", "audio_manager").Logger(),
	}, nil
}

func (am *AudioManager) Start(ctx context.Context) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if am.running {
		return types.ErrAlreadyRunning
	}

	if err := am.deviceManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start device manager: %w", err)
	}

	if err := am.captureEngine.Start(ctx); err != nil {
		return fmt.Errorf("failed to start capture engine: %w", err)
	}

	go am.healthMonitorLoop(ctx)

	am.running = true
	am.logger.Info().Msg("Audio manager started")

	return nil
}

func (am *AudioManager) ListDevices() ([]types.DeviceInfo, error) {
	if !am.running {
		return nil, types.ErrNotRunning
	}

	devices := am.deviceManager.GetDevices()

	return devices, nil
}

func (am *AudioManager) Stop() error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if !am.running {
		return nil
	}

	if err := am.captureEngine.Stop(); err != nil {
		am.logger.Error().Err(err).Msg("Failed to stop capture engine")
	}

	if err := am.deviceManager.Stop(); err != nil {
		am.logger.Error().Err(err).Msg("Failed to stop device manager")
	}

	am.running = false
	am.logger.Info().Msg("Audio manager stopped")

	return nil
}

func (am *AudioManager) ExtractAudio(ctx context.Context, req types.ExtractionRequest) (*types.AudioSegment, error) {
	if !am.running {
		return nil, types.ErrNotRunning
	}

	startTime := time.Now()
	segment, err := am.extractor.Extract(ctx, req)
	extractionTime := time.Since(startTime)

	// Record metrics
	success := err == nil
	am.metricsCollector.RecordExtraction(extractionTime, success)

	// Update performance tracking
	atomic.AddInt64(&am.totalExtractions, 1)
	if !success {
		atomic.AddInt64(&am.failedExtractions, 1)
	}

	// Update average extraction time (simple moving average)
	am.mu.Lock()
	if am.avgExtractionTime == 0 {
		am.avgExtractionTime = extractionTime
	} else {
		am.avgExtractionTime = (am.avgExtractionTime + extractionTime) / 2
	}
	am.mu.Unlock()

	return segment, err
}

func (am *AudioManager) GetHealth() types.SystemHealth {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.health
}

func (am *AudioManager) GetStats() types.CaptureStats {
	return am.captureEngine.GetStats()
}

func (am *AudioManager) healthMonitorLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			am.updateSystemHealth()
		}
	}
}

func (am *AudioManager) updateSystemHealth() {
	am.mu.Lock()
	defer am.mu.Unlock()

	captureStats := am.captureEngine.GetStats()
	bufferStats := am.ringBuffer.GetStats()
	deviceHealth := am.deviceManager.GetHealth()

	processorStats := types.ProcessorStats{
		ProcessedSamples:  captureStats.SamplesProcessed,
		ProcessingLatency: captureStats.AverageLatency,
		// TODO: placeholders while the real stats are implemented
		QualityScore:       0.8,
		VoiceDetectionRate: 0.7,
		NoiseReductionGain: 0.3,
	}

	overallStatus := am.calculateOverallStatus(captureStats, bufferStats, deviceHealth, processorStats)

	am.health = types.SystemHealth{
		CaptureHealth:   captureStats,
		BufferHealth:    bufferStats,
		DeviceHealth:    deviceHealth,
		ProcessorHealth: processorStats,
		OverallStatus:   overallStatus,
		LastUpdate:      time.Now(),
	}

	select {
	case am.healthChan <- am.health:
	default:
	}
}

func (am *AudioManager) calculateOverallStatus(
	capture types.CaptureStats,
	buffer types.BufferStats,
	device types.DeviceHealth,
	processor types.ProcessorStats,
) types.HealthStatus {
	if !device.IsConnected {
		return types.HealthStatusFailed
	}

	if device.ErrorCount > 10 {
		return types.HealthStatusCritical
	}

	if capture.DroppedSamples > capture.SamplesProcessed/10 {
		return types.HealthStatusWarning
	}

	if buffer.UtilizationPercent > 90.0 {
		return types.HealthStatusWarning
	}

	return types.HealthStatusHealthy
}

func NewAudioExtractor(config types.ExtractionConfig, ringBuffer *buffer.RingBuffer, logger zerolog.Logger) *AudioExtractor {
	return &AudioExtractor{
		config:      config,
		ringBuffer:  ringBuffer,
		activeTasks: make(map[string]*ExtractionTask),
		semaphore:   make(chan struct{}, config.MaxConcurrent),
		logger:      logger.With().Str("component", "audio_extractor").Logger(),
	}
}

func (ae *AudioExtractor) Extract(ctx context.Context, req types.ExtractionRequest) (*types.AudioSegment, error) {
	select {
	case ae.semaphore <- struct{}{}:
		defer func() { <-ae.semaphore }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	if req.Duration == 0 {
		req.Duration = ae.config.DefaultDuration
	}

	if req.EndTime.IsZero() {
		req.EndTime = time.Now()
	}

	segment, err := ae.ringBuffer.Extract(req.Duration, req.EndTime)
	if err != nil {
		return nil, fmt.Errorf("failed to extract audio: %w", err)
	}

	// Note: If format is "wav", the actual WAV conversion should happen
	// at the point of export/use, not here. The segment should always
	// contain raw float32 samples for further processing.

	return segment, nil
}

// GetMetrics returns current system metrics
func (am *AudioManager) GetMetrics() types.SystemMetrics {
	metrics := am.metricsCollector.GetMetrics()
	metrics.LastUpdate = time.Now()
	return metrics
}

// GetPerformanceStats returns performance statistics
func (am *AudioManager) GetPerformanceStats() map[string]interface{} {
	am.mu.RLock()
	defer am.mu.RUnlock()

	totalExtractions := atomic.LoadInt64(&am.totalExtractions)
	failedExtractions := atomic.LoadInt64(&am.failedExtractions)
	successRate := float64(0)
	if totalExtractions > 0 {
		successRate = float64(totalExtractions-failedExtractions) / float64(totalExtractions) * 100
	}

	return map[string]interface{}{
		"total_extractions":    totalExtractions,
		"failed_extractions":   failedExtractions,
		"success_rate_percent": successRate,
		"avg_extraction_ms":    am.avgExtractionTime.Milliseconds(),
		"is_running":           am.running,
		"overall_health":       am.health.OverallStatus.String(),
	}
}

// ExportSegmentToWAV exports an audio segment to WAV format
func (am *AudioManager) ExportSegmentToWAV(segment *types.AudioSegment) ([]byte, error) {
	return processing.ConvertToWAV(segment, am.config.Extraction.OutputSampleRate, am.config.Extraction.OutputChannels)
}

// ExtractAudioConcurrent extracts multiple audio segments concurrently
func (am *AudioManager) ExtractAudioConcurrent(ctx context.Context, requests []types.ExtractionRequest) ([]*types.AudioSegment, []error) {
	if !am.running {
		errors := make([]error, len(requests))
		for i := range errors {
			errors[i] = types.ErrNotRunning
		}
		return nil, errors
	}

	segments := make([]*types.AudioSegment, len(requests))
	errors := make([]error, len(requests))

	var wg sync.WaitGroup
	for i, req := range requests {
		wg.Add(1)
		go func(index int, request types.ExtractionRequest) {
			defer wg.Done()
			segment, err := am.ExtractAudio(ctx, request)
			segments[index] = segment
			errors[index] = err
		}(i, req)
	}

	wg.Wait()
	return segments, errors
}

// ProcessAudioSegment processes an audio segment using the enhanced processor
func (am *AudioManager) ProcessAudioSegment(segment *types.AudioSegment) error {
	if !am.running {
		return types.ErrNotRunning
	}

	startTime := time.Now()
	processed, err := am.processor.Process(segment.Data, segment.StartTime)
	processingTime := time.Since(startTime)

	if err != nil {
		return fmt.Errorf("failed to process audio segment: %w", err)
	}

	// Update segment data with processed audio
	segment.Data = processed

	// Calculate quality score (placeholder - would use actual quality assessment)
	qualityScore := am.calculateQualityScore(processed)
	segment.Metadata.Quality = qualityScore

	// Record processing metrics
	am.metricsCollector.RecordProcessing(int64(len(processed)), processingTime, qualityScore)

	return nil
}

// calculateQualityScore calculates a quality score for processed audio
func (am *AudioManager) calculateQualityScore(samples []float32) float64 {
	if len(samples) == 0 {
		return 0.0
	}

	// Simple quality assessment based on RMS and dynamic range
	var sum float32
	var max, min float32 = samples[0], samples[0]

	for _, sample := range samples {
		sum += sample * sample
		if sample > max {
			max = sample
		}
		if sample < min {
			min = sample
		}
	}

	rms := float64(sum) / float64(len(samples))
	dynamicRange := float64(max - min)

	// Normalize to 0-1 range (this is a simplified calculation)
	qualityScore := (rms + dynamicRange) / 2.0
	if qualityScore > 1.0 {
		qualityScore = 1.0
	}

	return qualityScore
}

// UpdateConfiguration updates the audio manager configuration
func (am *AudioManager) UpdateConfiguration(config types.AudioConfig) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Update processor configuration
	if err := am.processor.UpdateConfig(config.Processing); err != nil {
		return fmt.Errorf("failed to update processor config: %w", err)
	}

	am.config = config
	am.logger.Info().Msg("Audio manager configuration updated")

	return nil
}
