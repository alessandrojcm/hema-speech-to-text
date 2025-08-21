package templates

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/your-org/hema-replay-system/pkg/commentary/context"
)

// PromptTemplate represents a template for generating commentary prompts
type PromptTemplate struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Template    string            `json:"template"`
	Variables   []string          `json:"variables"`
	MaxTokens   int               `json:"max_tokens"`
	Temperature float32           `json:"temperature"`
	Metadata    map[string]string `json:"metadata"`
}

// TemplateManager manages prompt templates and their execution
type TemplateManager struct {
	templates map[string]*PromptTemplate
	compiled  map[string]*template.Template
}

// NewTemplateManager creates a new template manager
func NewTemplateManager() *TemplateManager {
	return &TemplateManager{
		templates: make(map[string]*PromptTemplate),
		compiled:  make(map[string]*template.Template),
	}
}

// RegisterTemplate registers a new template with the manager
func (tm *TemplateManager) RegisterTemplate(tmpl *PromptTemplate) error {
	if tmpl.ID == "" {
		return fmt.Errorf("template ID cannot be empty")
	}

	if tmpl.Template == "" {
		return fmt.Errorf("template content cannot be empty")
	}

	// Compile the template to check for syntax errors
	compiled, err := template.New(tmpl.ID).Parse(tmpl.Template)
	if err != nil {
		return fmt.Errorf("failed to compile template %s: %w", tmpl.ID, err)
	}

	tm.templates[tmpl.ID] = tmpl
	tm.compiled[tmpl.ID] = compiled

	return nil
}

// GetTemplate retrieves a template by ID
func (tm *TemplateManager) GetTemplate(id string) (*PromptTemplate, bool) {
	tmpl, exists := tm.templates[id]
	return tmpl, exists
}

// ListTemplates returns all registered template IDs
func (tm *TemplateManager) ListTemplates() []string {
	ids := make([]string, 0, len(tm.templates))
	for id := range tm.templates {
		ids = append(ids, id)
	}
	return ids
}

// ExecuteTemplate executes a template with the given data
func (tm *TemplateManager) ExecuteTemplate(id string, data interface{}) (string, error) {
	compiled, exists := tm.compiled[id]
	if !exists {
		return "", fmt.Errorf("template %s not found", id)
	}

	var buf bytes.Buffer
	if err := compiled.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", id, err)
	}

	return strings.TrimSpace(buf.String()), nil
}

// TemplateData represents the data structure for template execution
type TemplateData struct {
	Transcription string              `json:"transcription"`
	Confidence    float32             `json:"confidence"`
	Timestamp     time.Time           `json:"timestamp"`
	MatchState    *context.MatchState `json:"match_state,omitempty"`
	Context       map[string]string   `json:"context,omitempty"`
	AudioMetrics  *AudioMetrics       `json:"audio_metrics,omitempty"`
	PreviousCalls []string            `json:"previous_calls,omitempty"`
}

// AudioMetrics represents audio quality metrics from the speech system
type AudioMetrics struct {
	Volume       float32 `json:"volume"`
	Clarity      float32 `json:"clarity"`
	NoiseLevel   float32 `json:"noise_level"`
	VoicePresent bool    `json:"voice_present"`
}

// ValidateTemplateData validates template data for completeness
func ValidateTemplateData(data *TemplateData) error {
	if data == nil {
		return fmt.Errorf("template data cannot be nil")
	}

	if data.Transcription == "" {
		return fmt.Errorf("transcription text is required")
	}

	if data.Confidence < 0 || data.Confidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1, got %f", data.Confidence)
	}

	return nil
}

// DefaultTemplateManager TemplateRegistry provides a global registry for templates
var DefaultTemplateManager = NewTemplateManager()

// RegisterDefaultTemplate registers a template with the default manager
func RegisterDefaultTemplate(tmpl *PromptTemplate) error {
	return DefaultTemplateManager.RegisterTemplate(tmpl)
}

// GetDefaultTemplate retrieves a template from the default manager
func GetDefaultTemplate(id string) (*PromptTemplate, bool) {
	return DefaultTemplateManager.GetTemplate(id)
}

// ExecuteDefaultTemplate executes a template using the default manager
func ExecuteDefaultTemplate(id string, data interface{}) (string, error) {
	return DefaultTemplateManager.ExecuteTemplate(id, data)
}
