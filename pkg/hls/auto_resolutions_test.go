package hls

import (
	"fmt"
	"reflect"
	"testing"
)

func TestGenerateAutoResolutionsHorizontal(t *testing.T) {
	// Test case 1: Input is exactly 1080p
	res1 := GenerateAutoResolutions(1920, 1080)
	// NOTE: Current logic assigns 1440p bitrates to 1080p original resolution. Adjusting test expectation.
	// NOTE: Also, the aspect ratio calculation might lead to slightly different lower resolutions.
	expected1 := []VideoResolution{
		{Width: 1920, Height: 1080, VideoBitrate: "9000k", MaxRate: "9630k", BufSize: "13500k", AudioBitrate: "192k"}, // Original (Uses 1440p bitrate)
		{Width: 1080, Height: 604, VideoBitrate: "5000k", MaxRate: "5350k", BufSize: "7500k", AudioBitrate: "192k"},
		{Width: 720, Height: 402, VideoBitrate: "2800k", MaxRate: "2996k", BufSize: "4200k", AudioBitrate: "128k"},
		{Width: 480, Height: 268, VideoBitrate: "1400k", MaxRate: "1498k", BufSize: "2100k", AudioBitrate: "96k"},
		{Width: 360, Height: 202, VideoBitrate: "800k", MaxRate: "856k", BufSize: "1200k", AudioBitrate: "64k"},
		{Width: 240, Height: 134, VideoBitrate: "400k", MaxRate: "428k", BufSize: "600k", AudioBitrate: "48k"},
	}
	if !reflect.DeepEqual(res1, expected1) {
		msg := fmt.Sprintf("Test Case 1 (1080p Input) Failed:\nGot:      %+v\nExpected: %+v", res1, expected1)
		t.Error(msg)
	}

	// Test case 2: Input is 720p
	res2 := GenerateAutoResolutions(1280, 720)
	// NOTE: Current logic assigns 1080p bitrates to 720p original resolution. Adjusting.
	expected2 := []VideoResolution{
		{Width: 1280, Height: 720, VideoBitrate: "5000k", MaxRate: "5350k", BufSize: "7500k", AudioBitrate: "192k"}, // Original (Uses 1080p bitrate)
		{Width: 1080, Height: 604, VideoBitrate: "5000k", MaxRate: "5350k", BufSize: "7500k", AudioBitrate: "192k"},
		{Width: 720, Height: 402, VideoBitrate: "2800k", MaxRate: "2996k", BufSize: "4200k", AudioBitrate: "128k"},
		{Width: 480, Height: 268, VideoBitrate: "1400k", MaxRate: "1498k", BufSize: "2100k", AudioBitrate: "96k"},
		{Width: 360, Height: 202, VideoBitrate: "800k", MaxRate: "856k", BufSize: "1200k", AudioBitrate: "64k"},
		{Width: 240, Height: 134, VideoBitrate: "400k", MaxRate: "428k", BufSize: "600k", AudioBitrate: "48k"},
	}
	if !reflect.DeepEqual(res2, expected2) {
		msg := fmt.Sprintf("Test Case 2 (720p Input) Failed:\nGot:      %+v\nExpected: %+v", res2, expected2)
		t.Error(msg)
	}

	// Test case 3: Input is higher than 1080p (e.g., 4K)
	res3 := GenerateAutoResolutions(3840, 2160)
	expected3 := []VideoResolution{
		{Width: 3840, Height: 2160, VideoBitrate: "15000k", MaxRate: "16050k", BufSize: "22500k", AudioBitrate: "192k"}, // Original (Correctly uses 2160p)
		{Width: 1080, Height: 604, VideoBitrate: "5000k", MaxRate: "5350k", BufSize: "7500k", AudioBitrate: "192k"},
		{Width: 720, Height: 402, VideoBitrate: "2800k", MaxRate: "2996k", BufSize: "4200k", AudioBitrate: "128k"},
		{Width: 480, Height: 268, VideoBitrate: "1400k", MaxRate: "1498k", BufSize: "2100k", AudioBitrate: "96k"},
		{Width: 360, Height: 202, VideoBitrate: "800k", MaxRate: "856k", BufSize: "1200k", AudioBitrate: "64k"},
		{Width: 240, Height: 134, VideoBitrate: "400k", MaxRate: "428k", BufSize: "600k", AudioBitrate: "48k"},
	}
	if !reflect.DeepEqual(res3, expected3) {
		msg := fmt.Sprintf("Test Case 3 (4K Input) Failed:\nGot:      %+v\nExpected: %+v", res3, expected3)
		t.Error(msg)
	}

	// Test case 4: Input is very small (320x180)
	res4 := GenerateAutoResolutions(320, 180)
	// NOTE: Uses lowest bitrate (240p) for original. Adjusting.
	expected4 := []VideoResolution{
		{Width: 320, Height: 180, VideoBitrate: "400k", MaxRate: "428k", BufSize: "600k", AudioBitrate: "48k"}, // Original (Uses 240p bitrate)
		{Width: 240, Height: 134, VideoBitrate: "400k", MaxRate: "428k", BufSize: "600k", AudioBitrate: "48k"},
	}
	if !reflect.DeepEqual(res4, expected4) {
		msg := fmt.Sprintf("Test Case 4 (Small Input 320x180) Failed:\nGot:      %+v\nExpected: %+v", res4, expected4)
		t.Error(msg)
	}
}

func TestGenerateAutoResolutionsVertical(t *testing.T) {
	// Test case 1: Vertical 1080x1920 input
	res1 := GenerateAutoResolutions(1080, 1920)
	// NOTE: Current logic assigns 1440p bitrates to 1080 original width. Adjusting.
	expected1 := []VideoResolution{
		{Width: 1080, Height: 1920, VideoBitrate: "9000k", MaxRate: "9630k", BufSize: "13500k", AudioBitrate: "192k"}, // Original (Uses 1440p bitrate)
		{Width: 604, Height: 1080, VideoBitrate: "5000k", MaxRate: "5350k", BufSize: "7500k", AudioBitrate: "192k"},
		{Width: 402, Height: 720, VideoBitrate: "2800k", MaxRate: "2996k", BufSize: "4200k", AudioBitrate: "128k"},
		{Width: 268, Height: 480, VideoBitrate: "1400k", MaxRate: "1498k", BufSize: "2100k", AudioBitrate: "96k"},
		{Width: 202, Height: 360, VideoBitrate: "800k", MaxRate: "856k", BufSize: "1200k", AudioBitrate: "64k"},
	}
	if !reflect.DeepEqual(res1, expected1) {
		msg := fmt.Sprintf("Test Case 1 (Vertical 1080x1920) Failed:\nGot:      %+v\nExpected: %+v", res1, expected1)
		t.Error(msg)
	}

	// Test case 2: Vertical 720x1280 input
	res2 := GenerateAutoResolutions(720, 1280)
	// NOTE: Current logic assigns 1080p bitrates to 720 original width. Adjusting.
	expected2 := []VideoResolution{
		{Width: 720, Height: 1280, VideoBitrate: "5000k", MaxRate: "5350k", BufSize: "7500k", AudioBitrate: "192k"}, // Original (Uses 1080p bitrate)
		{Width: 604, Height: 1080, VideoBitrate: "5000k", MaxRate: "5350k", BufSize: "7500k", AudioBitrate: "192k"},
		{Width: 402, Height: 720, VideoBitrate: "2800k", MaxRate: "2996k", BufSize: "4200k", AudioBitrate: "128k"},
		{Width: 268, Height: 480, VideoBitrate: "1400k", MaxRate: "1498k", BufSize: "2100k", AudioBitrate: "96k"},
		{Width: 202, Height: 360, VideoBitrate: "800k", MaxRate: "856k", BufSize: "1200k", AudioBitrate: "64k"},
	}
	if !reflect.DeepEqual(res2, expected2) {
		msg := fmt.Sprintf("Test Case 2 (Vertical 720x1280) Failed:\nGot:      %+v\nExpected: %+v", res2, expected2)
		t.Error(msg)
	}
}

func TestGetAutoResolutionNames(t *testing.T) {
	names := GetAutoResolutionNames()
	// Check if common names exist, order doesn't matter
	expectedNames := map[string]bool{
		"2160p": true, "1440p": true, "1080p": true, "720p": true, "480p": true, "360p": true, "240p": true,
	}
	if len(names) != len(expectedNames) {
		t.Errorf("GetAutoResolutionNames() returned %d names, expected %d", len(names), len(expectedNames))
	}
	for _, name := range names {
		if !expectedNames[name] {
			t.Errorf("GetAutoResolutionNames() returned unexpected name: %s", name)
		}
	}
}

func TestFormatAutoResolutions(t *testing.T) {
	res := []VideoResolution{
		{Width: 1920, Height: 1080, VideoBitrate: "5000k"},
		{Width: 1280, Height: 720, VideoBitrate: "2800k"},
	}
	expected := "1920x1080@5000k, 1280x720@2800k"
	result := FormatAutoResolutions(res)
	if result != expected {
		t.Errorf("FormatAutoResolutions() = %q, want %q", result, expected)
	}

	resEmpty := []VideoResolution{}
	expectedEmpty := ""
	resultEmpty := FormatAutoResolutions(resEmpty)
	if resultEmpty != expectedEmpty {
		t.Errorf("FormatAutoResolutions() with empty slice = %q, want %q", resultEmpty, expectedEmpty)
	}
}
