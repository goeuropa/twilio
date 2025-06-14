# Voice Feature Enhancement: Endpoint-Specific XML Templates

## Overview
Replace the current programmatic TwiML XML generation with endpoint-specific embedded XML template documents using Go's `embed` module. Each voice endpoint will have its own dedicated template that handles all possible response scenarios for that endpoint.

## Current State Analysis

### Code to be Removed
1. **Functions in `formatters/response.go`:**
   - `GenerateTwiMLVoice(message string) (string, error)` (lines 76-87)
   - `GenerateTwiMLGather(prompt string, action string, numDigits int) (string, error)` (lines 89-105)

2. **Models in `models/types.go`:**
   - `TwiMLResponse` struct (lines 98-103)
   - `Gather` struct (lines 105-110)

### Voice Endpoints and Their Response Scenarios

#### `/voice` endpoint (`HandleVoiceStart` function)
- **Success case**: Welcome message with gather for stop ID input
- **Error cases**: Invalid request format, TwiML generation failure

#### `/voice/find_stop` endpoint (`HandleFindStop` function)
- **Success cases**: 
  - Single stop found → arrival information
  - Multiple stops found → disambiguation menu
- **Error cases**:
  - Invalid request format
  - Empty digits received
  - Invalid stop ID format
  - OneBusAway API failure
  - No stops found
  - Disambiguation session failure
  - No active selection
  - Invalid choice range

## Implementation Plan

### Phase 1: Template Design and Creation
**Duration**: 3-4 hours

1. **Create template directory structure:**
   ```
   templates/
   ├── voice_start.xml
   ├── voice_find_stop.xml
   ├── voice_error.xml
   └── voice_disambiguation.xml
   ```

2. **Design specific-purpose templates:**
   - `voice_start.xml`: Welcome message with gather for stop ID input
   - `voice_find_stop.xml`: Success response with arrival information
   - `voice_error.xml`: Error response for any voice endpoint
   - `voice_disambiguation.xml`: Disambiguation menu with gather for user choice

3. **Template purposes:**
   - Each template has a single, specific purpose
   - Go code determines which template to render based on the response scenario
   - No conditional logic within templates - they are pure presentation

### Phase 2: Template Engine Implementation
**Duration**: 4-5 hours

1. **Create new template manager:**
   - New file: `formatters/voice_templates.go`
   - Struct: `VoiceTemplateManager` with embedded templates
   - Use `//go:embed` directive to embed template files

2. **Template functions to implement:**
   - `RenderVoiceStart(ctx VoiceStartContext) (string, error)`
   - `RenderVoiceFindStop(ctx VoiceFindStopContext) (string, error)`
   - `RenderVoiceError(ctx VoiceErrorContext) (string, error)`
   - `RenderVoiceDisambiguation(ctx VoiceDisambiguationContext) (string, error)`

3. **Template context structures:**
   ```go
   type VoiceStartContext struct {
       WelcomePrompt string
   }
   
   type VoiceFindStopContext struct {
       ArrivalsMessage string
   }
   
   type VoiceErrorContext struct {
       ErrorMessage string
   }
   
   type VoiceDisambiguationContext struct {
       DisambiguationPrompt string
   }
   ```

### Phase 3: Voice Handler Migration
**Duration**: 3-4 hours

1. **Update imports in `handlers/voice.go`:**
   - Remove dependencies on `models.TwiMLResponse` and `models.Gather`
   - Add dependency on new voice template manager

2. **Refactor `HandleVoiceStart` function:**
   - Replace TwiML generation with conditional template selection
   - On success: call `templateManager.RenderVoiceStart(ctx)`
   - On error: call `templateManager.RenderVoiceError(ctx)`

3. **Refactor `HandleFindStop` function:**
   - Replace TwiML generation with conditional template selection based on scenario:
     - Single stop found → `templateManager.RenderVoiceFindStop(ctx)`
     - Multiple stops found → `templateManager.RenderVoiceDisambiguation(ctx)`
     - Any error condition → `templateManager.RenderVoiceError(ctx)`
   - Update URL routing from `/voice/input` to `/voice/find_stop`

4. **Update VoiceHandler struct:**
   - Add `VoiceTemplateManager` field
   - Initialize in `NewVoiceHandler()`
   - Handle template rendering errors appropriately

### Phase 4: Code Cleanup
**Duration**: 1 hour

1. **Remove deprecated code:**
   - Delete `GenerateTwiMLVoice` and `GenerateTwiMLGather` from `formatters/response.go`
   - Delete `TwiMLResponse` and `Gather` structs from `models/types.go`
   - Clean up unused imports

2. **Update error handling:**
   - Ensure consistent error messages across templates
   - Maintain backward compatibility for existing voice flows

### Phase 5: Testing Implementation
**Duration**: 5-6 hours

1. **Unit tests for voice template manager (`formatters/voice_templates_test.go`):**
   - Test `RenderVoiceStart` with welcome context
   - Test `RenderVoiceFindStop` with arrivals context
   - Test `RenderVoiceError` with error context
   - Test `RenderVoiceDisambiguation` with disambiguation context
   - Test XML output validation and proper escaping
   - Test error handling for malformed templates
   - Test embedded file loading

2. **Update existing tests (`formatters/response_test.go`):**
   - Remove `TestGenerateTwiMLVoice` and `TestGenerateTwiMLGather`
   - Add comprehensive template-based tests
   - Ensure XML output format matches previous implementation

3. **Voice handler tests (`handlers/voice_test.go`):**
   - Test `HandleVoiceStart` calls correct template (start vs error)
   - Test `HandleFindStop` calls correct template (find_stop/disambiguation/error)
   - Test error handling when template rendering fails
   - Mock template manager for isolated testing

4. **Template validation tests:**
   - Verify each template produces valid XML
   - Test XML parsing with edge case data
   - Ensure templates handle empty/nil values gracefully
   - Test all four templates with their respective contexts

### Phase 6: Documentation and Validation
**Duration**: 1-2 hours

1. **Update code documentation:**
   - Add godoc comments for new template functions
   - Document template file structure and usage
   - Update README if necessary

2. **Manual testing:**
   - Test voice calls with ngrok webhook
   - Verify TwiML responses are identical to previous implementation
   - Test edge cases and error scenarios

3. **Performance validation:**
   - Benchmark template rendering vs. struct marshaling
   - Verify embedded templates don't significantly impact binary size
   - Test memory usage with concurrent template rendering

## Template File Examples

### `templates/voice_start.xml`
```xml
<?xml version="1.0" encoding="UTF-8"?>
<Response>
    <Gather numDigits="6" action="/voice/find_stop" method="POST">
        <Say>{{.WelcomePrompt}}</Say>
    </Gather>
</Response>
```

### `templates/voice_find_stop.xml`
```xml
<?xml version="1.0" encoding="UTF-8"?>
<Response>
    <Say>{{.ArrivalsMessage}}</Say>
</Response>
```

### `templates/voice_error.xml`
```xml
<?xml version="1.0" encoding="UTF-8"?>
<Response>
    <Say>{{.ErrorMessage}}</Say>
</Response>
```

### `templates/voice_disambiguation.xml`
```xml
<?xml version="1.0" encoding="UTF-8"?>
<Response>
    <Gather numDigits="1" action="/voice/find_stop" method="POST">
        <Say>{{.DisambiguationPrompt}}</Say>
    </Gather>
</Response>
```

## Risk Assessment and Mitigation

### High Risk
1. **Template rendering errors breaking voice calls**
   - *Mitigation*: Comprehensive error handling with fallback responses
   - *Testing*: Extensive error scenario testing

2. **XML escaping issues with dynamic content**
   - *Mitigation*: Use Go's `html/template` package for automatic escaping
   - *Testing*: Test with special characters and edge cases

### Medium Risk
1. **Performance regression from template parsing**
   - *Mitigation*: Pre-parse templates at startup, benchmark performance
   - *Testing*: Load testing and performance comparison

2. **Breaking existing voice flows**
   - *Mitigation*: Maintain identical XML output format
   - *Testing*: Comprehensive regression testing

### Low Risk
1. **Binary size increase from embedded templates**
   - *Mitigation*: Templates are small text files, minimal impact expected
   - *Testing*: Monitor binary size before/after

## Success Criteria

1. **Functional Requirements:**
   - All existing voice functionality works identically
   - TwiML output format is unchanged
   - No regression in voice call handling

2. **Code Quality:**
   - Removal of XML generation models and functions
   - Clean separation of markup and logic
   - Maintainable template structure

3. **Testing:**
   - 100% test coverage for new template manager
   - All existing tests pass with template system
   - Integration tests verify end-to-end functionality

4. **Performance:**
   - No significant performance degradation
   - Template rendering completes within existing response time requirements

## Estimated Total Duration: 15-19 hours

This enhancement will improve code maintainability by separating TwiML markup from Go logic while providing endpoint-specific templates that handle all response scenarios. The approach eliminates generic template functions in favor of dedicated templates for each voice endpoint, making the voice flow logic clearer and easier to maintain.