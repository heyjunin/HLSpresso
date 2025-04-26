package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/heyjunin/HLSpresso/pkg/errors"
	"github.com/heyjunin/HLSpresso/pkg/progress"
	"github.com/heyjunin/HLSpresso/pkg/transcoder"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute) // Timeout for transcoding from stream
	defer cancel()

	// Publicly available test video
	remoteURL := "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerBlazes.mp4"
	outputDir := "output_hls_remote_stream"

	reporter := progress.NewReporter()
	defer reporter.Close()

	go func() {
		for p := range reporter.Updates() {
			fmt.Printf("Progress: %.1f%% (Step: %s, Stage: %s)\r", p.Percentage, p.Step, p.Stage)
		}
		fmt.Println() // New line after progress finishes
	}()

	log.Printf("Starting remote HLS transcoding (streaming input) for: %s", remoteURL)

	opts := transcoder.Options{
		InputPath:          remoteURL,
		OutputPath:         outputDir,
		OutputType:         transcoder.HLSOutput,
		StreamFromURL:      true, // <<<--- Enable streaming input from URL
		HLSSegmentDuration: 4,
		AllowOverwrite:     true,
		// DownloadDir is ignored when StreamFromURL is true
	}

	// When StreamFromURL is true, New() does not need/create a downloader.
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

	log.Printf("Remote HLS (streaming) transcoding finished successfully!")
	log.Printf("Master Playlist: %s", masterPlaylist)
	log.Printf("Output Directory: %s", outputDir)
}
