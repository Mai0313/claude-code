package install

import (
	"errors"
	"fmt"
	"runtime"

	"claude_analysis/cmd/installer/internal/env"
	"claude_analysis/cmd/installer/internal/platform"
)

// GetNpmPath returns the npm executable path with platform-specific fallback
func GetNpmPath() string {
	if runtime.GOOS == "windows" {
		return platform.GetWindowsNpmPath()
	}
	return platform.GetNpmPath()
}

// InstallOrUpdateClaude installs/updates Claude CLI
func InstallOrUpdateClaude() error {
	fmt.Println("🤖 Installing/Updating Claude Code CLI...")

	if err := installClaudeCLI(); err != nil {
		return fmt.Errorf("failed to install/update Claude CLI: %w", err)
	}

	fmt.Println("✅ Claude Code CLI installation/update completed!")
	return InstallClaudeAnalysisBinary()
}

// installClaudeCLI installs the @anthropic-ai/claude-code package using npm.
// It first tries the default npm registry, and if that fails, it looks for a fallback registry from the available environments.
// It verifies the installation by checking if the `claude --version` command works.
func installClaudeCLI() error {
	baseArgs := []string{"install", "-g", "@anthropic-ai/claude-code@latest", "--no-color"}

	// --- 步驟 1: 嘗試使用預設 registry 安裝 ---
	fmt.Println("📦 Attempting to install @anthropic-ai/claude-code with default registry...")
	err := platform.RunLoggedCmd(GetNpmPath(), baseArgs...)

	// 如果第一次嘗試就成功，直接進行驗證並返回
	if err == nil {
		fmt.Println("✅ Installation with default registry succeeded.")
		// 驗證安裝
		if verifyErr := verifyClaudeInstalled(); verifyErr != nil {
			return fmt.Errorf("installation verification failed: %w", verifyErr)
		}
		fmt.Println("✅ Claude CLI installed successfully!")
		return nil
	}

	// --- 步驟 2: 如果第一次失敗，則尋找備用 registry 重試 ---
	fmt.Printf("⚠️ Default registry failed: %v. Looking for a fallback...\n", err)

	chosen := env.SelectAvailableURL()
	if chosen.RegistryURL == "" {
		// 如果沒有找到備用 registry，返回第一次嘗試的錯誤
		return fmt.Errorf("npm install failed with default registry and no fallback registry is available: %w", err)
	}

	// 建立帶有 registry 的新參數
	retryArgs := append(baseArgs, "--registry="+chosen.RegistryURL)
	fmt.Printf("📦 Retrying installation with registry: %s\n", chosen.RegistryURL)

	// 執行重試
	if retryErr := platform.RunLoggedCmd(GetNpmPath(), retryArgs...); retryErr != nil {
		// 如果重試也失敗，返回重試時的錯誤
		return fmt.Errorf("npm install also failed on retry with registry %s: %w", chosen.RegistryURL, retryErr)
	}

	// --- 成功後的驗證 ---
	// 如果重試成功，進行驗證
	if verifyErr := verifyClaudeInstalled(); verifyErr != nil {
		return fmt.Errorf("installation verification failed after retry: %w", verifyErr)
	}

	fmt.Println("✅ Claude CLI installed successfully!")
	return nil
}

// verifyClaudeInstalled checks if the claude CLI is installed by running `claude --version`.
func verifyClaudeInstalled() error {
	if path, ok := platform.FindClaudeBinary(); ok {
		return platform.RunLoggedCmd(path, "--version")
	}
	return errors.New("claude CLI not found after installation")
}

// RunFullInstall performs the complete installation process
func RunFullInstall() error {
	fmt.Println("🚀 Starting full Claude Code installation...")

	// 1) Node.js check/install guidance
	if err := InstallNodeJS(); err != nil {
		return err
	}

	// 2) Install @anthropic-ai/claude-code with registry fallbacks
	// and move claude_analysis to ~/.claude with platform-specific name
	if err := InstallOrUpdateClaude(); err != nil {
		return err
	}

	fmt.Println("🎉 Installation completed successfully!")
	fmt.Println("🔧 Automatically switching to GAISF API Key configuration...")
	return nil
}
