package engine

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/your-org/hema-replay-system/pkg/commentary/types"
)

// FallbackGenerator generates fallback commentary using rule-based templates
type FallbackGenerator struct {
	config           *types.CommentaryConfig
	logger           zerolog.Logger
	rules            []FallbackRule
	templateLibrary  map[string][]string
	recentResponses  []string // Track recent responses for variety
	maxRecentHistory int
	randomizer       *rand.Rand
}

// FallbackRule represents a pattern-matching rule for fallback generation
type FallbackRule struct {
	Pattern       *regexp.Regexp
	Keywords      []string
	Templates     []string
	Category      string
	Priority      int
	Confidence    float32
	MinConfidence float32 // Minimum input confidence required for this rule
}

// NewFallbackGenerator creates a new fallback generator
func NewFallbackGenerator(config *types.CommentaryConfig, logger zerolog.Logger) (*FallbackGenerator, error) {
	generator := &FallbackGenerator{
		config:           config,
		logger:           logger.With().Str("component", "fallback-generator").Logger(),
		recentResponses:  make([]string, 0),
		maxRecentHistory: 10,
		randomizer:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	// Initialize rules and templates
	generator.initializeRules()
	generator.initializeTemplates()

	return generator, nil
}

// Generate generates fallback commentary for the given input
func (fg *FallbackGenerator) Generate(input types.TranscriptionInput) (*types.Commentary, error) {
	startTime := time.Now()

	// Find matching rules
	matchingRules := fg.findMatchingRules(input.Text, input.Confidence)

	if len(matchingRules) == 0 {
		// Use generic fallback
		return fg.generateGenericFallback(input, startTime), nil
	}

	// Select best rule based on priority and confidence
	selectedRule := fg.selectBestRule(matchingRules)

	// Generate commentary using the selected rule
	commentary := fg.generateFromRule(selectedRule, input, startTime)

	// Add to recent responses for variety tracking
	fg.addToRecent(commentary.Text)

	return commentary, nil
}

// findMatchingRules finds rules that match the input transcription
func (fg *FallbackGenerator) findMatchingRules(transcription string, confidence float32) []FallbackRule {
	var matching []FallbackRule
	lowerText := strings.ToLower(transcription)

	for _, rule := range fg.rules {
		// Check minimum confidence requirement
		if confidence < rule.MinConfidence {
			continue
		}

		// Check pattern match
		if rule.Pattern != nil && rule.Pattern.MatchString(lowerText) {
			matching = append(matching, rule)
			continue
		}

		// Check keyword match
		keywordMatches := 0
		for _, keyword := range rule.Keywords {
			if strings.Contains(lowerText, strings.ToLower(keyword)) {
				keywordMatches++
			}
		}

		// Require at least one keyword match for keyword-based rules
		if len(rule.Keywords) > 0 && keywordMatches > 0 {
			// Adjust rule confidence based on keyword match ratio
			matchRatio := float32(keywordMatches) / float32(len(rule.Keywords))
			adjustedRule := rule
			adjustedRule.Confidence = rule.Confidence * matchRatio
			matching = append(matching, adjustedRule)
		}
	}

	return matching
}

// selectBestRule selects the best matching rule based on priority and confidence
func (fg *FallbackGenerator) selectBestRule(rules []FallbackRule) FallbackRule {
	if len(rules) == 1 {
		return rules[0]
	}

	// Sort by priority first, then by confidence
	bestRule := rules[0]
	bestScore := float32(bestRule.Priority)*10 + bestRule.Confidence

	for _, rule := range rules[1:] {
		score := float32(rule.Priority)*10 + rule.Confidence
		if score > bestScore {
			bestRule = rule
			bestScore = score
		}
	}

	return bestRule
}

// generateFromRule generates commentary using a specific rule
func (fg *FallbackGenerator) generateFromRule(rule FallbackRule, input types.TranscriptionInput, startTime time.Time) *types.Commentary {
	// Select template with variety consideration
	template := fg.selectTemplate(rule.Templates, rule.Category)

	// Replace variables in template
	text := fg.populateTemplate(template, input)

	// Create commentary object
	commentary := &types.Commentary{
		ID:          generateCommentaryID(),
		Text:        text,
		DisplayText: fg.formatForDisplay(text),
		Source:      "fallback",
		Confidence:  rule.Confidence,
		Timestamp:   time.Now(),

		InputText:       input.Text,
		InputConfidence: input.Confidence,
		TemplateID:      rule.Category,

		GenerationLatency: time.Since(startTime),
		CacheHit:          false,

		QualityScore:     rule.Confidence,
		RelevanceScore:   rule.Confidence,
		ValidationPassed: true, // Fallback content is pre-validated

		Metadata: types.CommentaryMetadata{
			Category:        rule.Category,
			Priority:        "medium", // Fallback is always medium priority
			TargetTone:      "neutral",
			ProcessingSteps: []string{"fallback_generation"},
			Fallbacks:       []string{"rule_based_fallback"},
			ExtraData:       make(map[string]string),
		},
	}

	// Add match context if available
	if input.Context != nil {
		commentary.Metadata.MatchContext = input.Context
	}

	return commentary
}

// generateGenericFallback generates a generic fallback when no rules match
func (fg *FallbackGenerator) generateGenericFallback(input types.TranscriptionInput, startTime time.Time) *types.Commentary {
	// Generic templates for unknown situations
	genericTemplates := []string{
		"The director makes a call.",
		"An action takes place on the piste.",
		"The fencers continue their bout.",
		"The match proceeds.",
	}

	template := genericTemplates[fg.randomizer.Intn(len(genericTemplates))]
	text := fg.populateTemplate(template, input)

	return &types.Commentary{
		ID:          generateCommentaryID(),
		Text:        text,
		DisplayText: fg.formatForDisplay(text),
		Source:      "fallback",
		Confidence:  0.3, // Low confidence for generic fallback
		Timestamp:   time.Now(),

		InputText:       input.Text,
		InputConfidence: input.Confidence,
		TemplateID:      "generic_fallback",

		GenerationLatency: time.Since(startTime),
		CacheHit:          false,

		QualityScore:     0.3,
		RelevanceScore:   0.2,
		ValidationPassed: true,

		Metadata: types.CommentaryMetadata{
			Category:        "generic",
			Priority:        "low",
			TargetTone:      "neutral",
			ProcessingSteps: []string{"generic_fallback"},
			Fallbacks:       []string{"generic_template"},
			ExtraData:       make(map[string]string),
		},
	}
}

// selectTemplate selects a template with variety consideration
func (fg *FallbackGenerator) selectTemplate(templates []string, category string) string {
	if len(templates) == 0 {
		return "The action continues."
	}

	if len(templates) == 1 {
		return templates[0]
	}

	// Filter out recently used templates for variety
	available := make([]string, 0)
	for _, template := range templates {
		if !fg.wasRecentlyUsed(template) {
			available = append(available, template)
		}
	}

	// If all templates were recently used, use all of them
	if len(available) == 0 {
		available = templates
	}

	// Select random template from available options
	return available[fg.randomizer.Intn(len(available))]
}

// populateTemplate replaces variables in templates with actual values
func (fg *FallbackGenerator) populateTemplate(template string, input types.TranscriptionInput) string {
	text := template

	// Replace common variables
	text = strings.ReplaceAll(text, "{{transcription}}", input.Text)

	if input.Context != nil {
		text = strings.ReplaceAll(text, "{{score_red}}", fmt.Sprintf("%d", input.Context.ScoreRed))
		text = strings.ReplaceAll(text, "{{score_blue}}", fmt.Sprintf("%d", input.Context.ScoreBlue))
		text = strings.ReplaceAll(text, "{{current_score}}", fmt.Sprintf("%d-%d", input.Context.ScoreRed, input.Context.ScoreBlue))
		text = strings.ReplaceAll(text, "{{last_scorer}}", input.Context.LastScorer)
		text = strings.ReplaceAll(text, "{{period}}", fmt.Sprintf("%d", input.Context.Period))
	}

	// Replace with default values if context is not available
	text = strings.ReplaceAll(text, "{{score_red}}", "0")
	text = strings.ReplaceAll(text, "{{score_blue}}", "0")
	text = strings.ReplaceAll(text, "{{current_score}}", "0-0")
	text = strings.ReplaceAll(text, "{{last_scorer}}", "")
	text = strings.ReplaceAll(text, "{{period}}", "1")

	return text
}

// wasRecentlyUsed checks if a template was recently used
func (fg *FallbackGenerator) wasRecentlyUsed(template string) bool {
	for _, recent := range fg.recentResponses {
		compareLength := len(template)
		if compareLength > 20 {
			compareLength = 20
		}
		if strings.Contains(recent, template[:compareLength]) {
			return true
		}
	}
	return false
}

// addToRecent adds a response to the recent history
func (fg *FallbackGenerator) addToRecent(response string) {
	fg.recentResponses = append(fg.recentResponses, response)

	// Keep only the most recent responses
	if len(fg.recentResponses) > fg.maxRecentHistory {
		fg.recentResponses = fg.recentResponses[1:]
	}
}

// formatForDisplay formats text for display (same logic as in generator.go)
func (fg *FallbackGenerator) formatForDisplay(text string) string {
	if len(text) > 120 {
		text = text[:117] + "..."
	}
	return text
}

// initializeRules sets up the pattern-matching rules for fallback generation
func (fg *FallbackGenerator) initializeRules() {
	fg.rules = []FallbackRule{
		// Point scoring rules
		{
			Pattern:       regexp.MustCompile(`\b(point|score|touch|hit).*(left|right|red|blue)\b`),
			Keywords:      []string{"point", "score", "touch", "hit", "left", "right", "red", "blue"},
			Templates:     fg.templateLibrary["point_scored"],
			Category:      "scoring",
			Priority:      10,
			Confidence:    0.8,
			MinConfidence: 0.5,
		},
		// Double hit rules
		{
			Pattern:       regexp.MustCompile(`\b(double|both|simultaneous)\b`),
			Keywords:      []string{"double", "both", "simultaneous"},
			Templates:     fg.templateLibrary["double_hit"],
			Category:      "rules",
			Priority:      9,
			Confidence:    0.75,
			MinConfidence: 0.6,
		},
		// No point rules
		{
			Pattern:       regexp.MustCompile(`\b(no.?point|nothing|miss|parry)\b`),
			Keywords:      []string{"no point", "nothing", "miss", "parry"},
			Templates:     fg.templateLibrary["no_point"],
			Category:      "neutral",
			Priority:      7,
			Confidence:    0.7,
			MinConfidence: 0.5,
		},
		// Halt rules
		{
			Pattern:       regexp.MustCompile(`\b(halt|stop|break|pause)\b`),
			Keywords:      []string{"halt", "stop", "break", "pause"},
			Templates:     fg.templateLibrary["halt"],
			Category:      "control",
			Priority:      8,
			Confidence:    0.9,
			MinConfidence: 0.7,
		},
		// Warning/Card rules
		{
			Pattern:       regexp.MustCompile(`\b(card|warning|yellow|red|penalty)\b`),
			Keywords:      []string{"card", "warning", "yellow", "red", "penalty"},
			Templates:     fg.templateLibrary["card_warning"],
			Category:      "discipline",
			Priority:      9,
			Confidence:    0.8,
			MinConfidence: 0.6,
		},
		// Afterblow rules
		{
			Pattern:       regexp.MustCompile(`\b(afterblow|after|counter)\b`),
			Keywords:      []string{"afterblow", "after", "counter"},
			Templates:     fg.templateLibrary["afterblow"],
			Category:      "rules",
			Priority:      8,
			Confidence:    0.75,
			MinConfidence: 0.6,
		},
		// Match end rules
		{
			Pattern:       regexp.MustCompile(`\b(match|bout|end|finish|victory|win)\b`),
			Keywords:      []string{"match", "bout", "end", "finish", "victory", "win"},
			Templates:     fg.templateLibrary["match_end"],
			Category:      "conclusion",
			Priority:      10,
			Confidence:    0.8,
			MinConfidence: 0.7,
		},
	}
}

// initializeTemplates sets up the template library for fallback generation
func (fg *FallbackGenerator) initializeTemplates() {
	fg.templateLibrary = map[string][]string{
		"point_scored": {
			"A point is awarded.",
			"The score changes.",
			"A successful hit connects.",
			"The touch is acknowledged.",
			"A clean point is scored.",
		},
		"double_hit": {
			"Both fencers hit simultaneously.",
			"A double touch occurs.",
			"No point awarded for the double hit.",
			"The exchange results in a double.",
		},
		"no_point": {
			"No point is awarded.",
			"The action results in no score.",
			"The attempt is unsuccessful.",
			"No touch is registered.",
		},
		"halt": {
			"The match is halted.",
			"Action is stopped.",
			"The director calls halt.",
			"Play is paused.",
		},
		"card_warning": {
			"A warning is issued.",
			"Disciplinary action is taken.",
			"The director shows a card.",
			"A penalty is awarded.",
		},
		"afterblow": {
			"An afterblow is called.",
			"A counter-attack lands.",
			"The riposte connects after the initial hit.",
		},
		"match_end": {
			"The match concludes.",
			"The bout is finished.",
			"Victory is decided.",
			"The final result is determined.",
		},
	}
}
