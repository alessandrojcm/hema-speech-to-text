# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a planned HEMA Tournament Replay & Commentary System project designed to automatically trigger instant replays with commentator-style text overlays during live-streamed HEMA tournaments, based on judges' verbal decisions.

## Current Status

**Phase 1 Scaffolding: COMPLETE** ✅

The project has completed the initial scaffolding phase with:
- Complete Go module setup (`github.com/your-org/hema-replay-system`)
- Full directory structure implementation
- All core dependencies installed and configured
- Comprehensive build system with Makefile
- Configuration management system
- Documentation and test infrastructure

**Phase 1 Step 1: COMPLETE** ✅

Step 1 foundation components have been implemented:
- Configuration System (`internal/config/config.go`) - YAML config loading with validation
- Logging System (`pkg/logger/logger.go`) - Structured logging with zerolog
- Main Application Entry Point (`cmd/replay-system/main.go`) - CLI and lifecycle management
- Comprehensive test coverage for all components

**Next**: Ready for Step 2 implementation (OBS WebSocket Integration)

The complete implementation plan is in `instructions/phase-1-implementation.md`.

## Planned Architecture

The system will be built in Go with the following key components:

### Core Structure (Phase 1 Step 1: IMPLEMENTED)
- `cmd/replay-system/main.go` - Main application entry point ✅
- `internal/config/` - Configuration loading and validation ✅
- `pkg/logger/` - Structured logging setup ✅
- `config/settings.yaml` - Configuration file ✅
- `config/settings.example.yaml` - Example configuration with documentation ✅

### Core Structure (Phase 1 Step 2+: PLANNED)
- `internal/obs/` - OBS WebSocket client wrapper
- `internal/replay/` - Replay buffer and queue management
- `internal/text/` - Text overlay management
- `internal/scene/` - Scene management

### Future Structure (Planned for Phase 2+)
- `pkg/audio/` - Audio capture and ring buffer management
- `pkg/whisper/` - whisper.cpp integration for speech-to-text
- `pkg/summarizer/` - Local LLM integration for caption generation
- `pkg/pipeline/` - Main processing pipeline orchestration
- `config/vocab.txt` - HEMA-specific vocabulary for speech recognition

### Key Technologies
- Go (v1.20+) as the primary language
- OBS Studio with WebSocket plugin for replay control
- whisper.cpp with Metal support for M4 chip acceleration
- ollama or llama.cpp for local LLM inference
- Virtual Audio Cable/BlackHole for audio routing
- portaudio-go for direct audio capture

## Development Commands

### Current Phase 1 Commands (Implemented)

```bash
# Development workflow
make deps            # Install dependencies
make build           # Build the application
make test            # Run tests
make coverage        # Run tests with coverage
make dev             # Quick development cycle (fmt, vet, test, build)
make run             # Build and run the application
make run-config      # Build and run with config file
make clean           # Clean build artifacts

# Code quality
make fmt             # Format code
make vet             # Vet code
make lint            # Run linting (if golint is installed)

# Utilities
make watch           # Watch for changes and rebuild (requires entr)
make help            # Show all available commands
```

### Future Phase 2+ Commands (Planned)

```bash
# Install whisper.cpp with Metal support
git clone https://github.com/ggerganov/whisper.cpp
cd whisper.cpp
make clean && WHISPER_METAL=1 make

# Install ollama for local LLM
curl -fsSL https://ollama.com/install.sh | sh
ollama pull mistral:7b-instruct

# Install audio tools (macOS)
brew install blackhole-2ch portaudio
```

## System Requirements

- macOS with MacBook Pro M4 (target platform)
- OBS Studio v28+
- Go v1.20+
- Metal support for whisper.cpp acceleration
- Audio routing software (BlackHole or Virtual Audio Cable)

## Performance Targets

- End-to-end latency: < 3 seconds (trigger to overlay)
- Transcription accuracy: > 85% for common HEMA calls
- System availability: 99.9% during tournament
- CPU usage: < 60% sustained on M4
- Fallback rate: < 10% of triggers

## Key Implementation Notes

- Uses continuous audio capture with ring buffer to eliminate file I/O latency
- Implements local-only processing (whisper.cpp + local LLM) for reliability
- Includes smart caching for common phrases to bypass LLM processing
- Features graceful degradation with circuit breaker pattern and fallbacks
- Implements replay queue to handle rapid scoring without conflicts
- Leverages Metal optimization for M4 chip capabilities

## File Contents

### Implementation Documentation
- `instructions/hema-replay-system-v2.md` - Complete implementation plan and architecture
- `instructions/phase-1-implementation.md` - Detailed Phase 1 implementation guide
- `instructions/phase-2-*.md` through `instructions/phase-6-*.md` - Future phase plans

### Project Structure
- `cmd/replay-system/` - Main application entry point ✅
- `internal/config/` - Configuration loading and validation ✅
- `internal/obs/` - OBS WebSocket client wrapper (planned)
- `internal/replay/` - Replay buffer and queue management (planned)
- `internal/text/` - Text overlay management (planned)
- `internal/scene/` - Scene management (planned)
- `pkg/logger/` - Shared logging package ✅
- `config/` - Configuration files (settings.yaml, settings.example.yaml) ✅
- `assets/test/` - Test assets and configuration
- `docs/` - API documentation and project docs
- `Makefile` - Build and development automation
- `README.md` - Project overview and usage instructions

### Media Files
- `*.mp4`, `*.wav`, `*.mp3` - Test audio/video files for development and testing

### Current Dependencies
- `github.com/andreykaipov/goobs` - OBS WebSocket client
- `github.com/spf13/viper` - Configuration management
- `github.com/spf13/cobra` - CLI framework
- `github.com/rs/zerolog` - Structured logging
- `github.com/stretchr/testify` - Testing framework

## Implementation Status

### ✅ Phase 1 Scaffolding (COMPLETE)
- Go module structure
- Directory organization
- Build system (Makefile)
- Configuration files
- Documentation structure
- Test infrastructure

### ✅ Phase 1 Step 1 Implementation (COMPLETE)
- Configuration system implementation ✅
- Logging framework setup ✅
- Main application entry point ✅
- Comprehensive test coverage ✅

### 🔄 Phase 1 Step 2+ Implementation (NEXT)
- OBS WebSocket client wrapper
- Replay buffer management
- Text overlay system
- Scene management

### 📋 Future Phases (PLANNED)
- **Phase 2**: Audio capture and ring buffer
- **Phase 3**: Speech recognition integration
- **Phase 4**: Commentary generation
- **Phase 5**: Automated pipeline integration
- **Phase 6**: Production optimization

## Next Steps

**Step 1 Complete**: Configuration System, Logging, and Main Application are implemented and tested.

**Step 2 Ready**: The project is ready for Step 2 implementation (OBS WebSocket Integration). Follow the detailed guide in `instructions/phase-1-implementation.md` starting with the OBS WebSocket Integration section.