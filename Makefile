# Makefile for post_hook Go application

# Build directory
BUILD_DIR := build
BIN_NAME := claude_analysis

# Default target
.PHONY: all
all: build

# Build the application
.PHONY: build
build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BIN_NAME) ./cmd/claude_analysis

# Build for multiple platforms
.PHONY: build-all build_linux_amd64 build_linux_arm64 build_windows_amd64 build_darwin_amd64 build_darwin_arm64
build-all: build_linux_amd64 build_linux_arm64 build_windows_amd64 build_darwin_amd64 build_darwin_arm64

build_linux_amd64:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BIN_NAME)-linux-amd64 ./cmd/claude_analysis

build_linux_arm64:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(BIN_NAME)-linux-arm64 ./cmd/claude_analysis

build_windows_amd64:
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BIN_NAME)-windows-amd64.exe ./cmd/claude_analysis

build_darwin_amd64:
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BIN_NAME)-darwin-amd64 ./cmd/claude_analysis

build_darwin_arm64:
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BIN_NAME)-darwin-arm64 ./cmd/claude_analysis

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)

# Run the application (for testing)
.PHONY: run
run: build
	./$(BUILD_DIR)/$(BIN_NAME)

# Install to system (optional)
.PHONY: install
install: build
	sudo cp $(BUILD_DIR)/$(BIN_NAME) /usr/local/bin/$(BIN_NAME)

# Format code
.PHONY: fmt
fmt:
	go fmt ./...

# Test (if you add tests later)
.PHONY: test
test:
	go test -v ./...

.PHONY: test-verbose
test-verbose:
	go test -v ./tests -run TestParser_FromTestConversationJSONL_PrintsFullPayload -count=1

# Show help
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  build     - Build the application for current platform"
	@echo "  build-all - Build for multiple platforms (Linux, Windows, macOS)"
	@echo "  clean     - Remove build artifacts"
	@echo "  run       - Build and run the application"
	@echo "  install   - Install binary to /usr/local/bin"
	@echo "  fmt       - Format Go code"
	@echo "  test      - Run tests"
	@echo "  help      - Show this help message"
