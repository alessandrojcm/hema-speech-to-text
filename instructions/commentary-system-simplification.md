# Commentary System Simplification Plan

## Overview

The current commentary generation system is vastly over-complicated with multiple layers of templates, fallbacks, dynamic prompt building, and complex context management. This document outlines a plan to **completely replace** the current approach with a simplified system.

## Current System Problems

The existing system includes:
- **Complex Template System** (`templates/` package) - Multiple HEMA-specific templates with dynamic selection
- **Fallback Generator** (`engine/fallback.go`) - Rule-based fallback with pattern matching  
- **Dynamic Prompt Builder** (`prompt/builder.go`) - Context enrichment, template selection, complex data structures
- **Quality Validator** (`engine/validator.go`) - Multiple validation layers with scoring
- **Context Manager** (`context/manager.go`) - Match state tracking, call history, enrichment
- **Integration Manager** (`integration/manager.go`) - Processing pipeline with filtering

## New Simplified Architecture

**Goal**: `Transcription → Simple Static Prompt → LLM → Basic Validation → Accept or Discard`

### Core Principles
1. **No fallbacks** - If LLM can't generate good commentary, discard it
2. **No templates** - Use single static prompt format
3. **No dynamic prompts** - Same prompt structure for all inputs
4. **Minimal validation** - Basic checks only
5. **Faithful translations only** - Focus on explaining judge calls, not generating commentary

### Simplified Flow
```
Speech Transcription 
  ↓ (confidence > threshold)
Simple Static Prompt ("Explain this HEMA judge call briefly: {transcription}")
  ↓  
LLM Generation
  ↓
Basic Validation (non-empty, contains HEMA terms, reasonable length)
  ↓
Accept → Display Commentary  OR  Reject → No Output (silent discard)
```

## Implementation Plan

### Phase 1: Create Simple Prompt System
1. **Replace** `prompt/builder.go` with simple static prompt function
2. **Remove** template dependency
3. **Use static prompt**: `"Explain this HEMA judge call briefly: {transcription}"`

### Phase 2: Simplify Generator  
1. **Remove** fallback generator dependency from `engine/generator.go`
2. **Remove** prompt builder complexity - use simple static prompt
3. **Keep** basic LLM integration but remove retry/fallback logic
4. **Policy**: If LLM fails or returns poor quality → discard (no fallback)

### Phase 3: Simplify Validator
1. **Keep** only basic checks in `engine/validator.go`:
   - Non-empty output
   - Basic relevance (contains HEMA keywords)
   - Length within bounds (20-200 characters)
2. **Remove** complex scoring, profanity filters, pattern matching, coherence analysis

### Phase 4: Simplify Integration
1. **Remove** context enrichment from `integration/manager.go`  
2. **Remove** duplicate filtering, match state tracking
3. **Keep** basic confidence threshold filtering
4. **Direct path**: transcription → simple prompt → LLM → basic validation → accept/discard

### Phase 5: Clean Up Types
1. **Simplify** `types/commentary.go` - remove template/fallback metadata
2. **Remove** unused context types
3. **Keep** basic Commentary struct with minimal fields:
   - ID, Text, Source, Confidence, Timestamp
   - Remove: TemplateID, PromptUsed, ValidationPassed, complex Metadata

### Phase 6: Remove Unused Packages
1. **Delete** entire `templates/` package  
2. **Delete** entire `context/` package
3. **Delete** `engine/fallback.go`
4. **Delete** `prompt/builder.go` and `prompt/builder_test.go`
5. **Update** imports across the codebase

## Files to Change/Remove

### Files to REMOVE entirely:
- `engine/fallback.go` - Complex fallback system
- `prompt/builder.go` + `prompt/builder_test.go` - Dynamic prompt building
- `templates/` (entire package) - Template system
- `context/manager.go` + `context/manager_test.go` - Complex context management

### Files to SIMPLIFY:
- `engine/generator.go` - Remove fallback logic, template selection, complex validation
- `engine/validator.go` - Keep only basic validation (empty text, basic relevance)
- `integration/manager.go` - Remove context enrichment, filtering, complex processing
- `integration/end_to_end_test.go` - Update tests for simplified system
- `types/commentary.go` - Simplify types, remove template/fallback metadata

## Expected Benefits

1. **Drastically reduced complexity** - from ~15 files to ~5 files
2. **Faster processing** - no template selection, context enrichment, fallback chains
3. **Easier maintenance** - simple, predictable behavior
4. **Better reliability** - fewer failure points, clearer success/failure modes
5. **Clearer intent** - focus on faithful judge call explanations only

## Static Prompt Design

**Single prompt template**:
```
"You are a HEMA (Historical European Martial Arts) expert. Briefly explain what this judge call means in 1-2 sentences: '{transcription}'"
```

**Validation criteria**:
- Output length: 20-200 characters
- Contains at least one HEMA-related term
- Is not empty or whitespace-only

## Migration Strategy

1. **No compatibility mode** - complete replacement of current system
2. **Update all integration points** to use simplified API
3. **Remove all template/fallback configuration options**
4. **Simplify configuration to minimal required settings**

## Success Criteria

- ✅ No template system
- ✅ No fallback mechanisms  
- ✅ No dynamic prompt building
- ✅ No complex context management
- ✅ Single static prompt for all inputs
- ✅ Simple accept/discard based on basic validation
- ✅ Faithful judge call explanations only
- ✅ Silent failure (no output) when LLM cannot provide good explanation