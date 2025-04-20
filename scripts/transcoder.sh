#!/bin/bash

# HLSpresso - Helper script for video transcoding
# This script is a wrapper around the HLSpresso binary to facilitate common use

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Help function
function show_help {
    echo -e "${BLUE}☕ HLSpresso - Helper script for video transcoding${NC}"
    echo 
    echo "Usage:"
    echo "  ./HLSpresso.sh -i INPUT -o OUTPUT [OPTIONS]"
    echo 
    echo "Options:"
    echo "  -i, --input FILE      Input video file or URL"
    echo "  -o, --output DIR      Output directory or file for MP4"
    echo "  -t, --type TYPE       Output type: 'hls' (default) or 'mp4'"
    echo "  -d, --duration SEC    HLS segment duration in seconds (default: 10)"
    echo "  -p, --playlist TYPE   HLS playlist type: 'vod' (default) or 'event'"
    echo "  -r, --resolutions RES List of resolutions for HLS"
    echo "                        Format: widthxheight:v:bitrate:a:bitrate,..."
    echo "  -f, --ffmpeg PATH     Path to ffmpeg binary"
    echo "  -R, --remote          Download input file from URL"
    echo "  -D, --download-dir DIR Directory to save downloaded files (default: downloads)"
    echo "  -O, --overwrite       Overwrite existing files"
    echo "  -h, --help            Show this help"
    echo 
    echo "Examples:"
    echo "  ./HLSpresso.sh -i video.mp4 -o output_dir"
    echo "  ./HLSpresso.sh -i video.mp4 -o output.mp4 -t mp4"
    echo "  ./HLSpresso.sh -i http://example.com/video.mp4 -o output_dir -R"
    echo "  ./HLSpresso.sh -i video.mp4 -o output_dir -d 6 -p vod"
    echo "  ./HLSpresso.sh -i video.mp4 -o output_dir -r \"1920x1080:v:5000k:a:192k,1280x720:v:2800k:a:128k\""
    echo 
}

# Check if binary exists
function check_binary {
    # Find the binary
    if [ -x "./HLSpresso" ]; then
        BINARY="./HLSpresso"
    elif command -v HLSpresso > /dev/null; then
        BINARY="HLSpresso"
    else
        echo -e "${RED}Error: HLSpresso binary not found.${NC}"
        echo "Run 'make build' or 'go build -o HLSpresso cmd/transcoder/main.go' to compile."
        exit 1
    fi
}

# Convert resolution format
function convert_resolution_format {
    local input="$1"
    local output=""
    
    # Input example: 1920x1080:v:5000k:a:192k
    # Output example: 1920x1080:5000k:5350k:7500k:192k
    
    IFS=',' read -ra RESOLUTIONS <<< "$input"
    for res in "${RESOLUTIONS[@]}"; do
        if [[ "$res" =~ ([0-9]+x[0-9]+):v:([0-9]+k):a:([0-9]+k) ]]; then
            width_height="${BASH_REMATCH[1]}"
            video_bitrate="${BASH_REMATCH[2]}"
            audio_bitrate="${BASH_REMATCH[3]}"
            
            # Calculate default values for maxrate and bufsize
            # maxrate = bitrate * 1.07
            # bufsize = bitrate * 1.5
            video_bitrate_num=${video_bitrate%k}
            maxrate=$((video_bitrate_num * 107 / 100))
            bufsize=$((video_bitrate_num * 15 / 10))
            
            # Add to output
            if [ -n "$output" ]; then
                output="$output,"
            fi
            output="${output}${width_height}:${video_bitrate}:${maxrate}k:${bufsize}k:${audio_bitrate}"
        else
            echo -e "${YELLOW}Warning: Unrecognized resolution format: $res. Using value directly.${NC}"
            if [ -n "$output" ]; then
                output="$output,"
            fi
            output="${output}${res}"
        fi
    done
    
    echo "$output"
}

# Check if there are arguments
if [ $# -eq 0 ]; then
    show_help
    exit 0
fi

# Process arguments
INPUT=""
OUTPUT=""
TYPE="hls"
DURATION=""
PLAYLIST=""
RESOLUTIONS=""
FFMPEG=""
REMOTE=""
DOWNLOAD_DIR=""
OVERWRITE=""

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    key="$1"
    case $key in
        -h|--help)
            show_help
            exit 0
            ;;
        -i|--input)
            INPUT="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT="$2"
            shift 2
            ;;
        -t|--type)
            TYPE="$2"
            shift 2
            ;;
        -d|--duration)
            DURATION="$2"
            shift 2
            ;;
        -p|--playlist)
            PLAYLIST="$2"
            shift 2
            ;;
        -r|--resolutions)
            RESOLUTIONS="$2"
            shift 2
            ;;
        -f|--ffmpeg)
            FFMPEG="$2"
            shift 2
            ;;
        -R|--remote)
            REMOTE="--remote"
            shift
            ;;
        -D|--download-dir)
            DOWNLOAD_DIR="$2"
            shift 2
            ;;
        -O|--overwrite)
            OVERWRITE="--overwrite"
            shift
            ;;
        *)
            echo -e "${RED}Error: Unknown option $1${NC}"
            show_help
            exit 1
            ;;
    esac
done

# Check required arguments
if [ -z "$INPUT" ]; then
    echo -e "${RED}Error: Input file not specified.${NC}"
    show_help
    exit 1
fi

if [ -z "$OUTPUT" ]; then
    echo -e "${RED}Error: Output directory/file not specified.${NC}"
    show_help
    exit 1
fi

# Check binary
check_binary

# Build command
CMD="$BINARY -i \"$INPUT\" -o \"$OUTPUT\" -t $TYPE"

if [ -n "$DURATION" ]; then
    CMD="$CMD --hls-segment-duration $DURATION"
fi

if [ -n "$PLAYLIST" ]; then
    CMD="$CMD --hls-playlist-type $PLAYLIST"
fi

if [ -n "$RESOLUTIONS" ]; then
    CONVERTED_RES=$(convert_resolution_format "$RESOLUTIONS")
    CMD="$CMD --hls-resolutions \"$CONVERTED_RES\""
fi

if [ -n "$FFMPEG" ]; then
    CMD="$CMD --ffmpeg \"$FFMPEG\""
fi

if [ -n "$REMOTE" ]; then
    CMD="$CMD $REMOTE"
fi

if [ -n "$DOWNLOAD_DIR" ]; then
    CMD="$CMD --download-dir \"$DOWNLOAD_DIR\""
fi

if [ -n "$OVERWRITE" ]; then
    CMD="$CMD $OVERWRITE"
fi

# Run command
echo -e "${BLUE}☕ HLSpresso - Starting transcoding${NC}"
echo -e "${YELLOW}Executing: $CMD${NC}"
echo

# Execute the command - we use eval to properly handle quotes
eval $CMD

exit_code=$?
if [ $exit_code -eq 0 ]; then
    echo
    echo -e "${GREEN}✅ Transcoding completed successfully!${NC}"
    exit 0
else
    echo
    echo -e "${RED}❌ Transcoding failed with exit code $exit_code${NC}"
    exit $exit_code
fi 