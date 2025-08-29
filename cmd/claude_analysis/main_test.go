package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestClaudeAnalysis_PathMode_OutputToStdout(t *testing.T) {
	// Get the path to the built binary
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to get caller info")
	}
	// Navigate from cmd/claude_analysis to project root
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	binaryPath := filepath.Join(projectRoot, "build", "claude_analysis")

	// Check if binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skipf("Binary not found at %s, skipping integration test", binaryPath)
	}

	jsonlPath := filepath.Join(projectRoot, "examples", "test_conversation.jsonl")

	// Test --path parameter with output to stdout
	cmd := exec.Command(binaryPath, "--path", jsonlPath, "--skip-update-check")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Verify the output is valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("Output is not valid JSON: %v\nOutput: %s", err, string(output))
	}

	// Verify basic structure
	if result["user"] == nil || result["records"] == nil {
		t.Errorf("Missing required fields in output: %+v", result)
	}

	// Verify records array exists and is not empty
	records, ok := result["records"].([]interface{})
	if !ok || len(records) == 0 {
		t.Errorf("Expected non-empty records array, got: %+v", result["records"])
	}
}

func TestClaudeAnalysis_PathMode_OutputToFile(t *testing.T) {
	// Get the path to the built binary
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to get caller info")
	}
	// Navigate from cmd/claude_analysis to project root
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	binaryPath := filepath.Join(projectRoot, "build", "claude_analysis")

	// Check if binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skipf("Binary not found at %s, skipping integration test", binaryPath)
	}

	jsonlPath := filepath.Join(projectRoot, "examples", "test_conversation.jsonl")
	outputPath := filepath.Join(t.TempDir(), "test_output.json")

	// Test --path and --output parameters
	cmd := exec.Command(binaryPath, "--path", jsonlPath, "--output", outputPath, "--skip-update-check")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, string(output))
	}

	// Verify output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatalf("Output file was not created: %s", outputPath)
	}

	// Verify file content is valid JSON
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Output file is not valid JSON: %v\nContent: %s", err, string(data))
	}

	// Verify basic structure
	if result["user"] == nil || result["records"] == nil {
		t.Errorf("Missing required fields in output: %+v", result)
	}
}

func TestClaudeAnalysis_PathMode_FileNotFound(t *testing.T) {
	// Get the path to the built binary
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to get caller info")
	}
	// Navigate from cmd/claude_analysis to project root
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	binaryPath := filepath.Join(projectRoot, "build", "claude_analysis")

	// Check if binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skipf("Binary not found at %s, skipping integration test", binaryPath)
	}

	nonExistentPath := "/path/that/does/not/exist.jsonl"

	// Test with non-existent file
	cmd := exec.Command(binaryPath, "--path", nonExistentPath, "--skip-update-check")
	output, err := cmd.CombinedOutput()

	// Should exit with error
	if err == nil {
		t.Fatalf("Expected command to fail, but it succeeded")
	}

	// Check error message contains expected text
	outputStr := string(output)
	if !strings.Contains(outputStr, "does not exist") {
		t.Errorf("Expected error message about file not existing, got: %s", outputStr)
	}

	// Check that it outputs error JSON
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		if strings.Contains(line, `"status": "error"`) {
			var errorResult map[string]interface{}
			if err := json.Unmarshal([]byte(line), &errorResult); err == nil {
				if errorResult["status"] == "error" {
					return // Found expected error JSON
				}
			}
		}
	}
	t.Errorf("Expected error JSON output, got: %s", outputStr)
}

func TestClaudeAnalysis_VersionFlag(t *testing.T) {
	// Get the path to the built binary
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to get caller info")
	}
	// Navigate from cmd/claude_analysis to project root
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	binaryPath := filepath.Join(projectRoot, "build", "claude_analysis")

	// Check if binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skipf("Binary not found at %s, skipping integration test", binaryPath)
	}

	// Test --version flag
	cmd := exec.Command(binaryPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Version command failed: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Claude Analysis Tool") || !strings.Contains(outputStr, "Version:") {
		t.Errorf("Expected version information, got: %s", outputStr)
	}
}

func TestClaudeAnalysis_HelpFlag(t *testing.T) {
	// Get the path to the built binary
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to get caller info")
	}
	// Navigate from cmd/claude_analysis to project root
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	binaryPath := filepath.Join(projectRoot, "build", "claude_analysis")

	// Check if binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skipf("Binary not found at %s, skipping integration test", binaryPath)
	}

	// Test --help flag
	cmd := exec.Command(binaryPath, "--help")
	output, _ := cmd.CombinedOutput()
	// --help typically exits with status 2, so we don't check err

	outputStr := string(output)
	expectedFlags := []string{"-path", "-output", "-version", "-check-update", "-skip-update-check", "-o11y_base_url"}

	for _, flag := range expectedFlags {
		if !strings.Contains(outputStr, flag) {
			t.Errorf("Expected help to contain flag %s, got: %s", flag, outputStr)
		}
	}
}

func TestClaudeAnalysis_CheckUpdateFlag(t *testing.T) {
	// Get the path to the built binary
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to get caller info")
	}
	// Navigate from cmd/claude_analysis to project root
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	binaryPath := filepath.Join(projectRoot, "build", "claude_analysis")

	// Check if binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skipf("Binary not found at %s, skipping integration test", binaryPath)
	}

	// Test --check-update flag
	cmd := exec.Command(binaryPath, "--check-update")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Check update command failed: %v", err)
	}

	// Should output JSON with version information
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("Check update output is not valid JSON: %v\nOutput: %s", err, string(output))
	}

	// Should contain currentVersion field
	if result["currentVersion"] == nil && result["current_version"] == nil {
		t.Errorf("Expected currentVersion or current_version field in update check result: %+v", result)
	}
}

// Test that parseJSONLFile function handles directory creation properly
func TestClaudeAnalysis_OutputDirectoryCreation(t *testing.T) {
	// Get the path to the built binary
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to get caller info")
	}
	// Navigate from cmd/claude_analysis to project root
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	binaryPath := filepath.Join(projectRoot, "build", "claude_analysis")

	// Check if binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skipf("Binary not found at %s, skipping integration test", binaryPath)
	}

	jsonlPath := filepath.Join(projectRoot, "examples", "test_conversation.jsonl")

	// Create output path in non-existent directory
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "nested", "dir", "output.json")

	// Test that the tool creates the directory structure
	cmd := exec.Command(binaryPath, "--path", jsonlPath, "--output", outputPath, "--skip-update-check")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, string(output))
	}

	// Verify output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatalf("Output file was not created: %s", outputPath)
	}

	// Verify directory was created
	outputDir := filepath.Dir(outputPath)
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Fatalf("Output directory was not created: %s", outputDir)
	}
}
