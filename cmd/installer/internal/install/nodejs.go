package install

import (
	"errors"
	"fmt"
	"runtime"

	"claude_analysis/cmd/installer/internal/platform"
)

// InstallNodeJS checks for Node.js and installs if necessary
func InstallNodeJS() error {
	fmt.Println("📦 Step 1: Checking Node.js...")
	if !platform.CheckNodeVersion() {
		if platform.IsCommandAvailable("node") {
			fmt.Println("⚡ Node.js found but version is less than 22. Upgrading...")
		} else {
			fmt.Println("📦 Node.js not found. Installing...")
		}

		switch runtime.GOOS {
		case "windows":
			if err := platform.InstallNodeWindows(); err != nil {
				return fmt.Errorf("failed to install Node.js on Windows: %w", err)
			}
		case "darwin":
			if err := installNodeDarwin(); err != nil {
				return fmt.Errorf("failed to install Node.js on macOS: %w", err)
			}
		case "linux":
			if err := installNodeLinux(); err != nil {
				return fmt.Errorf("failed to install Node.js on Linux: %w", err)
			}
		default:
			return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
		}
	}

	fmt.Println("✅ Node.js version >= 22 found. Skipping Node.js installation.")
	return nil
}

func installNodeDarwin() error {
	// Try Homebrew first
	if platform.IsCommandAvailable("brew") {
		// Try node@22 then fallback to node
		if err := platform.RunLoggedCmd("brew", "install", "node@22"); err == nil {
			_ = platform.RunLoggedCmd("brew", "link", "--overwrite", "--force", "node@22")
			return nil
		}
		if err := platform.RunLoggedCmd("brew", "install", "node"); err == nil {
			return nil
		}
	}
	// Fallback: prompt user to install manually
	fmt.Println("❌ Unable to install Node.js automatically on macOS. Please install Node.js LTS from https://nodejs.org/ and re-run this installer.")
	return errors.New("node.js not installed")
}

func installNodeLinux() error {
	// Try common package managers
	if platform.IsCommandAvailable("apt-get") {
		_ = platform.RunLoggedCmd("sudo", "apt-get", "update")
		if err := platform.RunLoggedCmd("sudo", "apt-get", "install", "-y", "nodejs", "npm"); err == nil {
			return nil
		}
		// Try NodeSource for Node 22
		if platform.IsCommandAvailable("curl") {
			if err := platform.RunLoggedShell("curl -fsSL https://deb.nodesource.com/setup_22.x | sudo -E bash -"); err == nil {
				if err := platform.RunLoggedCmd("sudo", "apt-get", "install", "-y", "nodejs"); err == nil {
					return nil
				}
			}
		}
	}
	if platform.IsCommandAvailable("dnf") {
		_ = platform.RunLoggedCmd("sudo", "dnf", "-y", "module", "enable", "nodejs:22")
		if err := platform.RunLoggedCmd("sudo", "dnf", "-y", "install", "nodejs"); err == nil {
			return nil
		}
	}
	if platform.IsCommandAvailable("yum") {
		if err := platform.RunLoggedCmd("sudo", "yum", "-y", "install", "nodejs", "npm"); err == nil {
			return nil
		}
	}
	if platform.IsCommandAvailable("pacman") {
		if err := platform.RunLoggedCmd("sudo", "pacman", "-Sy", "--noconfirm", "nodejs", "npm"); err == nil {
			return nil
		}
	}
	fmt.Println("❌ Unable to install Node.js automatically on Linux. Please install Node.js LTS (v22) from https://nodejs.org/ and re-run this installer.")
	return errors.New("node.js not installed")
}
