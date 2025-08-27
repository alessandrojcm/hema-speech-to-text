package vocabulary

import (
	"fmt"
	"sort"
	"strings"
)

// BuildInitialPrompt creates an initial prompt from high-confidence HEMA terms
func (hv *HEMAVocabulary) BuildInitialPrompt() string {
	hv.mu.RLock()
	defer hv.mu.RUnlock()

	// Collect high-boost terms (boost >= 1.8)
	var highBoostTerms []string
	for term, vocabTerm := range hv.terms {
		if vocabTerm.Boost >= 1.8 {
			highBoostTerms = append(highBoostTerms, term)
		}
	}

	// Sort by boost value (descending) and take top 30 terms
	sort.Slice(highBoostTerms, func(i, j int) bool {
		return hv.terms[highBoostTerms[i]].Boost > hv.terms[highBoostTerms[j]].Boost
	})

	if len(highBoostTerms) > 30 {
		highBoostTerms = highBoostTerms[:30]
	}

	// Build a natural-sounding prompt that includes key terms
	// This helps prime Whisper to recognize these terms
	promptParts := []string{
		"Tournament bout with longsword and rapier.",
		"Judge calls: halt, point, double, afterblow, no-touch.",
		"Techniques: bind, riposte, thrust, cut, parry.",
	}

	// Add high-boost terms if available
	if len(highBoostTerms) > 0 {
		promptParts = append(promptParts, fmt.Sprintf("Terms: %s.", strings.Join(highBoostTerms, ", ")))
	}

	return strings.Join(promptParts, " ")
}

// BuildContextualPrompt creates a context-specific prompt based on recent terms
func (hv *HEMAVocabulary) BuildContextualPrompt(recentTerms []string) string {
	hv.mu.RLock()
	defer hv.mu.RUnlock()

	// Get categories of recent terms to build contextual prompt
	categoryCount := make(map[string]int)
	for _, term := range recentTerms {
		if vocabTerm, exists := hv.terms[strings.ToLower(term)]; exists {
			categoryCount[vocabTerm.Category]++
		}
	}

	// Find most common category
	var dominantCategory string
	maxCount := 0
	for category, count := range categoryCount {
		if count > maxCount {
			maxCount = count
			dominantCategory = category
		}
	}

	// Build category-specific prompt
	if dominantCategory != "" {
		categoryTerms := hv.GetTermsByCategory(dominantCategory)
		if len(categoryTerms) > 15 {
			categoryTerms = categoryTerms[:15] // Limit to top 15 terms
		}

		var contextPrompt string
		switch dominantCategory {
		case "judge_calls":
			contextPrompt = fmt.Sprintf("Judge officiating fencing bout. Calls: %s.", strings.Join(categoryTerms, ", "))
		case "techniques":
			contextPrompt = fmt.Sprintf("Fencing techniques and actions: %s.", strings.Join(categoryTerms, ", "))
		case "weapons":
			contextPrompt = fmt.Sprintf("HEMA weapons and equipment: %s.", strings.Join(categoryTerms, ", "))
		default:
			contextPrompt = fmt.Sprintf("HEMA %s terms: %s.", dominantCategory, strings.Join(categoryTerms, ", "))
		}

		return contextPrompt + " " + hv.BuildInitialPrompt()
	}

	// Fall back to standard prompt
	return hv.BuildInitialPrompt()
}

// GetPromptTerms returns the terms that would be included in the initial prompt
func (hv *HEMAVocabulary) GetPromptTerms() []string {
	hv.mu.RLock()
	defer hv.mu.RUnlock()

	var highBoostTerms []string
	for term, vocabTerm := range hv.terms {
		if vocabTerm.Boost >= 1.8 {
			highBoostTerms = append(highBoostTerms, term)
		}
	}

	// Sort by boost value (descending)
	sort.Slice(highBoostTerms, func(i, j int) bool {
		return hv.terms[highBoostTerms[i]].Boost > hv.terms[highBoostTerms[j]].Boost
	})

	if len(highBoostTerms) > 30 {
		highBoostTerms = highBoostTerms[:30]
	}

	return highBoostTerms
}
