//go:build !noaudio
// +build !noaudio

package capture

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/your-org/hema-replay-system/pkg/audio/buffer"
	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

// BenchmarkAsyncProcessingThroughput measures async processing throughput
func BenchmarkAsyncProcessingThroughput(b *testing.B) {
	logger := zerolog.Nop()
	config := types.DefaultAudioConfig()
	ringBuffer := buffer.NewRingBuffer(config.Buffer, config.Device.SampleRate, config.Device.Channels, 32)

	engine, err := NewCaptureEngine(config.Device, config.Processing, ringBuffer, logger)
	if err != nil {
		b.Fatalf("Failed to create capture engine: %v", err)
	}

	engine.initChannels()
	engine.initPerformance()

	frameSize := 512
	processedCount := int64(0)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start processing loop
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case frame, ok := <-engine.rawAudioChan:
				if !ok {
					return
				}

				// Simulate minimal processing
				if engine.processor != nil {
					engine.processor.Process(frame.Data, frame.Timestamp)
				}

				atomic.AddInt64(&processedCount, 1)
				putAudioFrame(frame)
			}
		}
	}()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		frame := getAudioFrame(frameSize)
		frame.Sequence = uint64(i + 1)
		frame.Timestamp = time.Now()

		// Fill with test data
		for j := 0; j < frameSize; j++ {
			frame.Data[j] = float32(math.Sin(2 * math.Pi * float64(j) / float64(frameSize)))
		}

		select {
		case engine.rawAudioChan <- frame:
		case <-ctx.Done():
			putAudioFrame(frame)
			break
		}
	}

	close(engine.rawAudioChan)
	wg.Wait()

	finalCount := atomic.LoadInt64(&processedCount)
	b.ReportMetric(float64(finalCount), "frames/op")
}

// BenchmarkMemoryPoolPerformance measures memory pool efficiency
func BenchmarkMemoryPoolPerformance(b *testing.B) {
	frameSizes := []int{256, 512, 1024, 2048, 4096}

	for _, size := range frameSizes {
		b.Run(fmt.Sprintf("FrameSize_%d", size), func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				frame := getAudioFrame(size)

				// Fill with data to simulate real usage
				for j := 0; j < size; j++ {
					frame.Data[j] = float32(i + j)
				}
				frame.Sequence = uint64(i)
				frame.Timestamp = time.Now()

				putAudioFrame(frame)
			}
		})
	}
}

// BenchmarkChannelBackpressureHandling measures frame dropping performance
func BenchmarkChannelBackpressureHandling(b *testing.B) {
	channelSizes := []int{10, 20, 50, 100}

	for _, chanSize := range channelSizes {
		b.Run(fmt.Sprintf("ChannelSize_%d", chanSize), func(b *testing.B) {
			rawAudioChan := make(chan *AudioFrame, chanSize)
			droppedFrames := int64(0)
			sentFrames := int64(0)

			// Fill channel to near capacity
			for i := 0; i < chanSize-2; i++ {
				frame := getAudioFrame(512)
				rawAudioChan <- frame
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				frame := getAudioFrame(512)
				frame.Sequence = uint64(i + 1)

				select {
				case rawAudioChan <- frame:
					atomic.AddInt64(&sentFrames, 1)
				default:
					// Simulate frame dropping
					select {
					case oldFrame := <-rawAudioChan:
						putAudioFrame(oldFrame)
						atomic.AddInt64(&droppedFrames, 1)
					default:
					}

					select {
					case rawAudioChan <- frame:
						atomic.AddInt64(&sentFrames, 1)
					default:
						putAudioFrame(frame)
						atomic.AddInt64(&droppedFrames, 1)
					}
				}
			}

			// Clean up
			for len(rawAudioChan) > 0 {
				frame := <-rawAudioChan
				putAudioFrame(frame)
			}

			sent := atomic.LoadInt64(&sentFrames)
			dropped := atomic.LoadInt64(&droppedFrames)
			b.ReportMetric(float64(sent), "sent/op")
			b.ReportMetric(float64(dropped), "dropped/op")
		})
	}
}

// BenchmarkConcurrentProcessing measures multi-goroutine processing performance
func BenchmarkConcurrentProcessing(b *testing.B) {
	processorCounts := []int{1, 2, 4, 8}

	for _, numProcessors := range processorCounts {
		b.Run(fmt.Sprintf("Processors_%d", numProcessors), func(b *testing.B) {
			rawAudioChan := make(chan *AudioFrame, RawAudioChannelSize*2)
			processedCount := int64(0)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Start processing goroutines
			var wg sync.WaitGroup
			for i := 0; i < numProcessors; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for {
						select {
						case <-ctx.Done():
							return
						case frame, ok := <-rawAudioChan:
							if !ok {
								return
							}

							// Simulate processing work
							var sum float32
							for _, sample := range frame.Data {
								sum += sample * sample
							}

							atomic.AddInt64(&processedCount, 1)
							putAudioFrame(frame)
						}
					}
				}()
			}

			b.ResetTimer()
			b.ReportAllocs()

			// Send frames
			for i := 0; i < b.N; i++ {
				frame := getAudioFrame(512)
				frame.Sequence = uint64(i + 1)

				// Fill with test data
				for j := 0; j < 512; j++ {
					frame.Data[j] = float32(math.Sin(2 * math.Pi * float64(j) / 512))
				}

				select {
				case rawAudioChan <- frame:
				case <-ctx.Done():
					putAudioFrame(frame)
					break
				}
			}

			close(rawAudioChan)
			wg.Wait()

			finalCount := atomic.LoadInt64(&processedCount)
			b.ReportMetric(float64(finalCount), "frames/op")
			b.ReportMetric(float64(finalCount)/float64(numProcessors), "frames_per_processor/op")
		})
	}
}

// BenchmarkLatencyMeasurement measures end-to-end processing latency
func BenchmarkLatencyMeasurement(b *testing.B) {
	rawAudioChan := make(chan *AudioFrame, RawAudioChannelSize)
	latencies := make([]time.Duration, 0, b.N)
	var latencyMu sync.Mutex

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start processing goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case frame, ok := <-rawAudioChan:
				if !ok {
					return
				}

				latency := time.Since(frame.Timestamp)

				latencyMu.Lock()
				latencies = append(latencies, latency)
				latencyMu.Unlock()

				putAudioFrame(frame)
			}
		}
	}()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		frame := getAudioFrame(512)
		frame.Timestamp = time.Now()
		frame.Sequence = uint64(i + 1)

		select {
		case rawAudioChan <- frame:
		case <-ctx.Done():
			putAudioFrame(frame)
			break
		}
	}

	close(rawAudioChan)
	wg.Wait()

	// Calculate latency statistics
	if len(latencies) > 0 {
		var totalLatency time.Duration
		for _, latency := range latencies {
			totalLatency += latency
		}
		avgLatency := totalLatency / time.Duration(len(latencies))
		b.ReportMetric(float64(avgLatency.Nanoseconds()), "avg_latency_ns/op")
	}
}

// BenchmarkFastPathActivation measures fast path switching performance
func BenchmarkFastPathActivation(b *testing.B) {
	rawAudioChan := make(chan *AudioFrame, RawAudioChannelSize)
	var fastPathEnabled uint32

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Fill channel to trigger fast path
		for j := 0; j < DropThreshold+1 && len(rawAudioChan) < cap(rawAudioChan); j++ {
			frame := getAudioFrame(512)
			rawAudioChan <- frame
		}

		// Simulate fast path logic
		queueDepth := len(rawAudioChan)
		if queueDepth > DropThreshold {
			atomic.StoreUint32(&fastPathEnabled, 1)
		} else if queueDepth < DropThreshold/2 {
			atomic.StoreUint32(&fastPathEnabled, 0)
		}

		// Drain some frames
		for j := 0; j < 5 && len(rawAudioChan) > 0; j++ {
			frame := <-rawAudioChan
			putAudioFrame(frame)
		}

		// Check fast path again
		queueDepth = len(rawAudioChan)
		if queueDepth < DropThreshold/2 {
			atomic.StoreUint32(&fastPathEnabled, 0)
		}
	}

	// Clean up
	for len(rawAudioChan) > 0 {
		frame := <-rawAudioChan
		putAudioFrame(frame)
	}

	b.ReportMetric(float64(atomic.LoadUint32(&fastPathEnabled)), "fast_path_state")
}

// BenchmarkAsyncVsSyncProcessing compares async vs synchronous processing patterns
func BenchmarkAsyncVsSyncProcessing(b *testing.B) {
	frameSize := 512
	numFrames := 1000

	b.Run("Synchronous", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Simulate synchronous processing
			for j := 0; j < numFrames; j++ {
				frame := getAudioFrame(frameSize)

				// Simulate processing work
				var sum float32
				for _, sample := range frame.Data {
					sum += sample * sample
				}

				putAudioFrame(frame)
			}
		}
	})

	b.Run("Asynchronous", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			rawAudioChan := make(chan *AudioFrame, RawAudioChannelSize)
			processedCount := int64(0)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

			// Start async processor
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-ctx.Done():
						return
					case frame, ok := <-rawAudioChan:
						if !ok {
							return
						}

						// Simulate processing work
						var sum float32
						for _, sample := range frame.Data {
							sum += sample * sample
						}

						atomic.AddInt64(&processedCount, 1)
						putAudioFrame(frame)
					}
				}
			}()

			// Send frames
			for j := 0; j < numFrames; j++ {
				frame := getAudioFrame(frameSize)
				select {
				case rawAudioChan <- frame:
				default:
					putAudioFrame(frame)
				}
			}

			close(rawAudioChan)
			wg.Wait()
			cancel()
		}
	})
}

// BenchmarkMemoryUsageComparison compares memory usage patterns
func BenchmarkMemoryUsageComparison(b *testing.B) {
	frameSize := 1024
	numFrames := 10000

	b.Run("WithPool", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			frames := make([]*AudioFrame, numFrames)

			// Allocate from pool
			for j := 0; j < numFrames; j++ {
				frames[j] = getAudioFrame(frameSize)
				frames[j].Sequence = uint64(j)
				frames[j].Timestamp = time.Now()
			}

			// Return to pool
			for j := 0; j < numFrames; j++ {
				putAudioFrame(frames[j])
			}
		}
	})

	b.Run("WithoutPool", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			frames := make([]*AudioFrame, numFrames)

			// Allocate directly
			for j := 0; j < numFrames; j++ {
				frames[j] = &AudioFrame{
					Data:      make([]float32, frameSize),
					Sequence:  uint64(j),
					Timestamp: time.Now(),
				}
			}

			// Frames will be garbage collected
			_ = frames
		}
	})
}

// BenchmarkCPUCoreUtilization measures CPU utilization across cores
func BenchmarkCPUCoreUtilization(b *testing.B) {
	numCores := runtime.NumCPU()
	b.Logf("Running on %d CPU cores", numCores)

	frameSize := 512
	channelSize := RawAudioChannelSize * numCores

	rawAudioChan := make(chan *AudioFrame, channelSize)
	processedCount := int64(0)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start one processor per CPU core
	var wg sync.WaitGroup
	for i := 0; i < numCores; i++ {
		wg.Add(1)
		go func(coreID int) {
			defer wg.Done()
			runtime.LockOSThread() // Pin goroutine to OS thread

			for {
				select {
				case <-ctx.Done():
					return
				case frame, ok := <-rawAudioChan:
					if !ok {
						return
					}

					// CPU-intensive work to utilize core
					for k := 0; k < 1000; k++ {
						var sum float32
						for _, sample := range frame.Data {
							sum += sample * float32(k) * 0.001
						}
					}

					atomic.AddInt64(&processedCount, 1)
					putAudioFrame(frame)
				}
			}
		}(i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		frame := getAudioFrame(frameSize)
		frame.Sequence = uint64(i + 1)

		select {
		case rawAudioChan <- frame:
		case <-ctx.Done():
			putAudioFrame(frame)
			break
		}
	}

	close(rawAudioChan)
	wg.Wait()

	finalCount := atomic.LoadInt64(&processedCount)
	b.ReportMetric(float64(finalCount)/float64(numCores), "frames_per_core/op")
}
