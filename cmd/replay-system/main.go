package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-audio/wav"
	"github.com/your-org/hema-replay-system/internal/config"
	"github.com/your-org/hema-replay-system/internal/obs"
	"github.com/your-org/hema-replay-system/internal/replay"
	"github.com/your-org/hema-replay-system/internal/scene"
	"github.com/your-org/hema-replay-system/internal/text"
	"github.com/your-org/hema-replay-system/pkg/audio"
	"github.com/your-org/hema-replay-system/pkg/audio/capture"
	"github.com/your-org/hema-replay-system/pkg/audio/debug"
	"github.com/your-org/hema-replay-system/pkg/audio/processing"
	audioTypes "github.com/your-org/hema-replay-system/pkg/audio/types"
	commentaryContext "github.com/your-org/hema-replay-system/pkg/commentary/context"
	commentaryEngine "github.com/your-org/hema-replay-system/pkg/commentary/engine"
	commentaryPrompt "github.com/your-org/hema-replay-system/pkg/commentary/prompt"
	commentaryTemplates "github.com/your-org/hema-replay-system/pkg/commentary/templates"
	commentaryTypes "github.com/your-org/hema-replay-system/pkg/commentary/types"
	llmEngine "github.com/your-org/hema-replay-system/pkg/llm/engine"
	"github.com/your-org/hema-replay-system/pkg/logger"
	"github.com/your-org/hema-replay-system/pkg/pipeline"
	"github.com/your-org/hema-replay-system/pkg/speech/engine"
	speechTypes "github.com/your-org/hema-replay-system/pkg/speech/types"
)

type Application struct {
	config         *config.Config
	logger         *logger.Logger
	obsClient      *obs.Client
	replayMgr      *replay.Manager
	textMgr        *text.Manager
	sceneMgr       *scene.Manager
	audioMgr       *audio.AudioManager
	speechMgr      *engine.SpeechManager
	pipelineMgr    *pipeline.Manager
	audioProcessor *processing.AudioProcessor
	speechOnly     bool
	audioFile      string
	pipelineMode   bool

	// Debug settings
	debugAudio     bool
	debugOutputDir string
	vadDebug       bool

	// Commentary system components
	llmEngine         *llmEngine.ModelEngine
	commentaryGen     *commentaryEngine.CommentaryGenerator
	commentaryEnabled bool
}

func main() {
	var configPath string
	var speechOnly bool
	var audioFile string
	var listDevices bool
	var pipelineMode bool
	var debugAudio bool
	var debugOutputDir string
	var vadDebug bool
	flag.StringVar(&configPath, "config", "", "Path to configuration file")
	flag.BoolVar(&speechOnly, "speech-only", false, "Run only speech recognition system (for testing)")
	flag.StringVar(&audioFile, "audio-file", "", "Process a single audio file and exit (for testing)")
	flag.BoolVar(&listDevices, "list-devices", false, "List available audio devices")
	flag.BoolVar(&pipelineMode, "pipeline", false, "Run the complete VAD-driven pipeline system")
	flag.BoolVar(&debugAudio, "debug-audio", false, "Save audio segments for debugging")
	flag.StringVar(&debugOutputDir, "debug-output", "./debug_audio", "Directory for debug audio files")
	flag.BoolVar(&vadDebug, "vad-debug", false, "Enable detailed VAD logging")
	flag.Parse()

	if err := run(configPath, speechOnly, audioFile, listDevices, pipelineMode, debugAudio, debugOutputDir, vadDebug); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(configPath string, speechOnly bool, audioFile string, listDevices bool, pipelineMode bool, debugAudio bool, debugOutputDir string, vadDebug bool) error {
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

	if listDevices {
		manager := capture.NewDeviceManager(cfg.Audio.Device, log.WithComponent("device_manager").Logger)
		manager.Start(context.Background())
		devices := manager.GetDevices()
		for _, device := range devices {
			if device.MaxOutputChannels > 0 {
				continue
			}
			log.Info().Msgf("Device ID: %d, Name: %s, Channels: %d", device.ID, device.Name, device.MaxInputChannels)
		}
		return nil
	}

	// Create application
	app := &Application{
		config:         cfg,
		logger:         log,
		speechOnly:     speechOnly,
		audioFile:      audioFile,
		pipelineMode:   pipelineMode,
		debugAudio:     debugAudio,
		debugOutputDir: debugOutputDir,
		vadDebug:       vadDebug,
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

	if a.audioFile != "" {
		a.logger.Info().Str("audio_file", a.audioFile).Msg("Initializing Audio File Processing Mode")
		return a.initializeAudioFileMode(ctx)
	}

	if a.pipelineMode {
		a.logger.Info().Msg("Initializing VAD-Driven Pipeline System")
		return a.initializePipelineMode(ctx)
	}

	if a.speechOnly {
		a.logger.Info().Msg("Initializing Speech Recognition System (Speech-Only Mode)")
		return a.initializeSpeechOnly(ctx)
	}

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

func (a *Application) initializeSpeechOnly(ctx context.Context) error {
	// Try to initialize audio manager (may fail in noaudio build)
	audioMgr, err := audio.NewAudioManager(a.config.Audio, a.logger.WithComponent("audio").Logger)
	if err != nil {
		a.logger.Warn().Err(err).Msg("Audio manager not available - running in test mode without real audio")
	} else {
		a.audioMgr = audioMgr
		// Start audio capture with long-lived context (not the timeout context)
		// Audio capture needs to run for the lifetime of the application
		longLivedCtx := context.Background()
		if err := a.audioMgr.Start(longLivedCtx); err != nil {
			a.logger.Warn().Err(err).Msg("Failed to start audio manager - continuing without audio")
			a.audioMgr = nil
		} else if a.config.Audio.Processing.EnablePreprocessing {
			// Initialize audio processor for VAD filtering in speech-only mode
			sampleRate := a.config.Audio.Device.SampleRate
			channels := a.config.Audio.Device.Channels
			audioProcessor, err := processing.NewAudioProcessor(a.config.Audio.Processing, sampleRate, channels, a.logger.WithComponent("audio_processor").Logger)
			if err != nil {
				a.logger.Warn().Err(err).Msg("Failed to initialize audio processor - VAD filtering disabled")
			} else {
				a.audioProcessor = audioProcessor
				a.logger.Info().Msg("Audio processor initialized - VAD filtering enabled")
			}
		}
	}

	// Initialize speech manager
	speechMgr, err := engine.NewSpeechManager(a.config.Speech, a.logger.WithComponent("speech").Logger)
	if err != nil {
		return fmt.Errorf("failed to create speech manager: %w", err)
	}
	a.speechMgr = speechMgr

	// Start speech recognition with timeout context (initialization only)
	if err := a.speechMgr.Start(ctx); err != nil {
		return fmt.Errorf("failed to start speech manager: %w", err)
	}

	// Setup debug saver if enabled
	if a.debugAudio {
		debugSaver := debug.NewSegmentSaver(a.debugOutputDir, true, a.logger.WithComponent("debug_saver").Logger)
		a.speechMgr.SetDebugSaver(debugSaver)
		a.logger.Info().Str("output_dir", a.debugOutputDir).Msg("Debug audio saving enabled")
	}

	// Initialize commentary system
	if err := a.initializeCommentarySystem(ctx); err != nil {
		a.logger.Warn().Err(err).Msg("Failed to initialize commentary system - continuing without commentary")
		a.commentaryEnabled = false
	} else {
		a.commentaryEnabled = true
		a.logger.Info().Msg("Commentary system initialized successfully")
	}

	if a.audioMgr != nil {
		a.logger.Info().Msg("Speech recognition system initialized successfully with audio capture")
	} else {
		a.logger.Info().Msg("Speech recognition system initialized in test mode (no audio capture)")
	}
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
	if a.audioFile != "" {
		return a.processAudioFile(ctx)
	}

	if a.pipelineMode {
		return a.pipelineLoop(ctx)
	}

	if a.speechOnly {
		return a.speechOnlyLoop(ctx)
	}

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

// initializePipelineMode initializes the pipeline system
func (a *Application) initializePipelineMode(ctx context.Context) error {
	// Initialize audio manager first
	audioMgr, err := audio.NewAudioManager(a.config.Audio, a.logger.WithComponent("audio").Logger)
	if err != nil {
		return fmt.Errorf("failed to create audio manager: %w", err)
	}
	a.audioMgr = audioMgr

	// Start audio capture with long-lived context
	longLivedCtx := context.Background()
	if err := a.audioMgr.Start(longLivedCtx); err != nil {
		return fmt.Errorf("failed to start audio manager: %w", err)
	}

	// Create pipeline manager with all required configurations (no audio config needed)
	pipelineConfig := &pipeline.PipelineManagerConfig{
		Speech:   a.config.Speech,
		VAD:      a.config.Pipeline.VAD,
		Pipeline: a.config.Pipeline.Pipeline,
	}

	// Set defaults for pipeline configuration
	pipelineConfig.SetDefaults()

	// Initialize pipeline manager with existing audio manager
	pipelineMgr, err := pipeline.NewManager(a.audioMgr, pipelineConfig, a.logger.WithComponent("pipeline").Logger)
	if err != nil {
		return fmt.Errorf("failed to create pipeline manager: %w", err)
	}
	a.pipelineMgr = pipelineMgr

	// Start pipeline manager with the same long-lived context
	if err := a.pipelineMgr.Start(longLivedCtx); err != nil {
		return fmt.Errorf("failed to start pipeline manager: %w", err)
	}

	a.logger.Info().Msg("VAD-driven pipeline system initialized successfully")
	return nil
}

// initializeCommentarySystem initializes the commentary generation system
func (a *Application) initializeCommentarySystem(ctx context.Context) error {
	// Initialize LLM engine
	llmEngine, err := llmEngine.NewLlmEngine(&a.config.LLMConfig, ctx, a.logger.WithComponent("llm").Logger)
	if err != nil {
		return fmt.Errorf("failed to create LLM engine: %w", err)
	}
	a.llmEngine = llmEngine

	// Initialize template manager and load HEMA templates
	templateManager := commentaryTemplates.NewTemplateManager()

	// Register HEMA templates manually since the auto-registration uses the default manager
	for _, tmpl := range commentaryTemplates.HEMATemplates {
		if err := templateManager.RegisterTemplate(tmpl); err != nil {
			return fmt.Errorf("failed to register template %s: %w", tmpl.ID, err)
		}
	}

	// Initialize context manager
	contextManager := commentaryContext.NewContextManager(a.logger.WithComponent("context").Logger)

	// Initialize prompt builder
	builderConfig := commentaryPrompt.DefaultBuilderConfig()
	builderConfig.DefaultTemplate = "point_scored"

	promptBuilder := commentaryPrompt.NewBuilder(
		templateManager,
		contextManager,
		builderConfig,
		a.logger.WithComponent("prompt").Logger,
	)

	// Initialize commentary generator
	commentaryGen, err := commentaryEngine.NewCommentaryGenerator(
		llmEngine,
		promptBuilder,
		&a.config.Commentary,
		a.logger.WithComponent("commentary").Logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create commentary generator: %w", err)
	}
	a.commentaryGen = commentaryGen

	// Start commentary generator
	if err := a.commentaryGen.Start(); err != nil {
		return fmt.Errorf("failed to start commentary generator: %w", err)
	}

	a.logger.Info().Msg("Commentary system initialized successfully")
	return nil
}

// processCommentaryForTranscription generates commentary for a transcription result
func (a *Application) processCommentaryForTranscription(ctx context.Context, transcription *speechTypes.TranscriptionResult) error {
	// Create commentary request
	request := &commentaryTypes.CommentaryRequest{
		Input: commentaryTypes.TranscriptionInput{
			Text:       transcription.Text,
			Confidence: float32(transcription.Confidence),
			Timestamp:  time.Now(),
		},
		MaxLatency:  2 * time.Second, // Quick response for live commentary
		Quality:     commentaryTypes.QualityLevelBalanced,
		CachePolicy: commentaryTypes.CachePolicyDefault,
	}

	// Add audio quality if available from metadata
	if transcription.Metadata.AudioQuality > 0 {
		request.Input.AudioMetrics = &commentaryTypes.AudioQuality{
			SignalToNoise:   float32(transcription.Metadata.AudioQuality),
			Clarity:         float32(transcription.Metadata.AudioQuality),
			VoiceDetection:  true,
			BackgroundNoise: 1.0 - float32(transcription.Metadata.AudioQuality),
			Distortion:      0.2,
		}
	}

	// Generate commentary
	response, err := a.commentaryGen.Generate(ctx, request)
	if err != nil {
		return fmt.Errorf("commentary generation failed: %w", err)
	}

	if !response.Success {
		return fmt.Errorf("commentary generation unsuccessful: %s", response.Error)
	}

	// Log and display the commentary result
	if response.Commentary != nil {
		a.logger.Info().
			Str("judge_call", transcription.Text).
			Str("commentary", response.Commentary.Text).
			Float32("confidence", response.Commentary.Confidence).
			Dur("generation_time", response.Latency).
			Str("source", response.Commentary.Source).
			Msg("Commentary generated")

		// Print to stdout for easy viewing during testing
		fmt.Printf("\n=== LIVE COMMENTARY ===\n")
		fmt.Printf("Judge: %s\n", transcription.Text)
		fmt.Printf("Commentary: %s\n", response.Commentary.Text)
		fmt.Printf("Confidence: %.2f | Generation Time: %v\n", response.Commentary.Confidence, response.Latency)
		fmt.Printf("Source: %s\n", response.Commentary.Source)
		if response.Commentary.TemplateID != "" {
			fmt.Printf("Template: %s\n", response.Commentary.TemplateID)
		}
		fmt.Printf("========================\n")
	}

	return nil
}

// pipelineLoop runs the main pipeline processing loop
func (a *Application) pipelineLoop(ctx context.Context) error {
	a.logger.Info().Msg("Starting VAD-driven pipeline - listening for speech...")

	// Monitor pipeline errors
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-a.pipelineMgr.GetErrors():
				if err != nil {
					a.logger.Error().Err(err).Msg("Pipeline error")
				}
			}
		}
	}()

	// Log metrics periodically
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			a.logger.Info().Msg("Shutting down pipeline...")
			return a.shutdownPipeline()
		case <-ticker.C:
			// Log current metrics
			metrics := a.pipelineMgr.GetMetrics()
			if metrics != nil {
				a.logger.Info().
					Interface("metrics", metrics).
					Str("state", a.pipelineMgr.GetState().String()).
					Msg("Pipeline status")
			}
		}
	}
}

// shutdownPipeline shuts down the pipeline system
func (a *Application) shutdownPipeline() error {
	a.logger.Info().Msg("Cleaning up pipeline resources...")

	// Stop pipeline manager
	if a.pipelineMgr != nil {
		if err := a.pipelineMgr.Stop(); err != nil {
			a.logger.Error().Err(err).Msg("Error stopping pipeline manager")
		}
	}

	a.logger.Info().Msg("Pipeline shutdown complete")
	return nil
}

func (a *Application) speechOnlyLoop(ctx context.Context) error {
	a.logger.Info().Msg("Starting speech recognition loop - speak into your microphone")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			a.logger.Info().Msg("Shutting down speech recognition...")
			return a.shutdownSpeechOnly()
		case <-ticker.C:
			// Extract recent audio and transcribe
			if err := a.processRecentAudio(ctx); err != nil {
				a.logger.Error().Err(err).Msg("Error processing audio")
			}
		}
	}
}

func (a *Application) processRecentAudio(ctx context.Context) error {
	// Skip if no audio manager available (noaudio build)
	if a.audioMgr == nil {
		a.logger.Debug().Msg("No audio manager available - skipping audio processing")
		return nil
	}

	// Extract last 3 seconds of audio
	extractReq := audioTypes.ExtractionRequest{
		Duration: 3 * time.Second,
		EndTime:  time.Now(),
		Format:   "raw",
	}

	segment, err := a.audioMgr.ExtractAudio(ctx, extractReq)
	if err != nil {
		return fmt.Errorf("failed to extract audio: %w", err)
	}

	// Validate audio segment before transcription
	if segment == nil {
		a.logger.Debug().Msg("Extracted audio segment is nil, skipping transcription")
		return nil
	}

	if len(segment.Data) == 0 {
		a.logger.Debug().Msg("Extracted audio segment is empty, skipping transcription")
		return nil
	}

	// Check for minimum audio data (at least 100ms worth)
	minDuration := 100 * time.Millisecond
	if segment.Duration < minDuration {
		a.logger.Debug().
			Dur("duration", segment.Duration).
			Dur("min_duration", minDuration).
			Int("data_length", len(segment.Data)).
			Msg("Audio segment too short for transcription, skipping")
		return nil
	}

	// Additional check for minimum data size
	minBytes := 8000 // Minimum bytes for meaningful audio (about 50ms at 44.1kHz stereo 16-bit)
	if len(segment.Data) < minBytes {
		a.logger.Debug().
			Int("data_length", len(segment.Data)).
			Int("min_bytes", minBytes).
			Dur("duration", segment.Duration).
			Msg("Audio segment has insufficient data, skipping transcription")
		return nil
	}

	// Apply VAD filtering and quality assessment if audio processor is available
	if a.audioProcessor != nil {
		hasVoice := a.audioProcessor.DetectVoiceActivity(segment.Data)

		// Always assess quality metrics for logging
		qualityMetrics := a.audioProcessor.AssessAudioQuality(segment.Data)
		snr := 20.0 * math.Log10(qualityMetrics.RMSLevel/qualityMetrics.NoiseLevel)

		if !hasVoice {
			a.logger.Debug().
				Dur("duration", segment.Duration).
				Int("data_length", len(segment.Data)).
				Float64("quality", qualityMetrics.Quality).
				Float64("snr", snr).
				Float64("rms_level", qualityMetrics.RMSLevel).
				Float64("noise_level", qualityMetrics.NoiseLevel).
				Bool("has_voice_flag", qualityMetrics.HasVoice).
				Msg("No voice activity detected by VAD, skipping transcription")
			return nil
		}

		// Quality metrics already assessed above for logging

		// Define quality thresholds to filter out coughs, throat clearing, etc.
		// Made more permissive to allow real speech through
		const (
			minQuality    = 0.1   // Minimum overall quality score (lowered from 0.3)
			minSNR        = 3.0   // Minimum signal-to-noise ratio in dB (lowered from 8.0)
			minRMSLevel   = 0.005 // Minimum RMS level to ensure sufficient signal (lowered from 0.02)
			maxNoiseLevel = 0.8   // Maximum acceptable noise level (raised from 0.3)
		)

		// SNR already calculated above for logging

		// Check quality thresholds
		if qualityMetrics.Quality < minQuality {
			a.logger.Debug().
				Float64("quality", qualityMetrics.Quality).
				Float64("min_quality", minQuality).
				Msg("Audio quality too low, skipping transcription")
			return nil
		}

		if snr < minSNR {
			a.logger.Debug().
				Float64("snr", snr).
				Float64("min_snr", minSNR).
				Msg("Signal-to-noise ratio too low, skipping transcription")
			return nil
		}

		if qualityMetrics.RMSLevel < minRMSLevel {
			a.logger.Debug().
				Float64("rms_level", qualityMetrics.RMSLevel).
				Float64("min_rms", minRMSLevel).
				Msg("Audio signal too weak, skipping transcription")
			return nil
		}

		if qualityMetrics.NoiseLevel > maxNoiseLevel {
			a.logger.Debug().
				Float64("noise_level", qualityMetrics.NoiseLevel).
				Float64("max_noise", maxNoiseLevel).
				Msg("Noise level too high, skipping transcription")
			return nil
		}

		a.logger.Debug().
			Dur("duration", segment.Duration).
			Int("data_length", len(segment.Data)).
			Float64("quality", qualityMetrics.Quality).
			Float64("snr", snr).
			Float64("rms_level", qualityMetrics.RMSLevel).
			Float64("noise_level", qualityMetrics.NoiseLevel).
			Msg("Audio passed quality checks, proceeding with transcription")
	}

	// Transcribe the audio
	result, err := a.speechMgr.TranscribeAudio(ctx, segment)
	if err != nil {
		return fmt.Errorf("failed to transcribe audio: %w", err)
	}

	// Log the transcription result
	if result.Text != "" && result.Confidence > 0.5 {
		a.logger.Info().
			Str("text", result.Text).
			Float64("confidence", result.Confidence).
			Dur("duration", result.Duration).
			Msg("Transcription result")

		// Generate commentary if commentary system is enabled
		if a.commentaryEnabled {
			if err := a.processCommentaryForTranscription(ctx, result); err != nil {
				a.logger.Error().Err(err).Msg("Failed to generate commentary")
			}
		}
	}

	return nil
}

func (a *Application) shutdownSpeechOnly() error {
	a.logger.Info().Msg("Cleaning up speech recognition resources...")

	// Stop commentary generator
	if a.commentaryEnabled && a.commentaryGen != nil {
		if err := a.commentaryGen.Stop(); err != nil {
			a.logger.Error().Err(err).Msg("Error stopping commentary generator")
		}
	}

	// Stop LLM engine
	if a.llmEngine != nil {
		if err := a.llmEngine.Close(); err != nil {
			a.logger.Error().Err(err).Msg("Error closing LLM engine")
		}
	}

	// Stop speech manager
	if a.speechMgr != nil {
		if err := a.speechMgr.Stop(); err != nil {
			a.logger.Error().Err(err).Msg("Error stopping speech manager")
		}
	}

	// Stop audio processor
	if a.audioProcessor != nil {
		if err := a.audioProcessor.Close(); err != nil {
			a.logger.Error().Err(err).Msg("Error closing audio processor")
		}
	}

	// Stop audio manager
	if a.audioMgr != nil {
		if err := a.audioMgr.Stop(); err != nil {
			a.logger.Error().Err(err).Msg("Error stopping audio manager")
		}
	}

	a.logger.Info().Msg("Speech recognition shutdown complete")
	return nil
}

func (a *Application) shutdown() error {
	if a.audioFile != "" || a.speechOnly {
		return a.shutdownSpeechOnly()
	}
	if a.pipelineMode {
		return a.shutdownPipeline()
	}
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

// initializeAudioFileMode initializes the application for processing a single audio file
func (a *Application) initializeAudioFileMode(ctx context.Context) error {
	// Initialize speech manager only (no audio capture needed)
	speechMgr, err := engine.NewSpeechManager(a.config.Speech, a.logger.WithComponent("speech").Logger)
	if err != nil {
		return fmt.Errorf("failed to create speech manager: %w", err)
	}
	a.speechMgr = speechMgr

	// Start speech recognition
	if err := a.speechMgr.Start(ctx); err != nil {
		return fmt.Errorf("failed to start speech manager: %w", err)
	}

	// Setup debug saver if enabled
	if a.debugAudio {
		debugSaver := debug.NewSegmentSaver(a.debugOutputDir, true, a.logger.WithComponent("debug_saver").Logger)
		a.speechMgr.SetDebugSaver(debugSaver)
		a.logger.Info().Str("output_dir", a.debugOutputDir).Msg("Debug audio saving enabled")
	}

	a.logger.Info().Msg("Audio file processing mode initialized successfully")
	return nil
}

// processAudioFile processes a single audio file and outputs the transcription
func (a *Application) processAudioFile(ctx context.Context) error {
	a.logger.Info().Str("file", a.audioFile).Msg("Processing audio file")

	// Load audio file
	segment, err := a.loadAudioFile(a.audioFile)
	if err != nil {
		return fmt.Errorf("failed to load audio file: %w", err)
	}

	a.logger.Info().
		Int("data_length", len(segment.Data)).
		Dur("duration", segment.Duration).
		Int("sample_rate", segment.Metadata.SampleRate).
		Int("channels", segment.Metadata.Channels).
		Msg("Audio file loaded successfully")

	// Transcribe the audio
	result, err := a.speechMgr.TranscribeAudio(ctx, segment)
	if err != nil {
		return fmt.Errorf("failed to transcribe audio: %w", err)
	}

	// Output the results
	a.logger.Info().
		Str("text", result.Text).
		Float64("confidence", result.Confidence).
		Dur("processing_time", result.Duration).
		Str("language", result.Language).
		Int("segments", len(result.Segments)).
		Msg("Transcription completed")

	// Print detailed results to stdout
	fmt.Printf("\n=== TRANSCRIPTION RESULTS ===\n")
	fmt.Printf("File: %s\n", a.audioFile)
	fmt.Printf("Text: %s\n", result.Text)
	fmt.Printf("Confidence: %.2f\n", result.Confidence)
	fmt.Printf("Language: %s\n", result.Language)
	fmt.Printf("Processing Time: %v\n", result.Duration)
	fmt.Printf("Segments: %d\n", len(result.Segments))

	if len(result.Segments) > 0 {
		fmt.Printf("\n=== DETAILED SEGMENTS ===\n")
		for i, segment := range result.Segments {
			fmt.Printf("Segment %d: %s (%.2f confidence, %v - %v)\n",
				i+1, segment.Text, segment.Confidence, segment.StartTime, segment.EndTime)
		}
	}

	if len(result.Metadata.HEMATermsFound) > 0 {
		fmt.Printf("\n=== HEMA TERMS FOUND ===\n")
		for _, term := range result.Metadata.HEMATermsFound {
			fmt.Printf("- %s\n", term)
		}
	}

	fmt.Printf("\n=== METADATA ===\n")
	fmt.Printf("Model: %s\n", result.Metadata.ModelUsed)
	fmt.Printf("Processing Time: %v\n", result.Metadata.ProcessingTime)
	fmt.Printf("Token Count: %d\n", result.Metadata.TokenCount)
	fmt.Printf("Metal Accelerated: %t\n", result.Metadata.MetalAccelerated)
	fmt.Printf("Vocabulary Boost: %t\n", result.Metadata.VocabularyBoost)

	return nil
}

// loadAudioFile loads an audio file and converts it to an AudioSegment
func (a *Application) loadAudioFile(filePath string) (*audioTypes.AudioSegment, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("audio file does not exist: %s", filePath)
	}

	// Determine file type by extension
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".wav":
		return a.loadWAVFile(filePath)
	case ".raw", ".pcm":
		return a.loadRawFile(filePath)
	default:
		return nil, fmt.Errorf("unsupported audio file format: %s (supported: .wav, .raw, .pcm)", ext)
	}
}

// loadWAVFile loads a WAV file using go-audio/wav package
func (a *Application) loadWAVFile(filePath string) (*audioTypes.AudioSegment, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open WAV file: %w", err)
	}
	defer file.Close()

	// Create WAV decoder
	decoder := wav.NewDecoder(file)
	if decoder == nil {
		return nil, fmt.Errorf("failed to create WAV decoder")
	}

	// Read the full PCM buffer
	buffer, err := decoder.FullPCMBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to read WAV data: %w", err)
	}

	// Get audio parameters
	sampleRate := int(decoder.SampleRate)
	channels := int(decoder.NumChans)
	bitDepth := int(decoder.BitDepth)

	// Convert to float32 samples
	samples := buffer.AsFloat32Buffer().Data

	// Calculate duration
	samplesPerChannel := len(samples) / channels
	duration := time.Duration(float64(samplesPerChannel)/float64(sampleRate)) * time.Second

	// Create audio segment
	segment := &audioTypes.AudioSegment{
		ID:        fmt.Sprintf("file_%d", time.Now().UnixNano()),
		Data:      samples,
		StartTime: time.Now(),
		Duration:  duration,
		Metadata: audioTypes.SegmentMetadata{
			SampleRate: sampleRate,
			Channels:   channels,
			BitDepth:   bitDepth,
			Quality:    1.0, // Assume good quality for file input
		},
	}

	return segment, nil
}

// loadRawFile loads a raw PCM file (assumes 16-bit, 44.1kHz, stereo)
func (a *Application) loadRawFile(filePath string) (*audioTypes.AudioSegment, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open raw file: %w", err)
	}
	defer file.Close()

	// Read all audio data
	audioData, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read raw audio data: %w", err)
	}

	// Default parameters for raw files (can be made configurable)
	sampleRate := 44100
	channels := 2
	bitsPerSample := 16

	// Convert byte data to float32 samples
	samples, err := a.convertBytesToFloat32(audioData, bitsPerSample)
	if err != nil {
		return nil, fmt.Errorf("failed to convert audio data: %w", err)
	}

	// Calculate duration
	samplesPerChannel := len(audioData) / (channels * 2) // 2 bytes per 16-bit sample
	duration := time.Duration(float64(samplesPerChannel)/float64(sampleRate)) * time.Second

	// Create audio segment
	segment := &audioTypes.AudioSegment{
		ID:        fmt.Sprintf("file_%d", time.Now().UnixNano()),
		Data:      samples,
		StartTime: time.Now(),
		Duration:  duration,
		Metadata: audioTypes.SegmentMetadata{
			SampleRate: sampleRate,
			Channels:   channels,
			BitDepth:   bitsPerSample,
			Quality:    1.0, // Assume good quality for file input
		},
	}

	return segment, nil
}

// convertBytesToFloat32 converts raw audio bytes to float32 samples
func (a *Application) convertBytesToFloat32(data []byte, bitsPerSample int) ([]float32, error) {
	switch bitsPerSample {
	case 16:
		if len(data)%2 != 0 {
			return nil, fmt.Errorf("invalid data length for 16-bit samples: %d bytes", len(data))
		}

		samples := make([]float32, len(data)/2)
		for i := 0; i < len(samples); i++ {
			// Convert 16-bit PCM to float32 (-1.0 to 1.0)
			// Little-endian format: low byte first, high byte second
			sample := int16(data[i*2]) | int16(data[i*2+1])<<8
			samples[i] = float32(sample) / 32768.0
		}
		return samples, nil

	default:
		return nil, fmt.Errorf("unsupported bits per sample: %d (only 16-bit supported)", bitsPerSample)
	}
}
