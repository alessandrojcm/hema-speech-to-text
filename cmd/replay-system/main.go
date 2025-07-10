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
	"github.com/your-org/hema-replay-system/pkg/logger"
)

type Application struct {
	config *config.Config
	logger *logger.Logger
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
	a.logger.Info().Msg("Initializing HEMA Replay System")
	
	// TODO: Initialize OBS client when implemented
	// TODO: Initialize replay manager when implemented
	// TODO: Initialize text manager when implemented
	// TODO: Initialize scene manager when implemented
	
	a.logger.Info().Msg("Application initialized successfully")
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
			// TODO: Process any pending operations when components are implemented
		}
	}
}

func (a *Application) shutdown() error {
	a.logger.Info().Msg("Cleaning up resources...")
	
	// TODO: Cleanup OBS client when implemented
	// TODO: Cleanup other resources when implemented
	
	a.logger.Info().Msg("Shutdown complete")
	return nil
}