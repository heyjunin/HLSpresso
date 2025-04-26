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
	// Downloader might be needed if using NewWithDeps explicitly
	// "github.com/heyjunin/HLSpresso/pkg/downloader"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute) // Longer timeout for download + transcode
	defer cancel()

	// Publicly available test video
	remoteURL := "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4"
	outputDir := "output_hls_remote_download"
	downloadDir := "temp_downloads" // Directory for the intermediate download

	// Clean up download directory afterwards
	defer os.RemoveAll(downloadDir)

	reporter := progress.NewReporter()
	defer reporter.Close()

	go func() {
		for p := range reporter.Updates() {
			fmt.Printf("Progress: %.1f%% (Step: %s, Stage: %s)\r", p.Percentage, p.Step, p.Stage)
		}
		fmt.Println() // New line after progress finishes
	}()

	log.Printf("Starting remote HLS transcoding (download first) for: %s", remoteURL)

	opts := transcoder.Options{
		InputPath:          remoteURL,
		OutputPath:         outputDir,
		OutputType:         transcoder.HLSOutput,
		StreamFromURL:      false, // Explicitly false (default), so download occurs
		DownloadDir:        downloadDir,
		HLSSegmentDuration: 10,
		AllowOverwrite:     true,
	}

	// New() handles creating a default downloader for remote inputs when StreamFromURL is false.
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

	log.Printf("Remote HLS (download) transcoding finished successfully!")
	log.Printf("Master Playlist: %s", masterPlaylist)
	log.Printf("Output Directory: %s", outputDir)
}
