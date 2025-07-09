# HEMA Tournament Replay System

A real-time replay system for Historical European Martial Arts (HEMA) tournaments that automatically triggers instant replays with commentator-style text overlays during live-streamed events.

## Project Status

**Phase 1: Foundation & OBS Integration** - In Development

This project is currently in Phase 1 implementation, focusing on:
- ✅ Project scaffolding and structure
- ⏳ OBS WebSocket integration
- ⏳ Replay buffer management
- ⏳ Text overlay system
- ⏳ Scene management

## Quick Start

### Prerequisites

- Go 1.20 or later
- OBS Studio 28+ with WebSocket plugin enabled
- macOS (primary target platform)

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

2. Edit `config/settings.yaml` to match your OBS setup:
   ```yaml
   obs:
     host: "localhost"
     port: 4455
     password: ""  # Set if you have a password configured
   ```

3. Configure OBS Studio:
   - Enable WebSocket server in OBS (Tools → WebSocket Server Settings)
   - Set up your scenes: "Main" and "Replay"
   - Add a text source named "ReplayText"

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

### Current Implementation (Phase 1)

```
├── cmd/replay-system/     # Application entry point
├── internal/
│   ├── config/           # Configuration management
│   ├── obs/              # OBS WebSocket client
│   ├── replay/           # Replay buffer and queue management
│   ├── text/             # Text overlay management
│   └── scene/            # Scene management
├── pkg/logger/           # Structured logging
└── config/               # Configuration files
```

### Planned Features (Future Phases)

- **Phase 2**: Audio capture and ring buffer
- **Phase 3**: Speech recognition with whisper.cpp
- **Phase 4**: LLM-powered commentary generation
- **Phase 5**: Automated pipeline integration

## Development

### Available Make Targets

```bash
make help           # Show all available targets
make dev            # Quick development cycle (fmt, vet, test, build)
make test           # Run tests
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
├── pkg/logger/                 # Shared logging
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
make test                    # Run all tests
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

- **OBS Connection**: < 2 seconds
- **Replay Save**: < 1 second  
- **Text Update**: < 500ms
- **Scene Switch**: < 500ms
- **Manual Trigger**: < 100ms

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
- Uses [viper](https://github.com/spf13/viper) for configuration management
- Structured logging with [zerolog](https://github.com/rs/zerolog)
- Testing with [testify](https://github.com/stretchr/testify)