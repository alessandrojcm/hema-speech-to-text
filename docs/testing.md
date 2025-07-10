# Testing Guide

This document explains the different types of tests in the HEMA Replay System and how to run them.

## Test Types

### 1. Unit Tests
**Location**: `*_test.go` files throughout the codebase  
**Purpose**: Test individual components in isolation  
**Requirements**: None (all dependencies are mocked)  
**Run with**: `make test`

```bash
make test
```

These tests run quickly and don't require any external dependencies. They use mocks and stubs to isolate the code under test.

### 2. Integration Tests
**Location**: `internal/obs/integration_test.go`  
**Purpose**: Test real interactions with OBS Studio  
**Requirements**: OBS Studio running with WebSocket server enabled  
**Run with**: `make test-integration`

```bash
make test-integration
```

#### Setting up OBS for Integration Tests

1. **Start OBS Studio**
2. **Enable WebSocket Server**:
   - Go to `Tools` → `WebSocket Server Settings`
   - Check "Enable WebSocket server"
   - Default port: 4455
   - Password: Optional (leave blank for tests)
3. **Create Test Scenes** (for scene switching tests):
   - Create at least 2 scenes in OBS
   - Name them anything you like
4. **Create Test Text Source** (optional):
   - Add a Text (GDI+) source named "TestText"
   - This allows testing text overlay functionality

#### What Integration Tests Verify

- **Real OBS Connection**: Actually connects to OBS WebSocket server
- **Scene Operations**: 
  - Get current scene
  - List all scenes
  - Switch between scenes
  - Verify scene changes
- **Text Source Operations**:
  - Update text source content
  - Handle missing sources gracefully

### 3. Manual Testing
**Purpose**: Test the complete application flow  
**Requirements**: OBS Studio running  
**Run with**: `make run` or `make run-config`

```bash
# Test with default configuration
make run

# Test with custom configuration
make run-config
```

## Test Structure

```
tests/
├── unit tests (embedded with code)
│   ├── internal/config/config_test.go
│   ├── internal/obs/client_test.go
│   └── pkg/logger/logger_test.go
├── integration tests
│   └── internal/obs/integration_test.go
└── scripts/
    └── test-obs-integration.sh
```

## Running Specific Tests

### Run Only Unit Tests
```bash
go test ./... -short
```

### Run Only Integration Tests
```bash
go test -tags=integration ./internal/obs/
```

### Run Tests with Coverage
```bash
make coverage
```

### Run Tests for Specific Package
```bash
go test -v ./internal/obs/
go test -v ./internal/config/
go test -v ./pkg/logger/
```

## Test Output Examples

### Successful Unit Tests
```
=== RUN   TestNewClient
--- PASS: TestNewClient (0.00s)
=== RUN   TestClient_IsConnected
--- PASS: TestClient_IsConnected (0.00s)
PASS
ok      github.com/your-org/hema-replay-system/internal/obs    1.257s
```

### Integration Tests (OBS Running)
```
=== RUN   TestRealOBSConnection
--- PASS: TestRealOBSConnection (0.12s)
=== RUN   TestRealOBSSceneOperations
--- PASS: TestRealOBSSceneOperations (0.34s)
✅ All OBS integration tests completed!
```

### Integration Tests (OBS Not Running)
```
❌ OBS Studio is not running or WebSocket server is not enabled

To run this test:
1. Start OBS Studio
2. Go to Tools → WebSocket Server Settings
3. Enable WebSocket server (default port 4455)
4. Run this script again
```

## Continuous Integration

For CI/CD pipelines, use:

```bash
# Run only unit tests (fast, no external dependencies)
make test

# Skip integration tests in CI unless OBS container is available
go test ./... -short
```

## Troubleshooting

### "connection refused" Error
- OBS Studio is not running
- WebSocket server is not enabled in OBS
- Wrong port (default is 4455)

### Scene Switching Tests Fail
- Need at least 2 scenes in OBS
- Scene names might have changed

### Text Source Tests Fail
- Create a text source named "TestText" in OBS
- Ensure the source is in the current scene

### Permission Errors
- Make sure the test script is executable: `chmod +x scripts/test-obs-integration.sh`

## Future Test Additions

As the project grows, we plan to add:

- **Performance Tests**: Measure latency and throughput
- **Load Tests**: Test with high-frequency replay triggers
- **Error Recovery Tests**: Test reconnection and failover scenarios
- **End-to-End Tests**: Test complete workflow from audio input to replay output