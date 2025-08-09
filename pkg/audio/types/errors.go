package types

import "errors"

var (
	ErrDeviceNotFound       = errors.New("audio device not found")
	ErrDeviceUnavailable    = errors.New("audio device unavailable")
	ErrInvalidFormat        = errors.New("invalid audio format")
	ErrBufferOverrun        = errors.New("audio buffer overrun")
	ErrBufferUnderrun       = errors.New("audio buffer underrun")
	ErrInsufficientData     = errors.New("insufficient audio data")
	ErrExtractionTimeout    = errors.New("audio extraction timeout")
	ErrConcurrencyLimit     = errors.New("extraction concurrency limit exceeded")
	ErrAlreadyRunning       = errors.New("audio system already running")
	ErrNotRunning           = errors.New("audio system not running")
	ErrEmptyData            = errors.New("empty audio data")
	ErrInvalidConfig        = errors.New("invalid audio configuration")
	ErrStreamClosed         = errors.New("audio stream closed")
	ErrInitializationFailed = errors.New("audio system initialization failed")
)
