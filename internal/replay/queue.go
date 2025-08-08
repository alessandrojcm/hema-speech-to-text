package replay

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/your-org/hema-replay-system/internal/config"
	"github.com/your-org/hema-replay-system/internal/obs"
	"github.com/your-org/hema-replay-system/pkg/logger"
)

// ReplayRequest represents a request to save a replay clip.
type ReplayRequest struct {
	ID        string
	Message   string
	Timestamp time.Time
	// Phase 1: Simplified - removed priority and metadata
}

// ReplayStatus represents the processing state of a replay request.
type ReplayStatus int

const (
	ReplayPending    ReplayStatus = iota // Request is queued for processing
	ReplayProcessing                     // Request is currently being processed
	ReplayCompleted                      // Request completed successfully
	ReplayFailed                         // Request failed to process
)

type ReplayResult struct {
	Request   ReplayRequest
	Status    ReplayStatus
	StartTime time.Time
	EndTime   time.Time
	Error     error
}

// Queue manages a queue of replay requests with asynchronous processing.
type Queue struct {
	config    config.ReplayConfig
	obsClient *obs.Client
	logger    *logger.Logger
	buffer    *Buffer

	queue      []ReplayRequest
	processing map[string]*ReplayResult
	completed  []ReplayResult
	mu         sync.RWMutex

	// Channels
	requestChan chan ReplayRequest
	resultChan  chan ReplayResult
	stopChan    chan struct{}

	// Metrics
	totalRequests  int
	successCount   int
	failureCount   int
	avgProcessTime time.Duration
}

// NewQueue creates a new replay request queue with the given configuration.
func NewQueue(config config.ReplayConfig, obsClient *obs.Client, logger *logger.Logger) *Queue {
	buffer := NewBuffer(config, obsClient, logger)

	return &Queue{
		config:      config,
		obsClient:   obsClient,
		logger:      logger,
		buffer:      buffer,
		queue:       make([]ReplayRequest, 0),
		processing:  make(map[string]*ReplayResult),
		completed:   make([]ReplayResult, 0),
		requestChan: make(chan ReplayRequest, config.QueueSize),
		resultChan:  make(chan ReplayResult, config.QueueSize),
		stopChan:    make(chan struct{}),
	}
}

func (q *Queue) Start(ctx context.Context) error {
	// Start replay buffer
	if err := q.buffer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start replay buffer: %w", err)
	}

	// Start queue processor
	go q.processQueue(ctx)

	q.logger.Info().
		Int("queue_size", q.config.QueueSize).
		Msg("Replay queue started")

	return nil
}

func (q *Queue) Stop(ctx context.Context) error {
	close(q.stopChan)

	// Stop replay buffer
	if err := q.buffer.Stop(ctx); err != nil {
		q.logger.Error().Err(err).Msg("Failed to stop replay buffer")
	}

	q.logger.Info().Msg("Replay queue stopped")
	return nil
}

func (q *Queue) AddRequest(request ReplayRequest) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Check queue size
	if len(q.queue) >= q.config.QueueSize {
		return fmt.Errorf("queue full, cannot add request")
	}

	// Generate ID if not provided
	if request.ID == "" {
		request.ID = fmt.Sprintf("replay_%d", time.Now().UnixNano())
	}

	request.Timestamp = time.Now()
	q.queue = append(q.queue, request)
	q.totalRequests++

	q.logger.Debug().
		Str("id", request.ID).
		Str("message", request.Message).
		Int("queue_size", len(q.queue)).
		Msg("Replay request added")

	return nil
}

func (q *Queue) processQueue(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-q.stopChan:
			return
		case <-ticker.C:
			q.processNextRequest(ctx)
		}
	}
}

func (q *Queue) processNextRequest(ctx context.Context) {
	q.mu.Lock()

	// Check if we can process more requests (Phase 1: simplified to 1 concurrent)
	if len(q.processing) >= 1 {
		q.mu.Unlock()
		return
	}

	// Get next request
	if len(q.queue) == 0 {
		q.mu.Unlock()
		return
	}

	request := q.queue[0]
	q.queue = q.queue[1:]

	// Mark as processing
	result := &ReplayResult{
		Request:   request,
		Status:    ReplayProcessing,
		StartTime: time.Now(),
	}
	q.processing[request.ID] = result

	q.mu.Unlock()

	// Process request asynchronously
	go q.executeReplay(ctx, request)
}

func (q *Queue) executeReplay(ctx context.Context, request ReplayRequest) {
	defer func() {
		q.mu.Lock()
		if result, exists := q.processing[request.ID]; exists {
			result.EndTime = time.Now()
			if result.Error == nil {
				result.Status = ReplayCompleted
				q.successCount++
			} else {
				result.Status = ReplayFailed
				q.failureCount++
			}
			q.completed = append(q.completed, *result)
			delete(q.processing, request.ID)

			// Update average processing time
			processTime := result.EndTime.Sub(result.StartTime)
			if q.avgProcessTime == 0 {
				q.avgProcessTime = processTime
			} else {
				q.avgProcessTime = (q.avgProcessTime + processTime) / 2
			}
		}
		q.mu.Unlock()
	}()

	q.logger.Info().
		Str("id", request.ID).
		Str("message", request.Message).
		Msg("Processing replay request")

	// Execute the replay
	if err := q.buffer.Save(ctx); err != nil {
		q.handleReplayError(request, err)
		return
	}

	q.logger.Info().
		Str("id", request.ID).
		Str("message", request.Message).
		Msg("Replay processed successfully")
}

func (q *Queue) handleReplayError(request ReplayRequest, err error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if result, exists := q.processing[request.ID]; exists {
		result.Error = err
	}

	q.logger.Error().
		Str("id", request.ID).
		Err(err).
		Msg("Replay processing failed")
}

func (q *Queue) GetQueueInfo() QueueInfo {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return QueueInfo{
		QueueSize:       len(q.queue),
		ProcessingCount: len(q.processing),
		CompletedCount:  len(q.completed),
		TotalRequests:   q.totalRequests,
		SuccessCount:    q.successCount,
		FailureCount:    q.failureCount,
		AvgProcessTime:  q.avgProcessTime,
		BufferInfo:      q.buffer.GetInfo(),
	}
}

func (q *Queue) GetResults(limit int) []ReplayResult {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if limit <= 0 || limit > len(q.completed) {
		limit = len(q.completed)
	}

	results := make([]ReplayResult, limit)
	start := len(q.completed) - limit
	copy(results, q.completed[start:])

	return results
}

func (q *Queue) ClearCompleted() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.completed = make([]ReplayResult, 0)
	q.logger.Info().Msg("Completed replay results cleared")
}

type QueueInfo struct {
	QueueSize       int
	ProcessingCount int
	CompletedCount  int
	TotalRequests   int
	SuccessCount    int
	FailureCount    int
	AvgProcessTime  time.Duration
	BufferInfo      BufferInfo
}
