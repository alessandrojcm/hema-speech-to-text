# HEMA Tournament Replay System

A real-time replay system for Historical European Martial Arts (HEMA) tournaments that automatically triggers instant replays with commentator-style text overlays during live-streamed events.

## Project Status

**Phase 1: Foundation & OBS Integration** - ✅ **COMPLETE**
**Phase 2: Audio Capture System** - ✅ **COMPLETE**

### Completed Features:
- ✅ Project scaffolding and structure
- ✅ OBS WebSocket integration with real-time communication
- ✅ Replay buffer management with queue system
- ✅ Text overlay system with formatting and validation
- ✅ Scene management with automatic switching
- ✅ Audio capture system with PortAudio integration
- ✅ Ring buffer for continuous audio storage (60-second capacity)
- ✅ Audio processing pipeline with filters and quality assessment
- ✅ Device management with fallback support
- ✅ Performance monitoring and health tracking
- ✅ Error handling with circuit breakers and retry policies

### Next Phase:
- **Phase 3**: Speech recognition integration with whisper.cpp

## Quick Start

### Prerequisites

- Go 1.20 or later
- OBS Studio 28+ with WebSocket plugin enabled
- macOS (primary target platform)
- PortAudio (for audio capture - optional for basic functionality)

### Installation

```bash
# Clone the repository
git clone https://github.com/your-org/hema-replay-system
cd hema-replay-system

# Install dependencies
make deps

# Build the application
make build

# Run tests
make test
```

### Configuration

1. Copy the example configuration:
   ```bash
   cp config/settings.example.yaml config/settings.yaml
   ```

2. Edit `config/settings.yaml` to match your setup:
   ```yaml
   obs:
     host: "localhost"
     port: 4455
     password: ""  # Set if you have a password configured
   
   audio:
     device:
       name: "BlackHole 2ch"  # Audio capture device
       sample_rate: 44100
       channels: 2
     buffer:
       duration: "60s"        # Ring buffer duration
       segment_size: "1s"     # Segment size for processing
   ```

3. Configure OBS Studio:
   - Enable WebSocket server in OBS (Tools → WebSocket Server Settings)
   - Set up your scenes: "Main" and "Replay"
   - Add a text source named "ReplayText"

4. Configure Audio (Optional):
   - Install PortAudio: `brew install portaudio` (macOS)
   - Set up audio routing (e.g., BlackHole for virtual audio)
   - Configure your audio device in the settings

### Usage

```bash
# Run with default configuration
make run

# Run with specific configuration file
make run-config

# Or run the binary directly
./bin/hema-replay-system -config config/settings.yaml
```

## Architecture

### Current Implementation (Phases 1 & 2)

```
├── cmd/replay-system/     # Application entry point
├── internal/
│   ├── config/           # Configuration management
│   ├── obs/              # OBS WebSocket client
│   ├── replay/           # Replay buffer and queue management
│   ├── text/             # Text overlay management
│   └── scene/            # Scene management
├── pkg/
│   ├── audio/            # Audio capture system (Phase 2)
│   │   ├── buffer/       # Ring buffer with segment management
│   │   ├── capture/      # PortAudio integration & device management
│   │   ├── processing/   # Audio processing pipeline
│   │   ├── types/        # Audio types and configuration
│   │   └── internal/     # Error handling and metrics
│   └── logger/           # Structured logging
└── config/               # Configuration files
```

### Planned Features (Future Phases)

- **Phase 3**: Speech recognition with whisper.cpp
- **Phase 4**: LLM-powered commentary generation
- **Phase 5**: Automated pipeline integration
- **Phase 6**: Production optimization

## Development

### Available Make Targets

```bash
make help           # Show all available targets
make dev            # Quick development cycle (fmt, vet, test, build)
make test           # Run tests (uses noaudio build tag)
make test-audio     # Run tests with full audio support (requires PortAudio)
make coverage       # Run tests with coverage report
make watch          # Watch for changes and rebuild
make lint           # Run linting tools
```

### Project Structure

```
speech-to-text/
├── cmd/replay-system/          # Main application
├── internal/                   # Internal packages
│   ├── config/                # Configuration loading
│   ├── obs/                   # OBS integration
│   ├── replay/                # Replay management
│   ├── text/                  # Text overlays
│   └── scene/                 # Scene management
├── pkg/                        # Shared packages
│   ├── audio/                 # Audio capture system
│   │   ├── buffer/            # Ring buffer implementation
│   │   ├── capture/           # Audio capture and device management
│   │   ├── processing/        # Audio processing pipeline
│   │   ├── types/             # Audio types and configuration
│   │   └── internal/          # Error handling and metrics
│   └── logger/                # Structured logging
├── config/                     # Configuration files
├── assets/test/                # Test assets
├── docs/                       # Documentation
└── Makefile                    # Build automation
```

## Configuration

### Environment Variables

You can override configuration values using environment variables:

```bash
export HEMA_REPLAY_OBS_HOST=localhost
export HEMA_REPLAY_OBS_PORT=4455
export HEMA_REPLAY_OBS_PASSWORD=your_password
export HEMA_REPLAY_LOGGING_LEVEL=debug
```

### Configuration File

See `config/settings.example.yaml` for detailed configuration options.

## Testing

### Unit Tests

```bash
make test                    # Run all tests (uses noaudio build tag)
make test-audio             # Run tests with full audio support
make coverage               # Run tests with coverage
```

### Integration Tests

Integration tests require OBS Studio to be running:

```bash
# Start OBS Studio with WebSocket enabled
# Then run integration tests
go test -tags=integration ./...
```

## Performance Targets

### OBS Integration
- **OBS Connection**: < 2 seconds
- **Replay Save**: < 1 second  
- **Text Update**: < 500ms
- **Scene Switch**: < 500ms
- **Manual Trigger**: < 100ms

### Audio System
- **Audio Capture Latency**: < 50ms
- **Ring Buffer Write**: < 10ms
- **Audio Extraction**: < 100ms
- **Processing Pipeline**: < 20ms per segment
- **Device Failover**: < 2 seconds

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the test suite: `make test`
6. Submit a pull request

### Code Style

- Follow Go best practices
- Use `make fmt` to format code
- Run `make vet` and `make lint` before submitting
- Write comprehensive tests for new features

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support and questions:
- Open an issue on GitHub
- Check the documentation in the `docs/` directory
- Review the example configuration files

## Acknowledgments

- Built with [goobs](https://github.com/andreykaipov/goobs) for OBS WebSocket integration
- Audio capture with [PortAudio Go bindings](https://github.com/gordonklaus/portaudio)
- Uses [viper](https://github.com/spf13/viper) for configuration management
- Structured logging with [zerolog](https://github.com/rs/zerolog)
- Testing with [testify](https://github.com/stretchr/testify)