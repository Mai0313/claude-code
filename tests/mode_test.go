package tests

import (
	"os"
	"testing"

	"claude_analysis/core/config"
)

func TestMode_DefaultSTOP_WithoutEnvAndDotenv(t *testing.T) {
	// Use empty temp dir as CWD (no .env)
	tmpDir := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	cfg := config.Default()
	// Mode field has been removed, this test can be simplified or removed
	if cfg.UserName == "" {
		t.Errorf("expected valid username, got empty string")
	}
}
