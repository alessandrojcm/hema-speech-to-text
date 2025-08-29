package engine

import (
	"strings"
	"unicode/utf8"

	"github.com/rs/zerolog"

	"github.com/your-org/hema-replay-system/pkg/commentary/types"
)

// QualityValidator validates the quality of generated commentary with basic checks only
type QualityValidator struct {
	config        *types.CommentaryConfig
	logger        zerolog.Logger
	hemaKeywords  []string
	minLength     int
	maxLength     int
	minConfidence float32
}

// ValidationResult represents the result of basic quality validation
type ValidationResult struct {
	IsValid    bool     `json:"is_valid"`
	Confidence float32  `json:"confidence"`
	Relevance  float32  `json:"relevance"`
	Issues     []string `json:"issues"`
}

// NewQualityValidator creates a new simplified quality validator
func NewQualityValidator(config *types.CommentaryConfig, logger zerolog.Logger) (*QualityValidator, error) {
	validator := &QualityValidator{
		config:        config,
		logger:        logger.With().Str("component", "quality-validator").Logger(),
		minLength:     config.MinOutputLength,
		maxLength:     config.MaxOutputLength,
		minConfidence: config.MinConfidence,
		hemaKeywords: []string{
			"point", "hit", "touch", "score", "double", "afterblow",
			"red", "blue", "attack", "defense",
			"parry", "riposte", "thrust", "cut", "pommel", "bind",
			"director", "judge", "halt", "warning", "card", "match",
			"shallow", "deep", "target", "ring", "pushed", "area", "tempo",
		},
	}

	return validator, nil
}

// Validate validates the quality of generated commentary with basic checks only
func (v *QualityValidator) Validate(text string, input types.TranscriptionInput) *ValidationResult {
	result := &ValidationResult{
		IsValid:    true,
		Confidence: 1.0,
		Relevance:  1.0,
		Issues:     []string{},
	}

	// Basic validation checks as specified in instructions:
	// 1. Non-empty output
	// 2. Basic relevance (contains HEMA keywords)
	// 3. Length within bounds (20-200 characters)

	// Check 1: Non-empty output
	trimmedText := strings.TrimSpace(text)
	if trimmedText == "" {
		result.IsValid = false
		result.Issues = append(result.Issues, "Output is empty or whitespace-only")
		result.Confidence = 0.0
		return result
	}

	// Check 2: Length within bounds
	length := utf8.RuneCountInString(trimmedText)
	if length < v.minLength {
		result.IsValid = false
		result.Issues = append(result.Issues, "Output too short")
		result.Confidence = 0.0
		return result
	}

	if length > v.maxLength {
		result.IsValid = false
		result.Issues = append(result.Issues, "Output too long")
		result.Confidence = 0.0
		return result
	}

	// Check 3: Basic relevance (contains HEMA keywords)
	relevanceScore := v.calculateRelevance(trimmedText)
	result.Relevance = relevanceScore

	if relevanceScore < 0.1 { // Must contain at least some HEMA terms
		result.IsValid = false
		result.Issues = append(result.Issues, "No HEMA-related terms found")
		result.Confidence = 0.0
		return result
	}

	// Calculate simple confidence based on length and relevance
	lengthScore := float32(1.0)
	if length < 50 {
		lengthScore = float32(length) / 50.0
	}

	result.Confidence = (lengthScore + relevanceScore) / 2.0

	return result
}

// calculateRelevance calculates basic relevance based on HEMA keyword presence
func (v *QualityValidator) calculateRelevance(text string) float32 {
	lowerText := strings.ToLower(text)
	words := strings.Fields(lowerText)

	if len(words) == 0 {
		return 0.0
	}

	// Count HEMA-related terms
	hemaTermCount := 0
	for _, word := range words {
		for _, keyword := range v.hemaKeywords {
			if strings.Contains(word, keyword) || strings.Contains(keyword, word) {
				hemaTermCount++
				break
			}
		}
	}

	// Simple relevance score based on ratio of HEMA terms to total words
	relevance := float32(hemaTermCount) / float32(len(words))

	// Cap at 1.0
	if relevance > 1.0 {
		relevance = 1.0
	}

	return relevance
}
