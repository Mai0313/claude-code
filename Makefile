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
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BIN_NAME) claude_analysis.go

# Build for multiple platforms
.PHONY: build-all
build-all:
	mkdir -p $(BUILD_DIR)
	# Build for current platform first
	go build -o $(BUILD_DIR)/$(BIN_NAME) claude_analysis.go
	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BIN_NAME)-linux-amd64 claude_analysis.go
	# Linux ARM64
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(BIN_NAME)-linux-arm64 claude_analysis.go
	# Windows AMD64
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BIN_NAME)-windows-amd64.exe claude_analysis.go
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BIN_NAME)-darwin-amd64 claude_analysis.go
	# macOS ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BIN_NAME)-darwin-arm64 claude_analysis.go

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
