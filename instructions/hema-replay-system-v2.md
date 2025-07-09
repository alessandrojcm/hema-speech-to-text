# HEMA Tournament Replay & Commentary System

A step-by-step implementation plan for an automated replay and commentary overlay system, using Go, OBS Studio, whisper.cpp, and local LLMs.

## 🎯 Project Goal

Automatically trigger instant replays with commentator-style text overlays during a live-streamed HEMA tournament, based on judges' verbal decisions.

---

## 🧭 Phase 1: Environment & Tooling Setup

### ✅ Step 1: Prepare Development Environment

- Install Go (v1.20+)
- Set up project structure:
  ```
  cmd/
    main.go
  pkg/
    obsclient/
    audio/
    whisper/
    summarizer/
    replay/
    pipeline/
  assets/
    test_audio/
  config/
    settings.yaml
    vocab.txt
  ```
- Install dependencies:
  - OBS Studio (v28+)
  - OBS WebSocket plugin (if not built-in)
  - `obs-websocket-go` or WebSocket client in Go
  - `whisper.cpp` (with Metal support for M4)
  - `ollama` or `llama.cpp` for local LLM
  - Virtual Audio Cable or BlackHole (macOS) for audio routing
  - `portaudio-go` for direct audio capture

### 🔍 Testing

- Verify OBS connection:
  - Trigger `GetVersion`
  - Change scene
- Test basic `SaveReplayBuffer` and text overlay change
- Verify whisper.cpp Metal acceleration on M4

---

## 🧱 Phase 2: Core Replay System

### ✅ Step 2: Replay Buffer & Text Overlay

Implement a Go module that:
- Connects to OBS WebSocket
- Manages replay buffer (30-60 second continuous recording)
- Triggers `SaveReplayBuffer` with configurable pre-roll
- Updates a text source
- Switches to a "Replay Scene" and returns after N seconds
- Implements replay queue to prevent overlaps

#### 🔧 Configuration
```yaml
replay:
  buffer_duration: 60  # seconds
  pre_roll_seconds: 5
  replay_duration: 10
  min_interval: 15     # prevent replay spam
```

#### 🔧 Best Practices
- Use OBS WebSocket commands:
  - `SaveReplayBuffer`
  - `SetInputTextSettings` or `SetInputSettings`
  - `SetSceneItemEnabled` for text visibility
- Implement state machine for replay management
- Add mutex for concurrent trigger protection

### 🔍 Testing

- Simulate rapid replay triggers
- Confirm replay timing includes pre-roll
- Verify queue handles multiple requests correctly

---

## 🗣 Phase 3: Audio Pipeline (Speech to Text)

### ✅ Step 3: Audio Capture Architecture

- Set up dedicated OBS audio source for judge microphone
- Use Virtual Audio Cable/BlackHole to create loopback
- Implement continuous ring buffer in Go:

```go
type AudioCapture struct {
    bufferSize   int        // 15 seconds
    sampleRate   int        // 16kHz
    deviceID     string
    ringBuffer   *RingBuffer
    outputChan   chan []byte
}
```

### ✅ Step 4: Whisper.cpp Integration

- Compile whisper.cpp with Metal support for M4
- Use streaming mode with 2-3 second chunks
- Load custom HEMA vocabulary file
- Implement confidence threshold checking

#### 🔧 Configuration
```yaml
whisper:
  model: "small"         # or "base" for lower latency
  language: "en"
  use_metal: true
  beam_size: 5
  confidence_threshold: 0.7
  vocab_boost_file: "config/vocab.txt"
```

#### 🔧 HEMA Vocabulary Enhancement
Create `config/vocab.txt`:
```
longsword
rapier
bind
riposte
nachreisen
zornhau
point
halt
double
afterblow
```

### 🔍 Testing

- Record test audio with HEMA terminology
- Verify Metal acceleration is active
- Test with venue noise simulation
- Benchmark latency (target: <2s)

---

## 🧠 Phase 4: Local LLM Summarization

### ✅ Step 5: Local LLM Integration

- Install `ollama` with Mistral 7B or Qwen 2.5
- Implement caching layer for common phrases
- Add timeout mechanism (2s max)
- Create prompt templates

#### 🔧 Implementation
```go
type Summarizer struct {
    client      *ollama.Client
    cache       map[string]string
    timeout     time.Duration
    templates   []string
}

func (s *Summarizer) Summarize(transcript string) (string, error) {
    // Check cache first
    if cached, ok := s.cache[transcript]; ok {
        return cached, nil
    }
    
    // Call LLM with timeout
    ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
    defer cancel()
    
    // Use template prompt
    prompt := fmt.Sprintf(
        "Convert this HEMA judge call to a brief TV-style commentary (max 10 words): '%s'",
        transcript,
    )
    
    // Fallback on timeout
}
```

#### 🔧 Fallback System
```yaml
fallback_captions:
  default:
    - "Point scored!"
    - "Excellent exchange!"
    - "Match continues..."
  by_keyword:
    "point": "Point to the fencer!"
    "double": "Double hit called!"
    "halt": "Halt! Judges confer..."
```

### 🔍 Testing

- Pre-populate cache with common calls
- Test LLM response time
- Verify fallback triggers on timeout
- Validate caption quality and length

---

## 🎛 Phase 5: Full Automation Logic

### ✅ Step 6: Trigger Mechanism

Start with manual triggers, then add voice activation:

1. **Manual Mode**: Hotkey or button trigger
2. **Semi-Auto**: Voice Activity Detection (VAD) + manual confirmation
3. **Full Auto**: VAD + keyword spotting ("point", "halt", "double")

#### 🔧 Pipeline Architecture
```go
type Pipeline struct {
    audio       *AudioCapture
    whisper     *WhisperClient
    summarizer  *Summarizer
    obs         *OBSClient
    circuit     *CircuitBreaker
    queue       *ReplayQueue
}

func (p *Pipeline) ProcessTrigger() {
    // 1. Grab last 10s of audio from ring buffer
    // 2. Run through whisper (concurrent)
    // 3. If confidence > threshold, summarize
    // 4. Queue replay with caption
    // 5. Execute when previous replay completes
}
```

#### 🔧 Circuit Breaker Pattern
```go
type CircuitBreaker struct {
    maxFailures    int
    resetTimeout   time.Duration
    fallbackFunc   func() string
    state          State
}
```

### 🔍 Testing

- Simulate component failures
- Verify graceful degradation
- Test queue under load
- Measure end-to-end latency

---

## 🧪 Phase 6: Venue Testing & Performance Optimization

### ✅ Step 7: Resource Management

Monitor and optimize for MacBook Pro M4:

```go
type ResourceMonitor struct {
    cpuThreshold    float64  // 80%
    memThreshold    float64  // 70%
    degradeModel    bool
}

func (r *ResourceMonitor) CheckAndAdjust() {
    if cpuUsage > r.cpuThreshold {
        // Switch to smaller whisper model
        // Increase LLM timeout
        // Reduce beam size
    }
}
```

### ✅ Step 8: Live Trial Checklist

- [ ] Test with actual tournament audio levels
- [ ] Verify judge mic placement (lapel mic recommended)
- [ ] Run full dress rehearsal with all weapons
- [ ] Test failover scenarios
- [ ] Benchmark system under tournament conditions
- [ ] Create operator quick-reference guide

---

## 🗂 Updated Implementation Order

| Step | Component                  | Goal                                      | Estimated Time |
|------|----------------------------|-------------------------------------------|----------------|
| 1    | OBS + Audio Setup          | OBS control + audio routing               | 1 day          |
| 2    | Replay Queue System        | Manage replays without overlap            | 1 day          |
| 3    | Audio Capture              | Ring buffer + VAD implementation          | 2 days         |
| 4    | Whisper.cpp Integration    | Metal-accelerated transcription           | 2 days         |
| 5    | Local LLM + Cache          | Fast summarization with fallbacks         | 2 days         |
| 6    | Pipeline + Circuit Breaker | Reliable full system integration          | 2 days         |
| 7    | Testing + Optimization     | Performance tuning for M4                 | 1 day          |

---

## 🔧 Key Improvements in This Version

1. **Continuous Audio Capture**: Ring buffer eliminates file I/O latency
2. **Local Everything**: No internet dependency with whisper.cpp + local LLM
3. **Smart Caching**: Common phrases bypass LLM entirely
4. **Graceful Degradation**: Circuit breaker + fallbacks ensure reliability
5. **Resource Awareness**: Automatic quality adjustment based on system load
6. **Replay Queue**: Handles rapid scoring without conflicts
7. **Metal Optimization**: Leverages M4 chip capabilities

---

## 📊 Performance Targets

- **End-to-end latency**: < 3 seconds (trigger to overlay)
- **Transcription accuracy**: > 85% for common calls
- **System availability**: 99.9% during tournament
- **CPU usage**: < 60% sustained on M4
- **Fallback rate**: < 10% of triggers

---

## 🚀 Quick Start Commands

```bash
# Install whisper.cpp with Metal
git clone https://github.com/ggerganov/whisper.cpp
cd whisper.cpp
make clean && WHISPER_METAL=1 make

# Install ollama
curl -fsSL https://ollama.com/install.sh | sh
ollama pull mistral:7b-instruct

# Install audio tools (macOS)
brew install blackhole-2ch portaudio

# Run the system
go run cmd/main.go -config config/settings.yaml
```

---

This updated design addresses all identified issues while maintaining the Go-based architecture and leveraging local models for reliability and performance.