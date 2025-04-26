package main

import (
	"context"

	// "fmt"
	"log"
	"time"

	"github.com/heyjunin/HLSpresso/pkg/errors"
	"github.com/heyjunin/HLSpresso/pkg/progress"
	"github.com/heyjunin/HLSpresso/pkg/transcoder"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	inputFile := "input_video.mp4" // Assumes the script copies test_video.mp4 here
	outputFile := "output_video.mp4"

	// createDummyFile(inputFile) // No longer needed
	// defer os.Remove(inputFile) // Cleanup is handled by the script

	reporter := progress.NewReporter()
	// defer reporter.Close() // No longer needed

	// Default reporter prints to console
	// go func() {
	// 	for p := range reporter.Updates() {
	// 		fmt.Printf(\"Progress: %.1f%%\\r\", p.Percentage)
	// 	}
	// 	fmt.Println()
	// }()

	log.Println("Starting local MP4 transcoding...")

	opts := transcoder.Options{
		InputPath:      inputFile,
		OutputPath:     outputFile,           // Specify the output file path
		OutputType:     transcoder.MP4Output, // <<<--- Set output type to MP4
		AllowOverwrite: true,
		// HLS specific options are ignored for MP4 output
		// FFmpegExtraParams: []string{"-crf", "18"}, // Optional: Add custom ffmpeg params
	}

	trans, err := transcoder.New(opts, reporter)
	if err != nil {
		log.Fatalf("Failed to create transcoder: %v", err)
	}

	resultPath, err := trans.Transcode(ctx)
	if err != nil {
		if sErr, ok := err.(*errors.StructuredError); ok {
			log.Fatalf("Transcoding failed [%s]: %s (Details: %v)", sErr.Code, sErr.Message, sErr.Details)
		} else {
			log.Fatalf("Transcoding failed: %v", err)
		}
	}

	log.Printf("MP4 transcoding finished successfully!")
	log.Printf("Output File: %s", resultPath)
}

// // Helper to create a dummy file for testing (REMOVED)
// func createDummyFile(path string) {
// 	if _, err := os.Stat(path); os.IsNotExist(err) {
// 		log.Printf(\"Creating dummy input file: %s\", path)
// 		f, err := os.Create(path)
// 		if err != nil {
// 			log.Fatalf(\"Failed to create dummy file: %v\", err)
// 		}
// 		_, _ = f.WriteString(\"dummy data\")
// 		f.Close()
// 	}
// }
