package internal

import (
	"sync"
	"time"

	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

type LatencyTracker struct {
	mu         sync.RWMutex
	samples    []time.Duration
	index      int
	count      int64
	sum        time.Duration
	min        time.Duration
	max        time.Duration
	windowSize int
}

func NewLatencyTracker(windowSize int) *LatencyTracker {
	return &LatencyTracker{
		windowSize: windowSize,
		samples:    make([]time.Duration, windowSize),
		min:        time.Duration(0),
		max:        time.Duration(0),
	}
}

func (lt *LatencyTracker) Record(latency time.Duration) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	if lt.count == 0 {
		lt.min = latency
		lt.max = latency
	} else {
		if latency < lt.min {
			lt.min = latency
		}
		if latency > lt.max {
			lt.max = latency
		}
	}

	if lt.count < int64(lt.windowSize) {
		lt.samples[lt.index] = latency
		lt.sum += latency
		lt.count++
	} else {
		lt.sum = lt.sum - lt.samples[lt.index] + latency
		lt.samples[lt.index] = latency
	}

	lt.index = (lt.index + 1) % lt.windowSize
}

func (lt *LatencyTracker) GetStats() types.LatencyStats {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	if lt.count == 0 {
		return types.LatencyStats{}
	}

	return types.LatencyStats{
		Average: time.Duration(int64(lt.sum) / lt.count),
		Min:     lt.min,
		Max:     lt.max,
		Count:   lt.count,
	}
}

func (lt *LatencyTracker) Reset() {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	lt.samples = make([]time.Duration, lt.windowSize)
	lt.index = 0
	lt.count = 0
	lt.sum = 0
	lt.min = 0
	lt.max = 0
}

type PerformanceMonitor struct {
	mu               sync.RWMutex
	samplesProcessed int64
	samplesDropped   int64
	extractionCount  int64
	errorCount       int64

	captureLatency    *LatencyTracker
	extractionLatency *LatencyTracker
	processingLatency *LatencyTracker

	memoryUsage       int64
	cpuUsage          float64
	bufferUtilization float64

	averageQuality     float64
	voiceDetectionRate float64

	startTime  time.Time
	lastUpdate time.Time
}

func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		captureLatency:    NewLatencyTracker(100),
		extractionLatency: NewLatencyTracker(50),
		processingLatency: NewLatencyTracker(100),
		startTime:         time.Now(),
		lastUpdate:        time.Now(),
	}
}

func (pm *PerformanceMonitor) RecordSamplesProcessed(count int64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.samplesProcessed += count
	pm.lastUpdate = time.Now()
}

func (pm *PerformanceMonitor) RecordSamplesDropped(count int64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.samplesDropped += count
	pm.lastUpdate = time.Now()
}

func (pm *PerformanceMonitor) RecordExtraction() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.extractionCount++
	pm.lastUpdate = time.Now()
}

func (pm *PerformanceMonitor) RecordError() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.errorCount++
	pm.lastUpdate = time.Now()
}

func (pm *PerformanceMonitor) RecordCaptureLatency(latency time.Duration) {
	pm.captureLatency.Record(latency)
	pm.mu.Lock()
	pm.lastUpdate = time.Now()
	pm.mu.Unlock()
}

func (pm *PerformanceMonitor) RecordExtractionLatency(latency time.Duration) {
	pm.extractionLatency.Record(latency)
	pm.mu.Lock()
	pm.lastUpdate = time.Now()
	pm.mu.Unlock()
}

func (pm *PerformanceMonitor) RecordProcessingLatency(latency time.Duration) {
	pm.processingLatency.Record(latency)
	pm.mu.Lock()
	pm.lastUpdate = time.Now()
	pm.mu.Unlock()
}

func (pm *PerformanceMonitor) UpdateResourceUsage(memoryUsage int64, cpuUsage float64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.memoryUsage = memoryUsage
	pm.cpuUsage = cpuUsage
	pm.lastUpdate = time.Now()
}

func (pm *PerformanceMonitor) UpdateBufferUtilization(utilization float64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.bufferUtilization = utilization
	pm.lastUpdate = time.Now()
}

func (pm *PerformanceMonitor) UpdateQualityMetrics(averageQuality, voiceDetectionRate float64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.averageQuality = averageQuality
	pm.voiceDetectionRate = voiceDetectionRate
	pm.lastUpdate = time.Now()
}

func (pm *PerformanceMonitor) GetMetrics() PerformanceMetrics {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	uptime := time.Since(pm.startTime)

	var samplesPerSecond float64
	if uptime.Seconds() > 0 {
		samplesPerSecond = float64(pm.samplesProcessed) / uptime.Seconds()
	}

	var dropRate float64
	if pm.samplesProcessed > 0 {
		dropRate = float64(pm.samplesDropped) / float64(pm.samplesProcessed) * 100.0
	}

	var errorRate float64
	if pm.extractionCount > 0 {
		errorRate = float64(pm.errorCount) / float64(pm.extractionCount) * 100.0
	}

	return PerformanceMetrics{
		Uptime:             uptime,
		SamplesProcessed:   pm.samplesProcessed,
		SamplesDropped:     pm.samplesDropped,
		SamplesPerSecond:   samplesPerSecond,
		DropRate:           dropRate,
		ExtractionCount:    pm.extractionCount,
		ErrorCount:         pm.errorCount,
		ErrorRate:          errorRate,
		CaptureLatency:     pm.captureLatency.GetStats(),
		ExtractionLatency:  pm.extractionLatency.GetStats(),
		ProcessingLatency:  pm.processingLatency.GetStats(),
		MemoryUsage:        pm.memoryUsage,
		CPUUsage:           pm.cpuUsage,
		BufferUtilization:  pm.bufferUtilization,
		AverageQuality:     pm.averageQuality,
		VoiceDetectionRate: pm.voiceDetectionRate,
		LastUpdate:         pm.lastUpdate,
	}
}

func (pm *PerformanceMonitor) Reset() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.samplesProcessed = 0
	pm.samplesDropped = 0
	pm.extractionCount = 0
	pm.errorCount = 0
	pm.memoryUsage = 0
	pm.cpuUsage = 0.0
	pm.bufferUtilization = 0.0
	pm.averageQuality = 0.0
	pm.voiceDetectionRate = 0.0
	pm.startTime = time.Now()
	pm.lastUpdate = time.Now()

	pm.captureLatency.Reset()
	pm.extractionLatency.Reset()
	pm.processingLatency.Reset()
}

type PerformanceMetrics struct {
	Uptime             time.Duration
	SamplesProcessed   int64
	SamplesDropped     int64
	SamplesPerSecond   float64
	DropRate           float64
	ExtractionCount    int64
	ErrorCount         int64
	ErrorRate          float64
	CaptureLatency     types.LatencyStats
	ExtractionLatency  types.LatencyStats
	ProcessingLatency  types.LatencyStats
	MemoryUsage        int64
	CPUUsage           float64
	BufferUtilization  float64
	AverageQuality     float64
	VoiceDetectionRate float64
	LastUpdate         time.Time
}

type ResourceMonitor struct {
	mu         sync.RWMutex
	monitoring bool
	stopChan   chan struct{}
	updateChan chan ResourceUsage
}

type ResourceUsage struct {
	MemoryUsage int64
	CPUUsage    float64
	Timestamp   time.Time
}

func NewResourceMonitor() *ResourceMonitor {
	return &ResourceMonitor{
		stopChan:   make(chan struct{}),
		updateChan: make(chan ResourceUsage, 10),
	}
}

func (rm *ResourceMonitor) Start() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.monitoring {
		return
	}

	rm.monitoring = true
	go rm.monitorLoop()
}

func (rm *ResourceMonitor) Stop() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if !rm.monitoring {
		return
	}

	close(rm.stopChan)
	rm.monitoring = false
}

func (rm *ResourceMonitor) GetUsage() <-chan ResourceUsage {
	return rm.updateChan
}

func (rm *ResourceMonitor) monitorLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-rm.stopChan:
			return
		case <-ticker.C:
			usage := rm.collectResourceUsage()
			select {
			case rm.updateChan <- usage:
			default:
			}
		}
	}
}

func (rm *ResourceMonitor) collectResourceUsage() ResourceUsage {
	return ResourceUsage{
		MemoryUsage: rm.getMemoryUsage(),
		CPUUsage:    rm.getCPUUsage(),
		Timestamp:   time.Now(),
	}
}

func (rm *ResourceMonitor) getMemoryUsage() int64 {
	return 0
}

func (rm *ResourceMonitor) getCPUUsage() float64 {
	return 0.0
}
