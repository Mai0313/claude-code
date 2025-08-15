package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Settings struct {
	Env                        map[string]string `json:"env"`
	IncludeCoAuthoredBy        bool              `json:"includeCoAuthoredBy"`
	EnableAllProjectMcpServers bool              `json:"enableAllProjectMcpServers"`
	Hooks                      map[string][]Hook `json:"hooks"`
}

type Hook struct {
	Matcher string       `json:"matcher,omitempty"`
	Hooks   []HookAction `json:"hooks,omitempty"`
	// For leaf action
	Type    string `json:"type,omitempty"`
	Command string `json:"command,omitempty"`
}

type HookAction = Hook

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "installer error:", err)
		os.Exit(1)
	}
}

func run() error {
	// 1) Node.js check/install guidance
	if !isCommandAvailable("node") {
		switch runtime.GOOS {
		case "windows":
			// Per requirement: prompt user to download MSI and exit.
			fmt.Println("Node.js not found. Please download and install Node.js LTS from:")
			fmt.Println("  https://nodejs.org/dist/v22.18.0/node-v22.18.0-arm64.msi")
			fmt.Println("After installation, re-run this installer.")
			return nil
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

	// 2) Install @anthropic-ai/claude-code with registry fallbacks
	registryUsed, err := installClaudeCLI()
	if err != nil {
		return err
	}

	// 3) Move claude_analysis to ~/.claude with platform-specific name
	destPath, err := installClaudeAnalysisBinary()
	if err != nil {
		return err
	}

	// 4) Generate settings.json to ~/.claude/settings.json
	if err := writeSettingsJSON(destPath, registryUsed); err != nil {
		return err
	}

	fmt.Println("Installation completed successfully.")
	return nil
}

func isCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func installNodeDarwin() error {
	// Try Homebrew first
	if isCommandAvailable("brew") {
		// Try node@22 then fallback to node
		if err := runCmdLogged("brew", "install", "node@22"); err == nil {
			_ = runCmdLogged("brew", "link", "--overwrite", "--force", "node@22")
			return nil
		}
		if err := runCmdLogged("brew", "install", "node"); err == nil {
			return nil
		}
	}
	// Fallback: prompt user to install manually
	fmt.Println("Unable to install Node.js automatically on macOS. Please install Node.js LTS from https://nodejs.org/ and re-run this installer.")
	return errors.New("node.js not installed")
}

func installNodeLinux() error {
	// Try common package managers
	if isCommandAvailable("apt-get") {
		_ = runCmdLogged("sudo", "apt-get", "update")
		if err := runCmdLogged("sudo", "apt-get", "install", "-y", "nodejs", "npm"); err == nil {
			return nil
		}
		// Try NodeSource for Node 22
		if isCommandAvailable("curl") {
			if err := runShellLogged("curl -fsSL https://deb.nodesource.com/setup_22.x | sudo -E bash -"); err == nil {
				if err := runCmdLogged("sudo", "apt-get", "install", "-y", "nodejs"); err == nil {
					return nil
				}
			}
		}
	}
	if isCommandAvailable("dnf") {
		_ = runCmdLogged("sudo", "dnf", "-y", "module", "enable", "nodejs:22")
		if err := runCmdLogged("sudo", "dnf", "-y", "install", "nodejs"); err == nil {
			return nil
		}
	}
	if isCommandAvailable("yum") {
		if err := runCmdLogged("sudo", "yum", "-y", "install", "nodejs", "npm"); err == nil {
			return nil
		}
	}
	if isCommandAvailable("pacman") {
		if err := runCmdLogged("sudo", "pacman", "-Sy", "--noconfirm", "nodejs", "npm"); err == nil {
			return nil
		}
	}
	fmt.Println("Unable to install Node.js automatically on Linux. Please install Node.js LTS (v22) from https://nodejs.org/ and re-run this installer.")
	return errors.New("node.js not installed")
}

func npmPath() string {
	// Prefer npm next to node if node is found
	if p, err := exec.LookPath("npm"); err == nil {
		return p
	}
	return "npm" // rely on PATH
}

func installClaudeCLI() (string, error) {
	registries := []string{"", "http://oa-mirror.mediatek.inc/repository/npm", "http://swrd-mirror.mediatek.inc/repository/npm"}
	var lastErr error
	for i, reg := range registries {
		args := []string{"install", "-g", "@anthropic-ai/claude-code"}
		if reg != "" {
			args = append(args, "--registry="+reg)
		}
		fmt.Printf("Installing @anthropic-ai/claude-code (attempt %d/%d)%s...\n", i+1, len(registries), func() string {
			if reg != "" {
				return " via registry=" + reg
			}
			return ""
		}())
		if err := runCmdLogged(npmPath(), args...); err != nil {
			lastErr = err
			continue
		}
		// Verify installation
		if err := verifyClaudeInstalled(); err != nil {
			lastErr = err
			continue
		}
		return reg, nil
	}
	return "", fmt.Errorf("failed to install @anthropic-ai/claude-code: %w", lastErr)
}

func verifyClaudeInstalled() error {
	// Try running "claude --version"; if not in PATH, attempt using npm bin -g
	if err := runCmdLogged("claude", "--version"); err == nil {
		return nil
	}
	// Try from npm bin -g
	out, err := exec.Command(npmPath(), "bin", "-g").Output()
	if err != nil {
		return fmt.Errorf("npm bin -g failed: %w", err)
	}
	binDir := strings.TrimSpace(string(out))
	claudePath := filepath.Join(binDir, exeName("claude"))
	if _, err := os.Stat(claudePath); err == nil {
		cmd := exec.Command(claudePath, "--version")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	return errors.New("claude CLI not found after installation")
}

func installClaudeAnalysisBinary() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to get home dir: %w", err)
	}
	targetDir := filepath.Join(homeDir, ".claude")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create %s: %w", targetDir, err)
	}

	// Determine source binary path: same directory as this installer
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("os.Executable failed: %w", err)
	}
	srcDir := filepath.Dir(exe)
	srcName := exeName("claude_analysis")
	srcPath := filepath.Join(srcDir, srcName)
	if _, err := os.Stat(srcPath); err != nil {
		return "", fmt.Errorf("expected %s next to installer: %w", srcName, err)
	}

	// Destination filename includes platform suffix
	platform := platformSuffix()
	destName := "claude_analysis-" + platform
	if runtime.GOOS == "windows" {
		destName += ".exe"
	}
	destPath := filepath.Join(targetDir, destName)

	// Prefer move (rename) to match "move" requirement; fallback to copy when cross-filesystem
	if err := os.Rename(srcPath, destPath); err != nil {
		if copyErr := copyFile(srcPath, destPath, 0o755); copyErr != nil {
			return "", fmt.Errorf("failed to install claude_analysis to %s: %w", destPath, copyErr)
		}
		// remove original if copy succeeded
		_ = os.Remove(srcPath)
	}
	fmt.Println("Installed claude_analysis to:", destPath)
	return destPath, nil
}

func platformSuffix() string {
	arch := runtime.GOARCH
	osname := runtime.GOOS
	switch osname {
	case "darwin", "linux", "windows":
		// ok
	default:
		// Fallback to generic
		return osname + "-" + arch
	}
	return osname + "-" + arch
}

func exeName(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

func writeSettingsJSON(installedBinaryPath string, registryUsed string) error {
	// Determine base URL by mirror rule first; if no mirror was used, fallback to connectivity
	var chosen string
	switch registryUsed {
	case "http://oa-mirror.mediatek.inc/repository/npm":
		chosen = "https://mlop-azure-gateway.mediatek.inc"
	case "http://swrd-mirror.mediatek.inc/repository/npm":
		chosen = "https://mlop-azure-rddmz.mediatek.inc"
	default:
		// Connectivity-based selection
		candidates := []string{
			"https://mlop-azure-gateway.mediatek.inc",
			"https://mlop-azure-rddmz.mediatek.inc",
		}
		chosen = candidates[0]
		for _, u := range candidates {
			if checkReachable(u, 3*time.Second) == nil {
				chosen = u
				break
			}
		}
	}

	// Hook command with tilde path and platform suffix
	hookPath := fmt.Sprintf("~/.claude/claude_analysis-%s", platformSuffix())
	if runtime.GOOS == "windows" {
		hookPath += ".exe"
	}

	settings := Settings{
		Env: map[string]string{
			"DISABLE_TELEMETRY":                        "1",
			"CLAUDE_CODE_USE_BEDROCK":                  "1",
			"ANTHROPIC_BEDROCK_BASE_URL":               chosen,
			"CLAUDE_CODE_ENABLE_TELEMETRY":             "1",
			"CLAUDE_CODE_SKIP_BEDROCK_AUTH":            "1",
			"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
		},
		IncludeCoAuthoredBy:        true,
		EnableAllProjectMcpServers: true,
		Hooks: map[string][]Hook{
			"Stop": {
				{
					Matcher: "*",
					Hooks: []Hook{
						{Type: "command", Command: hookPath},
					},
				},
			},
		},
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	homeDir, _ := os.UserHomeDir()
	targetDir := filepath.Join(homeDir, ".claude")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("failed to create %s: %w", targetDir, err)
	}
	target := filepath.Join(targetDir, "settings.json")
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", target, err)
	}
	fmt.Println("Wrote settings:", target)
	return nil
}

func checkReachable(rawURL string, timeout time.Duration) error {
	// Quick TCP connect check to 443 or 80 depending on scheme
	host := rawURL
	if strings.HasPrefix(rawURL, "https://") {
		host = strings.TrimPrefix(rawURL, "https://")
		host = strings.TrimSuffix(host, "/")
		host = net.JoinHostPort(host, "443")
	} else if strings.HasPrefix(rawURL, "http://") {
		host = strings.TrimPrefix(rawURL, "http://")
		host = strings.TrimSuffix(host, "/")
		host = net.JoinHostPort(host, "80")
	}
	d := net.Dialer{Timeout: timeout}
	conn, err := d.Dial("tcp", host)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}

func runCmdLogged(name string, args ...string) error {
	fmt.Printf("$ %s %s\n", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runShellLogged(script string) error {
	fmt.Printf("$ sh -lc %q\n", script)
	cmd := exec.Command("sh", "-lc", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

// Unused but handy: create zip buffer for potential future embedding
func zipBytes(name string, content []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, err := zw.Create(name)
	if err != nil {
		return nil, err
	}
	if _, err := f.Write(content); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
