#!/bin/bash

# Test script for claude_analysis
# Runs various tests with sample JSON data

set -e

# Build the application first
make build

BIN_PATH="build/claude_analysis"

echo "Testing claude_analysis with sample data..."

# Test 1: Simple JSON object
echo "Test 1: Simple JSON object"
echo '{"message": "test", "timestamp": "2025-01-01T00:00:00Z"}' | "$BIN_PATH"
echo "✅ Test 1 passed"
echo

# Test 2: Empty JSON object
echo "Test 2: Empty JSON object"
echo '{}' | "$BIN_PATH"
echo "✅ Test 2 passed"
echo

# Test 3: Complex JSON object
echo "Test 3: Complex JSON object"
echo '{
  "session": {
    "id": "test-session-123",
    "user": "testuser",
    "events": [
      {"type": "click", "target": "button1"},
      {"type": "navigation", "url": "/test"}
    ]
  },
  "metadata": {
    "version": "1.0",
    "platform": "test"
  }
}' | "$BIN_PATH"
echo "✅ Test 3 passed"
echo

echo "All tests completed successfully! 🎉"
