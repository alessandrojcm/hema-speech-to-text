# Phase 5 - Stage 2: Voice Activity Detection Integration

## Overview
This stage integrates Voice Activity Detection (VAD) with the existing audio system from Phase 2. We'll use the WebRTC VAD that's already part of the audio processing pipeline and enhance it for automatic speech detection triggering.

## Architecture

### VAD Integration Layer
```go
// pkg/pipeline/vad/detector.go
package vad

import (
    "context"
    "sync"
    "time"
    
    "github.com/rs/zerolog"
    "github.com/yourusername/hema-replay/pkg/audio"
    "github.com/yourusername/hema-replay/pkg/audio/processing"
)

// VADDetector wraps the existing audio.Manager's VAD capabilities
type VADDetector struct {
    audioManager    *audio.Manager
    processor       *processing.AudioProcessor // Existing processor with VAD
    
    // Detection state
    isActive        bool
    activityStart   time.Time
    silenceStart    time.Time
    
    // Configuration
    config          *Config
    
    // Output
    eventChan       chan VADEvent
    
    // Control
    stopChan        chan struct{}
    mu              sync.RWMutex
    logger          zerolog.Logger
}

type Config struct {
    MinSpeechDurationMs  int     `mapstructure:"min_speech_duration_ms"`  // Min duration to trigger
    MaxSilenceDurationMs int     `mapstructure:"max_silence_duration_ms"` // Max silence before end
    VADMode             int     `mapstructure:"vad_mode"`                // 0-3 (WebRTC VAD mode)
    BufferBeforeMs      int     `mapstructure:"buffer_before_ms"`        // Audio before speech
    BufferAfterMs       int     `mapstructure:"buffer_after_ms"`         // Audio after speech
}

type VADEvent struct {
    Type        EventType
    StartTime   time.Time
    EndTime     time.Time
    Duration    time.Duration
    Confidence  float32
    
    // Audio segment boundaries for extraction
    BufferStart time.Time  
    BufferEnd   time.Time
}

type EventType int

const (
    EventSpeechStart EventType = iota
    EventSpeechEnd
    EventSpeechSegment // Complete segment ready for processing
)

func NewVADDetector(audioManager *audio.Manager, config *Config, logger zerolog.Logger) *VADDetector {
    return &VADDetector{
        audioManager: audioManager,
        config:      config,
        eventChan:   make(chan VADEvent, 10),
        stopChan:    make(chan struct{}),
        logger:      logger.With().Str("component", "vad_detector").Logger(),
    }
}
```

### Continuous Monitoring
```go
// pkg/pipeline/vad/monitor.go
package vad

func (v *VADDetector) Start(ctx context.Context) error {
    v.logger.Info().Msg("Starting VAD monitoring")
    
    // Get audio processor from existing manager
    processor := v.audioManager.GetProcessor()
    if processor == nil {
        return fmt.Errorf("audio processor not available")
    }
    v.processor = processor
    
    // Start monitoring goroutine
    go v.monitorAudioStream(ctx)
    
    return nil
}

func (v *VADDetector) monitorAudioStream(ctx context.Context) {
    // Subscribe to processed audio from the existing audio.Manager
    audioChan := v.audioManager.Subscribe("vad_monitor")
    defer v.audioManager.Unsubscribe("vad_monitor")
    
    ticker := time.NewTicker(100 * time.Millisecond) // Check every 100ms
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
            
        case <-v.stopChan:
            return
            
        case <-ticker.C:
            // Check current VAD state from audio processor
            vadActive := v.checkVADState()
            v.handleVADState(vadActive)
        }
    }
}

func (v *VADDetector) checkVADState() bool {
    // Use the existing VAD from audio.processing.AudioProcessor
    metrics := v.audioManager.GetMetrics()
    
    // The audio processor already has VAD integrated
    if vadMetric, ok := metrics["vad_active"]; ok {
        return vadMetric.(bool)
    }
    
    return false
}

func (v *VADDetector) handleVADState(vadActive bool) {
    v.mu.Lock()
    defer v.mu.Unlock()
    
    now := time.Now()
    
    if vadActive && !v.isActive {
        // Speech started
        v.isActive = true
        v.activityStart = now
        v.logger.Debug().Time("start", now).Msg("Speech detected")
        
        // Send start event
        v.sendEvent(VADEvent{
            Type:      EventSpeechStart,
            StartTime: now,
        })
        
    } else if !vadActive && v.isActive {
        // Potential speech end - check silence duration
        if v.silenceStart.IsZero() {
            v.silenceStart = now
        }
        
        silenceDuration := now.Sub(v.silenceStart)
        if silenceDuration > time.Duration(v.config.MaxSilenceDurationMs)*time.Millisecond {
            // Speech ended
            v.handleSpeechEnd(now)
        }
        
    } else if vadActive && v.isActive {
        // Reset silence timer if speech resumes
        v.silenceStart = time.Time{}
    }
}

func (v *VADDetector) handleSpeechEnd(endTime time.Time) {
    speechDuration := endTime.Sub(v.activityStart)
    
    // Check minimum duration
    if speechDuration < time.Duration(v.config.MinSpeechDurationMs)*time.Millisecond {
        v.logger.Debug().
            Dur("duration", speechDuration).
            Msg("Speech too short, ignoring")
        v.resetState()
        return
    }
    
    // Calculate buffer boundaries
    bufferStart := v.activityStart.Add(-time.Duration(v.config.BufferBeforeMs) * time.Millisecond)
    bufferEnd := endTime.Add(time.Duration(v.config.BufferAfterMs) * time.Millisecond)
    
    // Send complete segment event
    event := VADEvent{
        Type:        EventSpeechSegment,
        StartTime:   v.activityStart,
        EndTime:     endTime,
        Duration:    speechDuration,
        BufferStart: bufferStart,
        BufferEnd:   bufferEnd,
        Confidence:  v.calculateConfidence(speechDuration),
    }
    
    v.sendEvent(event)
    v.logger.Info().
        Dur("duration", speechDuration).
        Time("start", v.activityStart).
        Time("end", endTime).
        Msg("Speech segment detected")
    
    v.resetState()
}

func (v *VADDetector) resetState() {
    v.isActive = false
    v.activityStart = time.Time{}
    v.silenceStart = time.Time{}
}

func (v *VADDetector) calculateConfidence(duration time.Duration) float32 {
    // Simple confidence based on duration
    // Longer, clearer speech = higher confidence
    if duration > 3*time.Second {
        return 0.95
    } else if duration > 1*time.Second {
        return 0.85
    } else {
        return 0.70
    }
}

func (v *VADDetector) sendEvent(event VADEvent) {
    select {
    case v.eventChan <- event:
    default:
        v.logger.Warn().Msg("VAD event channel full, dropping event")
    }
}

func (v *VADDetector) Events() <-chan VADEvent {
    return v.eventChan
}

func (v *VADDetector) Stop() {
    close(v.stopChan)
    close(v.eventChan)
}
```

### Pipeline Integration
```go
// pkg/pipeline/vad/integration.go
package vad

import (
    "github.com/yourusername/hema-replay/pkg/audio"
    "github.com/yourusername/hema-replay/pkg/pipeline"
)

// VADPipelineIntegration connects VAD to the pipeline manager
type VADPipelineIntegration struct {
    vadDetector     *VADDetector
    pipelineManager *pipeline.Manager
    audioManager    *audio.Manager
    
    logger          zerolog.Logger
}

func NewVADPipelineIntegration(
    pipelineManager *pipeline.Manager,
    audioManager *audio.Manager,
    config *Config,
    logger zerolog.Logger,
) *VADPipelineIntegration {
    
    vadDetector := NewVADDetector(audioManager, config, logger)
    
    return &VADPipelineIntegration{
        vadDetector:     vadDetector,
        pipelineManager: pipelineManager,
        audioManager:    audioManager,
        logger:         logger,
    }
}

func (vpi *VADPipelineIntegration) Start(ctx context.Context) error {
    // Start VAD detector
    if err := vpi.vadDetector.Start(ctx); err != nil {
        return fmt.Errorf("failed to start VAD detector: %w", err)
    }
    
    // Connect VAD events to pipeline
    go vpi.processVADEvents(ctx)
    
    return nil
}

func (vpi *VADPipelineIntegration) processVADEvents(ctx context.Context) {
    events := vpi.vadDetector.Events()
    
    for {
        select {
        case <-ctx.Done():
            return
            
        case event := <-events:
            if event.Type == EventSpeechSegment {
                vpi.handleSpeechSegment(event)
            }
        }
    }
}

func (vpi *VADPipelineIntegration) handleSpeechSegment(event VADEvent) {
    vpi.logger.Info().
        Dur("duration", event.Duration).
        Float32("confidence", event.Confidence).
        Msg("Processing speech segment")
    
    // Extract audio segment from buffer using existing audio.Manager
    audioSegment, err := vpi.audioManager.ExtractTimeRange(
        event.BufferStart,
        event.BufferEnd,
    )
    
    if err != nil {
        vpi.logger.Error().Err(err).Msg("Failed to extract audio segment")
        return
    }
    
    // Send to pipeline for processing
    vpi.pipelineManager.ProcessAudioSegment(audioSegment, map[string]interface{}{
        "vad_confidence": event.Confidence,
        "duration":       event.Duration,
        "start_time":     event.StartTime,
        "end_time":       event.EndTime,
    })
}
```

### Adaptive Threshold Management
```go
// pkg/pipeline/vad/adaptive.go
package vad

// AdaptiveThresholdManager adjusts VAD sensitivity based on environment
type AdaptiveThresholdManager struct {
    vadDetector     *VADDetector
    
    // Noise estimation
    noiseFloor      float32
    snrHistory      []float32
    
    // Adaptation parameters
    baseMode        int
    currentMode     int
    lastUpdate      time.Time
    
    mu              sync.RWMutex
}

func NewAdaptiveThresholdManager(vadDetector *VADDetector) *AdaptiveThresholdManager {
    return &AdaptiveThresholdManager{
        vadDetector: vadDetector,
        baseMode:    vadDetector.config.VADMode,
        currentMode: vadDetector.config.VADMode,
        snrHistory:  make([]float32, 0, 100),
    }
}

func (atm *AdaptiveThresholdManager) Update(metrics map[string]interface{}) {
    atm.mu.Lock()
    defer atm.mu.Unlock()
    
    // Get noise level from audio metrics
    if noiseLevel, ok := metrics["noise_level"].(float32); ok {
        atm.updateNoiseFloor(noiseLevel)
    }
    
    // Get SNR from audio metrics
    if snr, ok := metrics["snr"].(float32); ok {
        atm.snrHistory = append(atm.snrHistory, snr)
        if len(atm.snrHistory) > 100 {
            atm.snrHistory = atm.snrHistory[1:]
        }
    }
    
    // Adjust VAD mode based on environment
    if time.Since(atm.lastUpdate) > 5*time.Second {
        atm.adjustVADMode()
        atm.lastUpdate = time.Now()
    }
}

func (atm *AdaptiveThresholdManager) adjustVADMode() {
    if len(atm.snrHistory) < 10 {
        return
    }
    
    // Calculate average SNR
    var avgSNR float32
    for _, snr := range atm.snrHistory {
        avgSNR += snr
    }
    avgSNR /= float32(len(atm.snrHistory))
    
    // Adjust mode based on SNR
    // Lower SNR = noisier environment = less aggressive VAD
    var newMode int
    switch {
    case avgSNR < 10: // Very noisy
        newMode = 0 // Least aggressive
    case avgSNR < 20: // Moderately noisy
        newMode = 1
    case avgSNR < 30: // Relatively quiet
        newMode = 2
    default: // Very quiet
        newMode = 3 // Most aggressive
    }
    
    if newMode != atm.currentMode {
        atm.currentMode = newMode
        // Update VAD configuration in audio processor
        atm.vadDetector.processor.UpdateVADMode(newMode)
    }
}

func (atm *AdaptiveThresholdManager) updateNoiseFloor(level float32) {
    // Exponential moving average
    alpha := float32(0.1)
    atm.noiseFloor = alpha*level + (1-alpha)*atm.noiseFloor
}
```

## Configuration
```yaml
# config/vad.yaml
vad:
  # Detection parameters
  min_speech_duration_ms: 500    # Minimum speech to trigger
  max_silence_duration_ms: 1000  # Maximum silence before end
  
  # VAD mode (0-3, higher = more aggressive)
  vad_mode: 2
  
  # Buffer settings for context
  buffer_before_ms: 500  # Include audio before speech
  buffer_after_ms: 500   # Include audio after speech
  
  # Adaptive thresholds
  adaptive:
    enabled: true
    update_interval: 5s
    min_snr: 10.0
    target_snr: 20.0
```

## Integration with Pipeline Manager
```go
// Update to pkg/pipeline/manager.go
func (m *Manager) integrateVAD() error {
    // Create VAD integration
    vadIntegration := vad.NewVADPipelineIntegration(
        m,
        m.audioManager,
        m.config.VAD,
        m.logger,
    )
    
    // Start VAD monitoring
    if err := vadIntegration.Start(m.ctx); err != nil {
        return fmt.Errorf("failed to start VAD integration: %w", err)
    }
    
    m.vadIntegration = vadIntegration
    return nil
}

// Add method to process VAD-triggered audio
func (m *Manager) ProcessAudioSegment(segment *audio.Buffer, metadata map[string]interface{}) {
    // Transition to processing state
    m.state.Transition(EventSpeechDetected)
    
    // Store segment for processing
    m.currentSegment = segment
    m.currentMetadata = metadata
    
    // Process through pipeline
    go m.processSegmentThroughPipeline(segment, metadata)
}
```

## Testing
```go
// pkg/pipeline/vad/detector_test.go
func TestVADDetection(t *testing.T) {
    // Test with clean speech audio
    t.Run("DetectsCleanSpeech", func(t *testing.T) {
        audioManager := setupTestAudioManager(t)
        detector := NewVADDetector(audioManager, testConfig, testLogger)
        
        // Inject test audio with speech
        injectTestAudio(audioManager, "testdata/clean_speech.wav")
        
        // Should detect speech segment
        event := <-detector.Events()
        assert.Equal(t, EventSpeechSegment, event.Type)
        assert.Greater(t, event.Duration, 500*time.Millisecond)
    })
    
    // Test with noisy audio
    t.Run("HandlesNoisyAudio", func(t *testing.T) {
        // Test adaptive thresholds
    })
    
    // Test minimum duration filtering
    t.Run("FiltersShortSpeech", func(t *testing.T) {
        // Verify short bursts are ignored
    })
}

func TestVADPipelineIntegration(t *testing.T) {
    // Test VAD triggering pipeline processing
    // Test audio segment extraction
    // Test metadata propagation
}
```

## Implementation Tasks

### Priority 1 - Core VAD
- [ ] Implement VADDetector using existing audio.Manager
- [ ] Create continuous monitoring loop
- [ ] Add speech segment detection logic
- [ ] Implement minimum duration filtering

### Priority 2 - Pipeline Integration  
- [ ] Connect VAD events to pipeline
- [ ] Implement audio segment extraction
- [ ] Add metadata propagation
- [ ] Create configuration structure

### Priority 3 - Adaptive Features
- [ ] Implement adaptive threshold manager
- [ ] Add noise floor estimation
- [ ] Create SNR-based adjustment
- [ ] Add environment learning

## Success Criteria
1. Reliably detects speech in tournament audio
2. Filters out short noise bursts
3. Includes appropriate audio context (before/after)
4. Adapts to environment noise levels
5. Integrates seamlessly with existing audio system