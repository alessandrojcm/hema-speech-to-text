package engine

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/rs/zerolog"

	"github.com/your-org/hema-replay-system/pkg/commentary/types"
)

// QualityValidator validates the quality of generated commentary
type QualityValidator struct {
	config          *types.CommentaryConfig
	logger          zerolog.Logger
	profanityFilter *ProfanityFilter
	relevanceScorer *RelevanceScorer

	// Validation rules
	minLength          int
	maxLength          int
	minConfidence      float32
	qualityThreshold   float32
	relevanceThreshold float32

	// Pattern matchers
	validPatterns   []*regexp.Regexp
	invalidPatterns []*regexp.Regexp
}

// ValidationResult represents the result of quality validation
type ValidationResult struct {
	IsValid     bool               `json:"is_valid"`
	Confidence  float32            `json:"confidence"`
	Relevance   float32            `json:"relevance"`
	Issues      []string           `json:"issues"`
	Suggestions []string           `json:"suggestions"`
	Scores      map[string]float32 `json:"scores"`
}

// NewQualityValidator creates a new quality validator
func NewQualityValidator(config *types.CommentaryConfig, logger zerolog.Logger) (*QualityValidator, error) {
	validator := &QualityValidator{
		config:             config,
		logger:             logger.With().Str("component", "quality-validator").Logger(),
		minLength:          config.MinOutputLength,
		maxLength:          config.MaxOutputLength,
		minConfidence:      config.MinConfidence,
		qualityThreshold:   config.QualityThreshold,
		relevanceThreshold: config.RelevanceThreshold,
	}

	// Initialize profanity filter if enabled
	if config.EnableProfanityFilter {
		validator.profanityFilter = NewProfanityFilter()
	}

	// Initialize relevance scorer
	validator.relevanceScorer = NewRelevanceScorer()

	// Initialize validation patterns
	validator.initializePatterns()

	return validator, nil
}

// Validate validates the quality of generated commentary
func (v *QualityValidator) Validate(text string, input types.TranscriptionInput) *ValidationResult {
	result := &ValidationResult{
		IsValid:     true,
		Confidence:  1.0,
		Relevance:   1.0,
		Issues:      []string{},
		Suggestions: []string{},
		Scores:      make(map[string]float32),
	}

	// Basic length validation
	if !v.validateLength(text, result) {
		result.IsValid = false
	}

	// Content quality validation
	if !v.validateContent(text, result) {
		result.IsValid = false
	}

	// Profanity check
	if v.profanityFilter != nil && !v.validateProfanity(text, result) {
		result.IsValid = false
	}

	// Relevance check
	if !v.validateRelevance(text, input, result) {
		result.IsValid = false
	}

	// Calculate overall confidence
	v.calculateConfidence(result)

	// Final validation based on thresholds
	if result.Confidence < v.qualityThreshold || result.Relevance < v.relevanceThreshold {
		result.IsValid = false
		if result.Confidence < v.qualityThreshold {
			result.Issues = append(result.Issues, "Overall quality below threshold")
		}
		if result.Relevance < v.relevanceThreshold {
			result.Issues = append(result.Issues, "Relevance below threshold")
		}
	}

	return result
}

// validateLength checks if the text length is within acceptable bounds
func (v *QualityValidator) validateLength(text string, result *ValidationResult) bool {
	length := utf8.RuneCountInString(text)
	result.Scores["length"] = float32(length)

	if length < v.minLength {
		result.Issues = append(result.Issues, "Text too short")
		result.Suggestions = append(result.Suggestions, "Add more descriptive content")
		return false
	}

	if length > v.maxLength {
		result.Issues = append(result.Issues, "Text too long")
		result.Suggestions = append(result.Suggestions, "Reduce text length")
		return false
	}

	return true
}

// validateContent checks the content quality using various heuristics
func (v *QualityValidator) validateContent(text string, result *ValidationResult) bool {
	valid := true
	contentScore := float32(1.0)

	// Check for empty or whitespace-only text
	if strings.TrimSpace(text) == "" {
		result.Issues = append(result.Issues, "Empty or whitespace-only text")
		return false
	}

	// Check for repetitive content
	if v.isRepetitive(text) {
		result.Issues = append(result.Issues, "Repetitive content detected")
		result.Suggestions = append(result.Suggestions, "Vary the language and structure")
		contentScore -= 0.3
		valid = false
	}

	// Check for incomplete sentences
	if v.hasIncompleteSentences(text) {
		result.Issues = append(result.Issues, "Incomplete sentences detected")
		result.Suggestions = append(result.Suggestions, "Complete all sentences properly")
		contentScore -= 0.2
	}

	// Check for valid patterns
	validPatternScore := v.validatePatterns(text)
	contentScore *= validPatternScore
	result.Scores["patterns"] = validPatternScore

	// Check for coherence
	coherenceScore := v.assessCoherence(text)
	contentScore *= coherenceScore
	result.Scores["coherence"] = coherenceScore

	if coherenceScore < 0.5 {
		result.Issues = append(result.Issues, "Low coherence detected")
		result.Suggestions = append(result.Suggestions, "Improve logical flow and clarity")
		valid = false
	}

	result.Scores["content"] = contentScore
	return valid && contentScore >= 0.6
}

// validateProfanity checks for inappropriate content
func (v *QualityValidator) validateProfanity(text string, result *ValidationResult) bool {
	if v.profanityFilter.ContainsProfanity(text) {
		result.Issues = append(result.Issues, "Inappropriate content detected")
		result.Suggestions = append(result.Suggestions, "Remove inappropriate language")
		result.Scores["profanity"] = 0.0
		return false
	}

	result.Scores["profanity"] = 1.0
	return true
}

// validateRelevance checks if the content is relevant to the input
func (v *QualityValidator) validateRelevance(text string, input types.TranscriptionInput, result *ValidationResult) bool {
	relevanceScore := v.relevanceScorer.Score(text, input.Text)
	result.Relevance = relevanceScore
	result.Scores["relevance"] = relevanceScore

	if relevanceScore < v.relevanceThreshold {
		result.Issues = append(result.Issues, "Low relevance to input")
		result.Suggestions = append(result.Suggestions, "Focus more on the specific judge call")
		return false
	}

	return true
}

// calculateConfidence calculates the overall confidence score
func (v *QualityValidator) calculateConfidence(result *ValidationResult) {
	weights := map[string]float32{
		"content":   0.4,
		"relevance": 0.3,
		"coherence": 0.2,
		"patterns":  0.1,
	}

	totalScore := float32(0.0)
	totalWeight := float32(0.0)

	for metric, weight := range weights {
		if score, exists := result.Scores[metric]; exists {
			totalScore += score * weight
			totalWeight += weight
		}
	}

	if totalWeight > 0 {
		result.Confidence = totalScore / totalWeight
	}

	// Apply penalties for issues
	penalty := float32(len(result.Issues)) * 0.1
	result.Confidence = max(0.0, result.Confidence-penalty)
}

// Helper methods

func (v *QualityValidator) isRepetitive(text string) bool {
	words := strings.Fields(strings.ToLower(text))
	if len(words) < 6 {
		return false
	}

	wordCount := make(map[string]int)
	for _, word := range words {
		wordCount[word]++
	}

	// Check if any word appears more than 30% of the time
	threshold := len(words) * 3 / 10
	for _, count := range wordCount {
		if count > threshold {
			return true
		}
	}

	return false
}

func (v *QualityValidator) hasIncompleteSentences(text string) bool {
	// Simple check for sentences that don't end with proper punctuation
	sentences := regexp.MustCompile(`[.!?]+`).Split(text, -1)

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) > 20 && !regexp.MustCompile(`[.!?]$`).MatchString(text) {
			return true
		}
	}

	return false
}

func (v *QualityValidator) validatePatterns(text string) float32 {
	score := float32(1.0)

	// Check valid patterns (add score)
	for _, pattern := range v.validPatterns {
		if pattern.MatchString(text) {
			score += 0.1
		}
	}

	// Check invalid patterns (subtract score)
	for _, pattern := range v.invalidPatterns {
		if pattern.MatchString(text) {
			score -= 0.2
		}
	}

	return max(0.0, min(1.0, score))
}

func (v *QualityValidator) assessCoherence(text string) float32 {
	// Simple coherence assessment based on sentence structure
	sentences := regexp.MustCompile(`[.!?]+`).Split(text, -1)

	if len(sentences) == 0 {
		return 0.0
	}

	coherenceScore := float32(1.0)

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) == 0 {
			continue
		}

		// Check for basic sentence structure
		if len(strings.Fields(sentence)) < 3 {
			coherenceScore -= 0.2
		}

		// Check for connecting words (basic coherence)
		hasConnectors := regexp.MustCompile(`\b(and|but|however|therefore|thus|meanwhile|after|before|while|since)\b`).MatchString(strings.ToLower(sentence))
		if !hasConnectors && len(sentences) > 1 {
			coherenceScore -= 0.1
		}
	}

	return max(0.0, coherenceScore)
}

func (v *QualityValidator) initializePatterns() {
	// Patterns that indicate good HEMA commentary
	validPatternStrings := []string{
		`\b(point|score|hit|touch)\b`,
		`\b(left|right|red|blue)\b`,
		`\b(excellent|good|clean|solid)\b`,
		`\b(attack|defense|parry|riposte)\b`,
		`\b(fencer|director|judge)\b`,
	}

	// Patterns that indicate poor commentary
	invalidPatternStrings := []string{
		`\b(um|uh|er)\b`,
		`\.\.\.$`,                       // Trailing ellipsis
		`\b(I think|maybe|probably)\b`,  // Uncertain language
		`\b(very very|really really)\b`, // Repetitive intensifiers
	}

	// Compile valid patterns
	for _, pattern := range validPatternStrings {
		if compiled, err := regexp.Compile(pattern); err == nil {
			v.validPatterns = append(v.validPatterns, compiled)
		}
	}

	// Compile invalid patterns
	for _, pattern := range invalidPatternStrings {
		if compiled, err := regexp.Compile(pattern); err == nil {
			v.invalidPatterns = append(v.invalidPatterns, compiled)
		}
	}
}

// ProfanityFilter provides basic profanity filtering
type ProfanityFilter struct {
	blacklist []string
	patterns  []*regexp.Regexp
}

// NewProfanityFilter creates a new profanity filter
func NewProfanityFilter() *ProfanityFilter {
	// Basic profanity list - in a real implementation, this would be more comprehensive
	blacklist := []string{
		"damn", "hell", "crap", // Add actual words as needed
	}

	filter := &ProfanityFilter{
		blacklist: blacklist,
	}

	// Compile patterns
	for _, word := range blacklist {
		if pattern, err := regexp.Compile(`\b` + regexp.QuoteMeta(word) + `\b`); err == nil {
			filter.patterns = append(filter.patterns, pattern)
		}
	}

	return filter
}

// ContainsProfanity checks if text contains inappropriate content
func (pf *ProfanityFilter) ContainsProfanity(text string) bool {
	lowerText := strings.ToLower(text)

	for _, pattern := range pf.patterns {
		if pattern.MatchString(lowerText) {
			return true
		}
	}

	return false
}

// RelevanceScorer scores the relevance between generated commentary and input
type RelevanceScorer struct {
	hemaKeywords []string
}

// NewRelevanceScorer creates a new relevance scorer
func NewRelevanceScorer() *RelevanceScorer {
	return &RelevanceScorer{
		hemaKeywords: []string{
			"point", "hit", "touch", "score", "double", "afterblow",
			"red", "blue", "attack", "defense",
			"parry", "riposte", "thrust", "cut", "pommel", "bind",
			"director", "judge", "halt", "warning", "card", "match",
		},
	}
}

// Score calculates relevance between commentary and input transcription
func (rs *RelevanceScorer) Score(commentary, transcription string) float32 {
	commentaryWords := strings.Fields(strings.ToLower(commentary))
	transcriptionWords := strings.Fields(strings.ToLower(transcription))

	if len(commentaryWords) == 0 || len(transcriptionWords) == 0 {
		return 0.0
	}

	// Calculate word overlap
	transcriptionSet := make(map[string]bool)
	for _, word := range transcriptionWords {
		transcriptionSet[word] = true
	}

	overlap := 0
	hemaTerms := 0

	for _, word := range commentaryWords {
		// Check direct word overlap
		if transcriptionSet[word] {
			overlap++
		}

		// Check HEMA-specific terms
		for _, keyword := range rs.hemaKeywords {
			if strings.Contains(word, keyword) || strings.Contains(keyword, word) {
				hemaTerms++
				break
			}
		}
	}

	// Calculate relevance score
	wordOverlapScore := float32(overlap) / float32(len(commentaryWords))
	hemaRelevanceScore := float32(hemaTerms) / float32(len(commentaryWords))

	// Weighted combination
	relevance := (wordOverlapScore * 0.6) + (hemaRelevanceScore * 0.4)

	return min(1.0, relevance)
}

// Utility functions
func min(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
