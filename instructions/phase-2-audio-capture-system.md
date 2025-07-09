# Phase 2: Audio Capture System

## Overview
This phase implements a robust audio capture system with continuous recording capabilities and efficient ring buffer management. The MVP for this phase is a continuous audio capture system that can extract recent audio segments on demand for processing.

## Technical Requirements

### Dependencies from Phase 1
- Configuration management system
- Logging framework
- Project structure and error handling patterns

### New Dependencies
- Audio processing library (PortAudio Go bindings or similar)
- Audio format conversion libraries
- Virtual audio device management (BlackHole for macOS)
- Ring buffer implementation or data structures library

### Core Components to Implement

#### 1. Audio Device Management
- Audio device discovery and enumeration
- Device selection and configuration
- Audio format specification and validation
- Device state monitoring and error handling
- Support for virtual audio devices

#### 2. Ring Buffer Implementation
- Circular buffer for continuous audio storage
- Thread-safe read/write operations
- Configurable buffer size and duration
- Automatic overwrite of old data
- Efficient memory management

#### 3. Audio Capture Engine
- Continuous audio capture from selected device
- Real-time audio processing and storage
- Sample rate conversion and format handling
- Audio level monitoring and validation
- Performance optimization for low latency

#### 4. Audio Extraction System
- Extract audio segments from ring buffer
- Configurable extraction duration (last N seconds)
- Audio format conversion for downstream processing
- Timestamp tracking for extracted segments
- Concurrent extraction support

#### 5. Audio Processing Pipeline
- Audio preprocessing (normalization, filtering)
- Voice Activity Detection (VAD) preparation
- Audio quality assessment
- Noise reduction capabilities
- Format conversion for speech recognition

## Implementation Steps

### Step 1: Audio Device Infrastructure
1. Integrate audio library (PortAudio or equivalent)
2. Implement audio device discovery and enumeration
3. Create device selection and configuration system
4. Add audio format validation and conversion
5. Implement device state monitoring

### Step 2: Ring Buffer Core
1. Design ring buffer data structure
2. Implement thread-safe circular buffer operations
3. Add configurable buffer size management
4. Create automatic overwrite mechanism
5. Implement buffer state monitoring

### Step 3: Audio Capture Engine
1. Implement continuous audio capture loop
2. Add real-time audio processing
3. Integrate ring buffer storage
4. Create audio level monitoring
5. Add performance optimization

### Step 4: Audio Extraction System
1. Design extraction API and interface
2. Implement segment extraction from ring buffer
3. Add timestamp tracking and metadata
4. Create format conversion pipeline
5. Add concurrent extraction support

### Step 5: Audio Processing Pipeline
1. Implement audio preprocessing functions
2. Add Voice Activity Detection foundation
3. Create audio quality assessment
4. Add noise reduction capabilities
5. Implement format conversion for speech recognition

### Step 6: Integration and Testing
1. Integrate with Phase 1 configuration system
2. Add comprehensive error handling
3. Create audio capture testing utilities
4. Implement performance monitoring
5. Add graceful shutdown and cleanup

## Audio Configuration Schema

### Device Configuration
- Target audio device selection (name or ID)
- Fallback device options
- Audio format preferences (sample rate, channels, bit depth)
- Buffer size and latency settings
- Device monitoring and failover settings

### Ring Buffer Configuration
- Buffer duration (seconds of audio to store)
- Sample rate and format specifications
- Memory allocation strategy
- Overwrite policies and thresholds
- Performance optimization settings

### Extraction Configuration
- Default extraction duration
- Supported output formats
- Concurrent extraction limits
- Timestamp precision requirements
- Quality assessment thresholds

### Processing Configuration
- Audio preprocessing options
- Voice Activity Detection parameters
- Noise reduction settings
- Format conversion options
- Performance optimization flags

## Testing Requirements

### Unit Tests
- Audio device discovery and selection
- Ring buffer operations (read/write/overwrite)
- Audio format conversion functions
- Extraction logic and timestamp handling
- Audio processing pipeline functions

### Integration Tests
- Continuous audio capture with ring buffer
- Real-time audio extraction during capture
- Device switching and failover scenarios
- Memory usage under continuous operation
- Performance under high extraction frequency

### Manual Testing Scenarios
1. Test with various audio devices (microphone, line-in, virtual devices)
2. Verify continuous capture with different buffer sizes
3. Test extraction of segments while capture is active
4. Validate audio quality and format conversion
5. Test device disconnection and reconnection scenarios
6. Verify memory usage stays within bounds during long runs
7. Test concurrent extractions and performance impact

## Performance Requirements

### Capture Performance
- Continuous capture with minimal CPU usage
- Real-time processing without audio dropouts
- Memory usage proportional to buffer size
- Low latency for extraction requests
- Stable operation over extended periods

### Extraction Performance
- Extract 10-second segments in < 100ms
- Support concurrent extractions (up to 5 simultaneous)
- Minimal impact on ongoing capture
- Efficient format conversion
- Timestamp accuracy within 10ms

### Memory Management
- Ring buffer size configurable (default 60 seconds)
- Automatic cleanup of old audio data
- Memory usage should not grow over time
- Efficient memory allocation patterns
- Proper resource cleanup on shutdown

## Error Handling Requirements

### Audio Device Errors
- Handle device disconnection gracefully
- Automatic device failover when available
- Clear error reporting for device issues
- Recovery from temporary device unavailability
- Fallback to alternative devices

### Capture Errors
- Handle audio buffer overruns/underruns
- Manage system resource limitations
- Recovery from temporary audio interruptions
- Graceful degradation under resource pressure
- Comprehensive error logging

### Extraction Errors
- Handle invalid extraction requests
- Manage concurrent extraction limits
- Recovery from format conversion failures
- Timeout handling for long operations
- Clear error reporting to callers

## Integration Points

### Phase 1 Integration
- Use existing configuration management for audio settings
- Integrate with logging framework for audio events
- Follow established error handling patterns
- Use project structure for audio packages

### Phase 3 Preparation
- Design audio output format compatible with whisper.cpp
- Implement extraction API suitable for speech recognition
- Create audio quality metrics for transcription optimization
- Prepare audio preprocessing for speech recognition

## Success Criteria

### MVP Acceptance Criteria
1. Continuously capture audio from selected device
2. Store audio in ring buffer with configurable duration
3. Extract audio segments on demand (last N seconds)
4. Convert audio formats for downstream processing
5. Handle device errors and failover gracefully
6. Maintain stable operation over extended periods
7. Provide audio quality metrics and monitoring

### Performance Targets
- Continuous capture: < 5% CPU usage
- Audio extraction: < 100ms for 10-second segments
- Memory usage: Stable at configured buffer size
- Audio quality: No dropouts or artifacts
- Latency: < 50ms from capture to ring buffer

## API Design Guidelines

### Audio Capture Interface
Design clean interfaces for:
- Starting/stopping continuous capture
- Configuring audio parameters
- Monitoring capture health
- Handling device changes

### Extraction Interface
Create intuitive APIs for:
- Requesting audio segments by duration
- Specifying output format requirements
- Handling concurrent extraction requests
- Providing extraction metadata

## Notes for Implementation Agent
- Focus on thread safety for concurrent operations
- Implement proper resource management and cleanup
- Design for extensibility (different audio sources, formats)
- Consider macOS-specific audio requirements
- Implement comprehensive monitoring and debugging
- Create clear abstractions for audio operations
- Plan for integration with speech recognition systems
- Consider real-time performance requirements throughout