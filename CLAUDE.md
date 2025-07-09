# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a planned HEMA Tournament Replay & Commentary System project designed to automatically trigger instant replays with commentator-style text overlays during live-streamed HEMA tournaments, based on judges' verbal decisions.

## Current Status

The repository currently contains planning documentation and media files but no implemented code yet. The main documentation is in `hema-replay-system-v2.md` which outlines the complete implementation plan.

## Planned Architecture

The system will be built in Go with the following key components:

### Core Structure (Planned)
- `cmd/main.go` - Main application entry point
- `pkg/obsclient/` - OBS WebSocket client integration
- `pkg/audio/` - Audio capture and ring buffer management
- `pkg/whisper/` - whisper.cpp integration for speech-to-text
- `pkg/summarizer/` - Local LLM integration for caption generation
- `pkg/replay/` - Replay queue and management system
- `pkg/pipeline/` - Main processing pipeline orchestration
- `config/settings.yaml` - Configuration file
- `config/vocab.txt` - HEMA-specific vocabulary for speech recognition

### Key Technologies
- Go (v1.20+) as the primary language
- OBS Studio with WebSocket plugin for replay control
- whisper.cpp with Metal support for M4 chip acceleration
- ollama or llama.cpp for local LLM inference
- Virtual Audio Cable/BlackHole for audio routing
- portaudio-go for direct audio capture

## Development Commands

Based on the planning document, these commands will be relevant once implementation begins:

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

# Run the system (once implemented)
go run cmd/main.go -config config/settings.yaml
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

- `hema-replay-system-v2.md` - Complete implementation plan and architecture
- `*.mp4`, `*.wav`, `*.mp3` - Test audio/video files for development