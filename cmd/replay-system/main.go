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
	"github.com/your-org/hema-replay-system/internal/scene"
	"github.com/your-org/hema-replay-system/internal/text"
	"github.com/your-org/hema-replay-system/pkg/logger"
)

type Application struct {
	config    *config.Config
	logger    *logger.Logger
	obsClient *obs.Client
	replayMgr *replay.Manager
	textMgr   *text.Manager
	sceneMgr  *scene.Manager
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

	a.logger.Info().Msg("Initializing HEMA Replay System")

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

	// Initialize scene manager
	a.sceneMgr = scene.NewManager(a.config.Scene, a.obsClient, a.logger.WithComponent("scene"))
	if err := a.sceneMgr.Start(ctx); err != nil {
		return fmt.Errorf("failed to start scene manager: %w", err)
	}

	// Initialize text manager
	a.textMgr = text.NewManager(a.config.Text, a.obsClient, a.logger.WithComponent("text"))
	if err := a.textMgr.Start(ctx); err != nil {
		return fmt.Errorf("failed to start text manager: %w", err)
	}

	// Initialize replay manager
	a.replayMgr = replay.NewManager(a.config.Replay, a.obsClient, a.logger.WithComponent("replay"))
	if err := a.replayMgr.Start(ctx); err != nil {
		return fmt.Errorf("failed to start replay manager: %w", err)
	}

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
			// Process replay queue
			if a.replayMgr != nil {
				if err := a.replayMgr.ProcessQueue(ctx); err != nil {
					a.logger.Error().Err(err).Msg("Error processing replay queue")
				}
			}
		}
	}
}

func (a *Application) shutdown() error {
	a.logger.Info().Msg("Cleaning up resources...")

	// Cleanup replay manager
	if a.replayMgr != nil {
		if err := a.replayMgr.Stop(context.Background()); err != nil {
			a.logger.Error().Err(err).Msg("Error stopping replay manager")
		}
	}

	// Cleanup text manager
	if a.textMgr != nil {
		if err := a.textMgr.Stop(context.Background()); err != nil {
			a.logger.Error().Err(err).Msg("Error stopping text manager")
		}
	}

	// Cleanup scene manager
	if a.sceneMgr != nil {
		if err := a.sceneMgr.Stop(context.Background()); err != nil {
			a.logger.Error().Err(err).Msg("Error stopping scene manager")
		}
	}

	// Cleanup OBS client
	if a.obsClient != nil {
		if err := a.obsClient.Disconnect(); err != nil {
			a.logger.Error().Err(err).Msg("Error disconnecting from OBS")
		}
	}

	a.logger.Info().Msg("Shutdown complete")
	return nil
}
