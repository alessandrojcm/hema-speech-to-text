package templates

import (
	"fmt"
	"strings"

	"github.com/your-org/hema-replay-system/pkg/commentary/context"
)

// HEMATemplates contains all HEMA-specific prompt templates
var HEMATemplates = map[string]*PromptTemplate{
	"point_scored": {
		ID:   "point_scored",
		Name: "Point Scored Commentary",
		Template: `Convert this HEMA judge call into engaging TV commentary (max 2 sentences):
Judge call: "{{.Transcription}}"
Score: {{.MatchState.CurrentScore}}
{{if .MatchState.LastScorer}}Last scorer: {{.MatchState.LastScorer}}{{end}}
Commentary:`,
		Variables:   []string{"Transcription", "MatchState.CurrentScore", "MatchState.LastScorer"},
		MaxTokens:   50,
		Temperature: 0.7,
		Metadata: map[string]string{
			"category":    "scoring",
			"priority":    "high",
			"target_tone": "exciting",
		},
	},
	"double_hit": {
		ID:   "double_hit",
		Name: "Double Hit Explanation",
		Template: `Explain this double hit situation for TV viewers in simple terms:
Judge call: "{{.Transcription}}"
Match situation: {{.MatchState.ScoreRed}}-{{.MatchState.ScoreBlue}}
Brief explanation:`,
		Variables:   []string{"Transcription", "MatchState.ScoreRed", "MatchState.ScoreBlue"},
		MaxTokens:   60,
		Temperature: 0.6,
		Metadata: map[string]string{
			"category":    "rules",
			"priority":    "high",
			"target_tone": "educational",
		},
	},
	"no_point": {
		ID:   "no_point",
		Name: "No Point Called",
		Template: `Comment on this no-point situation in HEMA:
Judge call: "{{.Transcription}}"
Current action: {{.MatchState.LastAction}}
Commentary:`,
		Variables:   []string{"Transcription", "MatchState.LastAction"},
		MaxTokens:   40,
		Temperature: 0.5,
		Metadata: map[string]string{
			"category":    "neutral",
			"priority":    "medium",
			"target_tone": "analytical",
		},
	},
	"afterblow": {
		ID:   "afterblow",
		Name: "Afterblow Situation",
		Template: `Explain this afterblow call for viewers:
Judge call: "{{.Transcription}}"
This means: {{if eq .MatchState.LastScorer "left"}}the left fencer scored but took a counter-attack{{else if eq .MatchState.LastScorer "right"}}the right fencer scored but took a counter-attack{{else}}both fencers hit in rapid succession{{end}}
Commentary:`,
		Variables:   []string{"Transcription", "MatchState.LastScorer"},
		MaxTokens:   55,
		Temperature: 0.6,
		Metadata: map[string]string{
			"category":    "rules",
			"priority":    "high",
			"target_tone": "educational",
		},
	},
	"halt": {
		ID:   "halt",
		Name: "Match Halt",
		Template: `Brief comment on this match interruption:
Judge call: "{{.Transcription}}"
{{if .Context.reason}}Reason: {{.Context.reason}}{{end}}
Commentary:`,
		Variables:   []string{"Transcription", "Context.reason"},
		MaxTokens:   30,
		Temperature: 0.4,
		Metadata: map[string]string{
			"category":    "control",
			"priority":    "low",
			"target_tone": "neutral",
		},
	},
	"card_warning": {
		ID:   "card_warning",
		Name: "Card or Warning",
		Template: `Explain this disciplinary action:
Judge call: "{{.Transcription}}"
Match context: {{.MatchState.CurrentScore}}, Period {{.MatchState.Period}}
Commentary:`,
		Variables:   []string{"Transcription", "MatchState.CurrentScore", "MatchState.Period"},
		MaxTokens:   45,
		Temperature: 0.5,
		Metadata: map[string]string{
			"category":    "discipline",
			"priority":    "high",
			"target_tone": "serious",
		},
	},
	"match_end": {
		ID:   "match_end",
		Name: "Match Conclusion",
		Template: `Provide closing commentary for this match result:
Judge call: "{{.Transcription}}"
Final score: {{.MatchState.ScoreRed}}-{{.MatchState.ScoreBlue}}
{{if gt .MatchState.ScoreRed .MatchState.ScoreBlue}}Left fencer takes the victory{{else if gt .MatchState.ScoreBlue .MatchState.ScoreRed}}Right fencer claims the win{{else}}Match ends in a tie{{end}}
Closing commentary:`,
		Variables:   []string{"Transcription", "MatchState.ScoreRed", "MatchState.ScoreBlue"},
		MaxTokens:   70,
		Temperature: 0.8,
		Metadata: map[string]string{
			"category":    "conclusion",
			"priority":    "high",
			"target_tone": "celebratory",
		},
	},
	"technique_highlight": {
		ID:   "technique_highlight",
		Name: "Technique Highlight",
		Template: `Comment on this impressive technique:
Judge call: "{{.Transcription}}"
{{if .Context.technique}}Technique used: {{.Context.technique}}{{end}}
{{if eq .MatchState.LastScorer "left"}}Left fencer executed well{{else if eq .MatchState.LastScorer "right"}}Right fencer shows good form{{else}}Excellent technical display{{end}}
Commentary:`,
		Variables:   []string{"Transcription", "Context.technique", "MatchState.LastScorer"},
		MaxTokens:   55,
		Temperature: 0.8,
		Metadata: map[string]string{
			"category":    "technique",
			"priority":    "medium",
			"target_tone": "appreciative",
		},
	},
	"director_decision": {
		ID:   "director_decision",
		Name: "Director's Decision",
		Template: `Explain this director's ruling:
Judge call: "{{.Transcription}}"
Match situation: Period {{.MatchState.Period}}, {{.MatchState.ScoreRed}}-{{.MatchState.ScoreBlue}}
Explanation:`,
		Variables:   []string{"Transcription", "MatchState.Period", "MatchState.ScoreRed", "MatchState.ScoreBlue"},
		MaxTokens:   50,
		Temperature: 0.5,
		Metadata: map[string]string{
			"category":    "official",
			"priority":    "medium",
			"target_tone": "informative",
		},
	},
	"grappling_situation": {
		ID:   "grappling_situation",
		Name: "Grappling Commentary",
		Template: `Comment on this grappling exchange:
Judge call: "{{.Transcription}}"
{{if .Context.weapon}}Weapon: {{.Context.weapon}}{{end}}
Commentary:`,
		Variables:   []string{"Transcription", "Context.weapon"},
		MaxTokens:   45,
		Temperature: 0.7,
		Metadata: map[string]string{
			"category":    "grappling",
			"priority":    "medium",
			"target_tone": "tactical",
		},
	},
}

// TemplateSelector helps select the appropriate template based on context
type TemplateSelector struct {
	keywordMap      map[string][]string
	defaultTemplate string
}

// NewTemplateSelector creates a new template selector
func NewTemplateSelector() *TemplateSelector {
	return &TemplateSelector{
		keywordMap: map[string][]string{
			"point_scored":        {"point", "touch", "hit", "score", "left", "right"},
			"double_hit":          {"double", "both", "simultaneous", "together"},
			"no_point":            {"no point", "nothing", "miss", "parry"},
			"afterblow":           {"afterblow", "after", "counter", "riposte"},
			"halt":                {"halt", "stop", "break", "pause"},
			"card_warning":        {"card", "warning", "yellow", "red", "penalty"},
			"match_end":           {"match", "bout", "end", "finish", "victory", "defeat"},
			"technique_highlight": {"thrust", "cut", "slice", "pommel", "crossguard", "bind"},
			"director_decision":   {"director", "decision", "ruling", "call"},
			"grappling_situation": {"grapple", "grab", "wrestling", "close", "clinch"},
		},
		defaultTemplate: "point_scored",
	}
}

// SelectTemplate selects the most appropriate template based on transcription content
func (ts *TemplateSelector) SelectTemplate(transcription string, matchState *context.MatchState) string {
	transcriptionLower := strings.ToLower(transcription)

	// Score each template based on keyword matches
	scores := make(map[string]int)

	for templateID, keywords := range ts.keywordMap {
		score := 0
		for _, keyword := range keywords {
			if strings.Contains(transcriptionLower, strings.ToLower(keyword)) {
				score++
			}
		}
		scores[templateID] = score
	}

	// Find the template with the highest score
	maxScore := 0
	selectedTemplate := ts.defaultTemplate

	for templateID, score := range scores {
		if score > maxScore {
			maxScore = score
			selectedTemplate = templateID
		}
	}

	// Special logic based on match state
	if matchState != nil {
		// If it's the end of the match (high score), prefer match_end
		if matchState.ScoreRed >= 5 || matchState.ScoreBlue >= 5 {
			if strings.Contains(transcriptionLower, "match") || strings.Contains(transcriptionLower, "end") {
				selectedTemplate = "match_end"
			}
		}

		// If we detect specific situations, override
		if strings.Contains(transcriptionLower, "double") {
			selectedTemplate = "double_hit"
		} else if strings.Contains(transcriptionLower, "halt") {
			selectedTemplate = "halt"
		}
	}

	return selectedTemplate
}

// GetTemplatesByCategory returns templates filtered by category
func GetTemplatesByCategory(category string) map[string]*PromptTemplate {
	result := make(map[string]*PromptTemplate)

	for id, tmpl := range HEMATemplates {
		if tmpl.Metadata != nil && tmpl.Metadata["category"] == category {
			result[id] = tmpl
		}
	}

	return result
}

// GetTemplatesByPriority returns templates filtered by priority
func GetTemplatesByPriority(priority string) map[string]*PromptTemplate {
	result := make(map[string]*PromptTemplate)

	for id, tmpl := range HEMATemplates {
		if tmpl.Metadata != nil && tmpl.Metadata["priority"] == priority {
			result[id] = tmpl
		}
	}

	return result
}

// RegisterHEMATemplates registers all HEMA templates with the default template manager
func RegisterHEMATemplates() error {
	for _, tmpl := range HEMATemplates {
		if err := RegisterDefaultTemplate(tmpl); err != nil {
			return fmt.Errorf("failed to register template %s: %w", tmpl.ID, err)
		}
	}
	return nil
}

// init automatically registers HEMA templates when the package is imported
func init() {
	// Register all HEMA templates with the default template manager
	if err := RegisterHEMATemplates(); err != nil {
		// In a production system, you might want to handle this differently
		panic(fmt.Sprintf("Failed to initialize HEMA templates: %v", err))
	}
}
