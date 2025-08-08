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
}

type OBSConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
}

type ReplayConfig struct {
	BufferDuration time.Duration `mapstructure:"buffer_duration"`
	PreRollSeconds int           `mapstructure:"pre_roll_seconds"`
	MinInterval    time.Duration `mapstructure:"min_interval"`
	QueueSize      int           `mapstructure:"queue_size"`
}

type TextConfig struct {
	SourceName      string   `mapstructure:"source_name"`
	DefaultMessages []string `mapstructure:"default_messages"`
	MaxLength       int      `mapstructure:"max_length"`
}

type SceneConfig struct {
	MainScene   string `mapstructure:"main_scene"`
	ReplayScene string `mapstructure:"replay_scene"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
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

	// Replay defaults
	v.SetDefault("replay.buffer_duration", "60s")
	v.SetDefault("replay.pre_roll_seconds", 5)
	v.SetDefault("replay.min_interval", "15s")
	v.SetDefault("replay.queue_size", 10)

	// Text defaults
	v.SetDefault("text.source_name", "ReplayText")
	v.SetDefault("text.default_messages", []string{"Point scored!", "Excellent exchange!"})
	v.SetDefault("text.max_length", 100)

	// Scene defaults
	v.SetDefault("scene.main_scene", "Main")
	v.SetDefault("scene.replay_scene", "Replay")

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
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
