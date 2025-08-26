# Phase 5 - Stage 1: Pipeline Orchestration

## Overview
This stage creates the core orchestration layer that integrates all existing components from Phases 1-4 into a cohesive pipeline. The system continuously listens to audio, generates commentary, and updates OBS text overlays on a hidden replay scene. The operator maintains full control over when to trigger replays - the system never automatically switches scenes. We'll build upon the existing `audio.Manager`, `speech.Manager`, `commentary.Engine`, and `obs.Client` rather than recreating them.

## Design Philosophy
- **Operator Control**: The broadcast operator decides when to show replays, not the system
- **Continuous Updates**: Commentary is continuously generated and updated on a hidden OBS scene
- **No False Positives**: System never disrupts the broadcast with automatic scene switches
- **Ready When Needed**: When operator triggers replay, commentary is already prepared and visible

## Architecture

### Pipeline Manager
```go
// pkg/pipeline/manager.go
package pipeline

import (
    "context"
    "github.com/rs/zerolog"
    "github.com/yourusername/hema-replay/pkg/audio"
    "github.com/yourusername/hema-replay/pkg/speech"
    "github.com/yourusername/hema-replay/pkg/commentary"
    "github.com/yourusername/hema-replay/internal/obs"
)

type Manager struct {
    // Existing components - we reuse these, not recreate
    audioManager     *audio.Manager
    speechManager    *speech.Manager
    commentaryEngine *commentary.Engine
    obsClient        *obs.Client
    
    // New orchestration layer
    state            *StateManager
    eventBus         *EventBus
    metrics          *MetricsCollector
    commentaryBuffer *CommentaryBuffer  // Stores recent commentary for operator
    
    // Configuration
    config           *Config
    logger           zerolog.Logger
    
    // Control
    ctx              context.Context
    cancel           context.CancelFunc
    errorChan        chan error
}

func NewManager(cfg *Config, logger zerolog.Logger) (*Manager, error) {
    // Initialize existing components using their existing constructors
    audioManager, err := audio.NewManager(cfg.Audio, logger)
    if err != nil {
        return nil, fmt.Errorf("failed to create audio manager: %w", err)
    }
    
    speechManager, err := speech.NewManager(cfg.Speech, logger)
    if err != nil {
        return nil, fmt.Errorf("failed to create speech manager: %w", err)
    }
    
    commentaryEngine, err := commentary.NewEngine(cfg.Commentary, logger)
    if err != nil {
        return nil, fmt.Errorf("failed to create commentary engine: %w", err)
    }
    
    obsClient, err := obs.NewClient(cfg.OBS, logger)
    if err != nil {
        return nil, fmt.Errorf("failed to create OBS client: %w", err)
    }
    
    return &Manager{
        audioManager:     audioManager,
        speechManager:    speechManager,
        commentaryEngine: commentaryEngine,
        obsClient:        obsClient,
        state:           NewStateManager(),
        eventBus:        NewEventBus(),
        metrics:         NewMetricsCollector(),
        commentaryBuffer: NewCommentaryBuffer(cfg.CommentaryBufferSize),
        config:          cfg,
        logger:          logger.With().Str("component", "pipeline").Logger(),
        errorChan:       make(chan error, 10),
    }, nil
}
```

### State Management
```go
// pkg/pipeline/state.go
package pipeline

type State int

const (
    StateIdle State = iota
    StateListening    // Monitoring audio for speech
    StateProcessing   // Processing detected speech
    StateGenerating   // Generating commentary
    StateUpdating     // Updating OBS text overlay (no scene switching)
    StateError
    StateRecovering
)

type StateManager struct {
    current     State
    previous    State
    transitions map[StateTransition]TransitionFunc
    mu          sync.RWMutex
}

type StateTransition struct {
    From  State
    Event Event
}

type Event int

const (
    EventStart Event = iota
    EventSpeechDetected
    EventProcessingComplete
    EventCommentaryReady
    EventOverlayUpdated   // Text overlay updated on hidden scene
    EventError
    EventRecover
)

func (sm *StateManager) Transition(event Event) error {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    
    transition := StateTransition{
        From:  sm.current,
        Event: event,
    }
    
    handler, ok := sm.transitions[transition]
    if !ok {
        return fmt.Errorf("invalid transition from %v with event %v", sm.current, event)
    }
    
    newState, err := handler()
    if err != nil {
        return err
    }
    
    sm.previous = sm.current
    sm.current = newState
    return nil
}
```

### Event-Driven Processing
```go
// pkg/pipeline/events.go
package pipeline

type EventBus struct {
    subscribers map[EventType][]EventHandler
    mu          sync.RWMutex
}

type EventType string

const (
    EventTypeAudioReady      EventType = "audio.ready"
    EventTypeSpeechDetected  EventType = "speech.detected"
    EventTypeTranscriptReady EventType = "transcript.ready"
    EventTypeCommentaryReady EventType = "commentary.ready"
    EventTypeOverlayUpdated  EventType = "overlay.updated"  // Text updated on hidden scene
    EventTypeError          EventType = "error"
)

type PipelineEvent struct {
    Type      EventType
    Timestamp time.Time
    Data      interface{}
    Error     error
}

func (eb *EventBus) Publish(event PipelineEvent) {
    eb.mu.RLock()
    handlers := eb.subscribers[event.Type]
    eb.mu.RUnlock()
    
    for _, handler := range handlers {
        go handler(event) // Non-blocking
    }
}

func (eb *EventBus) Subscribe(eventType EventType, handler EventHandler) {
    eb.mu.Lock()
    defer eb.mu.Unlock()
    eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)
}
```

### Pipeline Processing Flow
```go
// pkg/pipeline/processor.go
package pipeline

func (m *Manager) Start(ctx context.Context) error {
    m.ctx, m.cancel = context.WithCancel(ctx)
    
    // Start all existing components
    if err := m.audioManager.Start(m.ctx); err != nil {
        return fmt.Errorf("failed to start audio manager: %w", err)
    }
    
    if err := m.speechManager.Start(m.ctx); err != nil {
        return fmt.Errorf("failed to start speech manager: %w", err)
    }
    
    if err := m.commentaryEngine.Start(m.ctx); err != nil {
        return fmt.Errorf("failed to start commentary engine: %w", err)
    }
    
    // Connect to OBS
    if err := m.obsClient.Connect(); err != nil {
        return fmt.Errorf("failed to connect to OBS: %w", err)
    }
    
    // Set initial state
    m.state.Transition(EventStart)
    
    // Start processing pipeline
    go m.processPipeline()
    
    return nil
}

func (m *Manager) processPipeline() {
    for {
        select {
        case <-m.ctx.Done():
            return
            
        default:
            switch m.state.Current() {
            case StateListening:
                // Will be implemented in Stage 2 with VAD
                // Continuously monitors audio for speech
                
            case StateProcessing:
                // Process audio through existing speech recognition
                if audioSegment := m.getAudioSegment(); audioSegment != nil {
                    m.processAudioSegment(audioSegment)
                }
                
            case StateGenerating:
                // Generate commentary using existing engine
                if transcript := m.getTranscript(); transcript != nil {
                    m.generateCommentary(transcript)
                }
                
            case StateUpdating:
                // Update text overlay on hidden replay scene (no scene switching)
                if commentary := m.getCommentary(); commentary != nil {
                    m.updateOverlay(commentary)
                }
            }
        }
    }
}

func (m *Manager) processAudioSegment(segment *audio.Buffer) {
    // Use existing speech manager
    result, err := m.speechManager.ProcessSegment(segment)
    if err != nil {
        m.handleError(fmt.Errorf("speech processing failed: %w", err))
        return
    }
    
    m.eventBus.Publish(PipelineEvent{
        Type:      EventTypeTranscriptReady,
        Timestamp: time.Now(),
        Data:      result,
    })
    
    m.state.Transition(EventProcessingComplete)
}

func (m *Manager) generateCommentary(transcript *speech.Result) {
    // Use existing commentary engine
    commentary, err := m.commentaryEngine.Generate(m.ctx, transcript)
    if err != nil {
        m.handleError(fmt.Errorf("commentary generation failed: %w", err))
        return
    }
    
    // Store in buffer for operator visibility
    m.commentaryBuffer.Add(commentary)
    
    m.eventBus.Publish(PipelineEvent{
        Type:      EventTypeCommentaryReady,
        Timestamp: time.Now(),
        Data:      commentary,
    })
    
    m.state.Transition(EventCommentaryReady)
}

func (m *Manager) updateOverlay(commentary *commentary.Result) {
    // Update text source on HIDDEN replay scene - no scene switching
    textSourceName := m.config.OBS.ReplayTextSource // e.g., "replay_commentary"
    
    err := m.obsClient.UpdateTextSource(textSourceName, commentary.Text)
    if err != nil {
        m.handleError(fmt.Errorf("overlay update failed: %w", err))
        return
    }
    
    // Optionally update additional metadata (timestamp, confidence, etc.)
    if m.config.ShowMetadata {
        metaSourceName := m.config.OBS.MetadataTextSource
        metadata := fmt.Sprintf("Confidence: %.0f%% | %s", 
            commentary.Confidence*100, 
            time.Now().Format("15:04:05"))
        m.obsClient.UpdateTextSource(metaSourceName, metadata)
    }
    
    m.eventBus.Publish(PipelineEvent{
        Type:      EventTypeOverlayUpdated,
        Timestamp: time.Now(),
        Data:      commentary,
    })
    
    // Return to listening state for continuous processing
    m.state.Transition(EventOverlayUpdated)
}
```

### Error Handling
```go
// pkg/pipeline/errors.go
package pipeline

type ErrorHandler struct {
    strategies  map[ErrorType]ErrorStrategy
    fallbacks   map[string]FallbackFunc
    logger      zerolog.Logger
}

type ErrorType string

const (
    ErrorTypeAudioCapture   ErrorType = "audio_capture"
    ErrorTypeSpeech        ErrorType = "speech_recognition"
    ErrorTypeCommentary    ErrorType = "commentary_generation"
    ErrorTypeOBS           ErrorType = "obs_communication"
    ErrorTypeTimeout       ErrorType = "timeout"
)

func (m *Manager) handleError(err error) {
    m.logger.Error().Err(err).Msg("Pipeline error occurred")
    
    // Determine error type
    errorType := m.classifyError(err)
    
    // Apply recovery strategy
    switch errorType {
    case ErrorTypeSpeech:
        // Fallback: Skip to next audio segment
        m.state.Transition(EventRecover)
        
    case ErrorTypeCommentary:
        // Fallback: Use default commentary
        m.useDefaultCommentary()
        
    case ErrorTypeOBS:
        // Attempt reconnection
        go m.attemptOBSReconnection()
        
    default:
        // Generic recovery
        m.state.Transition(EventError)
    }
    
    // Send to error channel for monitoring
    select {
    case m.errorChan <- err:
    default:
        // Don't block if channel is full
    }
}
```

### Metrics Collection
```go
// pkg/pipeline/metrics.go
package pipeline

type MetricsCollector struct {
    processingTimes map[string]*DurationMetric
    errorCounts     map[ErrorType]int64
    successCount    int64
    totalCount      int64
    mu              sync.RWMutex
}

type DurationMetric struct {
    count    int64
    total    time.Duration
    min      time.Duration
    max      time.Duration
    lastTime time.Duration
}

func (mc *MetricsCollector) RecordProcessingTime(stage string, duration time.Duration) {
    mc.mu.Lock()
    defer mc.mu.Unlock()
    
    metric, ok := mc.processingTimes[stage]
    if !ok {
        metric = &DurationMetric{min: duration, max: duration}
        mc.processingTimes[stage] = metric
    }
    
    metric.count++
    metric.total += duration
    metric.lastTime = duration
    
    if duration < metric.min {
        metric.min = duration
    }
    if duration > metric.max {
        metric.max = duration
    }
}

func (mc *MetricsCollector) GetStats() map[string]interface{} {
    mc.mu.RLock()
    defer mc.mu.RUnlock()
    
    stats := make(map[string]interface{})
    
    // Calculate success rate
    if mc.totalCount > 0 {
        stats["success_rate"] = float64(mc.successCount) / float64(mc.totalCount)
    }
    
    // Processing times
    for stage, metric := range mc.processingTimes {
        if metric.count > 0 {
            stats[stage+"_avg_ms"] = metric.total.Milliseconds() / metric.count
            stats[stage+"_min_ms"] = metric.min.Milliseconds()
            stats[stage+"_max_ms"] = metric.max.Milliseconds()
        }
    }
    
    // Error counts
    for errorType, count := range mc.errorCounts {
        stats["errors_"+string(errorType)] = count
    }
    
    return stats
}
```

## Configuration
```yaml
# config/pipeline.yaml
pipeline:
  # Processing settings
  max_concurrent_requests: 3
  processing_timeout: 5s
  
  # Commentary buffer
  commentary_buffer_size: 10  # Keep last 10 commentaries
  commentary_ttl: 5m          # Commentary expires after 5 minutes
  
  # State management
  state_persistence: true
  state_file: "pipeline_state.json"
  
  # Error handling
  max_retries: 3
  retry_delay: 1s
  fallback_enabled: true
  
  # Metrics
  metrics_enabled: true
  metrics_interval: 10s
  
  # Display options
  show_metadata: true         # Show confidence scores and timestamps
  
# Use existing component configs
audio:
  # Existing audio config from Phase 2
  
speech:
  # Existing speech config from Phase 3
  
commentary:
  # Existing commentary config from Phase 4
  
obs:
  # Existing OBS config from Phase 1
  replay_text_source: "replay_commentary"      # Text source on replay scene
  metadata_text_source: "replay_metadata"      # Optional metadata display
  replay_scene: "Replay Scene"                 # Hidden scene with overlays
```

## Implementation Tasks

### Priority 1 - Core Integration
- [x] Create pipeline package structure
- [x] Implement Manager that uses existing components
- [x] Create StateManager for pipeline states
- [x] Implement EventBus for component communication
- [x] Add basic error handling

### Priority 2 - Processing Flow
- [x] Connect audio.Manager output to pipeline
- [x] Integrate speech.Manager processing
- [x] Connect commentary.Engine generation
- [x] Link obs.Client replay triggering
- [x] Implement state transitions

### Priority 3 - Monitoring & Recovery
- [ ] Add MetricsCollector
- [ ] Implement error classification
- [ ] Create fallback mechanisms
- [ ] Add recovery strategies
- [ ] Implement graceful shutdown

## Testing Requirements
```go
// pkg/pipeline/manager_test.go
func TestPipelineIntegration(t *testing.T) {
    // Test component initialization
    // Test state transitions
    // Test error handling
    // Test metrics collection
}

func TestExistingComponentIntegration(t *testing.T) {
    // Verify audio.Manager integration
    // Verify speech.Manager integration
    // Verify commentary.Engine integration
    // Verify obs.Client integration
}
```

## Integration Points

### With Existing Components
- **audio.Manager**: Subscribe to audio buffer updates
- **speech.Manager**: Send audio segments for processing
- **commentary.Engine**: Generate commentary from transcripts
- **obs.Client**: Trigger replays with overlays
- **replay.Manager**: Coordinate replay timing

### Future Stages
- **Stage 2**: VAD integration will plug into StateListening
- **Stage 3**: Trigger mechanisms will initiate pipeline
- **Stage 4**: Keyword spotting will filter relevant audio
- **Stage 5**: Circuit breakers will wrap component calls

## Success Criteria
1. Successfully orchestrates existing components
2. Maintains state across pipeline stages
3. Handles component errors gracefully
4. Collects meaningful metrics
5. Provides clean integration points for future stages