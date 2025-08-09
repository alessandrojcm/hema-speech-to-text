package vocabulary

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestVocabularyFile(t *testing.T, content string) string {
	tmpDir := t.TempDir()
	vocabFile := filepath.Join(tmpDir, "test_vocab.txt")

	err := os.WriteFile(vocabFile, []byte(content), 0644)
	require.NoError(t, err)

	return vocabFile
}

func createTestLogger() zerolog.Logger {
	return zerolog.Nop() // No-op logger for tests
}

func TestNewHEMAVocabulary(t *testing.T) {
	logger := createTestLogger()
	vocab := NewHEMAVocabulary(logger)

	assert.NotNil(t, vocab)
	assert.NotNil(t, vocab.logger)
}

func TestHEMAVocabulary_LoadFromFile_ValidFile(t *testing.T) {
	content := `# HEMA Vocabulary Test File
# Format: term|category|boost|phonetic|context
point|scoring|1.2||
touch|scoring|1.2||
halt|control|1.5||
stop|control|1.5||
longsword|weapon|1.1||
rapier|weapon|1.1||
sabre|weapon|1.1||
thrust|action|1.3||
cut|action|1.3||
parry|action|1.3||
riposte|action|1.4||
head|target|1.0||
torso|target|1.0||
arm|target|1.0||
leg|target|1.0||
director|official|1.2||
judge|official|1.2||
referee|official|1.2||`

	vocabFile := createTestVocabularyFile(t, content)
	logger := createTestLogger()
	vocab := NewHEMAVocabulary(logger)

	err := vocab.LoadFromFile(vocabFile)
	assert.NoError(t, err)

	// Test that terms were loaded
	assert.True(t, vocab.IsHEMATerm("point"))
	assert.True(t, vocab.IsHEMATerm("longsword"))
	assert.True(t, vocab.IsHEMATerm("thrust"))
	assert.True(t, vocab.IsHEMATerm("director"))

	// Test case insensitivity
	assert.True(t, vocab.IsHEMATerm("POINT"))
	assert.True(t, vocab.IsHEMATerm("Point"))

	// Test non-HEMA terms
	assert.False(t, vocab.IsHEMATerm("hello"))
	assert.False(t, vocab.IsHEMATerm("world"))

	// Test boosts
	assert.Equal(t, 1.2, vocab.GetBoost("point"))
	assert.Equal(t, 1.5, vocab.GetBoost("halt"))
}
func TestHEMAVocabulary_LoadFromFile_NonExistentFile(t *testing.T) {
	logger := createTestLogger()
	vocab := NewHEMAVocabulary(logger)

	err := vocab.LoadFromFile("/nonexistent/path/vocab.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open vocabulary file")
}

func TestHEMAVocabulary_LoadFromFile_EmptyFile(t *testing.T) {
	vocabFile := createTestVocabularyFile(t, "")
	logger := createTestLogger()
	vocab := NewHEMAVocabulary(logger)

	err := vocab.LoadFromFile(vocabFile)
	assert.NoError(t, err)

	// Should handle empty file gracefully
	assert.False(t, vocab.IsHEMATerm("anything"))
}

func TestHEMAVocabulary_LoadFromFile_CommentsAndEmptyLines(t *testing.T) {
	content := `# This is a comment
# Another comment

point|scoring
# Comment in middle
touch|scoring

# Final comment`

	vocabFile := createTestVocabularyFile(t, content)
	logger := createTestLogger()
	vocab := NewHEMAVocabulary(logger)

	err := vocab.LoadFromFile(vocabFile)
	assert.NoError(t, err)

	// Should only load non-comment, non-empty lines
	assert.True(t, vocab.IsHEMATerm("point"))
	assert.True(t, vocab.IsHEMATerm("touch"))
	assert.False(t, vocab.IsHEMATerm("# This is a comment"))
	assert.False(t, vocab.IsHEMATerm(""))
}
func TestHEMAVocabulary_AddTerm(t *testing.T) {
	logger := createTestLogger()
	vocab := NewHEMAVocabulary(logger)

	// Initially should not have the term
	assert.False(t, vocab.IsHEMATerm("custom"))

	// Add the term
	customTerm := VocabularyTerm{
		Term:     "custom",
		Category: "test",
		Boost:    1.2,
	}
	vocab.AddTerm(customTerm)

	// Should now have the term
	assert.True(t, vocab.IsHEMATerm("custom"))
	assert.True(t, vocab.IsHEMATerm("CUSTOM")) // Case insensitive
}

func TestHEMAVocabulary_GetBoost_DefaultBoost(t *testing.T) {
	logger := createTestLogger()
	vocab := NewHEMAVocabulary(logger)

	pointTerm := VocabularyTerm{
		Term:     "point",
		Category: "scoring",
		Boost:    1.2,
	}
	vocab.AddTerm(pointTerm)

	boost := vocab.GetBoost("point")
	assert.Equal(t, 1.2, boost)
}

func TestHEMAVocabulary_UpdateBoost(t *testing.T) {
	logger := createTestLogger()
	vocab := NewHEMAVocabulary(logger)

	pointTerm := VocabularyTerm{
		Term:     "point",
		Category: "scoring",
		Boost:    1.2,
	}
	vocab.AddTerm(pointTerm)

	// Update boost
	vocab.UpdateBoost("point", 1.5)

	boost := vocab.GetBoost("point")
	assert.Equal(t, 1.5, boost)
}
func TestHEMAVocabulary_GetBoost_NonExistentTerm(t *testing.T) {
	logger := createTestLogger()
	vocab := NewHEMAVocabulary(logger)

	boost := vocab.GetBoost("nonexistent")
	assert.Equal(t, 1.0, boost) // Should return 1.0 for non-HEMA terms
}

func TestHEMAVocabulary_GetAllTerms(t *testing.T) {
	logger := createTestLogger()
	vocab := NewHEMAVocabulary(logger)

	terms := []VocabularyTerm{
		{Term: "point", Category: "scoring", Boost: 1.2},
		{Term: "touch", Category: "scoring", Boost: 1.2},
		{Term: "halt", Category: "control", Boost: 1.5},
	}

	for _, term := range terms {
		vocab.AddTerm(term)
	}

	allTerms := vocab.GetAllTerms()

	assert.Len(t, allTerms, len(terms))
	assert.Contains(t, allTerms, "point")
	assert.Contains(t, allTerms, "touch")
	assert.Contains(t, allTerms, "halt")
}

func TestHEMAVocabulary_GetStats(t *testing.T) {
	logger := createTestLogger()
	vocab := NewHEMAVocabulary(logger)

	terms := []VocabularyTerm{
		{Term: "point", Category: "scoring", Boost: 1.2},
		{Term: "touch", Category: "scoring", Boost: 1.2},
		{Term: "halt", Category: "control", Boost: 1.5},
	}

	for _, term := range terms {
		vocab.AddTerm(term)
	}

	stats := vocab.GetStats()

	assert.Contains(t, stats, "total_terms")
	assert.Contains(t, stats, "total_categories")
	assert.Equal(t, 3, stats["total_terms"])
	assert.Equal(t, 2, stats["total_categories"]) // scoring and control
}

func TestHEMAVocabulary_GetTermsByCategory(t *testing.T) {
	logger := createTestLogger()
	vocab := NewHEMAVocabulary(logger)

	terms := []VocabularyTerm{
		{Term: "point", Category: "scoring", Boost: 1.2},
		{Term: "touch", Category: "scoring", Boost: 1.2},
		{Term: "halt", Category: "control", Boost: 1.5},
	}

	for _, term := range terms {
		vocab.AddTerm(term)
	}

	scoringTerms := vocab.GetTermsByCategory("scoring")
	assert.Len(t, scoringTerms, 2)
	assert.Contains(t, scoringTerms, "point")
	assert.Contains(t, scoringTerms, "touch")

	controlTerms := vocab.GetTermsByCategory("control")
	assert.Len(t, controlTerms, 1)
	assert.Contains(t, controlTerms, "halt")

	// Non-existent category
	nonExistent := vocab.GetTermsByCategory("nonexistent")
	assert.Len(t, nonExistent, 0)
}

func TestHEMAVocabulary_CaseInsensitivity(t *testing.T) {
	logger := createTestLogger()
	vocab := NewHEMAVocabulary(logger)

	pointTerm := VocabularyTerm{
		Term:     "Point",
		Category: "scoring",
		Boost:    1.2,
	}
	vocab.AddTerm(pointTerm)

	// All variations should work
	assert.True(t, vocab.IsHEMATerm("point"))
	assert.True(t, vocab.IsHEMATerm("POINT"))
	assert.True(t, vocab.IsHEMATerm("Point"))
	assert.True(t, vocab.IsHEMATerm("pOiNt"))

	// Boost should also be case insensitive
	vocab.UpdateBoost("POINT", 1.8)
	assert.Equal(t, 1.8, vocab.GetBoost("point"))
	assert.Equal(t, 1.8, vocab.GetBoost("Point"))
}
func TestHEMAVocabulary_WhitespaceHandling(t *testing.T) {
	content := `  point|scoring  
	touch|scoring	
halt|control   
  stop|control`

	vocabFile := createTestVocabularyFile(t, content)
	logger := createTestLogger()
	vocab := NewHEMAVocabulary(logger)

	err := vocab.LoadFromFile(vocabFile)
	assert.NoError(t, err)

	// Should handle whitespace correctly
	assert.True(t, vocab.IsHEMATerm("point"))
	assert.True(t, vocab.IsHEMATerm("touch"))
	assert.True(t, vocab.IsHEMATerm("halt"))
	assert.True(t, vocab.IsHEMATerm("stop"))
}
func TestHEMAVocabulary_DuplicateTerms(t *testing.T) {
	logger := createTestLogger()
	vocab := NewHEMAVocabulary(logger)

	// Add same term multiple times (case variations)
	pointTerm1 := VocabularyTerm{Term: "point", Category: "scoring", Boost: 1.2}
	pointTerm2 := VocabularyTerm{Term: "point", Category: "scoring", Boost: 1.3} // Different boost
	pointTerm3 := VocabularyTerm{Term: "POINT", Category: "scoring", Boost: 1.4} // Different case

	vocab.AddTerm(pointTerm1)
	vocab.AddTerm(pointTerm2) // Should overwrite
	vocab.AddTerm(pointTerm3) // Should overwrite (case insensitive)

	stats := vocab.GetStats()
	assert.Equal(t, 1, stats["total_terms"]) // Should only count once

	assert.True(t, vocab.IsHEMATerm("point"))
	// Should have the last boost value
	assert.Equal(t, 1.4, vocab.GetBoost("point"))
}

func TestHEMAVocabulary_PhoneticVariations(t *testing.T) {
	logger := createTestLogger()
	vocab := NewHEMAVocabulary(logger)

	// Add term with phonetic variations
	pointTerm := VocabularyTerm{
		Term:     "point",
		Category: "scoring",
		Boost:    1.2,
		Phonetic: []string{"poynt", "pont"},
	}
	vocab.AddTerm(pointTerm)

	// Original term should work
	assert.True(t, vocab.IsHEMATerm("point"))
	assert.Equal(t, 1.2, vocab.GetBoost("point"))

	// Phonetic variations should also work
	assert.True(t, vocab.IsHEMATerm("poynt"))
	assert.True(t, vocab.IsHEMATerm("pont"))
	assert.Equal(t, 1.2, vocab.GetBoost("poynt"))
	assert.Equal(t, 1.2, vocab.GetBoost("pont"))
}
