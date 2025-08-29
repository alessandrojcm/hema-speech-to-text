package prompt

import "fmt"

// BuildSimplePrompt creates a simple static prompt for HEMA judge call explanation
func BuildSimplePrompt(transcription string) string {
	return fmt.Sprintf("You are a HEMA (Historical European Martial Arts) expert. Briefly explain what this judge call means in 1-2 sentences: '%s'", transcription)
}
