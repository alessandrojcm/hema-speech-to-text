package pipeline

import (
	"fmt"
	"sync"
	"time"

	"github.com/your-org/hema-replay-system/pkg/audio/types"
	"github.com/your-org/hema-replay-system/pkg/pipeline/vad"
	speechTypes "github.com/your-org/hema-replay-system/pkg/speech/types"
)

// SegmentStatus represents the processing status of a segment
type SegmentStatus int

const (
	SegmentStatusPending SegmentStatus = iota
	SegmentStatusProcessing
	SegmentStatusProcessed
	SegmentStatusFailed
)

// BufferedSegment represents a segment with its processing status and metadata
type BufferedSegment struct {
	ID          string
	Segment     *types.AudioSegment
	VADEvent    vad.VADEvent
	Status      SegmentStatus
	Result      *speechTypes.TranscriptionResult
	CreatedAt   time.Time
	ProcessedAt *time.Time
	Error       error
}

// SegmentBuffer manages audio segments pending and completed processing
type SegmentBuffer struct {
	segments  map[string]*BufferedSegment
	pending   []string // Queue of pending segment IDs
	processed []string // Queue of processed segment IDs
	maxSize   int
	mu        sync.RWMutex
	nextID    int64
}

// NewSegmentBuffer creates a new segment buffer
func NewSegmentBuffer(maxSize int) *SegmentBuffer {
	return &SegmentBuffer{
		segments:  make(map[string]*BufferedSegment),
		pending:   make([]string, 0),
		processed: make([]string, 0),
		maxSize:   maxSize,
		nextID:    1,
	}
}

// Add adds a new segment to the buffer and returns its ID
func (sb *SegmentBuffer) Add(segment *types.AudioSegment, vadEvent vad.VADEvent) string {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	// Generate unique ID
	id := fmt.Sprintf("segment_%d", sb.nextID)
	sb.nextID++

	// Create buffered segment
	bufferedSegment := &BufferedSegment{
		ID:        id,
		Segment:   segment,
		VADEvent:  vadEvent,
		Status:    SegmentStatusPending,
		CreatedAt: time.Now(),
	}

	// Add to segments map and pending queue
	sb.segments[id] = bufferedSegment
	sb.pending = append(sb.pending, id)

	// Clean up old segments if buffer is full
	sb.cleanup()

	return id
}

// GetPending returns all pending segments for processing
func (sb *SegmentBuffer) GetPending() []*AudioSegmentData {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	result := make([]*AudioSegmentData, 0, len(sb.pending))
	for _, id := range sb.pending {
		if segment, ok := sb.segments[id]; ok && segment.Status == SegmentStatusPending {
			result = append(result, &AudioSegmentData{
				SegmentID: id,
				Segment:   segment.Segment,
				VADEvent:  segment.VADEvent,
			})
		}
	}

	return result
}

// MarkProcessing marks a segment as currently being processed
func (sb *SegmentBuffer) MarkProcessing(id string) error {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	segment, ok := sb.segments[id]
	if !ok {
		return fmt.Errorf("segment %s not found", id)
	}

	if segment.Status != SegmentStatusPending {
		return fmt.Errorf("segment %s is not pending (status: %d)", id, segment.Status)
	}

	segment.Status = SegmentStatusProcessing
	return nil
}

// MarkProcessed marks a segment as processed with the result
func (sb *SegmentBuffer) MarkProcessed(id string, result *speechTypes.TranscriptionResult) error {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	segment, ok := sb.segments[id]
	if !ok {
		return fmt.Errorf("segment %s not found", id)
	}

	segment.Status = SegmentStatusProcessed
	segment.Result = result
	now := time.Now()
	segment.ProcessedAt = &now

	// Move from pending to processed
	sb.removeFromPending(id)
	sb.processed = append(sb.processed, id)

	return nil
}

// MarkFailed marks a segment as failed with an error
func (sb *SegmentBuffer) MarkFailed(id string, err error) error {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	segment, ok := sb.segments[id]
	if !ok {
		return fmt.Errorf("segment %s not found", id)
	}

	segment.Status = SegmentStatusFailed
	segment.Error = err
	now := time.Now()
	segment.ProcessedAt = &now

	// Move from pending to processed (even failed ones)
	sb.removeFromPending(id)
	sb.processed = append(sb.processed, id)

	return nil
}

// GetSegment returns a segment by ID
func (sb *SegmentBuffer) GetSegment(id string) (*BufferedSegment, bool) {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	segment, ok := sb.segments[id]
	return segment, ok
}

// GetProcessedResults returns all processed transcription results
func (sb *SegmentBuffer) GetProcessedResults() []*speechTypes.TranscriptionResult {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	results := make([]*speechTypes.TranscriptionResult, 0)
	for _, id := range sb.processed {
		if segment, ok := sb.segments[id]; ok && segment.Result != nil {
			results = append(results, segment.Result)
		}
	}

	return results
}

// GetPendingCount returns the number of pending segments
func (sb *SegmentBuffer) GetPendingCount() int {
	sb.mu.RLock()
	defer sb.mu.RUnlock()
	return len(sb.pending)
}

// GetProcessedCount returns the number of processed segments
func (sb *SegmentBuffer) GetProcessedCount() int {
	sb.mu.RLock()
	defer sb.mu.RUnlock()
	return len(sb.processed)
}

// GetTotalCount returns the total number of segments in buffer
func (sb *SegmentBuffer) GetTotalCount() int {
	sb.mu.RLock()
	defer sb.mu.RUnlock()
	return len(sb.segments)
}

// Clear clears all segments from the buffer
func (sb *SegmentBuffer) Clear() {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	sb.segments = make(map[string]*BufferedSegment)
	sb.pending = make([]string, 0)
	sb.processed = make([]string, 0)
}

// cleanup removes old segments when buffer exceeds maximum size
func (sb *SegmentBuffer) cleanup() {
	if len(sb.segments) <= sb.maxSize {
		return
	}

	// Remove oldest processed segments first
	removeCount := len(sb.segments) - sb.maxSize
	removed := 0

	// Remove from processed queue
	for i := 0; i < len(sb.processed) && removed < removeCount; i++ {
		id := sb.processed[i]
		delete(sb.segments, id)
		removed++
	}

	// Update processed queue
	if removed > 0 {
		if removed >= len(sb.processed) {
			sb.processed = make([]string, 0)
		} else {
			sb.processed = sb.processed[removed:]
		}
	}

	// If still over limit, remove oldest pending (this shouldn't normally happen)
	if len(sb.segments) > sb.maxSize && len(sb.pending) > 0 {
		remainingToRemove := len(sb.segments) - sb.maxSize
		for i := 0; i < len(sb.pending) && i < remainingToRemove; i++ {
			id := sb.pending[i]
			delete(sb.segments, id)
		}
		if remainingToRemove >= len(sb.pending) {
			sb.pending = make([]string, 0)
		} else {
			sb.pending = sb.pending[remainingToRemove:]
		}
	}
}

// removeFromPending removes an ID from the pending queue
func (sb *SegmentBuffer) removeFromPending(targetID string) {
	for i, id := range sb.pending {
		if id == targetID {
			// Remove from slice
			sb.pending = append(sb.pending[:i], sb.pending[i+1:]...)
			break
		}
	}
}

// GetStats returns buffer statistics
func (sb *SegmentBuffer) GetStats() map[string]interface{} {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	stats := map[string]interface{}{
		"total_segments":     len(sb.segments),
		"pending_segments":   len(sb.pending),
		"processed_segments": len(sb.processed),
		"max_size":           sb.maxSize,
		"next_id":            sb.nextID,
	}

	// Count by status
	statusCounts := map[SegmentStatus]int{}
	for _, segment := range sb.segments {
		statusCounts[segment.Status]++
	}

	stats["status_pending"] = statusCounts[SegmentStatusPending]
	stats["status_processing"] = statusCounts[SegmentStatusProcessing]
	stats["status_processed"] = statusCounts[SegmentStatusProcessed]
	stats["status_failed"] = statusCounts[SegmentStatusFailed]

	return stats
}
