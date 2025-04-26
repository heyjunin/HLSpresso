package hls

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNewGeneratorDefaults(t *testing.T) {
	opts := Options{}
	g := New(opts)

	if g.options.SegmentDuration != 10 {
		t.Errorf("Default SegmentDuration: got %d, want 10", g.options.SegmentDuration)
	}
	if g.options.PlaylistType != "vod" {
		t.Errorf("Default PlaylistType: got %q, want \"vod\"", g.options.PlaylistType)
	}
	if g.options.MasterPlaylist != "master.m3u8" {
		t.Errorf("Default MasterPlaylist: got %q, want \"master.m3u8\"", g.options.MasterPlaylist)
	}
	if g.options.SegmentFormat != "mpegts" {
		t.Errorf("Default SegmentFormat: got %q, want \"mpegts\"", g.options.SegmentFormat)
	}
	if g.options.FFmpegBinary != "ffmpeg" {
		t.Errorf("Default FFmpegBinary: got %q, want \"ffmpeg\"", g.options.FFmpegBinary)
	}
	if !reflect.DeepEqual(g.options.Resolutions, DefaultResolutions) {
		t.Errorf("Default Resolutions mismatch:\nGot:      %+v\nExpected: %+v", g.options.Resolutions, DefaultResolutions)
	}
}

func TestBuildFilterGraph(t *testing.T) {
	resolutions := []VideoResolution{
		{Width: 1280, Height: 720},
		{Width: 640, Height: 360},
	}
	numStreams := len(resolutions)

	expected := "[0:v]split=2[v0][v1]; [v0]scale=w=1280:h=720[v0out]; [v1]scale=w=640:h=360[v1out]"
	result := buildFilterGraph(numStreams, resolutions)

	if result != expected {
		t.Errorf("buildFilterGraph() failed:\nGot: %s\nWant: %s", result, expected)
	}

	// Teste com uma stream
	resolutionsSingle := []VideoResolution{{Width: 1920, Height: 1080}}
	expectedSingle := "[0:v]split=1[v0]; [v0]scale=w=1920:h=1080[v0out]"
	resultSingle := buildFilterGraph(1, resolutionsSingle)
	if resultSingle != expectedSingle {
		t.Errorf("buildFilterGraph() single stream failed:\nGot: %s\nWant: %s", resultSingle, expectedSingle)
	}
}

func TestBuildFFmpegArgs(t *testing.T) {
	opts := Options{
		InputFile:       "input.mp4",
		OutputDir:       "./output/hls",
		SegmentDuration: 6,
		PlaylistType:    "event",
		MasterPlaylist:  "index.m3u8",
		SegmentFormat:   "fmp4",
		FFmpegBinary:    "/usr/bin/ffmpeg",
		Resolutions: []VideoResolution{
			{Width: 1280, Height: 720, VideoBitrate: "2M", MaxRate: "2.2M", BufSize: "4M", AudioBitrate: "128k"},
			{Width: 640, Height: 360, VideoBitrate: "800k", MaxRate: "880k", BufSize: "1.6M", AudioBitrate: "64k"},
		},
		FFmpegExtraParams: []string{"-preset", "fast"},
	}
	g := New(opts)

	args := g.buildFFmpegArgs()

	// Verificar alguns argumentos chave
	argsMap := argsToMap(args)

	// Input
	if argsMap["-i"] != "input.mp4" {
		t.Errorf("Missing or incorrect -i argument: got %q", argsMap["-i"])
	}

	// Filter complex
	expectedFilter := "[0:v]split=2[v0][v1]; [v0]scale=w=1280:h=720[v0out]; [v1]scale=w=640:h=360[v1out]"
	if argsMap["-filter_complex"] != expectedFilter {
		t.Errorf("Incorrect -filter_complex:\nGot: %s\nWant: %s", argsMap["-filter_complex"], expectedFilter)
	}

	// Video/Audio maps and options (check a few)
	if !contains(args, "-map", "[v0out]") || !contains(args, "-b:v:0", "2M") || !contains(args, "-map", "a:0") || !contains(args, "-b:a:1", "64k") {
		t.Errorf("Missing or incorrect stream mapping/bitrate options in args: %v", args)
	}

	// HLS options
	if argsMap["-hls_time"] != "6" {
		t.Errorf("Incorrect -hls_time: got %q", argsMap["-hls_time"])
	}
	if argsMap["-hls_playlist_type"] != "event" {
		t.Errorf("Incorrect -hls_playlist_type: got %q", argsMap["-hls_playlist_type"])
	}
	if argsMap["-hls_segment_type"] != "fmp4" {
		t.Errorf("Incorrect -hls_segment_type: got %q", argsMap["-hls_segment_type"])
	}
	if argsMap["-master_pl_name"] != "index.m3u8" {
		t.Errorf("Incorrect -master_pl_name: got %q", argsMap["-master_pl_name"])
	}

	// Stream map
	expectedStreamMap := "v:0,a:0 v:1,a:1"
	if argsMap["-var_stream_map"] != expectedStreamMap {
		t.Errorf("Incorrect -var_stream_map: got %q, want %q", argsMap["-var_stream_map"], expectedStreamMap)
	}

	// Output pattern
	expectedOutputPattern := filepath.Join(opts.OutputDir, "stream_%v/playlist.m3u8")
	if !endsWith(args, expectedOutputPattern) {
		t.Errorf("Args should end with output pattern %q, but got: %v", expectedOutputPattern, args)
	}

	// Extra params
	if !contains(args, "-preset", "fast") {
		t.Errorf("Missing extra params in args: %v", args)
	}

}

// Helper function to convert args slice to a map for easier checking
// Note: assumes flags come before their values
func argsToMap(args []string) map[string]string {
	m := make(map[string]string)
	for i := 0; i < len(args)-1; i++ {
		if strings.HasPrefix(args[i], "-") {
			// Check if next arg is not a flag itself
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				m[args[i]] = args[i+1]
			}
		}
	}
	return m
}

// Helper function to check if a flag/value pair exists
func contains(args []string, flag, value string) bool {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag && args[i+1] == value {
			return true
		}
	}
	return false
}

// Helper function to check if slice ends with a specific value
func endsWith(args []string, value string) bool {
	if len(args) == 0 {
		return false
	}
	return args[len(args)-1] == value
}
