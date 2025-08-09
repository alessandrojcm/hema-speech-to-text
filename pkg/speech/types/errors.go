package types

import "fmt"

// SpeechError represents speech recognition specific errors
type SpeechError struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *SpeechError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("speech error [%s]: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("speech error [%s]: %s", e.Code, e.Message)
}

func (e *SpeechError) Unwrap() error {
	return e.Cause
}

// ErrorCode represents different types of speech recognition errors
type ErrorCode string

const (
	ErrorCodeModelLoad       ErrorCode = "MODEL_LOAD"
	ErrorCodeTranscription   ErrorCode = "TRANSCRIPTION"
	ErrorCodeAudioProcessing ErrorCode = "AUDIO_PROCESSING"
	ErrorCodeVocabulary      ErrorCode = "VOCABULARY"
	ErrorCodeTimeout         ErrorCode = "TIMEOUT"
	ErrorCodeMemory          ErrorCode = "MEMORY"
	ErrorCodeConfiguration   ErrorCode = "CONFIGURATION"
	ErrorCodeConcurrency     ErrorCode = "CONCURRENCY"
)

// NewSpeechError creates a new speech error
func NewSpeechError(code ErrorCode, message string, cause error) *SpeechError {
	return &SpeechError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// IsTimeout checks if the error is a timeout error
func IsTimeout(err error) bool {
	if speechErr, ok := err.(*SpeechError); ok {
		return speechErr.Code == ErrorCodeTimeout
	}
	return false
}

// IsModelError checks if the error is related to model loading
func IsModelError(err error) bool {
	if speechErr, ok := err.(*SpeechError); ok {
		return speechErr.Code == ErrorCodeModelLoad
	}
	return false
}
