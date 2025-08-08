package text

import (
	"strings"
	"unicode/utf8"

	"github.com/your-org/hema-replay-system/internal/config"
)

// Formatter handles text formatting operations including length limiting,
// whitespace cleanup, and UTF-8 safe truncation.
type Formatter struct {
	config config.TextConfig
}

// NewFormatter creates a new text formatter with the given configuration.
func NewFormatter(config config.TextConfig) *Formatter {
	return &Formatter{
		config: config,
	}
}

// Format cleans and formats text according to configuration constraints.
func (f *Formatter) Format(text string) string {
	// Phase 1: Simple formatting - just length limiting and basic cleanup
	formatted := f.cleanText(text)
	formatted = f.limitLength(formatted)
	return formatted
}

func (f *Formatter) cleanText(text string) string {
	// Remove excessive whitespace
	text = strings.TrimSpace(text)

	// Replace newlines with spaces (for single-line display)
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")
	text = strings.ReplaceAll(text, "\t", " ")

	// Replace multiple spaces with single space (after newline replacement)
	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}

	return text
}
func (f *Formatter) limitLength(text string) string {
	if f.config.MaxLength <= 0 {
		return text
	}

	// Use UTF-8 aware length limiting
	if utf8.RuneCountInString(text) <= f.config.MaxLength {
		return text
	}

	// Truncate and add ellipsis
	runes := []rune(text)
	if len(runes) > f.config.MaxLength-3 {
		return string(runes[:f.config.MaxLength-3]) + "..."
	}

	return string(runes[:f.config.MaxLength])
}

func (f *Formatter) FormatMessage(messageIndex int) (string, error) {
	if messageIndex < 0 || messageIndex >= len(f.config.DefaultMessages) {
		return "", nil
	}

	return f.Format(f.config.DefaultMessages[messageIndex]), nil
}

func (f *Formatter) ValidateText(text string) error {
	// Phase 1: Basic validation
	if strings.TrimSpace(text) == "" {
		return nil // Empty text is allowed (will clear overlay)
	}

	return nil
}

// Utility functions for future phases
func (f *Formatter) GetMaxLength() int {
	return f.config.MaxLength
}

func (f *Formatter) GetDefaultMessages() []string {
	return f.config.DefaultMessages
}
