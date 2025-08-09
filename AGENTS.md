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