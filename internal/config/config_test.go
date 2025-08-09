package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-org/hema-replay-system/pkg/audio/types"
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

replay:
  buffer_duration: "30s"
  pre_roll_seconds: 3
  min_interval: "12s"
  queue_size: 5

text:
  source_name: "TestText"
  default_messages:
    - "Test message 1"
    - "Test message 2"
  max_length: 50

scene:
  main_scene: "TestMain"
  replay_scene: "TestReplay"

audio:
  processing:
    resampler_type: "gosamplerate"
    vad_type: "webrtc"
    wav_exporter_type: "goaudio"
    fft_type: "gonum"
    resampler_quality: 0
    vad_mode: 3

logging:
  level: "debug"
  format: "console"
`,
			expectedConfig: &Config{
				OBS: OBSConfig{
					Host:     "test-host",
					Port:     4456,
					Password: "test-password",
				},
				Replay: ReplayConfig{
					BufferDuration: 30 * time.Second,
					PreRollSeconds: 3,
					MinInterval:    12 * time.Second,
					QueueSize:      5,
				},
				Text: TextConfig{
					SourceName:      "TestText",
					DefaultMessages: []string{"Test message 1", "Test message 2"},
					MaxLength:       50,
				},
				Scene: SceneConfig{
					MainScene:   "TestMain",
					ReplayScene: "TestReplay",
				},
				Audio: types.DefaultAudioConfig(),
				Logging: LoggingConfig{
					Level:  "debug",
					Format: "console",
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
		{
			name: "invalid buffer duration",
			configContent: `
replay:
  buffer_duration: "500ms"
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
				Audio: types.DefaultAudioConfig(),
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
		{
			name: "port too high",
			config: &Config{
				OBS: OBSConfig{
					Host: "localhost",
					Port: 70000,
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
		{
			name: "empty host",
			config: &Config{
				OBS: OBSConfig{
					Host: "",
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
			expectError: true,
		},
		{
			name: "buffer duration too short",
			config: &Config{
				OBS: OBSConfig{
					Host: "localhost",
					Port: 4455,
				},
				Replay: ReplayConfig{
					BufferDuration: 500 * time.Millisecond,
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
		{
			name: "negative max length",
			config: &Config{
				OBS: OBSConfig{
					Host: "localhost",
					Port: 4455,
				},
				Replay: ReplayConfig{
					BufferDuration: 60 * time.Second,
				},
				Text: TextConfig{
					MaxLength: -1,
				},
				Scene: SceneConfig{
					MainScene:   "Main",
					ReplayScene: "Replay",
				},
			},
			expectError: true,
		},
		{
			name: "empty main scene",
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
					MainScene:   "",
					ReplayScene: "Replay",
				},
			},
			expectError: true,
		},
		{
			name: "empty replay scene",
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
					ReplayScene: "",
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

func TestLoadDefaults(t *testing.T) {
	// Create empty config file
	tmpFile, err := os.CreateTemp("", "test_config_*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Write minimal config
	_, err = tmpFile.WriteString("# Empty config file\n")
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// Load config
	config, err := Load(tmpFile.Name())
	require.NoError(t, err)

	// Check defaults
	assert.Equal(t, "localhost", config.OBS.Host)
	assert.Equal(t, 4455, config.OBS.Port)
	assert.Equal(t, "", config.OBS.Password)

	assert.Equal(t, 60*time.Second, config.Replay.BufferDuration)
	assert.Equal(t, 5, config.Replay.PreRollSeconds)
	assert.Equal(t, 15*time.Second, config.Replay.MinInterval)
	assert.Equal(t, 10, config.Replay.QueueSize)

	assert.Equal(t, "ReplayText", config.Text.SourceName)
	assert.Equal(t, []string{"Point scored!", "Excellent exchange!"}, config.Text.DefaultMessages)
	assert.Equal(t, 100, config.Text.MaxLength)

	assert.Equal(t, "Main", config.Scene.MainScene)
	assert.Equal(t, "Replay", config.Scene.ReplayScene)

	assert.Equal(t, "info", config.Logging.Level)
	assert.Equal(t, "json", config.Logging.Format)
}

func TestLoadNoConfigFile(t *testing.T) {
	// Load config with empty path (should use default paths)
	config, err := Load("")

	// Should still succeed with defaults even if no config file is found
	require.NoError(t, err)
	assert.NotNil(t, config)

	// Check that defaults are applied
	assert.Equal(t, "localhost", config.OBS.Host)
	assert.Equal(t, 4455, config.OBS.Port)
}
