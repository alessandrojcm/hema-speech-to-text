# Phase 4: Commentary Generation - Low-Level Implementation Design

## Overview
This document provides the detailed implementation plan for the intelligent commentary generation system using go-llama.cpp bindings (from go-skynet) with small, efficient models (Qwen 2.5 3B or Gemma 2B) for real-time performance.

## Implementation Stages

### Stage 1: LLM Infrastructure Setup ✅ COMPLETED
**Timeline: 2-3 days** ✅ **COMPLETED**
**Dependencies: go-llama.cpp bindings from go-skynet, model files**

**Status**: ✅ **IMPLEMENTATION COMPLETE**
- ✅ LLM configuration types and validation
- ✅ LlamaEngine wrapper with placeholder implementation
- ✅ Comprehensive error handling system
- ✅ Thread-safe concurrent request handling
- ✅ Build tag support (nollm) for CI environments
- ✅ Complete test suite with 100% passing tests
- ✅ Ready for go-llama.cpp integration

#### 1.1 go-llama.cpp Integration
```go
// pkg/llm/types/config.go
type LLMConfig struct {
    ModelPath        string  `mapstructure:"model_path"`        // Path to GGUF model file
    ContextSize      int     `mapstructure:"context_size"`      // Default: 2048 for small models
    Threads          int     `mapstructure:"threads"`           // CPU threads
    Temperature      float32 `mapstructure:"temperature"`       // Default: 0.7
    TopP             float32 `mapstructure:"top_p"`             // Default: 0.9
    TopK             int     `mapstructure:"top_k"`             // Default: 40
    RepeatPenalty    float32 `mapstructure:"repeat_penalty"`    // Default: 1.1
    MaxTokens        int     `mapstructure:"max_tokens"`        // Default: 150
    Seed             int     `mapstructure:"seed"`              // For reproducibility
    UseGPU           bool    `mapstructure:"use_gpu"`           // GPU acceleration
    GPULayers        int     `mapstructure:"gpu_layers"`        // Layers to offload
    UseMMap          bool    `mapstructure:"use_mmap"`          // Memory-mapped I/O
    UseMlock         bool    `mapstructure:"use_mlock"`         // Lock model in memory
    EnableLowVRAM    bool    `mapstructure:"enable_low_vram"`   // Use more CPU/less GPU
}

// pkg/llm/engine/llama.go
type LlamaEngine struct {
    mu        sync.RWMutex
    llm       *llama.LLama
    config    *LLMConfig
    logger    zerolog.Logger
    metrics   *metrics.Collector
    options   []llama.PredictOption
}
```

#### 1.2 LLM Engine Implementation
```go
// pkg/llm/engine/llama.go
func NewLlamaEngine(config *LLMConfig, logger zerolog.Logger) (*LlamaEngine, error) {
    // Build options for model initialization
    modelOpts := []llama.ModelOption{
        llama.SetContext(config.ContextSize),
        llama.SetMMap(config.UseMMap),
    }
    
    if config.UseGPU {
        modelOpts = append(modelOpts, llama.SetGPULayers(config.GPULayers))
    }
    
    if config.EnableLowVRAM {
        modelOpts = append(modelOpts, llama.EnableLowVRAM)
    }
    
    // Initialize the model
    llm, err := llama.New(config.ModelPath, modelOpts...)
    if err != nil {
        return nil, fmt.Errorf("failed to load model: %w", err)
    }
    
    // Build prediction options
    predictOpts := []llama.PredictOption{
        llama.SetThreads(config.Threads),
        llama.SetTokens(config.MaxTokens),
        llama.SetTopP(config.TopP),
        llama.SetTopK(config.TopK),
        llama.SetTemperature(config.Temperature),
        llama.SetPenalty(config.RepeatPenalty),
        llama.SetSeed(config.Seed),
        llama.SetMlock(config.UseMlock),
    }
    
    return &LlamaEngine{
        llm:     llm,
        config:  config,
        logger:  logger,
        options: predictOpts,
    }, nil
}

func (e *LlamaEngine) Generate(prompt string) (string, error) {
    e.mu.RLock()
    defer e.mu.RUnlock()
    
    response, err := e.llm.Predict(prompt, e.options...)
    if err != nil {
        return "", fmt.Errorf("prediction failed: %w", err)
    }
    
    return response, nil
}

func (e *LlamaEngine) Close() {
    e.llm.Free()
}
```

#### 1.3 Key Implementation Files
- `pkg/llm/engine/llama.go` - go-llama.cpp wrapper implementation
- `pkg/llm/types/config.go` - Configuration structures
- `pkg/llm/types/errors.go` - Error definitions
- `binding/go-llama.cpp/` - Embedded go-llama.cpp library (built locally)

### Stage 2: Prompt Engineering System ✅ COMPLETED
**Timeline: 2 days** ✅ **COMPLETED**
**Focus: HEMA-specific prompt templates and optimization**

**Status**: ✅ **IMPLEMENTATION COMPLETE**
- ✅ Complete template system with 10 HEMA-specific templates
- ✅ Context management with MatchState and RingBuffer
- ✅ Dynamic prompt builder with template selection
- ✅ Thread-safe implementation with comprehensive tests
- ✅ Red/blue terminology for proper HEMA color system
- ✅ All deadlock issues resolved and race conditions fixed
- ✅ Ready for Stage 3: Smart Caching System

#### 2.1 Template System
```go
// pkg/commentary/templates/base.go
type PromptTemplate struct {
    ID          string
    Name        string
    Template    string
    Variables   []string
    MaxTokens   int
    Temperature float32
}

// pkg/commentary/templates/hema.go
var HEMATemplates = map[string]*PromptTemplate{
    "point_scored": {
        Template: `Convert this fencing judge call into engaging commentary (max 2 sentences):
Judge call: "{{.Transcription}}"
Score: {{.CurrentScore}}
Commentary:`,
        MaxTokens: 50,
    },
    "double_hit": {
        Template: `Explain this double hit situation for TV viewers:
Judge call: "{{.Transcription}}"
Brief explanation:`,
        MaxTokens: 60,
    },
}
```

#### 2.2 Context Management
```go
// pkg/commentary/context/manager.go
type ContextManager struct {
    matchState    *MatchState
    recentCalls   *RingBuffer
    fencerProfiles map[string]*FencerInfo
}

type MatchState struct {
    ScoreLeft    int
    ScoreRight   int
    Period       int
    TimeRemaining time.Duration
    LastAction   string
    LastScorer   string // "left", "right", or ""
}
```

#### 2.3 Key Implementation Files
- `pkg/commentary/templates/library.go` - Template collection
- `pkg/commentary/prompt/builder.go` - Dynamic prompt construction
- `pkg/commentary/context/enrichment.go` - Context enhancement

### Stage 3: Smart Caching System
**Timeline: 2 days**
**Focus: Multi-level caching for sub-100ms responses**

#### 3.1 Cache Architecture
```go
// pkg/commentary/cache/multilevel.go
type MultiLevelCache struct {
    l1      *MemoryCache    // Hot cache: exact matches
    l2      *FuzzyCache     // Warm cache: similar inputs
    l3      *DiskCache      // Cold cache: historical data
    hasher  *xxhash.Digest  // Fast hashing
}

// pkg/commentary/cache/memory.go
type MemoryCache struct {
    store    *ristretto.Cache
    ttl      time.Duration
    maxSize  int64
}

type CacheEntry struct {
    Key          string
    Commentary   string
    Confidence   float32
    GeneratedAt  time.Time
    HitCount     int
    LastAccessed time.Time
}
```

#### 3.2 Fuzzy Matching
```go
// pkg/commentary/cache/fuzzy.go
type FuzzyMatcher struct {
    threshold    float64 // Similarity threshold (0.85)
    normalizer   *TextNormalizer
    vectorizer   *TFIDFVectorizer
}

func (f *FuzzyMatcher) FindSimilar(input string) (*CacheEntry, float64) {
    normalized := f.normalizer.Normalize(input)
    vector := f.vectorizer.Vectorize(normalized)
    // Find nearest neighbor in vector space
}
```

#### 3.3 Key Implementation Files
- `pkg/commentary/cache/store.go` - Cache storage interface
- `pkg/commentary/cache/key.go` - Cache key generation
- `pkg/commentary/cache/metrics.go` - Cache performance tracking

### Stage 4: Commentary Generation Engine
**Timeline: 3 days**
**Focus: Core generation pipeline with quality control**

#### 4.1 Generation Pipeline
```go
// pkg/commentary/engine/generator.go
type CommentaryGenerator struct {
    llm         *LlamaEngine
    cache       *MultiLevelCache
    templates   *TemplateManager
    fallback    *FallbackSystem
    validator   *QualityValidator
    metrics     *GenerationMetrics
}

func (g *CommentaryGenerator) Generate(ctx context.Context, input TranscriptionInput) (*Commentary, error) {
    // 1. Check cache
    if cached := g.checkCache(input); cached != nil {
        return cached, nil
    }
    
    // 2. Build prompt
    prompt := g.buildPrompt(input)
    
    // 3. Generate with timeout
    resultChan := make(chan string, 1)
    errChan := make(chan error, 1)
    
    go func() {
        response, err := g.llm.Generate(prompt)
        if err != nil {
            errChan <- err
            return
        }
        resultChan <- response
    }()
    
    select {
    case result := <-resultChan:
        // 4. Validate quality
        if !g.validator.IsValid(result) {
            return g.fallback.Generate(input), nil
        }
        
        // 5. Cache result
        commentary := &Commentary{
            Text:      result,
            Source:    "llm",
            Timestamp: time.Now(),
        }
        g.cache.Store(input, commentary)
        return commentary, nil
        
    case err := <-errChan:
        g.logger.Error().Err(err).Msg("LLM generation failed")
        return g.fallback.Generate(input), nil
        
    case <-ctx.Done():
        g.logger.Warn().Msg("Generation timeout, using fallback")
        return g.fallback.Generate(input), nil
    }
}
```

#### 4.2 Quality Validation
```go
// pkg/commentary/validation/quality.go
type QualityValidator struct {
    minConfidence   float32
    maxLength       int
    minLength       int
    profanityFilter *ProfanityFilter
    relevanceScorer *RelevanceScorer
}

type ValidationResult struct {
    IsValid      bool
    Confidence   float32
    Issues       []string
    Suggestions  []string
}
```

#### 4.3 Key Implementation Files
- `pkg/commentary/engine/pipeline.go` - Main generation pipeline
- `pkg/commentary/validation/rules.go` - Validation rules
- `pkg/commentary/formatting/output.go` - Output formatting

### Stage 5: Intelligent Fallback System
**Timeline: 2 days**
**Focus: Rule-based fallback with context awareness**

#### 5.1 Fallback Engine
```go
// pkg/commentary/fallback/engine.go
type FallbackSystem struct {
    rules      *RuleEngine
    templates  map[string][]string
    selector   *ContextualSelector
    quality    *QualityScorer
}

// pkg/commentary/fallback/rules.go
type Rule struct {
    Pattern     *regexp.Regexp
    Keywords    []string
    Templates   []string
    Priority    int
    Confidence  float32
}

var HEMARules = []Rule{
    {
        Pattern:  regexp.MustCompile(`point.*(?:left|right)`),
        Keywords: []string{"point", "score", "touch"},
        Templates: []string{
            "Excellent attack scores the point!",
            "Point awarded after that exchange.",
            "Clean hit registers on the scoring box.",
        },
    },
}
```

#### 5.2 Contextual Selection
```go
// pkg/commentary/fallback/selector.go
type ContextualSelector struct {
    matchState *MatchState
    history    *CallHistory
    randomizer *rand.Rand
}

func (s *ContextualSelector) SelectTemplate(input string, rules []Rule) string {
    // Score-based selection with variety
    candidates := s.findCandidates(input, rules)
    weighted := s.applyContextWeights(candidates)
    return s.selectWithVariety(weighted)
}
```

#### 5.3 Key Implementation Files
- `pkg/commentary/fallback/library.go` - Fallback template library
- `pkg/commentary/fallback/matcher.go` - Pattern matching
- `pkg/commentary/fallback/variety.go` - Response variety management

### Stage 6: Integration Layer
**Timeline: 2 days**
**Focus: Connect with Phase 3 and system integration**

#### 6.1 Integration Manager
```go
// pkg/commentary/integration/manager.go
type IntegrationManager struct {
    speechInput   <-chan *speech.Transcription
    commentary    chan<- *Commentary
    generator     *CommentaryGenerator
    config        *IntegrationConfig
    metrics       *IntegrationMetrics
}

func (m *IntegrationManager) ProcessTranscription(t *speech.Transcription) {
    input := TranscriptionInput{
        Text:       t.Text,
        Confidence: t.Confidence,
        Timestamp:  t.Timestamp,
        AudioMetrics: t.AudioQuality,
    }
    
    commentary, err := m.generator.Generate(context.Background(), input)
    if err != nil {
        m.handleError(err)
        return
    }
    
    m.commentary <- commentary
}
```

#### 6.2 Output Formatting
```go
// pkg/commentary/output/formatter.go
type OutputFormatter struct {
    style      CommentaryStyle
    maxLength  int
    filters    []TextFilter
}

type CommentaryOutput struct {
    Text         string
    DisplayText  string // For OBS overlay
    Confidence   float32
    Source       string // "llm", "cache", "fallback"
    Latency      time.Duration
}
```

## Configuration Structure

```yaml
# config/commentary.yaml
commentary:
  llm:
    model_path: "models/qwen2.5-3b-instruct-q4_k_m.gguf"
    context_size: 2048
    batch_size: 512
    threads: 4
    temperature: 0.7
    top_p: 0.9
    max_tokens: 150
    use_gpu: true
    gpu_layers: 20
  
  cache:
    l1_size: 100MB
    l1_ttl: 5m
    l2_size: 500MB
    l2_ttl: 30m
    l3_path: "./cache/commentary"
    fuzzy_threshold: 0.85
  
  generation:
    timeout: 2s
    min_confidence: 0.7
    max_retries: 2
    concurrent_requests: 3
  
  fallback:
    enable: true
    confidence_threshold: 0.6
    variety_window: 10
    template_path: "./templates/fallback"
  
  validation:
    min_length: 10
    max_length: 200
    profanity_filter: true
    relevance_threshold: 0.75
```

## Directory Structure

```
go-llama.cpp/              # Git submodule for go-llama.cpp
├── llama.go
├── binding.cpp
├── binding.h
├── libbinding.a           # Built static library
└── ...
pkg/
├── llm/
│   ├── engine/
│   │   ├── llama.go       # Wrapper around go-llama.cpp
│   │   └── llama_test.go
│   ├── types/
│   │   ├── config.go
│   │   └── errors.go
├── commentary/
│   ├── engine/
│   │   ├── generator.go
│   │   └── pipeline.go
│   ├── templates/
│   │   ├── base.go
│   │   ├── hema.go
│   │   └── library.go
│   ├── cache/
│   │   ├── multilevel.go
│   │   ├── memory.go
│   │   ├── fuzzy.go
│   │   └── disk.go
│   ├── fallback/
│   │   ├── engine.go
│   │   ├── rules.go
│   │   └── selector.go
│   ├── validation/
│   │   ├── quality.go
│   │   └── rules.go
│   ├── context/
│   │   ├── manager.go
│   │   └── enrichment.go
│   ├── integration/
│   │   ├── manager.go
│   │   └── phase3.go
│   └── types/
│       ├── commentary.go
│       └── metrics.go
```

## Testing Strategy

### Unit Tests
```go
// pkg/commentary/engine/generator_test.go
func TestCommentaryGeneration(t *testing.T) {
    tests := []struct {
        name     string
        input    TranscriptionInput
        expected string
        source   string
    }{
        {
            name: "point_scored_from_cache",
            input: TranscriptionInput{
                Text: "Point left",
                Confidence: 0.95,
            },
            source: "cache",
        },
        {
            name: "double_hit_llm_generation",
            input: TranscriptionInput{
                Text: "Double hit no point",
                Confidence: 0.90,
            },
            source: "llm",
        },
    }
}
```

### Integration Tests
```go
// pkg/commentary/integration_test.go
func TestEndToEndCommentary(t *testing.T) {
    // Test full pipeline from transcription to commentary
}
```

### Performance Benchmarks
```go
// pkg/commentary/benchmark_test.go
func BenchmarkCommentaryGeneration(b *testing.B) {
    // Benchmark different paths: cache, LLM, fallback
}
```

## Performance Optimization

### Model Selection
- **Qwen 2.5 3B Instruct (Q4_K_M)**: ~2GB RAM, 30-50 tokens/sec on CPU
- **Gemma 2B (Q5_K_S)**: ~1.5GB RAM, 40-60 tokens/sec on CPU
- **Optimal quantization**: Q4_K_M for balance of size/quality

### Inference Optimization
- Use batch processing for multiple requests
- Implement prompt caching in llama.cpp
- GPU acceleration for first 20-30 layers
- CPU-only fallback for reliability

### Cache Optimization
- Bloom filter for fast negative lookups
- Vector similarity for fuzzy matching
- Async cache writes to prevent blocking
- Cache warming on startup

## Monitoring and Metrics

```go
// pkg/commentary/metrics/collector.go
type CommentaryMetrics struct {
    GenerationLatency   *prometheus.HistogramVec
    CacheHitRate        *prometheus.GaugeVec
    LLMTokensPerSecond  *prometheus.GaugeVec
    FallbackActivations *prometheus.CounterVec
    QualityScores       *prometheus.HistogramVec
    ErrorRate           *prometheus.CounterVec
}
```

## Error Handling Strategy

### Graceful Degradation Path
1. Try cache (L1 → L2 → L3)
2. Try LLM generation with timeout
3. Retry with relaxed parameters
4. Activate fallback system
5. Use generic safe response

### Error Recovery
- Automatic model reload on corruption
- Cache rebuild on consistency errors
- Fallback template refresh
- Metric-based circuit breakers

## Next Steps

### Implementation Order
1. **Week 1**: Stages 1-2 (LLM Infrastructure + Prompts)
2. **Week 2**: Stages 3-4 (Caching + Generation Engine)
3. **Week 3**: Stages 5-6 (Fallback + Integration)
4. **Week 4**: Testing, optimization, and production hardening

### Setup Instructions

#### 1. Add go-llama.cpp as a Git Submodule
```bash
# Add go-llama.cpp as a submodule (similar to whisper.cpp)
git submodule add https://github.com/go-skynet/go-llama.cpp go-llama.cpp
git submodule update --init --recursive

# Build the static library
cd go-llama.cpp
make clean
make libbinding.a

# Fix potential linker issues (if needed)
# Edit the Makefile to add -fPIE flags as described in the Medium article
```

#### 2. Import Configuration in Go Code
```go
// Use local import path for the embedded library
import llama "github.com/your-project/go-llama.cpp"
```

#### 3. Install Additional Dependencies
```bash
# Caching libraries
go get github.com/dgraph-io/ristretto
go get github.com/cespare/xxhash/v2

# Text processing
go get github.com/jdkato/prose/v2
go get github.com/kljensen/snowball
```

### Build Considerations

#### Potential Makefile Modifications for go-llama.cpp
If you encounter linker errors like `recompile with -fPIE`, modify `go-llama.cpp/Makefile`:

```makefile
# Add position-independent code flags
CMAKE_ARGS += -DCMAKE_POSITION_INDEPENDENT_CODE=ON -DCMAKE_C_FLAGS="-fPIE" -DCMAKE_CXX_FLAGS="-fPIE"

# Update CFLAGS and CXXFLAGS
CFLAGS   = -I./llama.cpp -I. -O3 -DNDEBUG -std=c11 -fPIC
CXXFLAGS = -I./llama.cpp -I. -I./llama.cpp/common -I./common -O3 -DNDEBUG -std=c++11 -fPIC

# Add -fPIE to binding.o compilation
binding.o: prepare
	$(CXX) $(CXXFLAGS) -fPIE -c binding.cpp -o binding.o
```

### Model Downloads
```bash
# Download Qwen 2.5 3B model (recommended)
wget https://huggingface.co/Qwen/Qwen2.5-3B-Instruct-GGUF/resolve/main/qwen2.5-3b-instruct-q4_k_m.gguf

# Alternative: Llama 2 7B Chat (as shown in the Medium article)
wget https://huggingface.co/TheBloke/Llama-2-7B-Chat-GGUF/resolve/main/llama-2-7b-chat.Q8_0.gguf

# Alternative: Gemma 2B model (lighter weight)
wget https://huggingface.co/google/gemma-2b-it-GGUF/resolve/main/gemma-2b-it-q5_k_s.gguf
```

**Note:** Ensure you download GGUF files that include the tokenizer to avoid `Could not find tokenizer.model` errors.

## Complete Integration Example

### Makefile Addition
```makefile
# Add to project Makefile similar to whisper.cpp
.PHONY: build-llama
build-llama:
	cd go-llama.cpp && make clean && make libbinding.a
	
build: build-llama
	go build -o bin/hema-replay-system cmd/replay-system/main.go
```

### Example Service Implementation
```go
// pkg/llm/service.go
package llm

import (
    "context"
    "fmt"
    "strings"
    "sync"
    
    llama "github.com/your-project/go-llama.cpp"
    "github.com/rs/zerolog"
)

type Service struct {
    mu      sync.RWMutex
    llm     *llama.LLama
    opts    []llama.PredictOption
    logger  zerolog.Logger
}

func NewService(modelPath string, config *LLMConfig, logger zerolog.Logger) (*Service, error) {
    // Initialize model with options
    modelOpts := []llama.ModelOption{
        llama.SetContext(config.ContextSize),
        llama.SetMMap(config.UseMMap),
    }
    
    if config.EnableLowVRAM {
        modelOpts = append(modelOpts, llama.EnableLowVRAM)
    }
    
    llm, err := llama.New(modelPath, modelOpts...)
    if err != nil {
        return nil, fmt.Errorf("failed to load model: %w", err)
    }
    
    // Set prediction options
    opts := []llama.PredictOption{
        llama.SetThreads(config.Threads),
        llama.SetTokens(config.MaxTokens),
        llama.SetTopP(config.TopP),
        llama.SetTemperature(config.Temperature),
        llama.SetSeed(config.Seed),
    }
    
    return &Service{
        llm:    llm,
        opts:   opts,
        logger: logger.With().Str("component", "llm").Logger(),
    }, nil
}

func (s *Service) GenerateCommentary(ctx context.Context, transcription string, matchState *MatchState) (string, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    // Build prompt based on match context
    prompt := s.buildPrompt(transcription, matchState)
    
    // Generate response
    response, err := s.llm.Predict(prompt, s.opts...)
    if err != nil {
        return "", fmt.Errorf("prediction failed: %w", err)
    }
    
    // Clean and validate response
    cleaned := strings.TrimSpace(response)
    
    return cleaned, nil
}

func (s *Service) Close() {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    if s.llm != nil {
        s.llm.Free()
    }
}
```

## Success Metrics

### Performance Targets
- P50 latency: < 100ms (cached)
- P50 latency: < 1.5s (generated)
- P99 latency: < 3s (all paths)
- Cache hit rate: > 70%
- Fallback rate: < 10%

### Quality Targets
- Commentary relevance: > 90%
- Style consistency: > 95%
- Error rate: < 1%
- User satisfaction: > 85%