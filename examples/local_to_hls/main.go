package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/heyjunin/HLSpresso/pkg/errors"
	"github.com/heyjunin/HLSpresso/pkg/progress"
	"github.com/heyjunin/HLSpresso/pkg/transcoder"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute) // 5 minute timeout
	defer cancel()

	// Assume input file exists at this path
	inputFile := "input_video.mp4"
	outputDir := "output_hls_local"

	// Create a dummy input file for the example to run
	createDummyFile(inputFile)
	defer os.Remove(inputFile) // Clean up dummy file

	// Create a simple console progress reporter
	reporter := progress.NewReporter() // Default reporter prints to console
	defer reporter.Close()

	go func() {
		for p := range reporter.Updates() {
			fmt.Printf("Progress: %.1f%%\r", p.Percentage)
		}
		fmt.Println() // New line after progress finishes
	}()

	log.Println("Starting local HLS transcoding...")

	opts := transcoder.Options{
		InputPath:          inputFile,
		OutputPath:         outputDir,
		OutputType:         transcoder.HLSOutput,
		HLSSegmentDuration: 6, // Example: 6 second segments
		AllowOverwrite:     true,
	}

	// Use the simple New constructor
	trans, err := transcoder.New(opts, reporter)
	if err != nil {
		log.Fatalf("Failed to create transcoder: %v", err)
	}

	masterPlaylist, err := trans.Transcode(ctx)
	if err != nil {
		if sErr, ok := err.(*errors.StructuredError); ok {
			log.Fatalf("Transcoding failed [%s]: %s (Details: %v)", sErr.Code, sErr.Message, sErr.Details)
		} else {
			log.Fatalf("Transcoding failed: %v", err)
		}
	}

	log.Printf("Local HLS transcoding finished successfully!")
	log.Printf("Master Playlist: %s", masterPlaylist)
	log.Printf("Output Directory: %s", outputDir)
}

// Helper to create a dummy file for testing
func createDummyFile(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("Creating dummy input file: %s", path)
		f, err := os.Create(path)
		if err != nil {
			log.Fatalf("Failed to create dummy file: %v", err)
		}
		// Write some minimal data to avoid potential issues with empty files in ffmpeg/ffprobe
		_, _ = f.WriteString("dummy data")
		f.Close()
	}
}
