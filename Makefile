# Makefile for post_hook Go application

# Build directory
BUILD_DIR := build
BIN_NAME := claude_analysis
INSTALLER_NAME := installer

# Default target
.PHONY: all
all: package-all

# Build the application
.PHONY: build
build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BIN_NAME) ./cmd/claude_analysis
	go build -o $(BUILD_DIR)/$(INSTALLER_NAME) ./cmd/installer

# Build for multiple platforms
.PHONY: build-all build_linux_amd64 build_linux_arm64 build_windows_amd64 build_darwin_amd64 build_darwin_arm64
build-all: build_linux_amd64 build_linux_arm64 build_windows_amd64 build_darwin_amd64 build_darwin_arm64

build_linux_amd64:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BIN_NAME)-linux-amd64 ./cmd/claude_analysis
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(INSTALLER_NAME)-linux-amd64 ./cmd/installer

build_linux_arm64:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(BIN_NAME)-linux-arm64 ./cmd/claude_analysis
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(INSTALLER_NAME)-linux-arm64 ./cmd/installer

build_windows_amd64:
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BIN_NAME)-windows-amd64.exe ./cmd/claude_analysis
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(INSTALLER_NAME)-windows-amd64.exe ./cmd/installer

build_darwin_amd64:
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BIN_NAME)-darwin-amd64 ./cmd/claude_analysis
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(INSTALLER_NAME)-darwin-amd64 ./cmd/installer

build_darwin_arm64:
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BIN_NAME)-darwin-arm64 ./cmd/claude_analysis
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(INSTALLER_NAME)-darwin-arm64 ./cmd/installer

# Packaging to Claude-Code-Installer-{platform}.zip
.PHONY: package-all package_linux_amd64 package_linux_arm64 package_windows_amd64 package_darwin_amd64 package_darwin_arm64
package-all: build-all package_linux_amd64 package_linux_arm64 package_windows_amd64 package_darwin_amd64 package_darwin_arm64

package_linux_amd64: build_linux_amd64
	@cp $(BUILD_DIR)/$(BIN_NAME)-linux-amd64 $(BUILD_DIR)/claude_analysis
	@cp $(BUILD_DIR)/$(INSTALLER_NAME)-linux-amd64 $(BUILD_DIR)/installer
	@cd $(BUILD_DIR) && cp ../cmd/installer/README*.md . && \
	  zip -q -9 "Claude-Code-Installer-linux-amd64.zip" claude_analysis installer README*.md && rm -f claude_analysis installer README*.md
	@rm -f $(BUILD_DIR)/$(BIN_NAME)-linux-amd64 $(BUILD_DIR)/$(INSTALLER_NAME)-linux-amd64

package_linux_arm64: build_linux_arm64
	@cp $(BUILD_DIR)/$(BIN_NAME)-linux-arm64 $(BUILD_DIR)/claude_analysis
	@cp $(BUILD_DIR)/$(INSTALLER_NAME)-linux-arm64 $(BUILD_DIR)/installer
	@cd $(BUILD_DIR) && cp ../cmd/installer/README*.md . && \
	  zip -q -9 "Claude-Code-Installer-linux-arm64.zip" claude_analysis installer README*.md && rm -f claude_analysis installer README*.md
	@rm -f $(BUILD_DIR)/$(BIN_NAME)-linux-arm64 $(BUILD_DIR)/$(INSTALLER_NAME)-linux-arm64

package_windows_amd64: build_windows_amd64
	@cp $(BUILD_DIR)/$(BIN_NAME)-windows-amd64.exe $(BUILD_DIR)/claude_analysis.exe
	@cp $(BUILD_DIR)/$(INSTALLER_NAME)-windows-amd64.exe $(BUILD_DIR)/installer.exe
	@cd $(BUILD_DIR) && cp ../cmd/installer/README*.md . && \
	  zip -q -9 "Claude-Code-Installer-windows-amd64.zip" claude_analysis.exe installer.exe README*.md && rm -f claude_analysis.exe installer.exe README*.md
	@rm -f $(BUILD_DIR)/$(BIN_NAME)-windows-amd64.exe $(BUILD_DIR)/$(INSTALLER_NAME)-windows-amd64.exe

package_darwin_amd64: build_darwin_amd64
	@cp $(BUILD_DIR)/$(BIN_NAME)-darwin-amd64 $(BUILD_DIR)/claude_analysis
	@cp $(BUILD_DIR)/$(INSTALLER_NAME)-darwin-amd64 $(BUILD_DIR)/installer
	@cd $(BUILD_DIR) && cp ../cmd/installer/README*.md . && \
	  zip -q -9 "Claude-Code-Installer-darwin-amd64.zip" claude_analysis installer README*.md && rm -f claude_analysis installer README*.md
	@rm -f $(BUILD_DIR)/$(BIN_NAME)-darwin-amd64 $(BUILD_DIR)/$(INSTALLER_NAME)-darwin-amd64

package_darwin_arm64: build_darwin_arm64
	@cp $(BUILD_DIR)/$(BIN_NAME)-darwin-arm64 $(BUILD_DIR)/claude_analysis
	@cp $(BUILD_DIR)/$(INSTALLER_NAME)-darwin-arm64 $(BUILD_DIR)/installer
	@cd $(BUILD_DIR) && cp ../cmd/installer/README*.md . && \
	  zip -q -9 "Claude-Code-Installer-darwin-arm64.zip" claude_analysis installer README*.md && rm -f claude_analysis installer README*.md
	@rm -f $(BUILD_DIR)/$(BIN_NAME)-darwin-arm64 $(BUILD_DIR)/$(INSTALLER_NAME)-darwin-arm64

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
	go test -cover -v ./...

.PHONY: test-verbose
test-verbose:
	go test -cover -v ./tests -run TestParser_FromTestConversationJSONL_PrintsFullPayload -count=1

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
