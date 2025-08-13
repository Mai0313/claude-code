#!/bin/bash

# Build script for claude_analysis
# This script provides additional build functionality beyond the Makefile

set -e

BUILD_DIR="build"
BIN_NAME="claude_analysis"

echo "Building claude_analysis..."

# Create build directory
mkdir -p "$BUILD_DIR"

# Get build information
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build with version information
go build \
    -ldflags "-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME -X main.GitCommit=$GIT_COMMIT" \
    -o "$BUILD_DIR/$BIN_NAME" \
    ./cmd/claude_analysis

echo "Build completed successfully!"
echo "Binary: $BUILD_DIR/$BIN_NAME"
echo "Version: $VERSION"
echo "Build time: $BUILD_TIME"
echo "Git commit: $GIT_COMMIT"
