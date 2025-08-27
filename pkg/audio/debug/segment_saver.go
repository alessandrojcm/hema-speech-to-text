package debug

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/rs/zerolog"
)

type SegmentSaver struct {
	outputDir string
	enabled   bool
	logger    zerolog.Logger
}

func NewSegmentSaver(outputDir string, enabled bool, logger zerolog.Logger) *SegmentSaver {
	if enabled {
		os.MkdirAll(outputDir, 0755)
	}

	return &SegmentSaver{
		outputDir: outputDir,
		enabled:   enabled,
		logger:    logger.With().Str("component", "segment_saver").Logger(),
	}
}

func (ss *SegmentSaver) SaveSegment(samples []float32, metadata map[string]interface{}) error {
	if !ss.enabled {
		return nil
	}

	timestamp := time.Now().Format("20060102_150405")
	segmentType := "unknown"
	if t, ok := metadata["type"].(string); ok {
		segmentType = t
	}

	filename := fmt.Sprintf("%s_%s.wav", timestamp, segmentType)
	filePath := filepath.Join(ss.outputDir, filename)

	// Also save metadata as JSON
	metaFile := fmt.Sprintf("%s_%s.json", timestamp, segmentType)
	metaPath := filepath.Join(ss.outputDir, metaFile)

	// Convert float32 to int for WAV encoding
	intSamples := make([]int, len(samples))
	for i, s := range samples {
		intSamples[i] = int(s * 32767)
	}

	// Save WAV file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create debug audio file: %w", err)
	}
	defer file.Close()

	encoder := wav.NewEncoder(file, 16000, 16, 1, 1)
	defer encoder.Close()

	buf := &audio.IntBuffer{
		Data:   intSamples,
		Format: &audio.Format{SampleRate: 16000, NumChannels: 1},
	}

	if err := encoder.Write(buf); err != nil {
		return fmt.Errorf("failed to write debug audio: %w", err)
	}

	// Save metadata as JSON
	if err := ss.saveMetadataJSON(metaPath, metadata); err != nil {
		ss.logger.Warn().Err(err).Str("meta_file", metaFile).Msg("Failed to save metadata JSON")
	}

	ss.logger.Debug().
		Str("file", filename).
		Int("samples", len(samples)).
		Interface("metadata", metadata).
		Msg("Saved debug audio segment")

	return nil
}

// saveMetadataJSON saves metadata to a JSON file
func (ss *SegmentSaver) saveMetadataJSON(filePath string, metadata map[string]interface{}) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create metadata file: %w", err)
	}
	defer file.Close()

	// Clean metadata to handle infinity and NaN values
	cleanedMetadata := ss.cleanMetadataForJSON(metadata)

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(cleanedMetadata); err != nil {
		return fmt.Errorf("failed to encode metadata as JSON: %w", err)
	}

	return nil
}

// cleanMetadataForJSON recursively cleans metadata to handle infinity and NaN values
func (ss *SegmentSaver) cleanMetadataForJSON(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		cleaned := make(map[string]interface{})
		for k, val := range v {
			cleaned[k] = ss.cleanMetadataForJSON(val)
		}
		return cleaned
	case map[string]float32:
		cleaned := make(map[string]interface{})
		for k, val := range v {
			cleaned[k] = ss.cleanFloatValue(float64(val))
		}
		return cleaned
	case []interface{}:
		cleaned := make([]interface{}, len(v))
		for i, val := range v {
			cleaned[i] = ss.cleanMetadataForJSON(val)
		}
		return cleaned
	case float64:
		return ss.cleanFloatValue(v)
	case float32:
		return ss.cleanFloatValue(float64(v))
	default:
		return v
	}
}

// cleanFloatValue converts infinity and NaN to JSON-safe values
func (ss *SegmentSaver) cleanFloatValue(val float64) interface{} {
	switch {
	case math.IsNaN(val):
		return "NaN"
	case math.IsInf(val, 1): // Positive infinity
		return "Infinity"
	case math.IsInf(val, -1): // Negative infinity
		return "-Infinity"
	default:
		return val
	}
}
