# Phase 8: Speech Recognition Improvements and Debugging

## Problem Statement
The current speech recognition system has several issues:
1. Capturing blank audio or non-speech sounds
2. Not properly utilizing HEMA vocabulary for better recognition
3. VAD triggering on ambient noise leading to false positives
4. No pre-filtering of low-quality audio segments

## Implementation Tasks

### 1. Implement Proper Whisper Vocabulary Boosting

**File to modify:** `pkg/speech/whisper/wrapper.go`

#### 1.1 Add Initial Prompt Support
- Modify the `configureContext` function to set an initial prompt using HEMA vocabulary
- Build the prompt from high-boost terms in the vocabulary file
- Implementation approach:

```go
// In configureContext function, add:
func (ww *WhisperWrapper) configureContext(context whisper.Context, params types.WhisperParams) error {
    // ... existing code ...
    
    // Set initial prompt with HEMA vocabulary
    if params.InitialPrompt != "" {
        context.SetInitialPrompt(params.InitialPrompt)
    }
    
    // Enable noise suppression
    context.SetSuppressBlank(true)     // Suppress blank outputs
    context.SetSuppressNonSpeechTokens(true) // Suppress non-speech tokens
    
    // Set token timestamps for better alignment
    context.SetTokenTimestamps(true)
    
    // ... rest of existing code ...
}
```

#### 1.2 Build HEMA Vocabulary Prompt
**File to create:** `pkg/speech/vocabulary/prompt_builder.go`

```go
package vocabulary

import (
    "fmt"
    "strings"
    "sort"
)

// BuildInitialPrompt creates an initial prompt from high-confidence HEMA terms
func (hv *HEMAVocabulary) BuildInitialPrompt() string {
    hv.mu.RLock()
    defer hv.mu.RUnlock()
    
    // Collect high-boost terms (boost >= 1.8)
    var highBoostTerms []string
    for term, vocabTerm := range hv.terms {
        if vocabTerm.Boost >= 1.8 {
            highBoostTerms = append(highBoostTerms, term)
        }
    }
    
    // Sort by boost value (descending) and take top 30 terms
    sort.Slice(highBoostTerms, func(i, j int) bool {
        return hv.terms[highBoostTerms[i]].Boost > hv.terms[highBoostTerms[j]].Boost
    })
    
    if len(highBoostTerms) > 30 {
        highBoostTerms = highBoostTerms[:30]
    }
    
    // Build a natural-sounding prompt that includes key terms
    // This helps prime Whisper to recognize these terms
    promptParts := []string{
        "Tournament bout with longsword and rapier.",
        "Judge calls: halt, point, double, afterblow, no-touch.",
        "Techniques: bind, riposte, thrust, cut, parry.",
        fmt.Sprintf("Terms: %s.", strings.Join(highBoostTerms, ", ")),
    }
    
    return strings.Join(promptParts, " ")
}
```

### 2. Add Noise Suppression and Token Filtering

**File to modify:** `pkg/speech/types/config.go`

Add new configuration options:
```go
type WhisperParams struct {
    // ... existing fields ...
    
    // Noise suppression
    SuppressBlank        bool    `mapstructure:"suppress_blank"`
    SuppressNonSpeech    bool    `mapstructure:"suppress_non_speech"`
    SuppressRegex        string  `mapstructure:"suppress_regex"`
    InitialPrompt        string  `mapstructure:"initial_prompt"`
    
    // Quality thresholds
    NoSpeechThreshold    float32 `mapstructure:"no_speech_threshold"`
    LogProbThreshold     float32 `mapstructure:"logprob_threshold"`
    
    // Token filtering
    MinTokenConfidence   float32 `mapstructure:"min_token_confidence"`
}
```

**Update config file:** `config/pipeline.example.yaml`
```yaml
speech:
  whisper:
    model_path: "models/ggml-base.bin"
    language: "en"
    suppress_blank: true
    suppress_non_speech: true
    suppress_regex: "\\[.*\\]|\\(.*\\)|um|uh|ah|eh"  # Filter out brackets, parentheses, filler words
    no_speech_threshold: 0.6   # Higher = more aggressive filtering
    logprob_threshold: -1.0     # Token log probability threshold
    min_token_confidence: 0.3   # Minimum confidence for tokens
    initial_prompt: ""  # Will be auto-generated from vocabulary
```

### 3. Implement Audio Quality Pre-filtering

**File to create:** `pkg/speech/preprocessing/quality_filter.go`

```go
package preprocessing

import (
    "math"
    "github.com/rs/zerolog"
)

type QualityFilter struct {
    minEnergy      float32
    minSNR         float32
    minVoiceRatio  float32
    logger         zerolog.Logger
}

func NewQualityFilter(logger zerolog.Logger) *QualityFilter {
    return &QualityFilter{
        minEnergy:     0.01,  // Minimum RMS energy
        minSNR:        3.0,   // Minimum signal-to-noise ratio in dB
        minVoiceRatio: 0.2,   // Minimum ratio of voiced frames
        logger:        logger.With().Str("component", "quality_filter").Logger(),
    }
}

// ShouldProcessSegment determines if an audio segment is worth processing
func (qf *QualityFilter) ShouldProcessSegment(samples []float32) (bool, map[string]float32) {
    metrics := qf.calculateMetrics(samples)
    
    // Check minimum energy (avoid silent segments)
    if metrics["rms_energy"] < qf.minEnergy {
        qf.logger.Debug().
            Float32("rms_energy", metrics["rms_energy"]).
            Msg("Segment rejected: energy too low")
        return false, metrics
    }
    
    // Check for sufficient voice content
    if metrics["voice_ratio"] < qf.minVoiceRatio {
        qf.logger.Debug().
            Float32("voice_ratio", metrics["voice_ratio"]).
            Msg("Segment rejected: insufficient voice content")
        return false, metrics
    }
    
    // Check signal-to-noise ratio
    if metrics["snr_db"] < qf.minSNR {
        qf.logger.Debug().
            Float32("snr_db", metrics["snr_db"]).
            Msg("Segment rejected: SNR too low")
        return false, metrics
    }
    
    return true, metrics
}

func (qf *QualityFilter) calculateMetrics(samples []float32) map[string]float32 {
    metrics := make(map[string]float32)
    
    // Calculate RMS energy
    var sumSquares float64
    for _, sample := range samples {
        sumSquares += float64(sample * sample)
    }
    metrics["rms_energy"] = float32(math.Sqrt(sumSquares / float64(len(samples))))
    
    // Calculate zero-crossing rate (indicator of voice vs noise)
    var zeroCrossings int
    for i := 1; i < len(samples); i++ {
        if (samples[i-1] >= 0 && samples[i] < 0) || (samples[i-1] < 0 && samples[i] >= 0) {
            zeroCrossings++
        }
    }
    metrics["zcr"] = float32(zeroCrossings) / float32(len(samples))
    
    // Estimate voice ratio using simple energy-based VAD
    frameSize := 160 // 10ms at 16kHz
    voicedFrames := 0
    totalFrames := len(samples) / frameSize
    
    for i := 0; i < len(samples)-frameSize; i += frameSize {
        frameEnergy := qf.calculateFrameEnergy(samples[i : i+frameSize])
        if frameEnergy > qf.minEnergy*2 {
            voicedFrames++
        }
    }
    
    metrics["voice_ratio"] = float32(voicedFrames) / float32(totalFrames)
    
    // Estimate SNR (simplified)
    metrics["snr_db"] = 20 * float32(math.Log10(float64(metrics["rms_energy"]/0.001)))
    
    return metrics
}

func (qf *QualityFilter) calculateFrameEnergy(frame []float32) float32 {
    var energy float32
    for _, sample := range frame {
        energy += sample * sample
    }
    return energy / float32(len(frame))
}
```

### 4. Add Debug Mode for Audio Analysis

**File to modify:** `cmd/replay-system/main.go`

Add new CLI flags:
```go
var (
    // ... existing flags ...
    debugAudio     = flag.Bool("debug-audio", false, "Save audio segments for debugging")
    debugOutputDir = flag.String("debug-output", "./debug_audio", "Directory for debug audio files")
    vadDebug       = flag.Bool("vad-debug", false, "Enable detailed VAD logging")
)
```

**File to create:** `pkg/audio/debug/segment_saver.go`

```go
package debug

import (
    "fmt"
    "os"
    "path/filepath"
    "time"
    
    "github.com/go-audio/audio"
    "github.com/go-audio/wav"
    "github.com/rs/zerolog"
)

type SegmentSaver struct {
    outputDir string
    enabled   bool
    logger    zerolog.Logger
}

func NewSegmentSaver(outputDir string, enabled bool, logger zerolog.Logger) *SegmentSaver {
    if enabled {
        os.MkdirAll(outputDir, 0755)
    }
    
    return &SegmentSaver{
        outputDir: outputDir,
        enabled:   enabled,
        logger:    logger.With().Str("component", "segment_saver").Logger(),
    }
}

func (ss *SegmentSaver) SaveSegment(samples []float32, metadata map[string]interface{}) error {
    if !ss.enabled {
        return nil
    }
    
    timestamp := time.Now().Format("20060102_150405")
    segmentType := "unknown"
    if t, ok := metadata["type"].(string); ok {
        segmentType = t
    }
    
    filename := fmt.Sprintf("%s_%s.wav", timestamp, segmentType)
    filepath := filepath.Join(ss.outputDir, filename)
    
    // Also save metadata as JSON
    metaFile := fmt.Sprintf("%s_%s.json", timestamp, segmentType)
    metaPath := filepath.Join(ss.outputDir, metaFile)
    
    // Convert float32 to int for WAV encoding
    intSamples := make([]int, len(samples))
    for i, s := range samples {
        intSamples[i] = int(s * 32767)
    }
    
    // Save WAV file
    file, err := os.Create(filepath)
    if err != nil {
        return fmt.Errorf("failed to create debug audio file: %w", err)
    }
    defer file.Close()
    
    encoder := wav.NewEncoder(file, 16000, 16, 1, 1)
    defer encoder.Close()
    
    buf := &audio.IntBuffer{
        Data:   intSamples,
        Format: &audio.Format{SampleRate: 16000, NumChannels: 1},
    }
    
    if err := encoder.Write(buf); err != nil {
        return fmt.Errorf("failed to write debug audio: %w", err)
    }
    
    ss.logger.Debug().
        Str("file", filename).
        Int("samples", len(samples)).
        Interface("metadata", metadata).
        Msg("Saved debug audio segment")
    
    return nil
}
```

### 5. Optimize VAD Parameters for HEMA Environment

**File to modify:** `config/pipeline.example.yaml`

```yaml
vad:
  min_speech_duration_ms: 500   # Increase from 250ms to reduce false triggers
  max_silence_duration_ms: 1000 # Allow longer pauses in speech
  vad_mode: 2                    # More aggressive mode (0-3, higher = more aggressive)
  buffer_before_ms: 200          # Capture context before speech
  buffer_after_ms: 300           # Capture context after speech
  
  # New parameters for better filtering
  energy_threshold: 0.02         # Minimum energy to consider as speech
  zero_crossing_threshold: 0.3   # Maximum ZCR for speech (vs music/noise)
  spectral_flatness_threshold: 0.5  # Differentiate speech from white noise
```

**File to modify:** `pkg/pipeline/vad/detector.go`

Add enhanced VAD logic:
```go
// Add to VADDetector struct
type VADDetector struct {
    // ... existing fields ...
    
    // Enhanced detection parameters
    energyThreshold          float32
    zeroCrossingThreshold    float32
    spectralFlatnessThreshold float32
    consecutiveSpeechFrames  int
    consecutiveSilenceFrames int
}

// Enhanced detection method
func (v *VADDetector) detectSpeechEnhanced(audioData []float32) bool {
    // First, use WebRTC VAD
    webrtcResult := v.processor.DetectVoiceActivity(audioData)
    
    // Then apply additional filters
    energy := v.calculateEnergy(audioData)
    zcr := v.calculateZeroCrossingRate(audioData)
    
    // Require both WebRTC detection AND energy threshold
    if !webrtcResult || energy < v.energyThreshold {
        v.consecutiveSilenceFrames++
        v.consecutiveSpeechFrames = 0
        return false
    }
    
    // Check if ZCR indicates speech (not music or noise)
    if zcr > v.zeroCrossingThreshold {
        v.consecutiveSilenceFrames++
        v.consecutiveSpeechFrames = 0
        return false
    }
    
    // Require consecutive frames for stability
    v.consecutiveSpeechFrames++
    v.consecutiveSilenceFrames = 0
    
    // Need at least 3 consecutive speech frames to trigger
    return v.consecutiveSpeechFrames >= 3
}
```

### 6. Integration Points

**File to modify:** `pkg/speech/engine/pipeline.go`

Update the processing pipeline to use all improvements:
```go
func (pp *ProcessingPipeline) Process(request *types.ProcessingRequest) (*types.TranscriptionResult, error) {
    // Step 1: Quality filtering
    if pp.qualityFilter != nil {
        shouldProcess, metrics := pp.qualityFilter.ShouldProcessSegment(request.AudioData)
        if !shouldProcess {
            pp.logger.Debug().
                Interface("metrics", metrics).
                Msg("Segment filtered out due to low quality")
            return nil, fmt.Errorf("audio quality too low: %v", metrics)
        }
    }
    
    // Step 2: Save debug audio if enabled
    if pp.debugSaver != nil {
        pp.debugSaver.SaveSegment(request.AudioData, map[string]interface{}{
            "type": "pre_whisper",
            "vad_confidence": request.VADConfidence,
        })
    }
    
    // Step 3: Prepare Whisper parameters with vocabulary
    whisperParams := pp.prepareWhisperParams(request)
    
    // Build initial prompt from vocabulary
    if pp.vocabulary != nil && request.UseVocabulary {
        whisperParams.InitialPrompt = pp.vocabulary.BuildInitialPrompt()
    }
    
    // Enable noise suppression
    whisperParams.SuppressBlank = true
    whisperParams.SuppressNonSpeech = true
    whisperParams.NoSpeechThreshold = 0.6
    
    // Step 4: Process with Whisper
    result, err := wrapper.Transcribe(audioData, whisperParams)
    
    // Step 5: Post-process and filter low-confidence tokens
    if result != nil {
        pp.filterLowConfidenceTokens(result)
    }
    
    return result, err
}
```

### 7. Testing and Validation

Create test cases to validate improvements:

**File to create:** `pkg/speech/testing/quality_test.go`

```go
package testing

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestQualityFiltering(t *testing.T) {
    tests := []struct {
        name          string
        audioFile     string
        shouldProcess bool
        description   string
    }{
        {
            name:          "silence",
            audioFile:     "testdata/silence.wav",
            shouldProcess: false,
            description:   "Silent audio should be filtered",
        },
        {
            name:          "white_noise",
            audioFile:     "testdata/white_noise.wav", 
            shouldProcess: false,
            description:   "Pure noise should be filtered",
        },
        {
            name:          "clear_speech",
            audioFile:     "testdata/clear_speech.wav",
            shouldProcess: true,
            description:   "Clear speech should pass",
        },
        {
            name:          "noisy_speech",
            audioFile:     "testdata/noisy_speech.wav",
            shouldProcess: true,
            description:   "Speech with background noise should pass if SNR is acceptable",
        },
    }
    
    // Run tests...
}
```

## Code to Eliminate/Simplify

### 1. Post-Processing Vocabulary Boosting (Can be Removed)

**File:** `pkg/speech/engine/pipeline.go` (Lines 100-130)

The current `applyVocabularyBoosting` function only applies confidence boosts after transcription:
```go
// This entire function can be REMOVED once we use initial_prompt
func (pp *ProcessingPipeline) applyVocabularyBoosting(result *types.TranscriptionResult) {
    // ... post-processing boost logic ...
}
```

**Reason:** With proper initial_prompt, Whisper will naturally recognize HEMA terms better during transcription, making post-processing unnecessary.

### 2. Simplified Vocabulary Manager

**File:** `pkg/speech/vocabulary/hema.go`

Can remove or simplify these methods once using initial_prompt:
```go
// These methods become less critical:
- GetBoost() - No longer needed for post-processing
- UpdateBoost() - Boost values used only for prompt building
- The boost application logic in the pipeline
```

### 3. Redundant Audio Processing

**File:** `pkg/audio/processing/processor.go`

With quality filtering, we can eliminate redundant processing:
```go
// Current code processes ALL audio regardless of quality
// Can add early exit:
if !meetsQualityThreshold(audioData) {
    return nil, ErrLowQualityAudio
}
```

### 4. Complex VAD State Management

**File:** `pkg/pipeline/vad/detector.go`

Current complex state tracking can be simplified:
```go
// Current complex state machine
type VADDetector struct {
    isActive      bool
    activityStart time.Time
    silenceStart  time.Time
    // ... many state variables
}

// Can be simplified to:
type VADDetector struct {
    consecutiveSpeechFrames   int
    consecutiveSilenceFrames  int
    lastSpeechTime            time.Time
}
```

### 5. Remove Unused FFT Processing

**Files to potentially remove:**
- `pkg/audio/processing/fft.go` - If not used for quality metrics
- `pkg/audio/processing/fft_test.go` 

**Reason:** The FFT processing appears to have import cycle issues and may not be necessary if we're using simpler energy-based quality metrics.

### 6. Consolidate Audio Processing Interfaces

**Current structure has redundancy:**
```
pkg/audio/processing/
├── processor.go           # Main processor
├── processor_integration_test.go
├── interfaces.go          # Can be consolidated
├── resampler_gosamplerate.go
├── resampler_test.go      # Has import cycle issues
├── vad_webrtc.go
├── wav_goaudio.go
└── quality.go            # Has import cycle issues
```

**Consolidation opportunity:** Merge interfaces.go into processor.go and fix import cycles.

### 7. Remove Duplicate Test Files with Import Cycles

**Files with import cycle errors that can be fixed or removed:**
```
pkg/audio/processing/converter_test.go
pkg/audio/processing/processor_test.go
pkg/audio/processing/quality.go
pkg/audio/processing/resampler_test.go
pkg/audio/processing/converter.go
```

**Fix:** Move these to a separate test package to avoid cycles:
```go
package processing_test  // Instead of package processing
```

## Refactoring Benefits

### Before (Complex Pipeline):
```
Audio → VAD Detection → Buffer → Processing → Whisper → Post-Process → Vocabulary Boost → Result
         (complex)     (all)    (redundant)            (ineffective)   (post-facto)
```

### After (Streamlined Pipeline):
```
Audio → Enhanced VAD → Quality Filter → Whisper (with prompt) → Result
        (simpler)      (early exit)     (pre-boosted)
```

### Lines of Code Reduction Estimate:
- Remove post-processing vocabulary boost: ~100 lines
- Simplify VAD state management: ~150 lines
- Remove redundant FFT if unused: ~200 lines
- Fix import cycles and consolidate: ~300 lines
- **Total potential reduction: ~750 lines**

## Breaking Changes - No Migration Needed

Since this is development software, we can make breaking changes directly:

### API Changes to Make Immediately:

1. **Remove `applyVocabularyBoosting()` from pipeline** - Delete entirely
2. **Change VADDetector struct** - Replace all state tracking with simple counters
3. **Delete FFT processing files** - Remove if not used for quality metrics
4. **Rename test packages** - Fix import cycles by using `_test` suffix
5. **Modify ProcessingRequest** - Add quality threshold fields
6. **Replace Vocabulary.GetBoost()** - Change to BuildInitialPrompt() only

### Files to Delete Immediately:
```bash
rm pkg/audio/processing/fft.go
rm pkg/audio/processing/fft_test.go  
rm pkg/audio/processing/quality.go  # Has import cycle
rm pkg/audio/processing/converter.go  # Has import cycle
```

### Simplified New API:
```go
// Old complex API - DELETE
type ProcessingPipeline struct {
    preprocessor *Preprocessor
    whisperModel *WhisperModel
    vocabulary   *HEMAVocabulary  
    postProcessor *PostProcessor  // DELETE
    booster      *VocabularyBooster // DELETE
}

// New simple API
type ProcessingPipeline struct {
    whisperModel *WhisperModel
    vocabulary   *HEMAVocabulary  // Only for initial prompt
    qualityFilter *QualityFilter  // NEW
}
```

No backward compatibility needed - just replace the old code entirely.

## Expected Improvements

1. **Reduced False Positives**: 60-80% reduction in blank/noise transcriptions
2. **Better HEMA Recognition**: 30-40% improvement in HEMA term accuracy
3. **Faster Processing**: Skip low-quality segments before Whisper
4. **Better Debugging**: Save problematic segments for analysis
5. **More Stable VAD**: Fewer triggers on ambient noise
6. **Cleaner Codebase**: ~750 fewer lines of code, no import cycles

## Configuration Checklist

- [ ] Update `config/pipeline.yaml` with new VAD parameters
- [ ] Set appropriate energy and quality thresholds based on environment
- [ ] Configure initial prompt with high-value HEMA terms
- [ ] Enable debug mode during testing to collect problem segments
- [ ] Tune suppress_regex to filter common non-speech patterns
- [ ] Adjust min_speech_duration_ms based on typical judge call length

## Monitoring and Metrics

Add metrics to track improvement:
- VAD trigger rate (should decrease)
- Blank transcription rate (should decrease)
- HEMA term recognition rate (should increase)
- Average confidence scores (should increase)
- Processing time per segment (may slightly increase due to filtering)

## Rollback Plan

If issues arise:
1. Disable quality filtering by setting thresholds to 0
2. Remove initial prompt to use default Whisper behavior
3. Disable suppress_blank and suppress_non_speech
4. Revert VAD parameters to original values

## Next Steps After Implementation

1. Collect debug audio segments from real tournament
2. Analyze false positive patterns
3. Fine-tune thresholds based on real data
4. Consider training custom acoustic model if needed
5. Potentially implement adaptive thresholds based on ambient noise level