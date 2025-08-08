package text

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-org/hema-replay-system/internal/config"
)

func TestNewFormatter(t *testing.T) {
	config := config.TextConfig{
		SourceName:      "TestText",
		DefaultMessages: []string{"Test message 1", "Test message 2"},
		MaxLength:       50,
	}

	formatter := NewFormatter(config)
	require.NotNil(t, formatter)
	assert.Equal(t, config, formatter.config)
}

func TestFormatter_Format(t *testing.T) {
	tests := []struct {
		name      string
		maxLength int
		input     string
		expected  string
	}{
		{
			name:      "normal text",
			maxLength: 100,
			input:     "Point scored!",
			expected:  "Point scored!",
		},
		{
			name:      "text with extra spaces",
			maxLength: 100,
			input:     "  Point   scored!  ",
			expected:  "Point scored!",
		},
		{
			name:      "text with newlines",
			maxLength: 100,
			input:     "Point\nscored!\r\nExcellent!",
			expected:  "Point scored! Excellent!",
		},
		{
			name:      "text with tabs",
			maxLength: 100,
			input:     "Point\tscored!\tExcellent!",
			expected:  "Point scored! Excellent!",
		},
		{
			name:      "text too long",
			maxLength: 10,
			input:     "This is a very long message that should be truncated",
			expected:  "This is...",
		},
		{
			name:      "text exactly at limit",
			maxLength: 13,
			input:     "Point scored!",
			expected:  "Point scored!",
		},
		{
			name:      "empty text",
			maxLength: 100,
			input:     "",
			expected:  "",
		},
		{
			name:      "whitespace only",
			maxLength: 100,
			input:     "   \n\t  ",
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := config.TextConfig{
				MaxLength: tt.maxLength,
			}

			formatter := NewFormatter(config)
			result := formatter.Format(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatter_FormatMessage(t *testing.T) {
	config := config.TextConfig{
		DefaultMessages: []string{
			"Point scored!",
			"Excellent exchange!",
			"Match continues...",
		},
		MaxLength: 100,
	}

	formatter := NewFormatter(config)

	tests := []struct {
		name         string
		messageIndex int
		expected     string
		expectError  bool
	}{
		{
			name:         "valid index 0",
			messageIndex: 0,
			expected:     "Point scored!",
			expectError:  false,
		},
		{
			name:         "valid index 1",
			messageIndex: 1,
			expected:     "Excellent exchange!",
			expectError:  false,
		},
		{
			name:         "valid index 2",
			messageIndex: 2,
			expected:     "Match continues...",
			expectError:  false,
		},
		{
			name:         "invalid negative index",
			messageIndex: -1,
			expected:     "",
			expectError:  false,
		},
		{
			name:         "invalid high index",
			messageIndex: 10,
			expected:     "",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := formatter.FormatMessage(tt.messageIndex)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestFormatter_ValidateText(t *testing.T) {
	config := config.TextConfig{
		MaxLength: 100,
	}

	formatter := NewFormatter(config)

	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "valid text",
			input:       "Point scored!",
			expectError: false,
		},
		{
			name:        "empty text",
			input:       "",
			expectError: false,
		},
		{
			name:        "whitespace only",
			input:       "   ",
			expectError: false,
		},
		{
			name:        "long text",
			input:       "This is a very long message that exceeds normal limits",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := formatter.ValidateText(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFormatter_UtilityMethods(t *testing.T) {
	config := config.TextConfig{
		DefaultMessages: []string{"Message 1", "Message 2"},
		MaxLength:       50,
	}

	formatter := NewFormatter(config)

	assert.Equal(t, 50, formatter.GetMaxLength())
	assert.Equal(t, []string{"Message 1", "Message 2"}, formatter.GetDefaultMessages())
}

func TestFormatter_UTF8Handling(t *testing.T) {
	config := config.TextConfig{
		MaxLength: 10,
	}

	formatter := NewFormatter(config)

	// Test with UTF-8 characters
	input := "🤺⚔️ Point scored! 🏆"
	result := formatter.Format(input)

	// Should be truncated but handle UTF-8 properly
	assert.True(t, len([]rune(result)) <= 10)
	assert.Contains(t, result, "...")
}
