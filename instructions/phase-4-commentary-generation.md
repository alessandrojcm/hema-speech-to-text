# Phase 4: Intelligent Commentary Generation

## Overview
This phase implements an intelligent commentary generation system that transforms raw judge call transcriptions into engaging, TV-style commentary using local LLM inference. The MVP for this phase is a system that generates contextually appropriate commentary with fast response times and intelligent fallback mechanisms.

## Technical Requirements

### Dependencies from Previous Phases
- Speech recognition transcription output (Phase 3)
- Configuration management and logging (Phase 1)
- Audio quality and confidence metrics

### New Dependencies
- Local LLM inference engine (ollama or llama.cpp)
- LLM model management system
- Caching infrastructure (in-memory and persistent)
- Template processing and formatting libraries
- Fallback text generation system

### Core Components to Implement

#### 1. Local LLM Integration
- ollama or llama.cpp client integration
- Model loading and management
- Inference request handling with timeouts
- Memory management for LLM operations
- Performance optimization for low latency

#### 2. Smart Caching System
- Multi-level caching architecture
- Cache key generation and validation
- TTL management and cache invalidation
- Persistent cache storage
- Cache hit rate optimization

#### 3. Prompt Engineering Framework
- Template-based prompt generation
- Context-aware prompt selection
- Dynamic prompt adaptation
- Prompt performance monitoring
- A/B testing infrastructure for prompts

#### 4. Commentary Generation Engine
- Text transformation pipeline
- Context analysis and enhancement
- Style consistency management
- Length and format control
- Quality assessment and validation

#### 5. Fallback System
- Rule-based fallback generation
- Keyword-based template matching
- Context-aware fallback selection
- Graceful degradation strategies
- Fallback quality assessment

## Implementation Steps

### Step 1: Local LLM Setup and Integration
1. Install and configure ollama with appropriate models
2. Create Go client for LLM communication
3. Implement model loading and management
4. Add timeout and error handling
5. Create performance monitoring

### Step 2: Caching Infrastructure
1. Design multi-level caching architecture
2. Implement in-memory cache with LRU eviction
3. Add persistent cache storage
4. Create cache key generation system
5. Implement cache performance monitoring

### Step 3: Prompt Engineering System
1. Create template-based prompt framework
2. Implement context-aware prompt selection
3. Add dynamic prompt adaptation
4. Create prompt performance tracking
5. Implement A/B testing for prompt optimization

### Step 4: Commentary Generation Core
1. Implement text transformation pipeline
2. Add context analysis and enhancement
3. Create style consistency management
4. Implement length and format control
5. Add quality assessment and validation

### Step 5: Fallback System Implementation
1. Create rule-based fallback generation
2. Implement keyword-based template matching
3. Add context-aware fallback selection
4. Create graceful degradation strategies
5. Implement fallback quality assessment

### Step 6: Integration and Optimization
1. Integrate with Phase 3 transcription output
2. Add comprehensive error handling
3. Create commentary testing utilities
4. Implement performance optimization
5. Add monitoring and analytics

## Configuration Schema

### LLM Configuration
- Model selection and parameters
- Inference timeout settings
- Context window management
- Temperature and generation parameters
- Performance optimization flags

### Caching Configuration
- Cache size limits and policies
- TTL settings for different content types
- Persistent cache storage location
- Cache invalidation rules
- Performance monitoring thresholds

### Prompt Configuration
- Template library organization
- Context selection rules
- Dynamic adaptation parameters
- Performance tracking settings
- A/B testing configuration

### Commentary Configuration
- Style and tone settings
- Length constraints and preferences
- Quality thresholds and validation
- Fallback trigger conditions
- Output format specifications

## Prompt Templates

### Basic Commentary Templates
- Point scoring announcements
- Double hit explanations
- Halt and restart calls
- Technical violation descriptions
- Match progression updates

### Context-Aware Templates
- Weapon-specific commentary
- Tournament phase considerations
- Fencer performance context
- Historical match context
- Crowd engagement elements

### Fallback Templates
- Generic positive commentary
- Neutral match descriptions
- Safety-focused messaging
- Technical explanations
- Audience engagement content

## Testing Requirements

### Unit Tests
- LLM client integration functions
- Caching system operations
- Prompt template processing
- Commentary generation pipeline
- Fallback system logic

### Integration Tests
- End-to-end commentary generation
- Cache performance and hit rates
- LLM timeout and error handling
- Fallback system activation
- Integration with Phase 3 transcription

### Manual Testing Scenarios
1. Test with various judge call transcriptions
2. Verify commentary quality and appropriateness
3. Test cache hit rates and performance
4. Validate fallback system activation
5. Test LLM timeout and error scenarios
6. Verify style consistency across outputs
7. Test with edge cases and unusual transcriptions

## Performance Requirements

### Generation Performance
- Commentary generation: < 2 seconds for standard calls
- Cache hit response: < 50ms
- LLM inference: < 3 seconds including timeout
- Fallback generation: < 100ms
- Concurrent generation: Support 3-5 simultaneous requests

### Quality Requirements
- Commentary appropriateness: > 95% for cached responses
- LLM generation quality: > 90% subjective rating
- Fallback quality: > 80% appropriateness
- Style consistency: > 95% across similar inputs
- Length compliance: 100% within specified limits

### Resource Management
- Memory usage: < 500MB for LLM client
- Cache memory: Configurable with LRU eviction
- CPU usage: < 30% during generation
- Disk usage: Efficient persistent cache storage
- Network usage: Minimal for local LLM inference

## Error Handling Requirements

### LLM Integration Errors
- Handle model loading failures
- Manage inference timeouts
- Recovery from generation errors
- Graceful handling of invalid responses
- Fallback activation on LLM failures

### Caching Errors
- Handle cache corruption
- Manage cache size limits
- Recovery from cache failures
- Fallback to generation on cache misses
- Cache invalidation error handling

### Generation Errors
- Handle prompt processing failures
- Manage template loading errors
- Recovery from formatting issues
- Validation of generated content
- Fallback activation on generation failures

## Caching Strategy

### Multi-Level Cache Design
- L1: In-memory cache for frequent responses
- L2: Persistent cache for common variations
- L3: Template-based cache for patterns
- Cache warming strategies
- Intelligent cache preloading

### Cache Key Generation
- Transcription content normalization
- Context-aware key generation
- Fuzzy matching for similar inputs
- Cache invalidation triggers
- Performance optimization

### Cache Management
- LRU eviction policies
- TTL-based invalidation
- Cache size monitoring
- Performance metrics tracking
- Automatic cache optimization

## Fallback System Design

### Fallback Triggers
- LLM timeout conditions
- Low confidence generation
- Invalid LLM responses
- System resource constraints
- Manual fallback activation

### Fallback Strategies
- Keyword-based template matching
- Context-aware selection
- Rule-based generation
- Static response library
- Graceful degradation paths

### Fallback Quality
- Appropriateness validation
- Context relevance scoring
- Style consistency checking
- Length compliance verification
- User preference alignment

## Integration Points

### Phase 3 Integration
- Use transcription output as input
- Leverage confidence scores for quality control
- Integrate with audio quality metrics
- Use context information for enhancement

### Phase 1 Integration
- Use text overlay system for commentary display
- Integrate with OBS scene management
- Leverage configuration management
- Use logging framework for monitoring

## Success Criteria

### MVP Acceptance Criteria
1. Generate appropriate commentary from judge call transcriptions
2. Maintain fast response times with caching
3. Provide intelligent fallback when LLM fails
4. Ensure style consistency across outputs
5. Handle various types of judge calls appropriately
6. Integrate seamlessly with transcription input
7. Provide comprehensive error handling and recovery

### Performance Targets
- Commentary generation: < 2 seconds
- Cache hit response: < 50ms
- Fallback generation: < 100ms
- Quality rating: > 90% for generated content
- Cache hit rate: > 70% for common calls

## API Design Guidelines

### Commentary Interface
Design clean interfaces for:
- Processing transcription input
- Configuring generation parameters
- Handling quality thresholds
- Managing cache operations

### Fallback Interface
Create intuitive APIs for:
- Fallback strategy selection
- Template management
- Quality assessment
- Performance monitoring

## Advanced Features

### Context Enhancement
- Match state awareness
- Fencer performance tracking
- Historical context integration
- Crowd engagement optimization

### Quality Optimization
- Automatic prompt tuning
- Generation quality scoring
- Style consistency enforcement
- User preference learning

## Notes for Implementation Agent
- Focus on local LLM performance optimization
- Implement comprehensive caching for production use
- Design robust fallback mechanisms
- Create thorough testing with actual judge calls
- Consider commentary quality and appropriateness
- Plan for prompt optimization and tuning
- Implement detailed performance monitoring
- Create clear abstractions for future enhancements
- Consider real-time performance requirements
- Test thoroughly with edge cases and unusual inputs