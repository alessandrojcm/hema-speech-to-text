package templates

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/your-org/hema-replay-system/pkg/commentary/context"
)

func TestTemplateManager_RegisterTemplate(t *testing.T) {
	tm := NewTemplateManager()

	tests := []struct {
		name        string
		template    *PromptTemplate
		expectError bool
	}{
		{
			name: "valid_template",
			template: &PromptTemplate{
				ID:       "test_template",
				Name:     "Test Template",
				Template: "Hello {{.Name}}!",
			},
			expectError: false,
		},
		{
			name: "empty_id",
			template: &PromptTemplate{
				Name:     "Test Template",
				Template: "Hello world!",
			},
			expectError: true,
		},
		{
			name: "empty_template",
			template: &PromptTemplate{
				ID:   "test_template",
				Name: "Test Template",
			},
			expectError: true,
		},
		{
			name: "invalid_template_syntax",
			template: &PromptTemplate{
				ID:       "test_template",
				Name:     "Test Template",
				Template: "Hello {{.Name}!", // Missing closing brace
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tm.RegisterTemplate(tt.template)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify template is registered
				retrieved, exists := tm.GetTemplate(tt.template.ID)
				assert.True(t, exists)
				assert.Equal(t, tt.template.ID, retrieved.ID)
				assert.Equal(t, tt.template.Name, retrieved.Name)
			}
		})
	}
}

func TestTemplateManager_ExecuteTemplate(t *testing.T) {
	tm := NewTemplateManager()

	// Register test template
	template := &PromptTemplate{
		ID:       "test_template",
		Name:     "Test Template",
		Template: "Hello {{.Name}}! Score: {{.Score}}",
	}
	err := tm.RegisterTemplate(template)
	require.NoError(t, err)

	tests := []struct {
		name        string
		templateID  string
		data        interface{}
		expected    string
		expectError bool
	}{
		{
			name:       "valid_execution",
			templateID: "test_template",
			data: map[string]interface{}{
				"Name":  "John",
				"Score": 5,
			},
			expected:    "Hello John! Score: 5",
			expectError: false,
		},
		{
			name:        "nonexistent_template",
			templateID:  "nonexistent",
			data:        map[string]interface{}{},
			expectError: true,
		},
		{
			name:       "missing_data_field",
			templateID: "test_template",
			data: map[string]interface{}{
				"Name": "John",
				// Missing Score
			},
			expected:    "Hello John! Score: <no value>",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tm.ExecuteTemplate(tt.templateID, tt.data)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestTemplateManager_ListTemplates(t *testing.T) {
	tm := NewTemplateManager()

	// Initially empty
	templates := tm.ListTemplates()
	assert.Empty(t, templates)

	// Add templates
	template1 := &PromptTemplate{
		ID:       "template1",
		Template: "Template 1",
	}
	template2 := &PromptTemplate{
		ID:       "template2",
		Template: "Template 2",
	}

	err := tm.RegisterTemplate(template1)
	require.NoError(t, err)
	err = tm.RegisterTemplate(template2)
	require.NoError(t, err)

	templates = tm.ListTemplates()
	assert.Len(t, templates, 2)
	assert.Contains(t, templates, "template1")
	assert.Contains(t, templates, "template2")
}

func TestValidateTemplateData(t *testing.T) {
	tests := []struct {
		name        string
		data        *TemplateData
		expectError bool
	}{
		{
			name: "valid_data",
			data: &TemplateData{
				Transcription: "Point left",
				Confidence:    0.95,
				Timestamp:     time.Now(),
			},
			expectError: false,
		},
		{
			name:        "nil_data",
			data:        nil,
			expectError: true,
		},
		{
			name: "empty_transcription",
			data: &TemplateData{
				Transcription: "",
				Confidence:    0.95,
			},
			expectError: true,
		},
		{
			name: "invalid_confidence_low",
			data: &TemplateData{
				Transcription: "Point left",
				Confidence:    -0.1,
			},
			expectError: true,
		},
		{
			name: "invalid_confidence_high",
			data: &TemplateData{
				Transcription: "Point left",
				Confidence:    1.1,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTemplateData(tt.data)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultTemplateManager(t *testing.T) {
	// Test that the default template manager exists and works
	template := &PromptTemplate{
		ID:       "test_default",
		Template: "Default test",
	}

	err := RegisterDefaultTemplate(template)
	assert.NoError(t, err)

	retrieved, exists := GetDefaultTemplate("test_default")
	assert.True(t, exists)
	assert.Equal(t, "test_default", retrieved.ID)

	result, err := ExecuteDefaultTemplate("test_default", nil)
	assert.NoError(t, err)
	assert.Equal(t, "Default test", result)
}

func TestTemplateDataTypes(t *testing.T) {
	// Test that we can create template data with all field types
	matchState := &context.MatchState{}
	audioMetrics := &AudioMetrics{
		Volume:       0.8,
		Clarity:      0.9,
		NoiseLevel:   0.1,
		VoicePresent: true,
	}

	data := &TemplateData{
		Transcription: "Point left",
		Confidence:    0.95,
		Timestamp:     time.Now(),
		MatchState:    matchState,
		Context:       map[string]string{"test": "value"},
		AudioMetrics:  audioMetrics,
		PreviousCalls: []string{"halt", "fence"},
	}

	// Validate the data structure
	assert.Equal(t, "Point left", data.Transcription)
	assert.Equal(t, float32(0.95), data.Confidence)
	assert.NotNil(t, data.MatchState)
	assert.NotNil(t, data.Context)
	assert.NotNil(t, data.AudioMetrics)
	assert.NotNil(t, data.PreviousCalls)

	// Test validation
	err := ValidateTemplateData(data)
	assert.NoError(t, err)
}

func TestAudioMetrics(t *testing.T) {
	metrics := &AudioMetrics{
		Volume:       0.75,
		Clarity:      0.85,
		NoiseLevel:   0.15,
		VoicePresent: true,
	}

	assert.Equal(t, float32(0.75), metrics.Volume)
	assert.Equal(t, float32(0.85), metrics.Clarity)
	assert.Equal(t, float32(0.15), metrics.NoiseLevel)
	assert.True(t, metrics.VoicePresent)
}

// Benchmark tests
func BenchmarkTemplateExecution(b *testing.B) {
	tm := NewTemplateManager()
	template := &PromptTemplate{
		ID:       "benchmark_template",
		Template: "Commentary for {{.Transcription}} with confidence {{.Confidence}}",
	}
	err := tm.RegisterTemplate(template)
	require.NoError(b, err)

	data := map[string]interface{}{
		"Transcription": "Point left",
		"Confidence":    0.95,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tm.ExecuteTemplate("benchmark_template", data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTemplateRegistration(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tm := NewTemplateManager()
		template := &PromptTemplate{
			ID:       "benchmark_template",
			Template: "Test template {{.Field}}",
		}
		err := tm.RegisterTemplate(template)
		if err != nil {
			b.Fatal(err)
		}
	}
}
