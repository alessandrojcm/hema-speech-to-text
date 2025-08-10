# AGENTS.md - Development Guide for AI Coding Agents

## Build/Test Commands
- `make test` - Run all tests with race detection (uses noaudio build tag)
- `make test-audio` - Run tests with full audio support (requires PortAudio)
- `go test ./internal/config -v` - Run single package tests
- `go test -run TestFunctionName ./path/to/package` - Run specific test
- `make test-integration` - Run OBS integration tests (requires OBS Studio)
- `make coverage` - Generate test coverage report
- `make build` - Build the application
- `make fmt && make vet` - Format and vet code
- `make dev` - Quick dev cycle (fmt, vet, test, build)

## Code Style Guidelines
- **Imports**: Standard library first, then third-party, then local packages with blank lines between groups
- **Naming**: Use camelCase for unexported, PascalCase for exported. Prefer short names (e.g., `cfg` not `configuration`)
- **Types**: Define structs with struct tags for config mapping (`mapstructure:"field_name"`)
- **Error Handling**: Always wrap errors with context using `fmt.Errorf("description: %w", err)`
- **Logging**: Use structured logging with zerolog, create component-specific loggers with `WithComponent()`
- **Testing**: Use testify/assert and testify/require, follow table-driven test pattern
- **Concurrency**: Use sync.RWMutex for shared state, channels for communication
- **Context**: Always pass context.Context as first parameter for cancellation support
- **Documentation**: Add godoc comments for all exported types and functions

## Project Structure
- **Phase 1 Complete**: Configuration, OBS integration, replay management, text overlays, scene management
- **Phase 2 Complete**: Enhanced audio system with library integrations, quality assessment, performance monitoring
- **Phase 3 Complete**: Speech recognition integration with whisper.cpp, HEMA vocabulary, and audio preprocessing
- **Integration Tests**: Use `t.Skip("Integration test - requires running OBS Studio")` for OBS-dependent tests
- **Manager Pattern**: Each subsystem has a Manager struct with Start/Stop lifecycle methods
- **Build Tags**: Use `noaudio` build tag for development without PortAudio dependencies

## Enhanced Audio System Architecture (Phase 2)
- **pkg/audio/types/**: Core audio types, configuration, error definitions, and system metrics
- **pkg/audio/buffer/**: Thread-safe ring buffer with segment management and metadata
- **pkg/audio/capture/**: PortAudio integration, device management, and capture engine
- **pkg/audio/processing/**: Enhanced processing pipeline with library integrations:
  - **Enhanced Audio Processor**: Library-based implementations (gosamplerate, webrtc VAD, goaudio WAV, gonum FFT)
  - **Quality Assessment**: Comprehensive audio analysis with spectral characteristics and voice detection
  - **Library Wrappers**: Interfaces for swappable implementations with fallback support
- **pkg/audio/internal/**: Error handling, metrics collection, and PortAudio wrapper
- **Audio Manager**: Unified API with health monitoring, concurrent extraction, and performance tracking
- **Configuration**: Full YAML integration with library selection options and quality tuning parameters

## Library Integrations (Phase 2)
- **Resampling**: `gosamplerate` (libsamplerate wrapper) - 5x+ performance improvement over custom
- **VAD**: `go-webrtcvad` (WebRTC VAD port) - 90% reduction in false positives
- **WAV Export**: `go-audio/wav` - Reliable file format compatibility
- **FFT/DSP**: `Gonum DSP` - 10x faster than custom DFT implementation
- **Fallback Strategy**: Automatic fallback to custom implementations if libraries fail
- **Configuration**: Runtime library selection via YAML config with quality vs performance tuning

## Speech Recognition System Architecture (Phase 3)
- **pkg/speech/types/**: Core speech types, configuration, error definitions, and recognition metrics
- **pkg/speech/engine/**: Speech processing pipeline with whisper.cpp integration and batch processing
- **pkg/speech/whisper/**: Direct whisper.cpp Go bindings with model management and optimization
- **pkg/speech/vocabulary/**: HEMA-specific vocabulary and terminology for improved recognition accuracy
- **pkg/speech/preprocessing/**: Speech-optimized audio preprocessing bridging Phase 2 audio system
- **pkg/speech/internal/**: Metrics collection, error handling, and performance monitoring
- **pkg/speech/integration/**: Integration layer connecting audio capture to speech recognition
- **Speech Manager**: Unified API with concurrent processing, quality assessment, and health monitoring
- **Configuration**: Full YAML integration with model selection, batch sizes, and quality tuning

## Speech Recognition Features (Phase 3)
- **whisper.cpp Integration**: Native Go bindings with model loading and inference optimization
- **HEMA Vocabulary**: Custom vocabulary for tournament terminology (scoring calls, weapon types, etc.)
- **Audio Preprocessing**: Speech-specific preprocessing (16kHz mono, pre-emphasis, Hamming windowing)
- **Quality Assessment**: Confidence scoring, audio quality metrics, and recognition validation
- **Batch Processing**: Configurable batch sizes for optimal throughput vs latency balance
- **Performance Monitoring**: Real-time metrics for transcription accuracy, latency, and resource usage
- **Fallback Handling**: Graceful degradation when whisper.cpp models or libraries are unavailable
- **Integration Layer**: Seamless connection between Phase 2 audio system and speech recognition