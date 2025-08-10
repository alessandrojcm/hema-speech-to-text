package types

import (
	"fmt"
	"time"

	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

// TranscriptionRequest represents a request for speech transcription
type TranscriptionRequest struct {
	ID                  string
	AudioSegment        *types.AudioSegment
	Language            string
	ModelSize           ModelSize
	UseVocabulary       bool
	ConfidenceThreshold float64
	MaxDuration         time.Duration
	Context             map[string]interface{}
}

// TranscriptionResult represents the result of speech transcription
type TranscriptionResult struct {
	ID          string
	Text        string
	Confidence  float64
	Language    string
	Duration    time.Duration
	Segments    []TranscriptionSegment
	Metadata    TranscriptionMetadata
	ProcessedAt time.Time
}

// TranscriptionSegment represents a segment of transcribed text
type TranscriptionSegment struct {
	Text       string
	StartTime  time.Duration
	EndTime    time.Duration
	Confidence float64
	Tokens     []Token
}

// Token represents a single transcribed token
type Token struct {
	Text       string
	Confidence float64
	StartTime  time.Duration
	EndTime    time.Duration
	IsHEMA     bool // Indicates if token is HEMA terminology
}

// TranscriptionMetadata contains processing metadata
type TranscriptionMetadata struct {
	ModelUsed        string
	ProcessingTime   time.Duration
	AudioQuality     float64
	MetalAccelerated bool
	VocabularyBoost  bool
	MemoryUsage      int64
	TokenCount       int
	HEMATermsFound   []string
}

// ModelSize represents whisper model sizes
type ModelSize int

const (
	ModelTiny ModelSize = iota
	ModelBase
	ModelSmall
	ModelMedium
	ModelLarge
)

func (ms ModelSize) String() string {
	switch ms {
	case ModelTiny:
		return "tiny"
	case ModelBase:
		return "base"
	case ModelSmall:
		return "small"
	case ModelMedium:
		return "medium"
	case ModelLarge:
		return "large"
	default:
		return "unknown"
	}
}

// UnmarshalText implements the encoding.TextUnmarshaler interface
func (ms *ModelSize) UnmarshalText(text []byte) error {
	switch string(text) {
	case "tiny":
		*ms = ModelTiny
	case "base":
		*ms = ModelBase
	case "small":
		*ms = ModelSmall
	case "medium":
		*ms = ModelMedium
	case "large":
		*ms = ModelLarge
	default:
		return fmt.Errorf("unknown model size: %s", string(text))
	}
	return nil
}

// WhisperParams represents parameters for whisper transcription
type WhisperParams struct {
	Language       string
	ThreadCount    int
	Temperature    float32
	BeamSize       int
	MaxTokens      int
	WordTimestamps bool
}
