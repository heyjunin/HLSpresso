.PHONY: build clean test test-unit test-e2e install

# Variables
BINARY_NAME=HLSpresso
BUILD_DIR=build
INSTALL_DIR=/usr/local/bin
GO_FILES=$(shell find . -name "*.go" -type f)

# Compilation
build:
	@echo "Compiling HLSpresso..."
	go build -o $(BINARY_NAME) cmd/transcoder/main.go

# Multi-platform compilation
build-all: clean
	@echo "Compiling for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 cmd/transcoder/main.go
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 cmd/transcoder/main.go
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 cmd/transcoder/main.go
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe cmd/transcoder/main.go
	@echo "Compilation finished. Binaries available in $(BUILD_DIR)/"

# Cleanup
clean:
	@echo "Cleaning generated files..."
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)
	rm -rf testdata/temp/* testdata/downloads/*

# Unit tests
test-unit:
	@echo "Running unit tests..."
	go test -v ./pkg/...

# End-to-end tests
test-e2e:
	@echo "Running end-to-end tests..."
	./scripts/run_e2e_tests.sh

# Example tests
test-examples:
	@echo "Running library examples tests..."
	./scripts/run_examples_tests.sh

# All tests
test: test-unit test-e2e

# Installation
install: build
	@echo "Installing HLSpresso in $(INSTALL_DIR)..."
	cp $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "HLSpresso installed successfully!"

# Uninstallation
uninstall:
	@echo "Uninstalling HLSpresso..."
	rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "HLSpresso uninstalled!"

# Help
help:
	@echo "Available commands:"
	@echo "  make build          - Compile HLSpresso"
	@echo "  make build-all      - Compile for multiple platforms"
	@echo "  make clean          - Remove generated files"
	@echo "  make test-unit      - Run unit tests"
	@echo "  make test-e2e       - Run end-to-end tests"
	@echo "  make test-examples  - Run library examples tests"
	@echo "  make test           - Run all tests"
	@echo "  make install        - Install HLSpresso on the system"
	@echo "  make uninstall      - Remove HLSpresso from the system" 