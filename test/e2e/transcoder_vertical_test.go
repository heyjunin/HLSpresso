package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/heyjunin/HLSpresso/pkg/hls"
	"github.com/heyjunin/HLSpresso/pkg/transcoder"
)

// TestTranscodeVerticalVideoToHLS tests the transcoding of a vertical video (portrait format)
// to HLS format, ensuring that vertical aspect ratios are maintained.
// This test specifically verifies:
// 1. That the vertical video is processed correctly maintaining its orientation (9:16)
// 2. That the generated resolutions maintain the vertical aspect ratio
// 3. That the master playlist contains the correct entries for each vertical resolution
// 4. That all variants and segments are properly generated
func TestTranscodeVerticalVideoToHLS(t *testing.T) {
	// Check if FFmpeg tools are installed
	if !checkFFmpegInstalled() {
		t.Skip("FFmpeg not found, skipping test")
		return
	}

	// Check if the test directory exists, otherwise create it
	setupTestDirectories(t)

	// Path to the vertical test video
	verticalVideoPath := "../../testdata/vertical/vertical_video.mp4"

	// Skip the test if the vertical video doesn't exist
	if _, err := os.Stat(verticalVideoPath); os.IsNotExist(err) {
		t.Skip("Vertical test video not found at: " + verticalVideoPath)
		return
	}

	// Output path
	outputPath := filepath.Join(testOutputDir, "vertical_hls")

	// Define resolutions maintaining the vertical aspect ratio (9:16)
	// Resolutions are defined as width x height to maintain consistency
	verticalResolutions := []hls.VideoResolution{
		{Width: 720, Height: 1280, VideoBitrate: "2800k", MaxRate: "2996k", BufSize: "4200k", AudioBitrate: "128k"}, // Original
		{Width: 540, Height: 960, VideoBitrate: "1400k", MaxRate: "1498k", BufSize: "2100k", AudioBitrate: "96k"},   // 540p
		{Width: 360, Height: 640, VideoBitrate: "800k", MaxRate: "856k", BufSize: "1200k", AudioBitrate: "64k"},     // 360p
	}

	// Create progress reporter for logs
	reporter := NewMockProgressReporter()

	// Transcoder configuration
	options := transcoder.Options{
		InputPath:          verticalVideoPath,
		IsRemoteInput:      false,
		OutputPath:         outputPath,
		OutputType:         transcoder.HLSOutput,
		HLSPlaylistType:    "vod",
		HLSSegmentDuration: 3,
		HLSResolutions:     verticalResolutions,
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Create transcoder
	trans, err := transcoder.New(options, reporter)
	if err != nil {
		t.Fatalf("Error creating transcoder: %v", err)
	}

	// Run transcoding
	masterPlaylistPath, err := trans.Transcode(ctx)
	if err != nil {
		t.Fatalf("Error during HLS transcoding: %v", err)
	}

	// Check if the master playlist was generated
	if _, err := os.Stat(masterPlaylistPath); os.IsNotExist(err) {
		t.Errorf("Master playlist was not generated at: %s", masterPlaylistPath)
	}

	// Check the content of the master playlist to ensure it contains all vertical resolutions
	checkVerticalMasterPlaylist(t, masterPlaylistPath, verticalResolutions)

	// Check if variant playlists were created for each resolution
	checkVariantPlaylists(t, outputPath, len(verticalResolutions))

	// Check if the progress events were generated correctly
	reporter.ValidateProgressEvents(t, false)
}

// checkVerticalMasterPlaylist checks if the master.m3u8 file contains the expected vertical resolutions
func checkVerticalMasterPlaylist(t *testing.T, masterPlaylistPath string, expectedResolutions []hls.VideoResolution) {
	lines := parseM3U8File(t, masterPlaylistPath)

	// Check if it contains the HLS header
	foundHeader := false
	for _, line := range lines {
		if line == "#EXTM3U" {
			foundHeader = true
			break
		}
	}
	if !foundHeader {
		t.Errorf("#EXTM3U header not found in master playlist")
	}

	// Check if there's the correct number of variants
	streamInfoLines := 0
	streamUriLines := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "#EXT-X-STREAM-INF:") {
			streamInfoLines++
		} else if strings.Contains(line, "stream_") && strings.Contains(line, "/playlist.m3u8") {
			streamUriLines++
		}
	}

	if streamInfoLines != len(expectedResolutions) {
		t.Errorf("Incorrect number of stream definitions in master playlist: found %d, expected %d",
			streamInfoLines, len(expectedResolutions))
	}

	if streamUriLines != len(expectedResolutions) {
		t.Errorf("Incorrect number of stream URIs in master playlist: found %d, expected %d",
			streamUriLines, len(expectedResolutions))
	}

	// Check if each vertical resolution is defined
	for _, resolution := range expectedResolutions {
		resolutionString := fmt.Sprintf("%d", resolution.Width)
		heightString := fmt.Sprintf("%d", resolution.Height)
		// For vertical videos, width < height
		if resolution.Width > resolution.Height {
			t.Errorf("Invalid resolution configuration for vertical video: width (%d) > height (%d)",
				resolution.Width, resolution.Height)
		}

		resolutionFound := false
		for _, line := range lines {
			if strings.HasPrefix(line, "#EXT-X-STREAM-INF:") &&
				strings.Contains(line, "RESOLUTION="+resolutionString+"x"+heightString) {
				resolutionFound = true
				break
			}
		}

		if !resolutionFound {
			t.Errorf("Vertical resolution %dx%d not found in master playlist",
				resolution.Width, resolution.Height)
		}
	}
}
