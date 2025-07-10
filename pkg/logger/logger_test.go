package logger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "json format",
			config: Config{
				Level:  "info",
				Format: "json",
			},
		},
		{
			name: "console format",
			config: Config{
				Level:  "debug",
				Format: "console",
			},
		},
		{
			name: "default format",
			config: Config{
				Level:  "warn",
				Format: "unknown",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(tt.config)
			require.NoError(t, err)
			assert.NotNil(t, logger)
		})
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		level    string
		expected string
	}{
		{"debug", "debug"},
		{"info", "info"},
		{"warn", "warn"},
		{"error", "error"},
		{"invalid", "info"}, // Should default to info
		{"", "info"},        // Should default to info
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			level := parseLevel(tt.level)
			assert.Equal(t, tt.expected, level.String())
		})
	}
}

func TestLoggerMethods(t *testing.T) {
	logger, err := New(Config{
		Level:  "debug",
		Format: "json",
	})
	require.NoError(t, err)

	// Test WithContext
	ctx := context.Background()
	contextLogger := logger.WithContext(ctx)
	assert.NotNil(t, contextLogger)

	// Test WithComponent
	componentLogger := logger.WithComponent("test")
	assert.NotNil(t, componentLogger)

	// Test WithError
	testErr := assert.AnError
	errorLogger := logger.WithError(testErr)
	assert.NotNil(t, errorLogger)

	// Test chaining
	chainedLogger := logger.WithComponent("test").WithError(testErr)
	assert.NotNil(t, chainedLogger)
}

func TestLoggerOutput(t *testing.T) {
	// Create a logger with json format for predictable output
	logger, err := New(Config{
		Level:  "debug",
		Format: "json",
	})
	require.NoError(t, err)

	// Test that logger can write messages
	logger.Info().Msg("test message")
	
	// Note: In a real implementation, we might want to capture stdout
	// but for this test we'll just verify the logger doesn't panic
	assert.NotNil(t, logger)
}