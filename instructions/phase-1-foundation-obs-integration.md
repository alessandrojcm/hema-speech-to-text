# Phase 1: Foundation & OBS Integration

## Overview
This phase establishes the foundational architecture for the HEMA Tournament Replay System and implements core OBS Studio integration capabilities. The MVP for this phase is a manual replay system that can save replay buffers and display customizable text overlays.

## Technical Requirements

### Go Project Structure
- Initialize Go module with appropriate module name
- Create standard Go project directory structure:
  - `cmd/` - Application entry points
  - `pkg/` - Reusable package code
  - `internal/` - Private application code
  - `config/` - Configuration files
  - `assets/` - Static assets and test files

### Required Dependencies
- OBS WebSocket Go client library (research and select appropriate library)
- Configuration management library (YAML support)
- Logging framework with structured logging capabilities
- Testing framework and assertion libraries

### Core Components to Implement

#### 1. Configuration Management
- Design configuration structure for OBS connection parameters
- Implement configuration loading from YAML files
- Support for environment variable overrides
- Validation of configuration values
- Default configuration values for development

#### 2. OBS WebSocket Client
- Create abstraction layer for OBS WebSocket communication
- Implement connection management with auto-reconnection
- Handle authentication if WebSocket password is configured
- Implement graceful connection handling and error recovery
- Create wrapper functions for common OBS operations

#### 3. Replay Buffer Management
- Design replay buffer configuration structure
- Implement replay buffer start/stop functionality
- Create replay saving mechanism with configurable pre-roll
- Handle replay buffer state management
- Implement queue system to prevent overlapping replays

#### 4. Text Overlay System
- Design text overlay configuration structure
- Implement text source creation and management
- Create text update functionality with formatting options
- Handle text visibility control
- Support for multiple text sources if needed

#### 5. Scene Management
- Implement scene switching capabilities
- Create replay scene management
- Handle scene state transitions
- Implement automatic scene return functionality
- Error handling for scene operations

## Implementation Steps

### Step 1: Project Foundation
1. Initialize Go module and create directory structure
2. Set up basic main.go with command-line argument handling
3. Implement configuration loading system
4. Set up structured logging throughout the application
5. Create basic error handling patterns

### Step 2: OBS WebSocket Integration
1. Research and integrate OBS WebSocket Go library
2. Create OBS client wrapper with connection management
3. Implement basic connection testing (GetVersion, GetSceneList)
4. Add authentication handling for secured OBS instances
5. Implement graceful shutdown and cleanup

### Step 3: Replay Buffer Core
1. Design replay buffer configuration schema
2. Implement replay buffer enable/disable functionality
3. Create save replay functionality with error handling
4. Add configurable pre-roll timing
5. Implement basic replay queue to prevent conflicts

### Step 4: Text Overlay Implementation
1. Design text overlay configuration schema
2. Implement text source discovery and management
3. Create text update functions with formatting support
4. Add text visibility control
5. Handle text source creation if not exists

### Step 5: Scene Management
1. Implement scene discovery and validation
2. Create scene switching functionality
3. Add replay scene management
4. Implement automatic scene return with timing
5. Handle scene operation errors gracefully

### Step 6: Manual Trigger System
1. Create manual trigger interface (CLI commands or hotkeys)
2. Implement end-to-end manual replay flow
3. Add customizable text message input
4. Create trigger queuing system
5. Add comprehensive error handling and logging

## Testing Requirements

### Unit Tests
- Configuration loading and validation
- OBS client connection and operation wrappers
- Replay buffer state management
- Text overlay functionality
- Scene management operations

### Integration Tests
- OBS WebSocket connection and authentication
- Replay buffer operations with actual OBS instance
- Text overlay updates with live OBS
- Scene switching and return functionality
- End-to-end manual replay flow

### Manual Testing Scenarios
1. Connect to OBS Studio with various authentication scenarios
2. Test replay buffer enable/disable cycles
3. Verify replay saving with different pre-roll settings
4. Test text overlay updates with various formatting
5. Verify scene switching and automatic return
6. Test rapid manual triggers to validate queuing
7. Test error scenarios (OBS disconnection, invalid scenes, etc.)

## Configuration Schema

### OBS Connection Configuration
- WebSocket host and port settings
- Authentication credentials (optional)
- Connection timeout and retry settings
- Reconnection parameters

### Replay Buffer Configuration
- Buffer duration settings
- Pre-roll timing configuration
- Replay duration settings
- Minimum interval between replays

### Text Overlay Configuration
- Text source names and settings
- Font and formatting options
- Position and visibility settings
- Default text messages

### Scene Configuration
- Main scene and replay scene names
- Scene switching timing
- Fallback scene handling

## Success Criteria

### MVP Acceptance Criteria
1. Successfully connect to OBS Studio via WebSocket
2. Enable and manage replay buffer functionality
3. Save replays with configurable pre-roll timing
4. Update text overlays with custom messages
5. Switch to replay scene and return automatically
6. Handle multiple manual triggers without conflicts
7. Provide comprehensive error handling and logging

### Performance Targets
- OBS connection establishment: < 2 seconds
- Replay buffer save operation: < 1 second
- Text overlay update: < 500ms
- Scene switching: < 500ms
- Manual trigger response: < 100ms

## Error Handling Requirements
- Graceful handling of OBS disconnection
- Recovery from invalid scene references
- Fallback behavior for missing text sources
- Timeout handling for WebSocket operations
- Comprehensive logging for debugging

## Dependencies for Next Phase
This phase provides the foundation for all subsequent phases:
- OBS client abstraction for replay management
- Configuration system for audio and processing settings
- Logging framework for system monitoring
- Basic project structure for additional components

## Notes for Implementation Agent
- Focus on creating robust, testable abstractions
- Prioritize error handling and graceful degradation
- Implement comprehensive logging for debugging
- Design interfaces that can be easily extended
- Consider thread safety for concurrent operations
- Create clear separation between OBS operations and business logic
- Implement proper resource cleanup and connection management