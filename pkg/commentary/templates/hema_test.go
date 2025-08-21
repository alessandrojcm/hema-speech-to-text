package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/your-org/hema-replay-system/pkg/commentary/context"
)

func TestHEMATemplates_Registration(t *testing.T) {
	// Test that all HEMA templates are registered during init
	expectedTemplates := []string{
		"point_scored",
		"double_hit",
		"no_point",
		"afterblow",
		"halt",
		"card_warning",
		"match_end",
		"technique_highlight",
		"director_decision",
		"grappling_situation",
	}

	for _, templateID := range expectedTemplates {
		template, exists := GetDefaultTemplate(templateID)
		assert.True(t, exists, "Template %s should be registered", templateID)
		assert.Equal(t, templateID, template.ID)
		assert.NotEmpty(t, template.Name)
		assert.NotEmpty(t, template.Template)
		assert.NotNil(t, template.Metadata)
	}
}

func TestHEMATemplates_Execution(t *testing.T) {
	tests := []struct {
		name         string
		templateID   string
		templateData *TemplateData
		shouldWork   bool
		contains     []string // Expected strings in output
	}{
		{
			name:       "point_scored_basic",
			templateID: "point_scored",
			templateData: &TemplateData{
				Transcription: "Point left",
				MatchState: &context.MatchState{
					CurrentScore: "2-1",
					LastScorer:   "left",
				},
			},
			shouldWork: true,
			contains:   []string{"Point left", "2-1"},
		},
		{
			name:       "double_hit",
			templateID: "double_hit",
			templateData: &TemplateData{
				Transcription: "Double hit no point",
				MatchState: &context.MatchState{
					ScoreRed:  2,
					ScoreBlue: 1,
				},
			},
			shouldWork: true,
			contains:   []string{"Double hit no point", "2-1"},
		},
		{
			name:       "match_end_with_fencers",
			templateID: "match_end",
			templateData: &TemplateData{
				Transcription: "Match complete",
				MatchState: &context.MatchState{
					ScoreRed:  5,
					ScoreBlue: 3,
				},
			},
			shouldWork: true,
			contains:   []string{"Match complete", "5-3", "Alice"},
		},
		{
			name:       "technique_highlight_with_style",
			templateID: "technique_highlight",
			templateData: &TemplateData{
				Transcription: "Beautiful thrust to the chest",
				Context:       map[string]string{"technique": "thrust"},
				MatchState:    &context.MatchState{LastScorer: "left"},
			},
			shouldWork: true,
			contains:   []string{"Beautiful thrust", "thrust", "aggressive"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExecuteDefaultTemplate(tt.templateID, tt.templateData)

			if tt.shouldWork {
				assert.NoError(t, err)
				assert.NotEmpty(t, result)

				for _, expectedString := range tt.contains {
					assert.Contains(t, result, expectedString,
						"Expected '%s' to contain '%s'", result, expectedString)
				}
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestTemplateSelector_SelectTemplate(t *testing.T) {
	selector := NewTemplateSelector()

	tests := []struct {
		name          string
		transcription string
		matchState    *context.MatchState
		expectedID    string
	}{
		{
			name:          "point_scored_detection",
			transcription: "Point left clean hit",
			matchState:    &context.MatchState{},
			expectedID:    "point_scored",
		},
		{
			name:          "double_hit_detection",
			transcription: "Double hit both fencers",
			matchState:    &context.MatchState{},
			expectedID:    "double_hit",
		},
		{
			name:          "halt_detection",
			transcription: "Halt stop the action",
			matchState:    &context.MatchState{},
			expectedID:    "halt",
		},
		{
			name:          "match_end_high_score",
			transcription: "Match is finished",
			matchState: &context.MatchState{
				ScoreRed:  5,
				ScoreBlue: 2,
			},
			expectedID: "match_end",
		},
		{
			name:          "afterblow_detection",
			transcription: "Afterblow from the right",
			matchState:    &context.MatchState{},
			expectedID:    "afterblow",
		},
		{
			name:          "card_warning_detection",
			transcription: "Yellow card for the left fencer",
			matchState:    &context.MatchState{},
			expectedID:    "card_warning",
		},
		{
			name:          "grappling_detection",
			transcription: "Grapple situation break them up",
			matchState:    &context.MatchState{},
			expectedID:    "grappling_situation",
		},
		{
			name:          "no_clear_match_uses_default",
			transcription: "Unclear situation",
			matchState:    &context.MatchState{},
			expectedID:    "point_scored", // Default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selected := selector.SelectTemplate(tt.transcription, tt.matchState)
			assert.Equal(t, tt.expectedID, selected)
		})
	}
}

func TestGetTemplatesByCategory(t *testing.T) {
	tests := []struct {
		category      string
		expectedCount int
		expectedIDs   []string
	}{
		{
			category:      "scoring",
			expectedCount: 1,
			expectedIDs:   []string{"point_scored"},
		},
		{
			category:      "rules",
			expectedCount: 2,
			expectedIDs:   []string{"double_hit", "afterblow"},
		},
		{
			category:      "discipline",
			expectedCount: 1,
			expectedIDs:   []string{"card_warning"},
		},
		{
			category:      "nonexistent",
			expectedCount: 0,
			expectedIDs:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			templates := GetTemplatesByCategory(tt.category)
			assert.Len(t, templates, tt.expectedCount)

			for _, expectedID := range tt.expectedIDs {
				assert.Contains(t, templates, expectedID)
			}
		})
	}
}

func TestGetTemplatesByPriority(t *testing.T) {
	tests := []struct {
		priority         string
		expectedContains []string
		shouldHaveItems  bool
	}{
		{
			priority:         "high",
			expectedContains: []string{"point_scored", "double_hit", "afterblow", "card_warning", "match_end"},
			shouldHaveItems:  true,
		},
		{
			priority:         "medium",
			expectedContains: []string{"technique_highlight", "director_decision", "grappling_situation"},
			shouldHaveItems:  true,
		},
		{
			priority:         "low",
			expectedContains: []string{"halt"},
			shouldHaveItems:  true,
		},
		{
			priority:        "nonexistent",
			shouldHaveItems: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.priority, func(t *testing.T) {
			templates := GetTemplatesByPriority(tt.priority)

			if tt.shouldHaveItems {
				assert.NotEmpty(t, templates)
				for _, expectedID := range tt.expectedContains {
					assert.Contains(t, templates, expectedID)
				}
			} else {
				assert.Empty(t, templates)
			}
		})
	}
}

func TestTemplateMetadata(t *testing.T) {
	// Test that all templates have required metadata
	for templateID, template := range HEMATemplates {
		t.Run(templateID, func(t *testing.T) {
			assert.NotNil(t, template.Metadata, "Template %s should have metadata", templateID)

			// Check required metadata fields
			assert.Contains(t, template.Metadata, "category", "Template %s should have category", templateID)
			assert.Contains(t, template.Metadata, "priority", "Template %s should have priority", templateID)
			assert.Contains(t, template.Metadata, "target_tone", "Template %s should have target_tone", templateID)

			// Validate metadata values
			category := template.Metadata["category"]
			assert.Contains(t, []string{"scoring", "rules", "neutral", "discipline", "conclusion", "technique", "official", "grappling", "control"},
				category, "Invalid category for template %s", templateID)

			priority := template.Metadata["priority"]
			assert.Contains(t, []string{"high", "medium", "low"},
				priority, "Invalid priority for template %s", templateID)

			tone := template.Metadata["target_tone"]
			assert.Contains(t, []string{"exciting", "educational", "analytical", "neutral", "serious", "celebratory", "appreciative", "informative", "tactical"},
				tone, "Invalid target_tone for template %s", templateID)
		})
	}
}

func TestTemplateVariables(t *testing.T) {
	// Test that all templates have their required variables documented
	for templateID, template := range HEMATemplates {
		t.Run(templateID, func(t *testing.T) {
			assert.NotEmpty(t, template.Variables, "Template %s should declare its variables", templateID)

			// Basic sanity check - all templates should use Transcription
			assert.Contains(t, template.Variables, "Transcription",
				"Template %s should use Transcription variable", templateID)
		})
	}
}

func TestTemplateTokenLimits(t *testing.T) {
	// Test that all templates have reasonable token limits
	for templateID, template := range HEMATemplates {
		t.Run(templateID, func(t *testing.T) {
			assert.Greater(t, template.MaxTokens, 0, "Template %s should have positive MaxTokens", templateID)
			assert.LessOrEqual(t, template.MaxTokens, 200, "Template %s MaxTokens should be reasonable", templateID)
		})
	}
}

func TestTemplateTemperature(t *testing.T) {
	// Test that all templates have reasonable temperature settings
	for templateID, template := range HEMATemplates {
		t.Run(templateID, func(t *testing.T) {
			assert.GreaterOrEqual(t, template.Temperature, float32(0.0), "Template %s temperature should be >= 0", templateID)
			assert.LessOrEqual(t, template.Temperature, float32(1.0), "Template %s temperature should be <= 1", templateID)
		})
	}
}

// Benchmark template selection
func BenchmarkTemplateSelector_SelectTemplate(b *testing.B) {
	selector := NewTemplateSelector()
	matchState := &context.MatchState{
		ScoreRed:  2,
		ScoreBlue: 1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = selector.SelectTemplate("Point left clean hit", matchState)
	}
}

// Benchmark template execution
func BenchmarkHEMATemplate_Execution(b *testing.B) {
	templateData := &TemplateData{
		Transcription: "Point left",
		MatchState: &context.MatchState{
			CurrentScore: "2-1",
			LastScorer:   "left",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ExecuteDefaultTemplate("point_scored", templateData)
		if err != nil {
			b.Fatal(err)
		}
	}
}
