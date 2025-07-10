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