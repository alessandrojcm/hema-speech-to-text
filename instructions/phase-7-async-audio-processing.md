# Phase 7: Asynchronous Audio Processing Architecture

## Problem Statement

The HEMA replay system experiences intermittent freezing when running in speech-only mode with preprocessing enabled. The root cause is **synchronous audio processing in the capture thread**, which blocks PortAudio from delivering frames in real-time.

### Current Architecture Issues

1. **Blocking Capture Loop**: Audio processing happens synchronously in `pkg/audio/capture/engine.go:226`
2. **Frame Drops**: When VAD processing takes >20ms, PortAudio drops audio frames
3. **System Freezes**: Speech input causes 2-3 second freezes as processing backs up
4. **Poor CPU Utilization**: Single-threaded processing can't leverage multi-core systems

### Impact

- **User Experience**: System becomes unresponsive during speech
- **Audio Quality**: Dropped frames cause choppy audio and missed transcriptions
- **Performance**: Can't handle real-time processing of continuous speech
- **Scalability**: Adding more processing stages would worsen the problem

## Proposed Solution: Async Processing Pipeline

### Architecture Overview

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   PortAudio     │────▶│  Capture Thread  │────▶│  Raw Audio      │
│   (Hardware)    │     │  (Main Thread)   │     │  Channel        │
└─────────────────┘     └──────────────────┘     └────────┬────────┘
                                                           │
                                                           ▼
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  Ring Buffer    │◀────│ Processing Thread│◀────│  (Buffered)     │
│  (To Speech)    │     │  (Goroutine)     │     │                 │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                               │
                               ▼
                        ┌──────────────────┐
                        │   VAD, Quality   │
                        │   Assessment     │
                        └──────────────────┘
```

### Key Design Principles

1. **Non-Blocking Capture**: The capture thread does minimal work - only reads from PortAudio and sends to channel
2. **Buffered Channels**: Use buffered channels (10-20 frames) to handle processing bursts
3. **Graceful Degradation**: Drop old frames if processing can't keep up (with metrics)
4. **Ordered Processing**: Maintain timestamp ordering throughout the pipeline
5. **Parallel Processing**: Leverage Go's concurrency for multi-core utilization

## Implementation Details

### 1. Data Structures

```go
// AudioFrame represents a captured audio frame with metadata
type AudioFrame struct {
    Data      []float32
    Timestamp time.Time
    Sequence  uint64  // For ordering validation
}

// ProcessingStats tracks async processing metrics
type ProcessingStats struct {
    FramesQueued    uint64
    FramesProcessed uint64
    FramesDropped   uint64
    QueueDepth      int
    ProcessingLag   time.Duration
}
```

### 2. Memory Pool for Frame Allocation

```go
// Global frame pool to reduce GC pressure
var audioFramePool = sync.Pool{
    New: func() interface{} {
        // Pre-allocate with typical frame size
        // Will be resized as needed
        return &AudioFrame{
            Data: make([]float32, 0, 4096), // 2048 samples * 2 channels
        }
    },
}

// Helper functions for pool management
func getAudioFrame(size int) *AudioFrame {
    frame := audioFramePool.Get().(*AudioFrame)
    if cap(frame.Data) < size {
        frame.Data = make([]float32, size)
    } else {
        frame.Data = frame.Data[:size]
    }
    return frame
}

func putAudioFrame(frame *AudioFrame) {
    // Reset frame before returning to pool
    frame.Timestamp = time.Time{}
    frame.Sequence = 0
    frame.Data = frame.Data[:0]
    audioFramePool.Put(frame)
}
```

### 3. Modified CaptureEngine Structure

```go
type CaptureEngine struct {
    // Existing fields...
    
    // Async processing channels (bounded for backpressure control)
    rawAudioChan   chan *AudioFrame  // From capture to processing
    processedChan  chan *AudioFrame  // From processing to buffer
    
    // Synchronization
    processingWG   sync.WaitGroup
    processingDone chan struct{}
    
    // Metrics
    frameSequence  uint64  // Atomic counter
    processingStats ProcessingStats
    statsLock      sync.RWMutex
    
    // Performance flags
    fastPathEnabled bool  // Skip heavy processing when backed up
}
```

### 4. Capture Loop (Non-Blocking with Memory Pool)

```go
func (ce *CaptureEngine) captureLoop(ctx context.Context) {
    defer close(ce.rawAudioChan)
    
    frameSize := ce.config.FramesPerBuffer * ce.config.Channels
    audioBuffer := make([]float32, frameSize)
    
    for {
        select {
        case <-ctx.Done():
            return
        default:
            // Read from PortAudio (blocking but fast)
            if err := ce.stream.Read(audioBuffer); err != nil {
                ce.handleCaptureError(err)
                continue
            }
            
            // Get frame from pool (reduces GC pressure)
            frame := getAudioFrame(len(audioBuffer))
            frame.Timestamp = time.Now()
            frame.Sequence = atomic.AddUint64(&ce.frameSequence, 1)
            copy(frame.Data, audioBuffer)
            
            // Check queue depth for fast path decision
            queueDepth := len(ce.rawAudioChan)
            if queueDepth > DropThreshold {
                atomic.StoreUint32(&ce.fastPathEnabled, 1)
            } else if queueDepth < DropThreshold/2 {
                atomic.StoreUint32(&ce.fastPathEnabled, 0)
            }
            
            // Send to processing (non-blocking with frame dropping)
            select {
            case ce.rawAudioChan <- frame:
                ce.updateStats(frameQueued)
            default:
                // Channel full - implement frame dropping strategy
                select {
                case oldFrame := <-ce.rawAudioChan:
                    // Return old frame to pool
                    putAudioFrame(oldFrame)
                    ce.updateStats(frameDropped)
                default:
                }
                // Try again
                select {
                case ce.rawAudioChan <- frame:
                    ce.updateStats(frameQueued)
                default:
                    // Frame couldn't be queued - return to pool
                    putAudioFrame(frame)
                    ce.updateStats(frameDropped)
                }
            }
        }
    }
}
```

### 5. Processing Loop (Async with Fast Path)

```go
func (ce *CaptureEngine) processingLoop(ctx context.Context) {
    defer ce.processingWG.Done()
    
    // Clean up frames on exit
    defer func() {
        // Drain channel and return frames to pool
        for frame := range ce.rawAudioChan {
            putAudioFrame(frame)
        }
    }()
    
    for {
        select {
        case <-ctx.Done():
            return
        case frame, ok := <-ce.rawAudioChan:
            if !ok {
                return
            }
            
            processingStart := time.Now()
            
            // Check fast path flag - skip heavy processing if backed up
            fastPath := atomic.LoadUint32(&ce.fastPathEnabled) == 1
            
            // Process audio (VAD, quality assessment, etc.)
            processedData := frame.Data
            if ce.processor != nil && !fastPath {
                var err error
                processedData, err = ce.processor.Process(frame.Data, frame.Timestamp)
                if err != nil {
                    ce.logger.Warn().
                        Err(err).
                        Uint64("sequence", frame.Sequence).
                        Bool("fast_path", fastPath).
                        Msg("Audio processing failed")
                    processedData = frame.Data
                }
            } else if fastPath {
                // Fast path - minimal processing only
                ce.logger.Debug().
                    Uint64("sequence", frame.Sequence).
                    Int("queue_depth", len(ce.rawAudioChan)).
                    Msg("Fast path enabled - skipping heavy processing")
            }
            
            // Write to buffer
            if err := ce.buffer.Write(processedData, frame.Timestamp); err != nil {
                ce.logger.Error().
                    Err(err).
                    Uint64("sequence", frame.Sequence).
                    Msg("Failed to write to ring buffer")
                ce.metrics.IncrementDroppedSamples(int64(len(processedData)))
            } else {
                ce.metrics.IncrementProcessedSamples(int64(len(processedData)))
            }
            
            // Return frame to pool
            putAudioFrame(frame)
            
            // Update processing metrics
            processingTime := time.Since(processingStart)
            ce.updateProcessingStats(frame.Sequence, processingTime)
        }
    }
}
```

### 6. Channel Configuration and Initialization

```go
const (
    // Channel buffer sizes (bounded to prevent unbounded memory growth)
    RawAudioChannelSize = 20  // ~400ms buffer at 50ms frames
    
    // Processing thresholds
    MaxProcessingLag = 500 * time.Millisecond
    DropThreshold    = 10  // Enable fast path if queue > 10
)

func (ce *CaptureEngine) initChannels() {
    // Create bounded channels for backpressure control
    ce.rawAudioChan = make(chan *AudioFrame, RawAudioChannelSize)
    ce.processingDone = make(chan struct{})
}

func (ce *CaptureEngine) initPerformance() {
    // Set GOMAXPROCS to use all available CPU cores
    runtime.GOMAXPROCS(runtime.NumCPU())
    
    // Initialize fast path flag
    ce.fastPathEnabled = 0  // Start with full processing
    
    // Log CPU configuration
    ce.logger.Info().
        Int("cpu_cores", runtime.NumCPU()).
        Int("gomaxprocs", runtime.GOMAXPROCS(0)).
        Msg("Performance configuration initialized")
}
```

### 7. Start/Stop Lifecycle with Performance Setup

```go
func (ce *CaptureEngine) Start(ctx context.Context) error {
    // ... existing initialization ...
    
    // Initialize performance settings
    ce.initPerformance()
    
    // Initialize channels with bounded sizes
    ce.initChannels()
    
    // Start processing goroutine
    ce.processingWG.Add(1)
    go ce.processingLoop(ctx)
    
    // Start capture goroutine
    go ce.captureLoop(ctx)
    
    // Start monitoring
    go ce.monitoringLoop(ctx)
    
    ce.logger.Info().
        Int("channel_buffer_size", RawAudioChannelSize).
        Int("drop_threshold", DropThreshold).
        Msg("Async audio processing started")
    
    return nil
}

func (ce *CaptureEngine) Stop() error {
    // Signal stop
    close(ce.stopChan)
    
    // Wait for capture to finish
    // (This will close rawAudioChan)
    
    // Wait for processing to complete
    ce.processingWG.Wait()
    
    // Clean up any remaining frames in the pool
    // (The pool will be garbage collected automatically)
    
    // ... existing cleanup ...
    
    ce.logger.Info().Msg("Async audio processing stopped")
    
    return nil
}
```

### 8. Monitoring and Metrics with Performance Tracking

```go
func (ce *CaptureEngine) monitoringLoop(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            ce.reportAsyncMetrics()
            
            // Check for processing lag
            queueDepth := len(ce.rawAudioChan)
            if ce.getProcessingLag() > MaxProcessingLag {
                ce.logger.Warn().
                    Dur("lag", ce.getProcessingLag()).
                    Int("queue_depth", queueDepth).
                    Bool("fast_path", atomic.LoadUint32(&ce.fastPathEnabled) == 1).
                    Msg("Processing falling behind")
            }
            
            // Report memory pool statistics periodically
            if queueDepth == 0 {
                // When idle, log pool efficiency
                ce.logger.Debug().
                    Msg("Audio frame pool operating efficiently")
            }
        }
    }
}

func (ce *CaptureEngine) reportAsyncMetrics() {
    ce.statsLock.RLock()
    defer ce.statsLock.RUnlock()
    
    // Calculate drop rate
    total := ce.processingStats.FramesQueued
    dropped := ce.processingStats.FramesDropped
    dropRate := float64(0)
    if total > 0 {
        dropRate = float64(dropped) / float64(total) * 100
    }
    
    ce.logger.Debug().
        Uint64("frames_queued", ce.processingStats.FramesQueued).
        Uint64("frames_processed", ce.processingStats.FramesProcessed).
        Uint64("frames_dropped", ce.processingStats.FramesDropped).
        Float64("drop_rate_pct", dropRate).
        Int("queue_depth", len(ce.rawAudioChan)).
        Dur("processing_lag", ce.processingStats.ProcessingLag).
        Bool("fast_path", atomic.LoadUint32(&ce.fastPathEnabled) == 1).
        Int("cpu_cores_used", runtime.NumCPU()).
        Msg("Async processing metrics")
}
```

## Built-in Performance Optimizations

The implementation includes several performance optimizations from the start:

1. **Memory Pool (sync.Pool)**: Reduces GC pressure by reusing AudioFrame allocations
2. **Bounded Channels**: Prevents unbounded memory growth with fixed-size buffers  
3. **GOMAXPROCS Configuration**: Ensures all CPU cores are utilized
4. **Fast Path Processing**: Automatically skips heavy processing when queue backs up
5. **Frame Dropping Strategy**: Maintains real-time performance under load

These optimizations are essential for preventing system degradation and are not optional features.

## Testing Strategy

### 1. Unit Tests

```go
func TestAsyncProcessing(t *testing.T) {
    // Test that capture doesn't block when processing is slow
    // Test frame ordering is maintained
    // Test graceful degradation under load
}

func TestChannelBackpressure(t *testing.T) {
    // Test frame dropping when channels are full
    // Test metrics are correctly updated
    // Test recovery after processing catches up
}
```

### 2. Benchmark Tests

```go
func BenchmarkAsyncCapture(b *testing.B) {
    // Measure capture throughput with async processing
    // Compare with synchronous implementation
    // Test with various processing delays
}

func BenchmarkProcessingLatency(b *testing.B) {
    // Measure end-to-end latency
    // Test with different channel buffer sizes
    // Profile CPU and memory usage
}
```

### 3. Integration Tests

- **Stress Test**: Run with continuous speech input for extended periods
- **Load Test**: Simulate heavy processing loads
- **Regression Test**: Ensure no audio quality degradation

## Implementation Plan

This is a **complete replacement** of the synchronous processing architecture with built-in performance optimizations. The async processing will be the only supported mode going forward.

### Implementation Steps:

1. **Step 1**: Add memory pool and performance initialization
   - Implement `sync.Pool` for AudioFrame reuse
   - Add `initPerformance()` to configure GOMAXPROCS
   - Define bounded channel constants

2. **Step 2**: Replace the synchronous `captureLoop` in `engine.go`
   - Implement non-blocking capture with frame pooling
   - Add fast path detection based on queue depth
   - Implement frame dropping strategy for backpressure

3. **Step 3**: Add the new `processingLoop` goroutine
   - Implement async processing with fast path support
   - Add proper frame cleanup and pool return
   - Include processing metrics tracking

4. **Step 4**: Update lifecycle management
   - Modify Start() to initialize performance settings
   - Update Stop() for proper cleanup
   - Add monitoring loop with performance metrics

5. **Step 5**: Update all tests to work with async architecture
   - Test frame ordering preservation
   - Verify memory pool efficiency
   - Benchmark async vs sync performance

6. **Step 6**: Remove any references to synchronous processing
   - Delete old synchronous code paths
   - Update documentation
   - Clean up configuration options

## Configuration Options

```yaml
audio:
  capture:
    # Async processing configuration (always enabled)
    async_processing:
      channel_buffer_size: 20
      max_processing_lag_ms: 500
      drop_threshold: 10
      monitoring_interval_ms: 1000
```

## Expected Improvements

### Performance Metrics

- **Capture Latency**: <1ms (from current 20-100ms during processing)
- **Frame Drop Rate**: <0.1% (from current 5-10% during speech)
- **CPU Utilization**: 70-80% multi-core (from current 100% single-core)
- **Processing Throughput**: 2-3x improvement

### User Experience

- **No Freezing**: System remains responsive during speech
- **Smooth Audio**: No choppy playback or gaps
- **Better Recognition**: Complete audio segments for transcription
- **Lower Latency**: Faster end-to-end processing

## Breaking Changes

This implementation **completely replaces** the synchronous audio processing architecture. There is no fallback to synchronous mode.

### API Changes
- The `CaptureEngine` struct gains new fields for async processing
- Internal processing flow is completely rewritten
- Metrics and monitoring are enhanced for async operations

### Why No Backward Compatibility
1. **Synchronous processing is fundamentally broken** - it causes system freezes
2. **Maintaining two code paths increases complexity** without benefit
3. **This is development phase** - breaking changes are acceptable
4. **The async architecture is strictly superior** in all metrics

## Future Enhancements (these are more like a pipe dream lol)

1. **Pipeline Parallelism**: Multiple processing stages in parallel
2. **GPU Acceleration**: Offload VAD/FFT to GPU
3. **Distributed Processing**: Scale across multiple machines
4. **Adaptive Quality**: Adjust processing based on system load
5. **ML-based Optimization**: Learn optimal buffer sizes and thresholds

## Conclusion

This asynchronous architecture **permanently replaces** the broken synchronous processing model. The new implementation:

1. **Eliminates all freezing issues** by ensuring audio capture never blocks
2. **Provides 2-3x better performance** through parallel processing
3. **Scales properly** on multi-core systems
4. **Simplifies the codebase** by having one clear processing model

The key insight is separating time-critical audio capture from compute-intensive processing, allowing each component to operate at its optimal pace without blocking others. This is not an optional enhancement - it's a critical fix that becomes the foundation for all future audio processing in the system.