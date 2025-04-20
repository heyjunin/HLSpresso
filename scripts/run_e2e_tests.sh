#!/bin/bash

# Script to run end-to-end tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Preparing environment for e2e tests${NC}"

# Base directory
BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$BASE_DIR"

# Check if FFmpeg is installed
if ! which ffmpeg > /dev/null; then
    echo -e "${RED}FFmpeg not found. Install FFmpeg to run e2e tests.${NC}"
    exit 1
fi

# Create test directories if they don't exist
mkdir -p testdata/temp testdata/downloads

# Compile the binary
echo -e "${YELLOW}Compiling HLSpresso binary...${NC}"
go build -o HLSpresso cmd/transcoder/main.go

if [ $? -ne 0 ]; then
    echo -e "${RED}Failed to compile binary. Aborting tests.${NC}"
    exit 1
fi

echo -e "${GREEN}Binary compiled successfully.${NC}"

# Run tests
echo -e "${YELLOW}Running e2e tests...${NC}"

# Tests for transcoder package
echo -e "${YELLOW}API tests (transcoder)...${NC}"
go test -v ./test/e2e -run TestTranscode

# CLI tests
echo -e "${YELLOW}CLI tests...${NC}"
go test -v ./test/e2e -run TestCLI

# Check result
if [ $? -eq 0 ]; then
    echo -e "${GREEN}All e2e tests passed!${NC}"
    
    # Clean temporary files
    read -p "Do you want to clean temporary test files? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}Cleaning temporary files...${NC}"
        rm -rf testdata/temp/*
        rm -rf testdata/downloads/*
        echo -e "${GREEN}Temporary files removed.${NC}"
    fi
    
    exit 0
else
    echo -e "${RED}Some tests failed. Check the logs above.${NC}"
    exit 1
fi 