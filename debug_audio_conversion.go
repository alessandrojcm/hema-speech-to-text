package main

import (
	"fmt"
	"log"
	"os"

	"github.com/go-audio/wav"
	"github.com/your-org/hema-replay-system/pkg/audio/processing"
	"github.com/your-org/hema-replay-system/pkg/audio/types"
)

func main() {

	// Read bria.wav
	file, err := os.Open("bria.wav")
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	// Decode WAV
	decoder := wav.NewDecoder(file)
	if !decoder.IsValidFile() {
		log.Fatalf("Invalid WAV file")
	}

	fullBuf, err := decoder.FullPCMBuffer()
	if err != nil {
		log.Fatalf("Failed to read full buffer: %v", err)
	}

	fmt.Printf("Original: %d samples, %d Hz, %d channels\n",
		len(fullBuf.Data), decoder.SampleRate, decoder.NumChans)

	// Convert to float32
	floatBuf := fullBuf.AsFloat32Buffer()
	samples := make([]float32, len(floatBuf.Data))
	for i, v := range floatBuf.Data {
		samples[i] = float32(v)
	}

	// Check samples
	var minSample, maxSample float32
	allZeros := true
	for _, s := range samples {
		if s != 0 {
			allZeros = false
		}
		if s > maxSample {
			maxSample = s
		}
		if s < minSample {
			minSample = s
		}
	}
	fmt.Printf("Float32 samples: %d, all zeros: %v, min: %f, max: %f\n",
		len(samples), allZeros, minSample, maxSample)

	// Create segment and convert to 16kHz mono for whisper
	segment := &types.AudioSegment{
		Data: samples,
		Metadata: types.SegmentMetadata{
			SampleRate: int(decoder.SampleRate),
			Channels:   int(decoder.NumChans),
			BitDepth:   16,
		},
	}

	// Convert to WAV (16kHz mono)
	wavData, err := processing.ConvertToWAV(segment, 16000, 1)
	if err != nil {
		log.Fatalf("Failed to convert to WAV: %v", err)
	}

	// Save the converted file
	if err := os.WriteFile("debug_converted.wav", wavData, 0644); err != nil {
		log.Fatalf("Failed to write file: %v", err)
	}

	fmt.Printf("Written debug_converted.wav (%d bytes)\n", len(wavData))

	// Verify the converted file
	verifyFile, err := os.Open("debug_converted.wav")
	if err != nil {
		log.Fatalf("Failed to open converted file: %v", err)
	}
	defer verifyFile.Close()

	verifyDecoder := wav.NewDecoder(verifyFile)
	if !verifyDecoder.IsValidFile() {
		fmt.Println("ERROR: Converted file is not valid WAV!")
		return
	}

	verifyBuf, err := verifyDecoder.FullPCMBuffer()
	if err != nil {
		log.Fatalf("Failed to read converted buffer: %v", err)
	}

	fmt.Printf("Converted: %d samples, %d Hz, %d channels\n",
		len(verifyBuf.Data), verifyDecoder.SampleRate, verifyDecoder.NumChans)

	// Check converted samples
	verifyFloats := verifyBuf.AsFloat32Buffer().Data
	allZeros = true
	minSample = 0
	maxSample = 0
	for _, s := range verifyFloats {
		v := float32(s)
		if v != 0 {
			allZeros = false
		}
		if v > maxSample {
			maxSample = v
		}
		if v < minSample {
			minSample = v
		}
	}
	fmt.Printf("Converted samples all zeros: %v, min: %f, max: %f\n", allZeros, minSample, maxSample)
}
