package buffer

import (
	"sync"
	"time"

	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

type RingBuffer struct {
	mu           sync.RWMutex
	data         []float32
	writePos     int64
	readPos      int64
	size         int64
	sampleRate   int
	channels     int
	bitDepth     int
	segmentSize  int64
	segments     []SegmentInfo
	overwritten  int64
	totalWritten int64
	startTime    time.Time
	lastWrite    time.Time
}

func NewRingBuffer(config types.BufferConfig, sampleRate, channels, bitDepth int) *RingBuffer {
	totalSamples := int64(config.Duration.Seconds() * float64(sampleRate) * float64(channels))
	segmentSamples := int64(config.SegmentSize.Seconds() * float64(sampleRate) * float64(channels))

	numSegments := totalSamples / segmentSamples
	if totalSamples%segmentSamples != 0 {
		numSegments++
	}

	return &RingBuffer{
		data:        make([]float32, totalSamples),
		size:        totalSamples,
		sampleRate:  sampleRate,
		channels:    channels,
		bitDepth:    bitDepth,
		segmentSize: segmentSamples,
		segments:    make([]SegmentInfo, numSegments),
		startTime:   time.Now(),
	}
}

func (rb *RingBuffer) Write(samples []float32, timestamp time.Time) error {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if len(samples) == 0 {
		return types.ErrEmptyData
	}

	startPos := rb.writePos % rb.size
	endPos := (rb.writePos + int64(len(samples))) % rb.size

	if endPos < startPos {
		firstPart := rb.size - startPos
		copy(rb.data[startPos:], samples[:firstPart])
		copy(rb.data[0:endPos], samples[firstPart:])
	} else {
		copy(rb.data[startPos:startPos+int64(len(samples))], samples)
	}

	rb.writePos += int64(len(samples))
	rb.totalWritten += int64(len(samples))
	rb.lastWrite = timestamp

	if rb.writePos > rb.size {
		rb.overwritten += int64(len(samples))
	}

	rb.updateSegmentMetadata(startPos, int64(len(samples)), timestamp)

	return nil
}

func (rb *RingBuffer) Extract(duration time.Duration, endTime time.Time) (*types.AudioSegment, error) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	sampleCount := int64(duration.Seconds() * float64(rb.sampleRate) * float64(rb.channels))

	// Always use the current write position as the reference point
	// This ensures we're extracting from the actual buffered data
	endPos := rb.writePos
	startPos := endPos - sampleCount

	// Check if we have enough data in the buffer
	// The buffer can only hold rb.size samples, so we can't extract more than what's available
	availableData := rb.writePos
	if availableData > rb.size {
		availableData = rb.size
	}

	if sampleCount > availableData {
		return nil, types.ErrInsufficientData
	}

	// Check if the requested data has been overwritten
	if startPos < 0 || (rb.writePos > rb.size && startPos < rb.writePos-rb.size) {
		return nil, types.ErrInsufficientData
	}

	data := make([]float32, sampleCount)
	if err := rb.extractRange(startPos, sampleCount, data); err != nil {
		return nil, err
	}

	metadata := calculateSegmentMetadata(data, rb.sampleRate, rb.channels, rb.bitDepth)

	segment := createAudioSegment(
		data,
		endTime.Add(-duration),
		endTime,
		metadata,
	)

	return segment, nil
}

func (rb *RingBuffer) GetStats() types.BufferStats {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	usedCapacity := rb.totalWritten
	if usedCapacity > rb.size {
		usedCapacity = rb.size
	}

	utilizationPercent := float64(usedCapacity) / float64(rb.size) * 100.0

	oldestSegmentAge := time.Duration(0)
	if !rb.lastWrite.IsZero() {
		oldestSegmentAge = time.Since(rb.lastWrite)
	}

	return types.BufferStats{
		TotalCapacity:      rb.size,
		UsedCapacity:       usedCapacity,
		UtilizationPercent: utilizationPercent,
		SegmentCount:       len(rb.segments),
		OldestSegmentAge:   oldestSegmentAge,
		WritePosition:      rb.writePos,
		ReadPosition:       rb.readPos,
		OverwriteCount:     rb.overwritten,
	}
}

// timeToPosition is deprecated - we now always use writePos as reference
// Keeping for compatibility but should be removed in future refactor
func (rb *RingBuffer) timeToPosition(t time.Time) int64 {
	// Always return current write position
	// Time-based positioning proved unreliable after long runs
	return rb.writePos
}

func (rb *RingBuffer) extractRange(startPos, sampleCount int64, dest []float32) error {
	if int64(len(dest)) < sampleCount {
		return types.ErrInsufficientData
	}

	actualStartPos := startPos % rb.size
	actualEndPos := (startPos + sampleCount) % rb.size

	if actualEndPos < actualStartPos {
		firstPart := rb.size - actualStartPos
		copy(dest[:firstPart], rb.data[actualStartPos:])
		copy(dest[firstPart:sampleCount], rb.data[0:actualEndPos])
	} else {
		copy(dest[:sampleCount], rb.data[actualStartPos:actualStartPos+sampleCount])
	}

	return nil
}

func (rb *RingBuffer) updateSegmentMetadata(startPos, sampleCount int64, timestamp time.Time) {
	segmentIndex := startPos / rb.segmentSize
	if segmentIndex >= int64(len(rb.segments)) {
		return
	}

	segment := &rb.segments[segmentIndex]
	segment.StartPos = startPos
	segment.EndPos = startPos + sampleCount
	segment.Timestamp = timestamp

	segmentData := make([]float32, rb.segmentSize)
	if err := rb.extractRange(startPos, rb.segmentSize, segmentData); err == nil {
		rms := calculateRMS(segmentData)
		segment.Quality = calculateQuality(rms, calculatePeak(segmentData), estimateNoise(segmentData), detectVoiceActivity(segmentData, rms))
		segment.HasVoice = detectVoiceActivity(segmentData, rms)
		segment.NoiseLevel = estimateNoise(segmentData)
	}
}
