# Phase 1 API Documentation

## Overview

This document describes the API and interfaces for Phase 1 of the HEMA Tournament Replay System.

## Configuration API

### Config Structure

```go
type Config struct {
    OBS     OBSConfig     `mapstructure:"obs"`
    Replay  ReplayConfig  `mapstructure:"replay"`
    Text    TextConfig    `mapstructure:"text"`
    Scene   SceneConfig   `mapstructure:"scene"`
    Logging LoggingConfig `mapstructure:"logging"`
}
```

### OBS Configuration

```go
type OBSConfig struct {
    Host     string `mapstructure:"host"`
    Port     int    `mapstructure:"port"`
    Password string `mapstructure:"password"`
}
```

### Replay Configuration

```go
type ReplayConfig struct {
    BufferDuration time.Duration `mapstructure:"buffer_duration"`
    PreRollSeconds int           `mapstructure:"pre_roll_seconds"`
    MinInterval    time.Duration `mapstructure:"min_interval"`
    QueueSize      int           `mapstructure:"queue_size"`
}
```

## OBS Client API

### Connection Management

```go
func (c *Client) Connect(ctx context.Context) error
func (c *Client) Disconnect() error
func (c *Client) IsConnected() bool
func (c *Client) GetStatus() ConnectionStatus
```

### Scene Operations

```go
func (c *Client) GetCurrentScene() (string, error)
func (c *Client) SetCurrentScene(sceneName string) error
func (c *Client) GetSceneList() ([]string, error)
```

### Replay Buffer Operations

```go
func (c *Client) StartReplayBuffer() error
func (c *Client) StopReplayBuffer() error
func (c *Client) SaveReplayBuffer() error
func (c *Client) GetReplayBufferStatus() (bool, error)
```

### Text Source Operations

```go
func (c *Client) UpdateTextSource(sourceName, text string) error
func (c *Client) SetSourceVisibility(sourceName string, visible bool) error
```

## Replay Management API

### Buffer Management

```go
func (b *Buffer) Start(ctx context.Context) error
func (b *Buffer) Stop(ctx context.Context) error
func (b *Buffer) Save(ctx context.Context) error
func (b *Buffer) GetInfo() BufferInfo
func (b *Buffer) IsReady() bool
```

### Queue Management

```go
func (q *Queue) Start(ctx context.Context) error
func (q *Queue) Stop(ctx context.Context) error
func (q *Queue) AddRequest(request ReplayRequest) error
func (q *Queue) GetQueueInfo() QueueInfo
func (q *Queue) GetResults(limit int) []ReplayResult
```

### Manager Interface

```go
func (m *Manager) Start(ctx context.Context) error
func (m *Manager) Stop(ctx context.Context) error
func (m *Manager) TriggerReplay(message string) error
func (m *Manager) GetStatus() ManagerStatus
```

## Data Structures

### ReplayRequest

```go
type ReplayRequest struct {
    ID        string
    Message   string
    Timestamp time.Time
}
```

### ReplayResult

```go
type ReplayResult struct {
    Request   ReplayRequest
    Status    ReplayStatus
    StartTime time.Time
    EndTime   time.Time
    Error     error
}
```

### Status Types

```go
type ReplayStatus int

const (
    ReplayPending ReplayStatus = iota
    ReplayProcessing
    ReplayCompleted
    ReplayFailed
)

type BufferStatus int

const (
    BufferStopped BufferStatus = iota
    BufferStarted
    BufferSaving
    BufferError
)
```

## Error Handling

All API methods return appropriate error types:

- `fmt.Errorf()` for standard errors
- Context cancellation support
- Proper error wrapping with `%w` verb
- Detailed error messages for debugging

## Usage Examples

### Basic Setup

```go
// Load configuration
config, err := config.Load("config/settings.yaml")
if err != nil {
    log.Fatal(err)
}

// Create OBS client
obsClient, err := obs.NewClient(config.OBS, logger)
if err != nil {
    log.Fatal(err)
}

// Connect to OBS
ctx := context.Background()
if err := obsClient.Connect(ctx); err != nil {
    log.Fatal(err)
}

// Create replay manager
replayMgr := replay.NewManager(config.Replay, obsClient, logger)
if err := replayMgr.Start(ctx); err != nil {
    log.Fatal(err)
}
```

### Triggering a Replay

```go
// Trigger a replay with a message
if err := replayMgr.TriggerReplay("Point scored!"); err != nil {
    log.Error("Failed to trigger replay", "error", err)
}
```

### Getting Status

```go
// Get replay manager status
status := replayMgr.GetStatus()
fmt.Printf("Queue size: %d\n", status.QueueInfo.QueueSize)
fmt.Printf("Processing: %d\n", status.QueueInfo.ProcessingCount)
fmt.Printf("Completed: %d\n", status.QueueInfo.CompletedCount)
```

## Testing

### Unit Tests

All components include comprehensive unit tests with mocking:

```go
func TestBuffer_Start(t *testing.T) {
    mockOBS := &MockOBSClient{}
    mockOBS.On("IsConnected").Return(true)
    mockOBS.On("StartReplayBuffer").Return(nil)
    
    buffer := NewBuffer(config, mockOBS, logger)
    err := buffer.Start(context.Background())
    
    assert.NoError(t, err)
    assert.Equal(t, BufferStarted, buffer.status)
}
```

### Integration Tests

Integration tests require a running OBS Studio instance:

```go
func TestClient_Connect(t *testing.T) {
    t.Skip("Integration test - requires running OBS Studio")
    
    // Test implementation here
}
```

## Performance Considerations

### Target Performance

- OBS Connection: < 2 seconds
- Replay Save: < 1 second
- Text Update: < 500ms
- Scene Switch: < 500ms

### Optimization Strategies

- Connection pooling for OBS WebSocket
- Efficient queue processing
- Minimal memory allocations
- Proper goroutine management

## Future Extensions

This API is designed to be extensible for future phases:

- Audio processing integration
- Speech recognition triggers
- Advanced text formatting
- Multi-scene management
- Performance analytics