package types

import (
	"time"
)

type AudioSample struct {
	Data       []float32
	Timestamp  time.Time
	Channels   int
	SampleRate int
}

type AudioSegment struct {
	ID        string
	Data      []float32
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Metadata  SegmentMetadata
}

type SegmentMetadata struct {
	SampleRate    int
	Channels      int
	BitDepth      int
	Quality       float64
	HasVoice      bool
	NoiseLevel    float64
	PeakAmplitude float64
	RMSLevel      float64
}

type DeviceInfo struct {
	ID                int
	Name              string
	MaxInputChannels  int
	MaxOutputChannels int
	DefaultSampleRate float64
	IsDefault         bool
	IsAvailable       bool
}

type ExtractionRequest struct {
	Duration time.Duration
	EndTime  time.Time
	Format   string
}

type CaptureStats struct {
	SamplesProcessed  int64
	DroppedSamples    int64
	AverageLatency    time.Duration
	BufferUtilization float64
	DeviceHealth      DeviceHealth
	LastUpdate        time.Time
}

type DeviceHealth struct {
	IsConnected   bool
	SignalLevel   float64
	NoiseFloor    float64
	LastHeartbeat time.Time
	ErrorCount    int
	WarningCount  int
}

type BufferStats struct {
	TotalCapacity      int64
	UsedCapacity       int64
	UtilizationPercent float64
	SegmentCount       int
	OldestSegmentAge   time.Duration
	WritePosition      int64
	ReadPosition       int64
	OverwriteCount     int64
}

type ProcessorStats struct {
	ProcessedSamples   int64
	ProcessingLatency  time.Duration
	QualityScore       float64
	VoiceDetectionRate float64
	NoiseReductionGain float64
}

type HealthStatus int

const (
	HealthStatusUnknown HealthStatus = iota
	HealthStatusHealthy
	HealthStatusWarning
	HealthStatusCritical
	HealthStatusFailed
)

func (hs HealthStatus) String() string {
	switch hs {
	case HealthStatusHealthy:
		return "healthy"
	case HealthStatusWarning:
		return "warning"
	case HealthStatusCritical:
		return "critical"
	case HealthStatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

type SystemHealth struct {
	CaptureHealth   CaptureStats
	BufferHealth    BufferStats
	DeviceHealth    DeviceHealth
	ProcessorHealth ProcessorStats
	OverallStatus   HealthStatus
	LastUpdate      time.Time
}

type LatencyStats struct {
	Average time.Duration
	Min     time.Duration
	Max     time.Duration
	Count   int64
}

type SystemMetrics struct {
	// Processing metrics
	TotalSamplesProcessed int64
	TotalProcessingTime   time.Duration
	AverageQualityScore   float64

	// VAD metrics
	VADDetections     int64
	VADFalsePositives int64

	// Extraction metrics
	TotalExtractions      int64
	FailedExtractions     int64
	ExtractionFailureRate float64
	AverageExtractionTime time.Duration

	// System metrics
	MemoryUsage int64
	CPUUsage    float64

	// Timestamp
	LastUpdate time.Time
}
