#!/bin/bash

# Exit immediately if a command exits with a non-zero status.
set -e

# Get the root directory of the project
PROJECT_ROOT=$(pwd)
EXAMPLES_DIR="$PROJECT_ROOT/examples"
TEST_VIDEO="$EXAMPLES_DIR/test_video.mp4"
INPUT_VIDEO_NAME="input_video.mp4"

# Check if examples directory exists
if [ ! -d "$EXAMPLES_DIR" ]; then
  echo "Error: Examples directory not found at $EXAMPLES_DIR" >&2
  exit 1
fi

# Check if test video exists
if [ ! -f "$TEST_VIDEO" ]; then
  echo "Error: Test video not found at $TEST_VIDEO" >&2
  exit 1
fi

# Ensure FFmpeg/FFprobe are available (basic check)
if ! command -v ffmpeg &> /dev/null || ! command -v ffprobe &> /dev/null; then
    echo "Error: ffmpeg and/or ffprobe could not be found. Please ensure they are installed and in your PATH." >&2
    exit 1
fi

echo "Running examples tests using $TEST_VIDEO..."

# Loop through each example directory
for example_dir in "$EXAMPLES_DIR"/*; do
  if [ -d "$example_dir" ]; then
    example_name=$(basename "$example_dir")
    echo "------------------------------------"
    echo "Running example: $example_name"
    echo "------------------------------------"

    cd "$example_dir"

    # Clean up any potential leftover mod files
    rm -f go.mod go.sum

    # Copy the test video into the example directory
    echo "Copying test video..."
    cp "$TEST_VIDEO" "./$INPUT_VIDEO_NAME"

    # Run go run from the example dir
    echo "Running go run ./main.go..."
    go run ./main.go

    # Check for expected output (basic check)
    echo "Checking for output..."
    output_ok=false
    case $example_name in
      local_to_hls)
        [ -f "output_hls_local/master.m3u8" ] && output_ok=true
        expected_output="output_hls_local/master.m3u8"
        ;;
      remote_download_to_hls)
        # This example doesn't use the local input video
        [ -f "output_hls_remote_download/master.m3u8" ] && output_ok=true
        expected_output="output_hls_remote_download/master.m3u8"
        ;;
      remote_stream_to_hls)
        # This example doesn't use the local input video
        [ -f "output_hls_remote_stream/master.m3u8" ] && output_ok=true
        expected_output="output_hls_remote_stream/master.m3u8"
        ;;
      custom_resolutions)
        [ -f "output_hls_custom_res/master.m3u8" ] && output_ok=true
        expected_output="output_hls_custom_res/master.m3u8"
        ;;
      to_mp4)
        [ -f "output_video.mp4" ] && output_ok=true
        expected_output="output_video.mp4"
        ;;
      *)
        echo "Warning: No output check defined for example $example_name"
        output_ok=true # Skip check for unknown examples
        ;;
    esac

    if [ "$output_ok" = false ]; then
      echo "Error: Expected output '$expected_output' not found for example '$example_name'" >&2
      cd "$PROJECT_ROOT" # Go back before exiting
      exit 1
    else
       echo "Output check passed."
    fi

    # Clean up generated files
    echo "Cleaning up example directory..."
    rm -rf output_* temp_downloads "./$INPUT_VIDEO_NAME" # Remove copied video

    # Go back to the project root directory
    cd "$PROJECT_ROOT"
    echo "Example $example_name finished successfully."
    echo
  fi
done

echo "------------------------------------"
echo "All examples tests passed successfully!"
echo "------------------------------------"

exit 0 