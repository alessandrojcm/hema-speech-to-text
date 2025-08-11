package main

import (
	"context"
	"flag"
	"fmt"
	"io"
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
	audioTypes "github.com/your-org/hema-replay-system/pkg/audio/types"
	"github.com/your-org/hema-replay-system/pkg/logger"
	"github.com/your-org/hema-replay-system/pkg/speech/engine"
)

type Application struct {
	config     *config.Config
	logger     *logger.Logger
	obsClient  *obs.Client
	replayMgr  *replay.Manager
	textMgr    *text.Manager
	sceneMgr   *scene.Manager
	audioMgr   *audio.AudioManager
	speechMgr  *engine.SpeechManager
	speechOnly bool
	audioFile  string
}

func main() {
	var configPath string
	var speechOnly bool
	var audioFile string
	flag.StringVar(&configPath, "config", "", "Path to configuration file")
	flag.BoolVar(&speechOnly, "speech-only", false, "Run only speech recognition system (for testing)")
	flag.StringVar(&audioFile, "audio-file", "", "Process a single audio file and exit (for testing)")
	flag.Parse()

	if err := run(configPath, speechOnly, audioFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(configPath string, speechOnly bool, audioFile string) error {
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
		config:     cfg,
		logger:     log,
		speechOnly: speechOnly,
		audioFile:  audioFile,
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
		// Start audio capture
		if err := a.audioMgr.Start(ctx); err != nil {
			a.logger.Warn().Err(err).Msg("Failed to start audio manager - continuing without audio")
			a.audioMgr = nil
		}
	}

	// Initialize speech manager
	speechMgr, err := engine.NewSpeechManager(a.config.Speech, a.logger.WithComponent("speech").Logger)
	if err != nil {
		return fmt.Errorf("failed to create speech manager: %w", err)
	}
	a.speechMgr = speechMgr

	// Start speech recognition
	if err := a.speechMgr.Start(ctx); err != nil {
		return fmt.Errorf("failed to start speech manager: %w", err)
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
	}

	return nil
}

func (a *Application) shutdownSpeechOnly() error {
	a.logger.Info().Msg("Cleaning up speech recognition resources...")

	// Stop speech manager
	if a.speechMgr != nil {
		if err := a.speechMgr.Stop(); err != nil {
			a.logger.Error().Err(err).Msg("Error stopping speech manager")
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
