package vocabulary

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/rs/zerolog"
)

// HEMAVocabulary manages HEMA-specific terminology and boosts
type HEMAVocabulary struct {
	terms      map[string]VocabularyTerm
	boosts     map[string]float64
	phonetic   map[string][]string // Phonetic variations
	categories map[string][]string // Term categories
	mu         sync.RWMutex
	logger     zerolog.Logger
}

// VocabularyTerm represents a HEMA vocabulary term
type VocabularyTerm struct {
	Term      string
	Category  string
	Boost     float64
	Phonetic  []string
	Context   []string
	Frequency int
}

// NewHEMAVocabulary creates a new HEMA vocabulary manager
func NewHEMAVocabulary(logger zerolog.Logger) *HEMAVocabulary {
	return &HEMAVocabulary{
		terms:      make(map[string]VocabularyTerm),
		boosts:     make(map[string]float64),
		phonetic:   make(map[string][]string),
		categories: make(map[string][]string),
		logger:     logger.With().Str("component", "hema_vocabulary").Logger(),
	}
}

// LoadFromFile loads HEMA vocabulary from a file
func (hv *HEMAVocabulary) LoadFromFile(filePath string) error {
	hv.mu.Lock()
	defer hv.mu.Unlock()

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open vocabulary file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if err := hv.parseLine(line, lineNum); err != nil {
			hv.logger.Warn().
				Err(err).
				Int("line", lineNum).
				Str("content", line).
				Msg("Failed to parse vocabulary line")
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading vocabulary file: %w", err)
	}

	hv.logger.Info().
		Int("terms_loaded", len(hv.terms)).
		Str("file_path", filePath).
		Msg("HEMA vocabulary loaded successfully")

	return nil
}

// parseLine parses a single line from the vocabulary file
// Format: term|category|boost|phonetic1,phonetic2|context1,context2
func (hv *HEMAVocabulary) parseLine(line string, lineNum int) error {
	parts := strings.Split(line, "|")
	if len(parts) < 2 {
		return fmt.Errorf("invalid format: expected at least 2 parts")
	}

	term := strings.TrimSpace(parts[0])
	category := strings.TrimSpace(parts[1])

	vocabTerm := VocabularyTerm{
		Term:     term,
		Category: category,
		Boost:    1.0, // Default boost
	}

	// Parse boost if provided
	if len(parts) > 2 && parts[2] != "" {
		var boost float64
		if _, err := fmt.Sscanf(parts[2], "%f", &boost); err == nil {
			vocabTerm.Boost = boost
		}
	}

	// Parse phonetic variations if provided
	if len(parts) > 3 && parts[3] != "" {
		phonetic := strings.Split(parts[3], ",")
		for i, p := range phonetic {
			phonetic[i] = strings.TrimSpace(p)
		}
		vocabTerm.Phonetic = phonetic
		hv.phonetic[term] = phonetic
	}

	// Parse context if provided
	if len(parts) > 4 && parts[4] != "" {
		context := strings.Split(parts[4], ",")
		for i, c := range context {
			context[i] = strings.TrimSpace(c)
		}
		vocabTerm.Context = context
	}

	hv.terms[term] = vocabTerm
	hv.boosts[term] = vocabTerm.Boost

	// Add to category mapping
	if hv.categories[category] == nil {
		hv.categories[category] = make([]string, 0)
	}
	hv.categories[category] = append(hv.categories[category], term)

	return nil
}

// GetBoost returns the boost value for a term
func (hv *HEMAVocabulary) GetBoost(term string) float64 {
	hv.mu.RLock()
	defer hv.mu.RUnlock()

	if boost, exists := hv.boosts[strings.ToLower(term)]; exists {
		return boost
	}

	// Check phonetic variations
	for originalTerm, variations := range hv.phonetic {
		for _, variation := range variations {
			if strings.EqualFold(variation, term) {
				return hv.boosts[originalTerm]
			}
		}
	}

	return 1.0 // Default boost
}

// IsHEMATerm checks if a term is HEMA-related
func (hv *HEMAVocabulary) IsHEMATerm(term string) bool {
	hv.mu.RLock()
	defer hv.mu.RUnlock()

	_, exists := hv.terms[strings.ToLower(term)]
	if exists {
		return true
	}

	// Check phonetic variations
	for _, variations := range hv.phonetic {
		for _, variation := range variations {
			if strings.EqualFold(variation, term) {
				return true
			}
		}
	}

	return false
}

// GetTermsByCategory returns all terms in a specific category
func (hv *HEMAVocabulary) GetTermsByCategory(category string) []string {
	hv.mu.RLock()
	defer hv.mu.RUnlock()

	if terms, exists := hv.categories[category]; exists {
		result := make([]string, len(terms))
		copy(result, terms)
		return result
	}

	return []string{}
}

// GetAllTerms returns all vocabulary terms
func (hv *HEMAVocabulary) GetAllTerms() map[string]VocabularyTerm {
	hv.mu.RLock()
	defer hv.mu.RUnlock()

	result := make(map[string]VocabularyTerm)
	for k, v := range hv.terms {
		result[k] = v
	}

	return result
}

// UpdateBoost updates the boost value for a term
func (hv *HEMAVocabulary) UpdateBoost(term string, boost float64) {
	hv.mu.Lock()
	defer hv.mu.Unlock()

	term = strings.ToLower(term)
	if vocabTerm, exists := hv.terms[term]; exists {
		vocabTerm.Boost = boost
		hv.terms[term] = vocabTerm
		hv.boosts[term] = boost
	}
}

// AddTerm adds a new term to the vocabulary
func (hv *HEMAVocabulary) AddTerm(term VocabularyTerm) {
	hv.mu.Lock()
	defer hv.mu.Unlock()

	termKey := strings.ToLower(term.Term)
	hv.terms[termKey] = term
	hv.boosts[termKey] = term.Boost

	if term.Phonetic != nil {
		hv.phonetic[termKey] = term.Phonetic
	}

	// Add to category mapping
	if hv.categories[term.Category] == nil {
		hv.categories[term.Category] = make([]string, 0)
	}
	hv.categories[term.Category] = append(hv.categories[term.Category], termKey)
}

// GetStats returns vocabulary statistics
func (hv *HEMAVocabulary) GetStats() map[string]interface{} {
	hv.mu.RLock()
	defer hv.mu.RUnlock()

	categoryStats := make(map[string]int)
	for category, terms := range hv.categories {
		categoryStats[category] = len(terms)
	}

	return map[string]interface{}{
		"total_terms":      len(hv.terms),
		"total_categories": len(hv.categories),
		"category_stats":   categoryStats,
		"phonetic_terms":   len(hv.phonetic),
	}
}
