package main

import (
	"context"
	// "fmt"
	"log"
	"time"

	"github.com/heyjunin/HLSpresso/pkg/errors"
	"github.com/heyjunin/HLSpresso/pkg/hls"
	"github.com/heyjunin/HLSpresso/pkg/progress"
	"github.com/heyjunin/HLSpresso/pkg/transcoder"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	inputFile := "input_video.mp4"
	outputDir := "output_hls_custom_res"

	reporter := progress.NewReporter()

	log.Println("Starting local HLS transcoding with custom resolutions...")

	// Define custom resolutions
	customResolutions := []hls.VideoResolution{
		{Width: 1280, Height: 720, VideoBitrate: "2500k", MaxRate: "2800k", BufSize: "3750k", AudioBitrate: "128k"},
		{Width: 854, Height: 480, VideoBitrate: "1200k", MaxRate: "1400k", BufSize: "1800k", AudioBitrate: "96k"},
		{Width: 640, Height: 360, VideoBitrate: "800k", MaxRate: "900k", BufSize: "1200k", AudioBitrate: "64k"},
	}

	opts := transcoder.Options{
		InputPath:      inputFile,
		OutputPath:     outputDir,
		OutputType:     transcoder.HLSOutput,
		HLSResolutions: customResolutions, // <<<--- Assign custom resolutions
		AllowOverwrite: true,
	}

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

	log.Printf("Custom resolution HLS transcoding finished successfully!")
	log.Printf("Master Playlist: %s", masterPlaylist)
	log.Printf("Output Directory: %s", outputDir)
}
