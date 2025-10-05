package logger

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Logger struct {
	zerolog.Logger
	file           *os.File
	bufferedWriter *bufio.Writer
	stopFlush      chan struct{}
	mu             sync.Mutex
}

type Config struct {
	Level    string
	Format   string
	FilePath string
}

func New(config Config) (*Logger, error) {
	level := parseLevel(config.Level)

	var logger zerolog.Logger
	var file *os.File
	var bufferedWriter *bufio.Writer
	var writer io.Writer = os.Stdout

	// If file path is provided, create file writer and multi-writer
	if config.FilePath != "" {
		var err error
		file, err = os.OpenFile(fmt.Sprintf("%s/%d.log", config.FilePath, time.Now().Unix()), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, err
		}

		// Create buffered writer for non-blocking file writes
		bufferedWriter = bufio.NewWriter(file)
		writer = io.MultiWriter(os.Stdout, bufferedWriter)
	}

	switch config.Format {
	case "json":
		logger = zerolog.New(writer).With().Timestamp().Logger()
	case "console":
		logger = log.Output(zerolog.ConsoleWriter{Out: writer})
	default:
		logger = zerolog.New(writer).With().Timestamp().Logger()
	}

	logger = logger.Level(level)

	l := &Logger{
		Logger:         logger,
		file:           file,
		bufferedWriter: bufferedWriter,
		stopFlush:      make(chan struct{}),
	}

	// Start periodic flush if file logging is enabled
	if bufferedWriter != nil {
		go l.periodicFlush()
	}

	return l, nil
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

func (l *Logger) Flush() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.bufferedWriter != nil {
		return l.bufferedWriter.Flush()
	}
	return nil
}

func (l *Logger) periodicFlush() {
	ticker := time.NewTicker(1 * time.Second) // Flush every second
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.mu.Lock()
			if l.bufferedWriter != nil {
				l.bufferedWriter.Flush()
			}
			l.mu.Unlock()
		case <-l.stopFlush:
			return
		}
	}
}

func (l *Logger) Close() error {
	// Stop periodic flushing
	if l.stopFlush != nil {
		close(l.stopFlush)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.bufferedWriter != nil {
		if err := l.bufferedWriter.Flush(); err != nil {
			return err
		}
	}
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}
