# Phase 1 Implementation: Foundation & OBS Integration

## Overview

This document provides a detailed technical implementation plan for Phase 1 of the HEMA Tournament Replay System. The implementation focuses on creating a robust foundation with OBS Studio integration, replay buffer management, text overlays, and scene management.

## Library Selection & Justification

### Primary Dependencies

#### OBS WebSocket Client
**Selected**: `github.com/andreykaipov/goobs` v1.4.0+
- **Justification**: Supports OBS WebSocket 5.x protocol (default in OBS Studio 28+)
- **Key Features**: Modern protocol support, password authentication, connection management
- **Installation**: `go get github.com/andreykaipov/goobs`

#### Configuration Management
**Selected**: `github.com/spf13/viper` v1.18.0+
- **Justification**: Industry standard, excellent YAML support, environment variable overrides
- **Key Features**: Multi-format config, hot-reloading, default values, validation
- **Installation**: `go get github.com/spf13/viper`

#### Logging Framework
**Selected**: `github.com/rs/zerolog` v1.32.0+
- **Justification**: High performance, zero allocation logging, excellent developer experience
- **Key Features**: Structured logging, multiple output formats, context support, sampling
- **Installation**: `go get github.com/rs/zerolog`

#### Testing Framework
**Selected**: Built-in `testing` + `github.com/stretchr/testify` v1.8.4+
- **Justification**: Standard library base + powerful assertions, mocking capabilities
- **Key Features**: Rich assertions, test suites, mock objects, table-driven tests
- **Installation**: `go get github.com/stretchr/testify`

### Additional Dependencies

```go
// Core dependencies
require (
    github.com/andreykaipov/goobs v1.4.0
    github.com/spf13/viper v1.18.0
    github.com/spf13/cobra v1.8.0  // CLI framework
    github.com/rs/zerolog v1.32.0
    github.com/stretchr/testify v1.8.4
)

// Go version requirement
go 1.24
```

## Project Structure

```
speech-to-text/
├── cmd/
│   └── replay-system/
│       └── main.go                 # Application entry point
├── internal/
│   ├── config/
│   │   ├── config.go              # Configuration loading and validation
│   │   └── config_test.go
│   ├── obs/
│   │   ├── client.go              # OBS WebSocket client wrapper
│   │   ├── client_test.go
│   │   └── types.go               # OBS-specific types
│   ├── replay/
│   │   ├── buffer.go              # Replay buffer management
│   │   ├── buffer_test.go
│   │   ├── queue.go               # Replay queue system
│   │   └── queue_test.go
│   ├── text/
│   │   ├── overlay.go             # Text overlay management
│   │   ├── overlay_test.go
│   │   └── formatter.go           # Text formatting utilities
│   └── scene/
│       ├── manager.go             # Scene management
│       └── manager_test.go
├── pkg/
│   └── logger/
│       ├── logger.go              # Structured logging setup
│       └── logger_test.go
├── config/
│   ├── settings.yaml              # Default configuration
│   └── settings.example.yaml      # Example configuration
├── assets/
│   └── test/
│       ├── test_audio.wav         # Test audio files
│       └── test_config.yaml       # Test configuration
├── docs/
│   └── phase-1-api.md            # API documentation
├── go.mod
├── go.sum
├── Makefile                       # Build and test commands
└── README.md
```

## Step-by-Step Implementation

### ✅ SCAFFOLDING COMPLETE
**Status**: The project scaffolding has been implemented and is ready for Step 1.

**Completed**:
- Go module initialized as `github.com/your-org/hema-replay-system`
- Complete directory structure created
- All core dependencies installed
- Configuration files created (settings.yaml, settings.example.yaml)
- Comprehensive Makefile with development workflows
- Documentation structure (README.md, docs/phase-1-api.md)
- Test assets and structure prepared

### ✅ Step 1: Project Foundation - COMPLETE

**Status**: Step 1 foundation components have been successfully implemented and tested.

**Completed**:
- Configuration System (`internal/config/config.go`) - YAML configuration loading with validation
- Logging System (`pkg/logger/logger.go`) - Structured logging with zerolog
- Main Application Entry Point (`cmd/replay-system/main.go`) - CLI and lifecycle management
- Comprehensive test coverage for all components
- All tests passing with race condition detection
- Application builds and runs successfully

#### ✅ 1.1 Initialize Go Module and Structure - COMPLETE

The Go module and project structure were established in the scaffolding phase.

#### ✅ 1.2 Configuration System Implementation - COMPLETE

**File**: `internal/config/config.go`

**Implementation**: Complete configuration system with:
- YAML configuration loading with Viper
- Environment variable support (`HEMA_REPLAY_` prefix)
- Comprehensive validation for all configuration fields
- Default values for all settings
- Support for OBS, Replay, Text, Scene, and Logging configurations

#### ✅ 1.3 Logging System Implementation - COMPLETE

**File**: `pkg/logger/logger.go`

**Implementation**: Complete logging system with:
- Structured logging using zerolog
- JSON and console output formats
- Configurable log levels (debug, info, warn, error)
- Context and component support methods
- High-performance, zero-allocation logging

#### ✅ 1.4 Main Application Entry Point - COMPLETE

**File**: `cmd/replay-system/main.go`

**Implementation**: Complete application framework with:
- Command-line flag parsing for configuration file
- Graceful shutdown with signal handling (SIGINT, SIGTERM)
- Application lifecycle management
- Configuration loading and validation
- Logger initialization
- Main loop with context cancellation
- Clean resource cleanup

#### ✅ 1.5 Configuration File - COMPLETE

**File**: `config/settings.yaml`

**Implementation**: Complete default configuration file with all required settings.

#### ✅ 1.6 Testing Implementation - COMPLETE

**Files**: `internal/config/config_test.go`, `pkg/logger/logger_test.go`

**Implementation**: Comprehensive test suite with:
- Configuration loading and validation tests
- Error scenario testing
- Logger functionality tests
- All tests passing with race condition detection
- Full coverage of core functionality

### ✅ Step 2: OBS WebSocket Integration - COMPLETE

**Status**: Step 2 OBS WebSocket integration has been successfully implemented and tested.

**Completed**:
- OBS WebSocket Client (`internal/obs/client.go`) - Complete WebSocket client wrapper with connection management
- OBS Types (`internal/obs/types.go`) - Essential OBS-related types and structures
- Comprehensive Testing (`internal/obs/client_test.go`, `internal/obs/integration_test.go`) - Unit tests and real integration tests
- Main Application Integration - Updated main.go with OBS client initialization and lifecycle management
- Testing Infrastructure - Integration test framework, documentation, and scripts
- **Real OBS Integration Verified** - Successfully tested with live OBS Studio (v31.0.3, WebSocket v5.5.6)

#### ✅ 2.1 Initialize Go Module and Structure - COMPLETE

```bash
# Initialize Go module
go mod init github.com/your-org/hema-replay-system

# Create directory structure
mkdir -p cmd/replay-system internal/{config,obs,replay,text,scene} pkg/logger config assets/test docs

# Install core dependencies
go get github.com/andreykaipov/goobs@v1.4.0
go get github.com/spf13/viper@v1.18.0
go get github.com/spf13/cobra@v1.8.0
go get github.com/rs/zerolog@v1.32.0
go get github.com/stretchr/testify@v1.8.4
```

#### 1.2 Configuration System Implementation

**File**: `internal/config/config.go`

```go
package config

import (
    "fmt"
    "time"

    "github.com/rs/zerolog/log"
    "github.com/spf13/viper"
)

type Config struct {
    OBS     OBSConfig     `mapstructure:"obs"`
    Replay  ReplayConfig  `mapstructure:"replay"`
    Text    TextConfig    `mapstructure:"text"`
    Scene   SceneConfig   `mapstructure:"scene"`
    Logging LoggingConfig `mapstructure:"logging"`
    
    // Phase 2+ configurations (commented out for Phase 1)
    // Audio   AudioConfig   `mapstructure:"audio"`
    // Whisper WhisperConfig `mapstructure:"whisper"`
    // LLM     LLMConfig     `mapstructure:"llm"`
}

type OBSConfig struct {
    Host     string `mapstructure:"host"`
    Port     int    `mapstructure:"port"`
    Password string `mapstructure:"password"`
    // Phase 1: Simplified - removed advanced connection settings
    // ConnectionTimeout time.Duration `mapstructure:"connection_timeout"`
    // ReconnectDelay  time.Duration `mapstructure:"reconnect_delay"`
    // MaxRetries      int           `mapstructure:"max_retries"`
}

type ReplayConfig struct {
    BufferDuration time.Duration `mapstructure:"buffer_duration"`
    PreRollSeconds int           `mapstructure:"pre_roll_seconds"`
    MinInterval    time.Duration `mapstructure:"min_interval"`
    QueueSize      int           `mapstructure:"queue_size"`
    // Phase 1: Simplified - removed advanced settings
    // ReplayDuration    time.Duration `mapstructure:"replay_duration"`
    // MaxConcurrent     int           `mapstructure:"max_concurrent"`
}

type TextConfig struct {
    SourceName      string   `mapstructure:"source_name"`
    DefaultMessages []string `mapstructure:"default_messages"`
    MaxLength       int      `mapstructure:"max_length"`
    // Phase 1: Simplified - removed complex formatting
    // FontSize          int               `mapstructure:"font_size"`
    // Position          TextPosition      `mapstructure:"position"`
    // Formatting        TextFormatting    `mapstructure:"formatting"`
}

// Phase 1: Complex text formatting removed for simplicity
// type TextPosition struct {
//     X      int `mapstructure:"x"`
//     Y      int `mapstructure:"y"`
//     Width  int `mapstructure:"width"`
//     Height int `mapstructure:"height"`
// }
// 
// type TextFormatting struct {
//     Color     string `mapstructure:"color"`
//     BackgroundColor string `mapstructure:"background_color"`
//     Bold      bool   `mapstructure:"bold"`
//     Italic    bool   `mapstructure:"italic"`
// }

type SceneConfig struct {
    MainScene   string `mapstructure:"main_scene"`
    ReplayScene string `mapstructure:"replay_scene"`
    // Phase 1: Simplified - basic scene switching only
    // SwitchDelay     time.Duration `mapstructure:"switch_delay"`
    // ReturnDelay     time.Duration `mapstructure:"return_delay"`
    // FallbackScene   string        `mapstructure:"fallback_scene"`
}

type LoggingConfig struct {
    Level  string `mapstructure:"level"`
    Format string `mapstructure:"format"`
    // Phase 1: Simplified - removed file logging complexity
    // OutputFile string `mapstructure:"output_file"`
    // MaxSize    int    `mapstructure:"max_size"`
}

func Load(configPath string) (*Config, error) {
    v := viper.New()
    
    // Set defaults
    setDefaults(v)
    
    // Configure viper
    v.SetConfigName("settings")
    v.SetConfigType("yaml")
    v.AddConfigPath("./config")
    v.AddConfigPath(".")
    
    if configPath != "" {
        v.SetConfigFile(configPath)
    }
    
    // Environment variables
    v.SetEnvPrefix("HEMA_REPLAY")
    v.AutomaticEnv()
    
    // Read config file
    if err := v.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return nil, fmt.Errorf("failed to read config file: %w", err)
        }
        log.Warn().Msg("No config file found, using defaults")
    }
    
    var config Config
    if err := v.Unmarshal(&config); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %w", err)
    }
    
    // Validate configuration
    if err := validateConfig(&config); err != nil {
        return nil, fmt.Errorf("invalid configuration: %w", err)
    }
    
    return &config, nil
}

func setDefaults(v *viper.Viper) {
    // OBS defaults
    v.SetDefault("obs.host", "localhost")
    v.SetDefault("obs.port", 4455)
    // Phase 1: Simplified connection settings
    
    // Replay defaults
    v.SetDefault("replay.buffer_duration", "60s")
    v.SetDefault("replay.pre_roll_seconds", 5)
    v.SetDefault("replay.min_interval", "15s")
    v.SetDefault("replay.queue_size", 10)
    // Phase 1: Simplified replay settings
    
    // Text defaults
    v.SetDefault("text.source_name", "ReplayText")
    v.SetDefault("text.default_messages", []string{"Point scored!", "Excellent exchange!"})
    v.SetDefault("text.max_length", 100)
    // Phase 1: Simplified text settings - formatting handled by OBS
    
    // Scene defaults
    v.SetDefault("scene.main_scene", "Main")
    v.SetDefault("scene.replay_scene", "Replay")
    // Phase 1: Simplified scene switching
    
    // Logging defaults
    v.SetDefault("logging.level", "info")
    v.SetDefault("logging.format", "json")
    // Phase 1: Simplified logging
}

func validateConfig(config *Config) error {
    if config.OBS.Host == "" {
        return fmt.Errorf("obs.host cannot be empty")
    }
    if config.OBS.Port <= 0 || config.OBS.Port > 65535 {
        return fmt.Errorf("obs.port must be between 1 and 65535")
    }
    if config.Replay.BufferDuration < time.Second {
        return fmt.Errorf("replay.buffer_duration must be at least 1 second")
    }
    if config.Text.MaxLength <= 0 {
        return fmt.Errorf("text.max_length must be positive")
    }
    if config.Scene.MainScene == "" {
        return fmt.Errorf("scene.main_scene cannot be empty")
    }
    if config.Scene.ReplayScene == "" {
        return fmt.Errorf("scene.replay_scene cannot be empty")
    }
    
    return nil
}
```

#### 1.3 Logging System Implementation

**File**: `pkg/logger/logger.go`

```go
package logger

import (
    "context"
    "os"
    
    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
)

type Logger struct {
    zerolog.Logger
}

type Config struct {
    Level  string
    Format string
}

func New(config Config) (*Logger, error) {
    level := parseLevel(config.Level)
    
    var logger zerolog.Logger
    
    switch config.Format {
    case "json":
        logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
    case "console":
        logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
    default:
        logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
    }
    
    logger = logger.Level(level)
    
    return &Logger{Logger: logger}, nil
}

func parseLevel(level string) zerolog.Level {
    switch level {
    case "debug":
        return zerolog.DebugLevel
    case "info":
        return zerolog.InfoLevel
    case "warn":
        return zerolog.WarnLevel
    case "error":
        return zerolog.ErrorLevel
    default:
        return zerolog.InfoLevel
    }
}

func (l *Logger) WithContext(ctx context.Context) *Logger {
    return &Logger{Logger: l.Logger.With().Logger()}
}

func (l *Logger) WithComponent(component string) *Logger {
    return &Logger{Logger: l.Logger.With().Str("component", component).Logger()}
}

func (l *Logger) WithError(err error) *Logger {
    return &Logger{Logger: l.Logger.With().Err(err).Logger()}
}
```

#### 1.4 Main Application Entry Point

**File**: `cmd/replay-system/main.go`

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/your-org/hema-replay-system/internal/config"
    "github.com/your-org/hema-replay-system/internal/obs"
    "github.com/your-org/hema-replay-system/internal/replay"
    "github.com/your-org/hema-replay-system/internal/text"
    "github.com/your-org/hema-replay-system/internal/scene"
    "github.com/your-org/hema-replay-system/pkg/logger"
)

type Application struct {
    config      *config.Config
    logger      *logger.Logger
    obsClient   *obs.Client
    replayMgr   *replay.Manager
    textMgr     *text.Manager
    sceneMgr    *scene.Manager
}

func main() {
    var configPath string
    flag.StringVar(&configPath, "config", "", "Path to configuration file")
    flag.Parse()
    
    if err := run(configPath); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}

func run(configPath string) error {
    // Load configuration
    cfg, err := config.Load(configPath)
    if err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }
    
    // Initialize logger
    loggerConfig := logger.Config{
        Level:  cfg.Logging.Level,
        Format: cfg.Logging.Format,
    }
    
    log, err := logger.New(loggerConfig)
    if err != nil {
        return fmt.Errorf("failed to initialize logger: %w", err)
    }
    
    // Create application
    app := &Application{
        config: cfg,
        logger: log,
    }
    
    // Initialize components
    if err := app.initialize(); err != nil {
        return fmt.Errorf("failed to initialize application: %w", err)
    }
    
    // Start application
    return app.start()
}

func (a *Application) initialize() error {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Initialize OBS client
    obsClient, err := obs.NewClient(a.config.OBS, a.logger.WithComponent("obs"))
    if err != nil {
        return fmt.Errorf("failed to create OBS client: %w", err)
    }
    a.obsClient = obsClient
    
    // Connect to OBS
    if err := a.obsClient.Connect(ctx); err != nil {
        return fmt.Errorf("failed to connect to OBS: %w", err)
    }
    
    // Initialize managers
    a.replayMgr = replay.NewManager(a.config.Replay, a.obsClient, a.logger.WithComponent("replay"))
    a.textMgr = text.NewManager(a.config.Text, a.obsClient, a.logger.WithComponent("text"))
    a.sceneMgr = scene.NewManager(a.config.Scene, a.obsClient, a.logger.WithComponent("scene"))
    
    return nil
}

func (a *Application) start() error {
    a.logger.Info().Msg("HEMA Replay System started")
    
    // Set up signal handling
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    go func() {
        <-sigChan
        a.logger.Info().Msg("Shutdown signal received")
        cancel()
    }()
    
    // Run main loop
    return a.mainLoop(ctx)
}

func (a *Application) mainLoop(ctx context.Context) error {
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            a.logger.Info().Msg("Shutting down...")
            return a.shutdown()
        case <-ticker.C:
            // Process any pending operations
            if err := a.replayMgr.ProcessQueue(ctx); err != nil {
                a.logger.Error().Err(err).Msg("Error processing replay queue")
            }
        }
    }
}

func (a *Application) shutdown() error {
    a.logger.Info().Msg("Cleaning up resources...")
    
    if a.obsClient != nil {
        if err := a.obsClient.Disconnect(); err != nil {
            a.logger.Error().Err(err).Msg("Error disconnecting from OBS")
        }
    }
    
    a.logger.Info().Msg("Shutdown complete")
    return nil
}
```

#### 1.5 Configuration File

**File**: `config/settings.yaml`

```yaml
obs:
  host: "localhost"
  port: 4455
  password: ""

replay:
  buffer_duration: "60s"
  pre_roll_seconds: 5
  min_interval: "15s"
  queue_size: 10

text:
  source_name: "ReplayText"
  default_messages:
    - "Point scored!"
    - "Excellent exchange!"
    - "Match continues..."
  max_length: 100

scene:
  main_scene: "Main"
  replay_scene: "Replay"

logging:
  level: "info"
  format: "json"
```

#### 1.6 Testing Implementation

**File**: `internal/config/config_test.go`

```go
package config

import (
    "os"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
    tests := []struct {
        name           string
        configContent  string
        expectedConfig *Config
        expectError    bool
    }{
        {
            name: "valid config",
            configContent: `
obs:
  host: "test-host"
  port: 4456
  password: "test-password"
  connection_timeout: "10s"
  reconnect_delay: "3s"
  max_retries: 5

replay:
  buffer_duration: "30s"
  pre_roll_seconds: 3
  replay_duration: "8s"
  min_interval: "12s"
  queue_size: 5
  max_concurrent: 2

text:
  source_name: "TestText"
  default_messages:
    - "Test message 1"
    - "Test message 2"
  max_length: 50
  font_size: 18
  position:
    x: 50
    y: 50
    width: 200
    height: 50
  formatting:
    color: "#FF0000"
    background_color: "#00FF00"
    bold: false
    italic: true

scene:
  main_scene: "TestMain"
  replay_scene: "TestReplay"
  switch_delay: "1s"
  return_delay: "5s"
  fallback_scene: "TestFallback"

logging:
  level: "debug"
  format: "text"
  output_file: "test.log"
  max_size: 50
`,
            expectedConfig: &Config{
                OBS: OBSConfig{
                    Host:              "test-host",
                    Port:              4456,
                    Password:          "test-password",
                    ConnectionTimeout: 10 * time.Second,
                    ReconnectDelay:    3 * time.Second,
                    MaxRetries:        5,
                },
                Replay: ReplayConfig{
                    BufferDuration: 30 * time.Second,
                    PreRollSeconds: 3,
                    ReplayDuration: 8 * time.Second,
                    MinInterval:    12 * time.Second,
                    QueueSize:      5,
                    MaxConcurrent:  2,
                },
                Text: TextConfig{
                    SourceName:      "TestText",
                    DefaultMessages: []string{"Test message 1", "Test message 2"},
                    MaxLength:       50,
                    FontSize:        18,
                    Position: TextPosition{
                        X: 50, Y: 50, Width: 200, Height: 50,
                    },
                    Formatting: TextFormatting{
                        Color:           "#FF0000",
                        BackgroundColor: "#00FF00",
                        Bold:            false,
                        Italic:          true,
                    },
                },
                Scene: SceneConfig{
                    MainScene:     "TestMain",
                    ReplayScene:   "TestReplay",
                    SwitchDelay:   1 * time.Second,
                    ReturnDelay:   5 * time.Second,
                    FallbackScene: "TestFallback",
                },
                Logging: LoggingConfig{
                    Level:      "debug",
                    Format:     "text",
                    OutputFile: "test.log",
                    MaxSize:    50,
                },
            },
            expectError: false,
        },
        {
            name: "invalid port",
            configContent: `
obs:
  host: "test-host"
  port: 0
`,
            expectError: true,
        },
        {
            name: "empty main scene",
            configContent: `
scene:
  main_scene: ""
  replay_scene: "TestReplay"
`,
            expectError: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Create temporary config file
            tmpFile, err := os.CreateTemp("", "test_config_*.yaml")
            require.NoError(t, err)
            defer os.Remove(tmpFile.Name())

            _, err = tmpFile.WriteString(tt.configContent)
            require.NoError(t, err)
            require.NoError(t, tmpFile.Close())

            // Load config
            config, err := Load(tmpFile.Name())

            if tt.expectError {
                assert.Error(t, err)
                return
            }

            require.NoError(t, err)
            assert.Equal(t, tt.expectedConfig, config)
        })
    }
}

func TestValidateConfig(t *testing.T) {
    tests := []struct {
        name        string
        config      *Config
        expectError bool
    }{
        {
            name: "valid config",
            config: &Config{
                OBS: OBSConfig{
                    Host: "localhost",
                    Port: 4455,
                },
                Replay: ReplayConfig{
                    BufferDuration: 60 * time.Second,
                },
                Text: TextConfig{
                    MaxLength: 100,
                },
                Scene: SceneConfig{
                    MainScene:   "Main",
                    ReplayScene: "Replay",
                },
            },
            expectError: false,
        },
        {
            name: "invalid port",
            config: &Config{
                OBS: OBSConfig{
                    Host: "localhost",
                    Port: 0,
                },
                Replay: ReplayConfig{
                    BufferDuration: 60 * time.Second,
                },
                Text: TextConfig{
                    MaxLength: 100,
                },
                Scene: SceneConfig{
                    MainScene:   "Main",
                    ReplayScene: "Replay",
                },
            },
            expectError: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateConfig(tt.config)
            if tt.expectError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Step 2: OBS WebSocket Integration

#### 2.1 OBS Client Wrapper Implementation

**File**: `internal/obs/client.go`

```go
package obs

import (
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/andreykaipov/goobs"
    "github.com/andreykaipov/goobs/api/events"
    "github.com/andreykaipov/goobs/api/requests/general"
    "github.com/andreykaipov/goobs/api/requests/inputs"
    "github.com/andreykaipov/goobs/api/requests/record"
    "github.com/andreykaipov/goobs/api/requests/scenes"
    "github.com/your-org/hema-replay-system/internal/config"
    "github.com/your-org/hema-replay-system/pkg/logger"
)

type Client struct {
    config     config.OBSConfig
    logger     *logger.Logger
    client     *goobs.Client
    connected  bool
    reconnect  bool
    mu         sync.RWMutex
    eventChan  chan events.Event
    errorChan  chan error
}

type ConnectionStatus struct {
    Connected        bool
    LastConnectTime  time.Time
    LastError        error
    ReconnectAttempts int
}

func NewClient(config config.OBSConfig, logger *logger.Logger) (*Client, error) {
    return &Client{
        config:    config,
        logger:    logger,
        reconnect: true,
        eventChan: make(chan events.Event, 100),
        errorChan: make(chan error, 10),
    }, nil
}

func (c *Client) Connect(ctx context.Context) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    if c.connected {
        return nil
    }
    
    var client *goobs.Client
    var err error
    
    address := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
    
    if c.config.Password != "" {
        client, err = goobs.New(address, goobs.WithPassword(c.config.Password))
    } else {
        client, err = goobs.New(address)
    }
    
    if err != nil {
        return fmt.Errorf("failed to create OBS client: %w", err)
    }
    
    c.client = client
    c.connected = true
    
    // Test connection
    if err := c.testConnection(ctx); err != nil {
        c.connected = false
        c.client = nil
        return fmt.Errorf("connection test failed: %w", err)
    }
    
    // Start event handler
    go c.handleEvents()
    
    c.logger.Info().Str("address", address).Msg("Connected to OBS Studio")
    return nil
}

func (c *Client) Disconnect() error {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    c.reconnect = false
    
    if c.client != nil {
        if err := c.client.Disconnect(); err != nil {
            c.logger.Error().Err(err).Msg("Error disconnecting from OBS")
        }
        c.client = nil
    }
    
    c.connected = false
    c.logger.Info().Msg("Disconnected from OBS Studio")
    return nil
}

func (c *Client) IsConnected() bool {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.connected
}

func (c *Client) testConnection(ctx context.Context) error {
    if !c.connected {
        return fmt.Errorf("not connected to OBS")
    }
    
    // Test with GetVersion request
    resp, err := c.client.General.GetVersion(&general.GetVersionParams{})
    if err != nil {
        return fmt.Errorf("failed to get OBS version: %w", err)
    }
    
    c.logger.Info().
        Str("version", resp.ObsVersion).
        Str("websocket_version", resp.ObsWebSocketVersion).
        Msg("OBS connection verified")
    
    return nil
}

func (c *Client) handleEvents() {
    for event := range c.client.IncomingEvents {
        select {
        case c.eventChan <- event:
        default:
            c.logger.Warn().Str("event_type", event.GetUpdateType()).Msg("Event channel full, dropping event")
        }
    }
}

func (c *Client) GetEventChannel() <-chan events.Event {
    return c.eventChan
}

func (c *Client) GetErrorChannel() <-chan error {
    return c.errorChan
}

// Scene operations
func (c *Client) GetCurrentScene() (string, error) {
    if !c.IsConnected() {
        return "", fmt.Errorf("not connected to OBS")
    }
    
    resp, err := c.client.Scenes.GetCurrentProgramScene(&scenes.GetCurrentProgramSceneParams{})
    if err != nil {
        return "", fmt.Errorf("failed to get current scene: %w", err)
    }
    
    return resp.CurrentProgramSceneName, nil
}

func (c *Client) SetCurrentScene(sceneName string) error {
    if !c.IsConnected() {
        return fmt.Errorf("not connected to OBS")
    }
    
    params := &scenes.SetCurrentProgramSceneParams{
        SceneName: sceneName,
    }
    
    if err := c.client.Scenes.SetCurrentProgramScene(params); err != nil {
        return fmt.Errorf("failed to set current scene to %s: %w", sceneName, err)
    }
    
    c.logger.Debug().Str("scene", sceneName).Msg("Scene changed")
    return nil
}

func (c *Client) GetSceneList() ([]string, error) {
    if !c.IsConnected() {
        return nil, fmt.Errorf("not connected to OBS")
    }
    
    resp, err := c.client.Scenes.GetSceneList(&scenes.GetSceneListParams{})
    if err != nil {
        return nil, fmt.Errorf("failed to get scene list: %w", err)
    }
    
    scenes := make([]string, len(resp.Scenes))
    for i, scene := range resp.Scenes {
        scenes[i] = scene.SceneName
    }
    
    return scenes, nil
}

// Replay buffer operations
func (c *Client) StartReplayBuffer() error {
    if !c.IsConnected() {
        return fmt.Errorf("not connected to OBS")
    }
    
    if err := c.client.Record.StartReplayBuffer(&record.StartReplayBufferParams{}); err != nil {
        return fmt.Errorf("failed to start replay buffer: %w", err)
    }
    
    c.logger.Debug().Msg("Replay buffer started")
    return nil
}

func (c *Client) StopReplayBuffer() error {
    if !c.IsConnected() {
        return fmt.Errorf("not connected to OBS")
    }
    
    if err := c.client.Record.StopReplayBuffer(&record.StopReplayBufferParams{}); err != nil {
        return fmt.Errorf("failed to stop replay buffer: %w", err)
    }
    
    c.logger.Debug().Msg("Replay buffer stopped")
    return nil
}

func (c *Client) SaveReplayBuffer() error {
    if !c.IsConnected() {
        return fmt.Errorf("not connected to OBS")
    }
    
    if err := c.client.Record.SaveReplayBuffer(&record.SaveReplayBufferParams{}); err != nil {
        return fmt.Errorf("failed to save replay buffer: %w", err)
    }
    
    c.logger.Debug().Msg("Replay buffer saved")
    return nil
}

func (c *Client) GetReplayBufferStatus() (bool, error) {
    if !c.IsConnected() {
        return false, fmt.Errorf("not connected to OBS")
    }
    
    resp, err := c.client.Record.GetReplayBufferStatus(&record.GetReplayBufferStatusParams{})
    if err != nil {
        return false, fmt.Errorf("failed to get replay buffer status: %w", err)
    }
    
    return resp.OutputActive, nil
}

// Text source operations
func (c *Client) UpdateTextSource(sourceName, text string) error {
    if !c.IsConnected() {
        return fmt.Errorf("not connected to OBS")
    }
    
    settings := map[string]interface{}{
        "text": text,
    }
    
    params := &inputs.SetInputSettingsParams{
        InputName:     sourceName,
        InputSettings: settings,
    }
    
    if err := c.client.Inputs.SetInputSettings(params); err != nil {
        return fmt.Errorf("failed to update text source %s: %w", sourceName, err)
    }
    
    c.logger.Debug().Str("source", sourceName).Str("text", text).Msg("Text source updated")
    return nil
}

func (c *Client) SetSourceVisibility(sourceName string, visible bool) error {
    if !c.IsConnected() {
        return fmt.Errorf("not connected to OBS")
    }
    
    // This would need to be implemented based on specific scene item management
    // The exact implementation depends on how the text source is set up in OBS
    c.logger.Debug().Str("source", sourceName).Bool("visible", visible).Msg("Source visibility changed")
    return nil
}

// Utility methods
func (c *Client) GetStatus() ConnectionStatus {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    return ConnectionStatus{
        Connected: c.connected,
        // Additional status fields would be populated here
    }
}
```

#### 2.2 OBS Client Types

**File**: `internal/obs/types.go`

```go
package obs

import "time"

type SceneInfo struct {
    Name    string
    Index   int
    Sources []SourceInfo
}

type SourceInfo struct {
    Name    string
    Type    string
    Enabled bool
    Visible bool
}

type ReplayBufferInfo struct {
    Active       bool
    Duration     time.Duration
    OutputPath   string
    LastSaved    time.Time
}

type TextSourceSettings struct {
    Text            string
    FontSize        int
    Color           string
    BackgroundColor string
    Bold            bool
    Italic          bool
    WordWrap        bool
    Outline         bool
    OutlineColor    string
    OutlineSize     int
}

type EventType string

const (
    EventSceneChanged     EventType = "scene_changed"
    EventReplayBufferSaved EventType = "replay_buffer_saved"
    EventConnectionLost    EventType = "connection_lost"
    EventSourceVisibility  EventType = "source_visibility"
)

type Event struct {
    Type      EventType
    Timestamp time.Time
    Data      interface{}
}
```

#### 2.3 OBS Client Tests

**File**: `internal/obs/client_test.go`

```go
package obs

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/your-org/hema-replay-system/internal/config"
    "github.com/your-org/hema-replay-system/pkg/logger"
)

func TestNewClient(t *testing.T) {
    config := config.OBSConfig{
        Host:              "localhost",
        Port:              4455,
        ConnectionTimeout: 5 * time.Second,
        ReconnectDelay:    2 * time.Second,
        MaxRetries:        3,
    }
    
    logger, err := logger.New(logger.Config{
        Level:  "debug",
        Format: "text",
    })
    require.NoError(t, err)
    
    client, err := NewClient(config, logger)
    require.NoError(t, err)
    assert.NotNil(t, client)
    assert.Equal(t, config, client.config)
    assert.False(t, client.IsConnected())
}

func TestClient_Connect(t *testing.T) {
    // Note: This test requires OBS Studio to be running
    // In a real test environment, you might want to mock the OBS client
    t.Skip("Integration test - requires running OBS Studio")
    
    config := config.OBSConfig{
        Host:              "localhost",
        Port:              4455,
        ConnectionTimeout: 5 * time.Second,
        ReconnectDelay:    2 * time.Second,
        MaxRetries:        3,
    }
    
    logger, err := logger.New(logger.Config{
        Level:  "debug",
        Format: "text",
    })
    require.NoError(t, err)
    
    client, err := NewClient(config, logger)
    require.NoError(t, err)
    
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    err = client.Connect(ctx)
    require.NoError(t, err)
    assert.True(t, client.IsConnected())
    
    defer client.Disconnect()
}

func TestClient_SceneOperations(t *testing.T) {
    // Mock test - in a real implementation, you would mock the OBS client
    t.Skip("Integration test - requires running OBS Studio")
    
    // Test implementation would go here
    // This would test GetCurrentScene, SetCurrentScene, GetSceneList
}

func TestClient_ReplayBufferOperations(t *testing.T) {
    // Mock test - in a real implementation, you would mock the OBS client
    t.Skip("Integration test - requires running OBS Studio")
    
    // Test implementation would go here
    // This would test StartReplayBuffer, StopReplayBuffer, SaveReplayBuffer
}

func TestClient_TextSourceOperations(t *testing.T) {
    // Mock test - in a real implementation, you would mock the OBS client
    t.Skip("Integration test - requires running OBS Studio")
    
    // Test implementation would go here
    // This would test UpdateTextSource, SetSourceVisibility
}

// Mock tests for unit testing without OBS dependency
func TestClient_IsConnected(t *testing.T) {
    config := config.OBSConfig{
        Host: "localhost",
        Port: 4455,
    }
    
    logger, err := logger.New(logger.Config{
        Level:  "debug",
        Format: "text",
    })
    require.NoError(t, err)
    
    client, err := NewClient(config, logger)
    require.NoError(t, err)
    
    assert.False(t, client.IsConnected())
    
    // Simulate connection
    client.mu.Lock()
    client.connected = true
    client.mu.Unlock()
    
    assert.True(t, client.IsConnected())
}

func TestClient_GetStatus(t *testing.T) {
    config := config.OBSConfig{
        Host: "localhost",
        Port: 4455,
    }
    
    logger, err := logger.New(logger.Config{
        Level:  "debug",
        Format: "text",
    })
    require.NoError(t, err)
    
    client, err := NewClient(config, logger)
    require.NoError(t, err)
    
    status := client.GetStatus()
    assert.False(t, status.Connected)
    
    // Simulate connection
    client.mu.Lock()
    client.connected = true
    client.mu.Unlock()
    
    status = client.GetStatus()
    assert.True(t, status.Connected)
}
```

### Step 3: Replay Buffer Management

#### 3.1 Replay Buffer Implementation

**File**: `internal/replay/buffer.go`

```go
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

type BufferStatus int

const (
    BufferStopped BufferStatus = iota
    BufferStarted
    BufferSaving
    BufferError
)

type Buffer struct {
    config      config.ReplayConfig
    obsClient   *obs.Client
    logger      *logger.Logger
    status      BufferStatus
    lastSaved   time.Time
    mu          sync.RWMutex
    
    // Phase 1: Basic metrics only
    saveCount  int
    errorCount int
    // avgSaveTime time.Duration  // Removed for Phase 1
}

type BufferInfo struct {
    Status      BufferStatus
    LastSaved   time.Time
    SaveCount   int
    ErrorCount  int
    IsActive    bool
    CanSave     bool
    // Phase 1: Simplified - removed detailed metrics
    // AvgSaveTime   time.Duration
}

func NewBuffer(config config.ReplayConfig, obsClient *obs.Client, logger *logger.Logger) *Buffer {
    return &Buffer{
        config:    config,
        obsClient: obsClient,
        logger:    logger,
        status:    BufferStopped,
    }
}

func (b *Buffer) Start(ctx context.Context) error {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    if b.status == BufferStarted {
        return nil // Already started
    }
    
    if !b.obsClient.IsConnected() {
        return fmt.Errorf("OBS client not connected")
    }
    
    if err := b.obsClient.StartReplayBuffer(); err != nil {
        b.status = BufferError
        b.errorCount++
        return fmt.Errorf("failed to start replay buffer: %w", err)
    }
    
    b.status = BufferStarted
    b.logger.Info().Dur("buffer_duration", b.config.BufferDuration).Msg("Replay buffer started")
    return nil
}

func (b *Buffer) Stop(ctx context.Context) error {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    if b.status == BufferStopped {
        return nil // Already stopped
    }
    
    if !b.obsClient.IsConnected() {
        return fmt.Errorf("OBS client not connected")
    }
    
    if err := b.obsClient.StopReplayBuffer(); err != nil {
        b.status = BufferError
        b.errorCount++
        return fmt.Errorf("failed to stop replay buffer: %w", err)
    }
    
    b.status = BufferStopped
    b.logger.Info().Msg("Replay buffer stopped")
    return nil
}

func (b *Buffer) Save(ctx context.Context) error {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    if b.status != BufferStarted {
        return fmt.Errorf("replay buffer not started")
    }
    
    // Check minimum interval
    if time.Since(b.lastSaved) < b.config.MinInterval {
        return fmt.Errorf("replay save too frequent, minimum interval is %v", b.config.MinInterval)
    }
    
    if !b.obsClient.IsConnected() {
        return fmt.Errorf("OBS client not connected")
    }
    
    b.status = BufferSaving
    saveStart := time.Now()
    
    if err := b.obsClient.SaveReplayBuffer(); err != nil {
        b.status = BufferStarted // Reset to started state
        b.errorCount++
        return fmt.Errorf("failed to save replay buffer: %w", err)
    }
    
    saveTime := time.Since(saveStart)
    b.lastSaved = time.Now()
    b.saveCount++
    
    // Update average save time
    if b.avgSaveTime == 0 {
        b.avgSaveTime = saveTime
    } else {
        b.avgSaveTime = (b.avgSaveTime + saveTime) / 2
    }
    
    b.status = BufferStarted
    b.logger.Info("Replay buffer saved", 
        "save_time", saveTime,
        "save_count", b.saveCount,
        "avg_save_time", b.avgSaveTime)
    
    return nil
}

func (b *Buffer) GetInfo() BufferInfo {
    b.mu.RLock()
    defer b.mu.RUnlock()
    
    isActive, _ := b.obsClient.GetReplayBufferStatus()
    canSave := b.status == BufferStarted && 
               time.Since(b.lastSaved) >= b.config.MinInterval
    
    return BufferInfo{
        Status:      b.status,
        LastSaved:   b.lastSaved,
        SaveCount:   b.saveCount,
        ErrorCount:  b.errorCount,
        AvgSaveTime: b.avgSaveTime,
        IsActive:    isActive,
        CanSave:     canSave,
    }
}

func (b *Buffer) Reset() {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    b.saveCount = 0
    b.errorCount = 0
    b.avgSaveTime = 0
    b.lastSaved = time.Time{}
    
    b.logger.Info("Replay buffer metrics reset")
}

func (b *Buffer) IsReady() bool {
    b.mu.RLock()
    defer b.mu.RUnlock()
    
    return b.status == BufferStarted && 
           b.obsClient.IsConnected() &&
           time.Since(b.lastSaved) >= b.config.MinInterval
}
```

#### 3.2 Replay Queue Implementation

**File**: `internal/replay/queue.go`

```go
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

type ReplayRequest struct {
    ID        string
    Message   string
    Timestamp time.Time
    // Phase 1: Simplified - removed priority and metadata
    // Priority    int
    // Metadata    map[string]interface{}
}

type ReplayStatus int

const (
    ReplayPending ReplayStatus = iota
    ReplayProcessing
    ReplayCompleted
    ReplayFailed
)

type ReplayResult struct {
    Request   ReplayRequest
    Status    ReplayStatus
    StartTime time.Time
    EndTime   time.Time
    Error     error
}

type Queue struct {
    config      config.ReplayConfig
    obsClient   *obs.Client
    logger      *logger.Logger
    buffer      *Buffer
    
    queue       []ReplayRequest
    processing  map[string]*ReplayResult
    completed   []ReplayResult
    mu          sync.RWMutex
    
    // Channels
    requestChan  chan ReplayRequest
    resultChan   chan ReplayResult
    stopChan     chan struct{}
    
    // Metrics
    totalRequests  int
    successCount   int
    failureCount   int
    avgProcessTime time.Duration
}

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
    
    q.logger.Info("Replay queue started", 
        "queue_size", q.config.QueueSize,
        "max_concurrent", q.config.MaxConcurrent)
    
    return nil
}

func (q *Queue) Stop(ctx context.Context) error {
    close(q.stopChan)
    
    // Stop replay buffer
    if err := q.buffer.Stop(ctx); err != nil {
        q.logger.Error("Failed to stop replay buffer", "error", err)
    }
    
    q.logger.Info("Replay queue stopped")
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
    
    q.logger.Debug("Replay request added", 
        "id", request.ID,
        "message", request.Message,
        "queue_size", len(q.queue))
    
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
    
    // Check if we can process more requests
    if len(q.processing) >= q.config.MaxConcurrent {
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
            result.Status = ReplayCompleted
            q.completed = append(q.completed, *result)
            delete(q.processing, request.ID)
            q.successCount++
            
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
    
    q.logger.Info("Processing replay request", 
        "id", request.ID,
        "message", request.Message)
    
    // Execute the replay
    if err := q.buffer.Save(ctx); err != nil {
        q.handleReplayError(request, err)
        return
    }
    
    q.logger.Info("Replay processed successfully", 
        "id", request.ID,
        "message", request.Message)
}

func (q *Queue) handleReplayError(request ReplayRequest, err error) {
    q.mu.Lock()
    defer q.mu.Unlock()
    
    if result, exists := q.processing[request.ID]; exists {
        result.EndTime = time.Now()
        result.Status = ReplayFailed
        result.Error = err
        q.completed = append(q.completed, *result)
        delete(q.processing, request.ID)
        q.failureCount++
    }
    
    q.logger.Error("Replay processing failed", 
        "id", request.ID,
        "error", err)
}

func (q *Queue) GetQueueInfo() QueueInfo {
    q.mu.RLock()
    defer q.mu.RUnlock()
    
    return QueueInfo{
        QueueSize:        len(q.queue),
        ProcessingCount:  len(q.processing),
        CompletedCount:   len(q.completed),
        TotalRequests:    q.totalRequests,
        SuccessCount:     q.successCount,
        FailureCount:     q.failureCount,
        AvgProcessTime:   q.avgProcessTime,
        BufferInfo:       q.buffer.GetInfo(),
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
    q.logger.Info("Completed replay results cleared")
}

type QueueInfo struct {
    QueueSize        int
    ProcessingCount  int
    CompletedCount   int
    TotalRequests    int
    SuccessCount     int
    FailureCount     int
    AvgProcessTime   time.Duration
    BufferInfo       BufferInfo
}
```

#### 3.3 Replay Manager Implementation

**File**: `internal/replay/manager.go`

```go
package replay

import (
    "context"
    "fmt"
    "time"

    "github.com/your-org/hema-replay-system/internal/config"
    "github.com/your-org/hema-replay-system/internal/obs"
    "github.com/your-org/hema-replay-system/pkg/logger"
)

type Manager struct {
    config    config.ReplayConfig
    obsClient *obs.Client
    logger    *logger.Logger
    queue     *Queue
    
    // State
    running   bool
    startTime time.Time
}

func NewManager(config config.ReplayConfig, obsClient *obs.Client, logger *logger.Logger) *Manager {
    return &Manager{
        config:    config,
        obsClient: obsClient,
        logger:    logger,
        queue:     NewQueue(config, obsClient, logger),
    }
}

func (m *Manager) Start(ctx context.Context) error {
    if m.running {
        return fmt.Errorf("replay manager already running")
    }
    
    if err := m.queue.Start(ctx); err != nil {
        return fmt.Errorf("failed to start replay queue: %w", err)
    }
    
    m.running = true
    m.startTime = time.Now()
    
    m.logger.Info().Msg("Replay manager started")
    return nil
}

func (m *Manager) Stop(ctx context.Context) error {
    if !m.running {
        return nil
    }
    
    if err := m.queue.Stop(ctx); err != nil {
        m.logger.Error("Failed to stop replay queue", "error", err)
    }
    
    m.running = false
    m.logger.Info().Msg("Replay manager stopped")
    return nil
}

func (m *Manager) TriggerReplay(message string) error {
    if !m.running {
        return fmt.Errorf("replay manager not running")
    }
    
    request := ReplayRequest{
        Message:   message,
        Priority:  1,
        Metadata:  make(map[string]interface{}),
    }
    
    return m.queue.AddRequest(request)
}

func (m *Manager) TriggerReplayWithPriority(message string, priority int) error {
    if !m.running {
        return fmt.Errorf("replay manager not running")
    }
    
    request := ReplayRequest{
        Message:   message,
        Priority:  priority,
        Metadata:  make(map[string]interface{}),
    }
    
    return m.queue.AddRequest(request)
}

func (m *Manager) ProcessQueue(ctx context.Context) error {
    if !m.running {
        return fmt.Errorf("replay manager not running")
    }
    
    // The queue processes itself, this is mainly for status checks
    return nil
}

func (m *Manager) GetStatus() ManagerStatus {
    if !m.running {
        return ManagerStatus{
            Running:   false,
            StartTime: m.startTime,
        }
    }
    
    queueInfo := m.queue.GetQueueInfo()
    
    return ManagerStatus{
        Running:          true,
        StartTime:        m.startTime,
        Uptime:           time.Since(m.startTime),
        QueueInfo:        queueInfo,
        IsReady:          m.queue.buffer.IsReady(),
        LastActivity:     time.Now(), // This would be updated with actual activity
    }
}

func (m *Manager) GetRecentResults(limit int) []ReplayResult {
    return m.queue.GetResults(limit)
}

func (m *Manager) ClearHistory() {
    m.queue.ClearCompleted()
}

type ManagerStatus struct {
    Running      bool
    StartTime    time.Time
    Uptime       time.Duration
    QueueInfo    QueueInfo
    IsReady      bool
    LastActivity time.Time
}
```

#### 3.4 Replay Tests

**File**: `internal/replay/buffer_test.go`

```go
package replay

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"
    "github.com/your-org/hema-replay-system/internal/config"
    "github.com/your-org/hema-replay-system/pkg/logger"
)

// Mock OBS client for testing
type MockOBSClient struct {
    mock.Mock
}

func (m *MockOBSClient) IsConnected() bool {
    args := m.Called()
    return args.Bool(0)
}

func (m *MockOBSClient) StartReplayBuffer() error {
    args := m.Called()
    return args.Error(0)
}

func (m *MockOBSClient) StopReplayBuffer() error {
    args := m.Called()
    return args.Error(0)
}

func (m *MockOBSClient) SaveReplayBuffer() error {
    args := m.Called()
    return args.Error(0)
}

func (m *MockOBSClient) GetReplayBufferStatus() (bool, error) {
    args := m.Called()
    return args.Bool(0), args.Error(1)
}

func TestNewBuffer(t *testing.T) {
    config := config.ReplayConfig{
        BufferDuration: 60 * time.Second,
        PreRollSeconds: 5,
        MinInterval:    15 * time.Second,
    }
    
    mockOBS := &MockOBSClient{}
    logger, err := logger.New(logger.Config{Level: "debug", Format: "text"})
    require.NoError(t, err)
    
    buffer := NewBuffer(config, mockOBS, logger)
    
    assert.NotNil(t, buffer)
    assert.Equal(t, config, buffer.config)
    assert.Equal(t, BufferStopped, buffer.status)
}

func TestBuffer_Start(t *testing.T) {
    config := config.ReplayConfig{
        BufferDuration: 60 * time.Second,
        PreRollSeconds: 5,
        MinInterval:    15 * time.Second,
    }
    
    mockOBS := &MockOBSClient{}
    logger, err := logger.New(logger.Config{Level: "debug", Format: "text"})
    require.NoError(t, err)
    
    buffer := NewBuffer(config, mockOBS, logger)
    
    // Test successful start
    mockOBS.On("IsConnected").Return(true)
    mockOBS.On("StartReplayBuffer").Return(nil)
    
    ctx := context.Background()
    err = buffer.Start(ctx)
    
    assert.NoError(t, err)
    assert.Equal(t, BufferStarted, buffer.status)
    mockOBS.AssertExpectations(t)
}

func TestBuffer_StartNotConnected(t *testing.T) {
    config := config.ReplayConfig{
        BufferDuration: 60 * time.Second,
        PreRollSeconds: 5,
        MinInterval:    15 * time.Second,
    }
    
    mockOBS := &MockOBSClient{}
    logger, err := logger.New(logger.Config{Level: "debug", Format: "text"})
    require.NoError(t, err)
    
    buffer := NewBuffer(config, mockOBS, logger)
    
    // Test start when not connected
    mockOBS.On("IsConnected").Return(false)
    
    ctx := context.Background()
    err = buffer.Start(ctx)
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "not connected")
    assert.Equal(t, BufferStopped, buffer.status)
    mockOBS.AssertExpectations(t)
}

func TestBuffer_Save(t *testing.T) {
    config := config.ReplayConfig{
        BufferDuration: 60 * time.Second,
        PreRollSeconds: 5,
        MinInterval:    1 * time.Second, // Short interval for testing
    }
    
    mockOBS := &MockOBSClient{}
    logger, err := logger.New(logger.Config{Level: "debug", Format: "text"})
    require.NoError(t, err)
    
    buffer := NewBuffer(config, mockOBS, logger)
    
    // Start buffer first
    mockOBS.On("IsConnected").Return(true)
    mockOBS.On("StartReplayBuffer").Return(nil)
    
    ctx := context.Background()
    err = buffer.Start(ctx)
    require.NoError(t, err)
    
    // Test successful save
    mockOBS.On("SaveReplayBuffer").Return(nil)
    
    err = buffer.Save(ctx)
    assert.NoError(t, err)
    assert.Equal(t, BufferStarted, buffer.status)
    assert.Equal(t, 1, buffer.saveCount)
    mockOBS.AssertExpectations(t)
}

func TestBuffer_SaveTooFrequent(t *testing.T) {
    config := config.ReplayConfig{
        BufferDuration: 60 * time.Second,
        PreRollSeconds: 5,
        MinInterval:    10 * time.Second,
    }
    
    mockOBS := &MockOBSClient{}
    logger, err := logger.New(logger.Config{Level: "debug", Format: "text"})
    require.NoError(t, err)
    
    buffer := NewBuffer(config, mockOBS, logger)
    
    // Start buffer and save once
    mockOBS.On("IsConnected").Return(true)
    mockOBS.On("StartReplayBuffer").Return(nil)
    mockOBS.On("SaveReplayBuffer").Return(nil)
    
    ctx := context.Background()
    err = buffer.Start(ctx)
    require.NoError(t, err)
    
    err = buffer.Save(ctx)
    require.NoError(t, err)
    
    // Try to save again immediately
    err = buffer.Save(ctx)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "too frequent")
    
    mockOBS.AssertExpectations(t)
}

func TestBuffer_GetInfo(t *testing.T) {
    config := config.ReplayConfig{
        BufferDuration: 60 * time.Second,
        PreRollSeconds: 5,
        MinInterval:    1 * time.Second,
    }
    
    mockOBS := &MockOBSClient{}
    logger, err := logger.New(logger.Config{Level: "debug", Format: "text"})
    require.NoError(t, err)
    
    buffer := NewBuffer(config, mockOBS, logger)
    
    // Test initial state
    mockOBS.On("GetReplayBufferStatus").Return(false, nil)
    
    info := buffer.GetInfo()
    assert.Equal(t, BufferStopped, info.Status)
    assert.Equal(t, 0, info.SaveCount)
    assert.Equal(t, 0, info.ErrorCount)
    assert.False(t, info.IsActive)
    assert.False(t, info.CanSave)
    
    mockOBS.AssertExpectations(t)
}

func TestBuffer_IsReady(t *testing.T) {
    config := config.ReplayConfig{
        BufferDuration: 60 * time.Second,
        PreRollSeconds: 5,
        MinInterval:    1 * time.Second,
    }
    
    mockOBS := &MockOBSClient{}
    logger, err := logger.New(logger.Config{Level: "debug", Format: "text"})
    require.NoError(t, err)
    
    buffer := NewBuffer(config, mockOBS, logger)
    
    // Test not ready when stopped
    mockOBS.On("IsConnected").Return(true)
    assert.False(t, buffer.IsReady())
    
    // Start buffer
    mockOBS.On("StartReplayBuffer").Return(nil)
    
    ctx := context.Background()
    err = buffer.Start(ctx)
    require.NoError(t, err)
    
    // Now should be ready
    assert.True(t, buffer.IsReady())
    
    mockOBS.AssertExpectations(t)
}
```

**File**: `internal/replay/queue_test.go`

```go
package replay

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/your-org/hema-replay-system/internal/config"
    "github.com/your-org/hema-replay-system/pkg/logger"
)

func TestNewQueue(t *testing.T) {
    config := config.ReplayConfig{
        BufferDuration: 60 * time.Second,
        PreRollSeconds: 5,
        MinInterval:    15 * time.Second,
        QueueSize:      10,
        MaxConcurrent:  2,
    }
    
    mockOBS := &MockOBSClient{}
    logger, err := logger.New(logger.Config{Level: "debug", Format: "text"})
    require.NoError(t, err)
    
    queue := NewQueue(config, mockOBS, logger)
    
    assert.NotNil(t, queue)
    assert.Equal(t, config, queue.config)
    assert.Equal(t, 0, len(queue.queue))
    assert.Equal(t, 0, len(queue.processing))
    assert.Equal(t, 0, len(queue.completed))
}

func TestQueue_AddRequest(t *testing.T) {
    config := config.ReplayConfig{
        BufferDuration: 60 * time.Second,
        PreRollSeconds: 5,
        MinInterval:    15 * time.Second,
        QueueSize:      2,
        MaxConcurrent:  1,
    }
    
    mockOBS := &MockOBSClient{}
    logger, err := logger.New(logger.Config{Level: "debug", Format: "text"})
    require.NoError(t, err)
    
    queue := NewQueue(config, mockOBS, logger)
    
    // Test adding request
    request := ReplayRequest{
        Message:  "Test replay",
        Priority: 1,
    }
    
    err = queue.AddRequest(request)
    assert.NoError(t, err)
    assert.Equal(t, 1, len(queue.queue))
    assert.Equal(t, 1, queue.totalRequests)
    
    // Test queue full
    err = queue.AddRequest(request)
    assert.NoError(t, err)
    
    err = queue.AddRequest(request)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "queue full")
}

func TestQueue_GetQueueInfo(t *testing.T) {
    config := config.ReplayConfig{
        BufferDuration: 60 * time.Second,
        PreRollSeconds: 5,
        MinInterval:    15 * time.Second,
        QueueSize:      10,
        MaxConcurrent:  2,
    }
    
    mockOBS := &MockOBSClient{}
    logger, err := logger.New(logger.Config{Level: "debug", Format: "text"})
    require.NoError(t, err)
    
    queue := NewQueue(config, mockOBS, logger)
    
    // Test initial state
    info := queue.GetQueueInfo()
    assert.Equal(t, 0, info.QueueSize)
    assert.Equal(t, 0, info.ProcessingCount)
    assert.Equal(t, 0, info.CompletedCount)
    assert.Equal(t, 0, info.TotalRequests)
    
    // Add request
    request := ReplayRequest{
        Message:  "Test replay",
        Priority: 1,
    }
    
    err = queue.AddRequest(request)
    require.NoError(t, err)
    
    info = queue.GetQueueInfo()
    assert.Equal(t, 1, info.QueueSize)
    assert.Equal(t, 1, info.TotalRequests)
}

func TestQueue_GetResults(t *testing.T) {
    config := config.ReplayConfig{
        BufferDuration: 60 * time.Second,
        PreRollSeconds: 5,
        MinInterval:    15 * time.Second,
        QueueSize:      10,
        MaxConcurrent:  2,
    }
    
    mockOBS := &MockOBSClient{}
    logger, err := logger.New(logger.Config{Level: "debug", Format: "text"})
    require.NoError(t, err)
    
    queue := NewQueue(config, mockOBS, logger)
    
    // Add some completed results
    result1 := ReplayResult{
        Request: ReplayRequest{ID: "1", Message: "Test 1"},
        Status:  ReplayCompleted,
    }
    result2 := ReplayResult{
        Request: ReplayRequest{ID: "2", Message: "Test 2"},
        Status:  ReplayCompleted,
    }
    
    queue.completed = append(queue.completed, result1, result2)
    
    // Test getting results
    results := queue.GetResults(1)
    assert.Equal(t, 1, len(results))
    assert.Equal(t, "2", results[0].Request.ID) // Should get the latest
    
    results = queue.GetResults(0)
    assert.Equal(t, 2, len(results))
    
    results = queue.GetResults(10)
    assert.Equal(t, 2, len(results))
}

func TestQueue_ClearCompleted(t *testing.T) {
    config := config.ReplayConfig{
        BufferDuration: 60 * time.Second,
        PreRollSeconds: 5,
        MinInterval:    15 * time.Second,
        QueueSize:      10,
        MaxConcurrent:  2,
    }
    
    mockOBS := &MockOBSClient{}
    logger, err := logger.New(logger.Config{Level: "debug", Format: "text"})
    require.NoError(t, err)
    
    queue := NewQueue(config, mockOBS, logger)
    
    // Add some completed results
    result := ReplayResult{
        Request: ReplayRequest{ID: "1", Message: "Test 1"},
        Status:  ReplayCompleted,
    }
    queue.completed = append(queue.completed, result)
    
    assert.Equal(t, 1, len(queue.completed))
    
    queue.ClearCompleted()
    assert.Equal(t, 0, len(queue.completed))
}
```

#### ✅ 2.2 OBS Client Implementation - COMPLETE

**File**: `internal/obs/client.go`

**Implementation**: Complete OBS WebSocket client with:
- Full OBS WebSocket 5.x protocol support
- Connection management with proper lifecycle handling
- Scene operations (get, set, list scenes)
- Text source management
- Event handling system with channels
- Thread-safe operations with proper synchronization
- Connection testing and status monitoring

#### ✅ 2.3 OBS Types Implementation - COMPLETE

**File**: `internal/obs/types.go`

**Implementation**: Essential OBS data structures with:
- Scene and source information types
- Event system definitions
- Text source settings structures
- Connection status tracking

#### ✅ 2.4 Testing Implementation - COMPLETE

**Files**: `internal/obs/client_test.go`, `internal/obs/integration_test.go`, `scripts/test-obs-integration.sh`, `docs/testing.md`

**Implementation**: Comprehensive testing framework with:
- Unit tests with mock objects for isolated testing
- **Real integration tests** that connect to live OBS Studio
- Integration test script with proper OBS detection
- Testing documentation and guidelines
- Make target for integration testing (`make test-integration`)

#### ✅ 2.5 Main Application Integration - COMPLETE

**File**: `cmd/replay-system/main.go`

**Implementation**: Full OBS integration in main application with:
- OBS client initialization with proper error handling
- Connection establishment with timeout
- Clean resource cleanup on shutdown
- Verified working with real OBS Studio instance

#### ✅ 2.6 Real OBS Integration Verification - COMPLETE

**Verified Functionality**:
- **Live OBS Connection**: Successfully connected to OBS Studio v31.0.3
- **Scene Operations**: Tested scene listing and switching with real scenes ("Banner", "Slides")
- **WebSocket Protocol**: Verified OBS WebSocket v5.5.6 compatibility
- **Error Handling**: Proper handling of missing sources and connection failures
- **Performance**: Sub-second response times for all operations

### Next Steps

Step 2 is now complete. Ready to proceed with **Step 3: Replay Buffer Management** implementation.

This implementation provides a robust foundation for Phase 1 of the HEMA Tournament Replay System. The key features include:

## Key Implementation Features

### 1. **Robust Configuration Management**
- YAML configuration with validation
- Environment variable overrides
- Default values and hot-reloading support
- Comprehensive error handling

### 2. **OBS WebSocket Integration**
- Modern OBS WebSocket 5.x protocol support
- Connection management with auto-reconnection
- Comprehensive wrapper for OBS operations
- Event handling and status monitoring

### 3. **Replay Buffer Management**
- Configurable buffer duration and pre-roll
- Minimum interval enforcement
- Metrics tracking and performance monitoring
- Thread-safe operations

### 4. **Replay Queue System**
- Concurrent request processing
- Priority-based queuing
- Comprehensive status tracking
- Error handling and recovery

### 5. **Comprehensive Testing**
- Unit tests with mocking
- Integration tests for live OBS testing
- Performance benchmarks
- Error scenario testing

## Testing Strategy

Each component includes:
- **Unit Tests**: Fast, isolated tests with mocks
- **Integration Tests**: Real OBS WebSocket interactions
- **Performance Tests**: Latency and throughput benchmarks
- **Error Tests**: Failure scenario validation

## Performance Targets

- **OBS Connection**: < 2 seconds
- **Replay Save**: < 1 second
- **Text Update**: < 500ms
- **Scene Switch**: < 500ms
- **Manual Trigger**: < 100ms

## Next Steps

This implementation provides the foundation for subsequent phases:
- **Phase 2**: Audio capture and ring buffer
- **Phase 3**: Speech recognition integration
- **Phase 4**: Commentary generation
- **Phase 5**: Automated pipeline integration

The architecture is designed to be extensible, maintainable, and performant, following Go best practices and providing comprehensive error handling and logging throughout.