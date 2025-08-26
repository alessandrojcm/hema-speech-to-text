package pipeline

import (
	"sync"
	"sync/atomic"
	"time"
)

// MetricsCollector collects and manages pipeline metrics
type MetricsCollector struct {
	// Processing metrics
	processingTimes map[string]*DurationMetric
	errorCounts     map[string]int64
	successCount    int64
	totalCount      int64

	// Pipeline specific metrics
	segmentsProcessed  int64
	segmentsFailed     int64
	vadDetections      int64
	vadFalsePositives  int64
	avgConfidenceScore float64
	lastUpdateTime     time.Time

	mu sync.RWMutex
}

// DurationMetric tracks duration statistics
type DurationMetric struct {
	count    int64
	total    time.Duration
	min      time.Duration
	max      time.Duration
	lastTime time.Duration
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		processingTimes: make(map[string]*DurationMetric),
		errorCounts:     make(map[string]int64),
		lastUpdateTime:  time.Now(),
	}
}

// RecordProcessingTime records processing time for a specific stage
func (mc *MetricsCollector) RecordProcessingTime(stage string, duration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	metric, ok := mc.processingTimes[stage]
	if !ok {
		metric = &DurationMetric{
			min: duration,
			max: duration,
		}
		mc.processingTimes[stage] = metric
	}

	atomic.AddInt64(&metric.count, 1)
	metric.total += duration
	metric.lastTime = duration

	if duration < metric.min {
		metric.min = duration
	}
	if duration > metric.max {
		metric.max = duration
	}

	mc.lastUpdateTime = time.Now()
}

// RecordSuccess records a successful operation
func (mc *MetricsCollector) RecordSuccess() {
	atomic.AddInt64(&mc.successCount, 1)
	atomic.AddInt64(&mc.totalCount, 1)
}

// RecordError records an error by type
func (mc *MetricsCollector) RecordError(err error) {
	atomic.AddInt64(&mc.totalCount, 1)

	mc.mu.Lock()
	defer mc.mu.Unlock()

	errorType := "unknown"
	if err != nil {
		errorType = "general" // Could be enhanced to classify error types
	}

	mc.errorCounts[errorType]++
	mc.lastUpdateTime = time.Now()
}

// RecordSegmentProcessed records a processed segment
func (mc *MetricsCollector) RecordSegmentProcessed(success bool) {
	if success {
		atomic.AddInt64(&mc.segmentsProcessed, 1)
	} else {
		atomic.AddInt64(&mc.segmentsFailed, 1)
	}
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

// UpdateConfidenceScore updates the average confidence score
func (mc *MetricsCollector) UpdateConfidenceScore(score float64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Simple moving average
	if mc.avgConfidenceScore == 0 {
		mc.avgConfidenceScore = score
	} else {
		mc.avgConfidenceScore = (mc.avgConfidenceScore + score) / 2.0
	}

	mc.lastUpdateTime = time.Now()
}

// GetStats returns current metrics snapshot
func (mc *MetricsCollector) GetStats() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	stats := make(map[string]interface{})

	// Success rate
	totalCount := atomic.LoadInt64(&mc.totalCount)
	successCount := atomic.LoadInt64(&mc.successCount)
	if totalCount > 0 {
		stats["success_rate"] = float64(successCount) / float64(totalCount)
	} else {
		stats["success_rate"] = 0.0
	}

	// Processing times
	for stage, metric := range mc.processingTimes {
		count := atomic.LoadInt64(&metric.count)
		if count > 0 {
			stats[stage+"_avg_ms"] = metric.total.Milliseconds() / count
			stats[stage+"_min_ms"] = metric.min.Milliseconds()
			stats[stage+"_max_ms"] = metric.max.Milliseconds()
			stats[stage+"_last_ms"] = metric.lastTime.Milliseconds()
			stats[stage+"_count"] = count
		}
	}

	// Error counts
	for errorType, count := range mc.errorCounts {
		stats["errors_"+errorType] = count
	}

	// Segment metrics
	stats["segments_processed"] = atomic.LoadInt64(&mc.segmentsProcessed)
	stats["segments_failed"] = atomic.LoadInt64(&mc.segmentsFailed)

	// VAD metrics
	stats["vad_detections"] = atomic.LoadInt64(&mc.vadDetections)
	stats["vad_false_positives"] = atomic.LoadInt64(&mc.vadFalsePositives)

	// Overall metrics
	stats["total_operations"] = totalCount
	stats["successful_operations"] = successCount
	stats["avg_confidence_score"] = mc.avgConfidenceScore
	stats["last_update"] = mc.lastUpdateTime

	return stats
}

// Reset resets all metrics
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Reset atomic counters
	atomic.StoreInt64(&mc.successCount, 0)
	atomic.StoreInt64(&mc.totalCount, 0)
	atomic.StoreInt64(&mc.segmentsProcessed, 0)
	atomic.StoreInt64(&mc.segmentsFailed, 0)
	atomic.StoreInt64(&mc.vadDetections, 0)
	atomic.StoreInt64(&mc.vadFalsePositives, 0)

	// Reset maps
	mc.processingTimes = make(map[string]*DurationMetric)
	mc.errorCounts = make(map[string]int64)
	mc.avgConfidenceScore = 0.0
	mc.lastUpdateTime = time.Now()
}
