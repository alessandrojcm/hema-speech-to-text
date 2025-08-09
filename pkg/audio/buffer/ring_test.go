package buffer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

func TestNewRingBuffer(t *testing.T) {
	config := types.BufferConfig{
		Duration:    10 * time.Second,
		SegmentSize: 1 * time.Second,
	}

	rb := NewRingBuffer(config, 44100, 2, 16)

	assert.NotNil(t, rb)
	assert.Equal(t, int64(44100*2*10), rb.size)
	assert.Equal(t, 44100, rb.sampleRate)
	assert.Equal(t, 2, rb.channels)
	assert.Equal(t, 16, rb.bitDepth)
}

func TestRingBufferWrite(t *testing.T) {
	config := types.BufferConfig{
		Duration:    2 * time.Second,
		SegmentSize: 1 * time.Second,
	}

	rb := NewRingBuffer(config, 44100, 1, 16)
	samples := make([]float32, 1000)
	for i := range samples {
		samples[i] = float32(i) / 1000.0
	}

	err := rb.Write(samples, time.Now())
	require.NoError(t, err)

	assert.Equal(t, int64(1000), rb.writePos)
	assert.Equal(t, int64(1000), rb.totalWritten)
}

func TestRingBufferWriteEmpty(t *testing.T) {
	config := types.BufferConfig{
		Duration:    2 * time.Second,
		SegmentSize: 1 * time.Second,
	}

	rb := NewRingBuffer(config, 44100, 1, 16)

	err := rb.Write([]float32{}, time.Now())
	assert.Equal(t, types.ErrEmptyData, err)
}

func TestRingBufferExtract(t *testing.T) {
	config := types.BufferConfig{
		Duration:    10 * time.Second,
		SegmentSize: 1 * time.Second,
	}

	rb := NewRingBuffer(config, 44100, 1, 16)

	samples := make([]float32, 44100)
	for i := range samples {
		samples[i] = float32(i) / 44100.0
	}

	timestamp := time.Now()
	err := rb.Write(samples, timestamp)
	require.NoError(t, err)

	segment, err := rb.Extract(1*time.Second, timestamp.Add(1*time.Second))
	require.NoError(t, err)
	assert.NotNil(t, segment)
	assert.Equal(t, 1*time.Second, segment.Duration)
	assert.Len(t, segment.Data, 44100)
}

func TestRingBufferExtractInsufficientData(t *testing.T) {
	config := types.BufferConfig{
		Duration:    2 * time.Second,
		SegmentSize: 1 * time.Second,
	}

	rb := NewRingBuffer(config, 44100, 1, 16)

	segment, err := rb.Extract(5*time.Second, time.Now())
	assert.Nil(t, segment)
	assert.Equal(t, types.ErrInsufficientData, err)
}

func TestRingBufferCircularOverwrite(t *testing.T) {
	config := types.BufferConfig{
		Duration:    1 * time.Second,
		SegmentSize: 500 * time.Millisecond,
	}

	rb := NewRingBuffer(config, 1000, 1, 16)

	samples1 := make([]float32, 600)
	for i := range samples1 {
		samples1[i] = 1.0
	}

	samples2 := make([]float32, 600)
	for i := range samples2 {
		samples2[i] = 2.0
	}

	timestamp1 := time.Now()
	err := rb.Write(samples1, timestamp1)
	require.NoError(t, err)

	timestamp2 := timestamp1.Add(600 * time.Millisecond)
	err = rb.Write(samples2, timestamp2)
	require.NoError(t, err)

	assert.Greater(t, rb.overwritten, int64(0))
}

func TestRingBufferGetStats(t *testing.T) {
	config := types.BufferConfig{
		Duration:    5 * time.Second,
		SegmentSize: 1 * time.Second,
	}

	rb := NewRingBuffer(config, 44100, 2, 16)

	samples := make([]float32, 1000)
	err := rb.Write(samples, time.Now())
	require.NoError(t, err)

	stats := rb.GetStats()
	assert.Equal(t, int64(44100*2*5), stats.TotalCapacity)
	assert.Equal(t, int64(1000), stats.UsedCapacity)
	assert.Greater(t, stats.UtilizationPercent, 0.0)
	assert.Equal(t, int64(0), stats.OverwriteCount)
}
