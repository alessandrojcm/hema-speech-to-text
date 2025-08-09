package buffer

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

func generateSegmentID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("seg_%x_%d", bytes, time.Now().UnixNano())
}

func createAudioSegment(data []float32, startTime, endTime time.Time, metadata types.SegmentMetadata) *types.AudioSegment {
	return &types.AudioSegment{
		ID:        generateSegmentID(),
		Data:      data,
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  endTime.Sub(startTime),
		Metadata:  metadata,
	}
}

func calculateSegmentMetadata(samples []float32, sampleRate, channels, bitDepth int) types.SegmentMetadata {
	if len(samples) == 0 {
		return types.SegmentMetadata{
			SampleRate: sampleRate,
			Channels:   channels,
			BitDepth:   bitDepth,
		}
	}

	rms := calculateRMS(samples)
	peak := calculatePeak(samples)
	noise := estimateNoise(samples)
	hasVoice := detectVoiceActivity(samples, rms)
	quality := calculateQuality(rms, peak, noise, hasVoice)

	return types.SegmentMetadata{
		SampleRate:    sampleRate,
		Channels:      channels,
		BitDepth:      bitDepth,
		Quality:       quality,
		HasVoice:      hasVoice,
		NoiseLevel:    noise,
		PeakAmplitude: peak,
		RMSLevel:      rms,
	}
}

func detectVoiceActivity(samples []float32, rms float64) bool {
	const vadThreshold = 0.01
	return rms > vadThreshold
}
