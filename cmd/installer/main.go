package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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
	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "installer error:", err)
	} else {
		fmt.Println("Installation completed successfully.")
	}
	pauseIfInteractive()
	if err != nil {
		os.Exit(1)
	}
}

func run() error {
	// 1) Node.js check/install guidance
	if !checkNodeVersion() {
		if isCommandAvailable("node") {
			fmt.Println("Node.js found but version is less than 22. Upgrading...")
		} else {
			fmt.Println("Node.js not found. Installing...")
		}

		switch runtime.GOOS {
		case "windows":
			// Per requirement: prompt user to download MSI and exit.
			fmt.Println("Node.js not found or version < 22. Please download and install Node.js LTS from:")
			fmt.Println("  https://nodejs.org/dist/v22.18.0/node-v22.18.0-arm64.msi")
			fmt.Println("After installation, re-run this installer.")
			return errors.New("node.js not installed; user action required")
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
	} else {
		fmt.Println("Node.js version >= 22 found. Skipping Node.js installation.")
	}

	// 2) Install @anthropic-ai/claude-code with registry fallbacks
	var registryUsed string
	if checkClaudeInstalled() {
		fmt.Println("Claude CLI already installed. Skipping installation.")
		registryUsed = "" // No registry was used since we skipped installation
	} else {
		var err error
		registryUsed, err = installClaudeCLI()
		if err != nil {
			return err
		}
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

	return nil
}

// pauseIfInteractive waits for Enter when stdin is a TTY so users can read output before the window closes.
func pauseIfInteractive() {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return
	}
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		// Not a terminal (piped or redirected); don't block
		return
	}
	fmt.Print("\nPress Enter to exit...")
	r := bufio.NewReader(os.Stdin)
	_, _ = r.ReadString('\n')
}

func isCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// checkNodeVersion returns true if Node.js is installed and version >= 22
func checkNodeVersion() bool {
	if !isCommandAvailable("node") {
		return false
	}

	out, err := exec.Command("node", "--version").Output()
	if err != nil {
		return false
	}

	version := strings.TrimSpace(string(out))
	// Remove 'v' prefix if present (e.g., "v22.1.0" -> "22.1.0")
	version = strings.TrimPrefix(version, "v")

	// Extract major version
	parts := strings.Split(version, ".")
	if len(parts) == 0 {
		return false
	}

	// Parse major version
	var major int
	if _, err := fmt.Sscanf(parts[0], "%d", &major); err != nil {
		return false
	}

	return major >= 22
}

// checkClaudeInstalled returns true if Claude CLI is already installed and working
func checkClaudeInstalled() bool {
	// Try running "claude --version"; if successful, it's installed
	if err := exec.Command("claude", "--version").Run(); err == nil {
		return true
	}

	// Try from npm bin -g location
	out, err := exec.Command(npmPath(), "bin", "-g").Output()
	if err != nil {
		return false
	}
	binDir := strings.TrimSpace(string(out))
	claudePath := filepath.Join(binDir, exeName("claude"))
	if _, err := os.Stat(claudePath); err == nil {
		if err := exec.Command(claudePath, "--version").Run(); err == nil {
			return true
		}
	}
	return false
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

// checkConnectivity tests connectivity to a base URL with both HTTPS and HTTP schemes
// Returns the working URL (with scheme) or empty string if none work
func checkConnectivity(baseURL string, timeout time.Duration) string {
	// Extract hostname from baseURL (remove any existing scheme)
	hostname := strings.TrimPrefix(baseURL, "https://")
	hostname = strings.TrimPrefix(hostname, "http://")
	hostname = strings.TrimSuffix(hostname, "/")

	// First try HTTPS
	httpsURL := "https://" + hostname
	if checkURLReachability(httpsURL, timeout) == nil {
		return httpsURL
	}

	// Then try HTTP
	httpURL := "http://" + hostname
	if checkURLReachability(httpURL, timeout) == nil {
		return httpURL
	}

	return ""
}

// checkURLReachability performs an HTTP HEAD request to test if URL is reachable
func checkURLReachability(url string, timeout time.Duration) error {
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}

	resp, err := client.Head(url)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return nil
	}

	return fmt.Errorf("HTTP status: %d", resp.StatusCode)
}

// selectRegistryURL checks connectivity and returns the best npm registry to use
func selectRegistryURL() string {
	registryHosts := []string{
		"oa-mirror.mediatek.inc/repository/npm",
		"swrd-mirror.mediatek.inc/repository/npm",
	}

	for _, host := range registryHosts {
		if workingURL := checkConnectivity(host, 3*time.Second); workingURL != "" {
			fmt.Printf("Found working registry: %s\n", workingURL)
			return workingURL
		}
	}

	return "" // Use default registry
}

// selectGaisfURL checks connectivity and returns the best MLOP URL to use
func selectGaisfURL() string {
	mlopHosts := []string{
		"mlop-azure-gateway.mediatek.inc",
		"mlop-azure-rddmz.mediatek.inc",
	}

	for _, host := range mlopHosts {
		if workingURL := checkConnectivity(host, 3*time.Second); workingURL != "" {
			fmt.Printf("Found working MLOP endpoint: %s\n", workingURL)
			return workingURL
		}
	}

	// Fallback: just use first option with HTTPS if no connectivity check worked
	return "https://mlop-azure-gateway.mediatek.inc"
}

func installClaudeCLI() (string, error) {
	// Use the best working registry found by selectRegistryURL
	registry := selectRegistryURL()

	args := []string{"install", "-g", "@anthropic-ai/claude-code"}
	if registry != "" {
		args = append(args, "--registry="+registry)
		fmt.Printf("Installing @anthropic-ai/claude-code via registry=%s...\n", registry)
	} else {
		fmt.Println("Installing @anthropic-ai/claude-code via default registry...")
	}

	if err := runCmdLogged(npmPath(), args...); err != nil {
		return "", fmt.Errorf("npm install failed: %w", err)
	}

	// Verify installation
	if err := verifyClaudeInstalled(); err != nil {
		return "", fmt.Errorf("installation verification failed: %w", err)
	}

	return registry, nil
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

	// Copy the binary to destination and keep the original
	if err := copyFile(srcPath, destPath, 0o755); err != nil {
		return "", fmt.Errorf("failed to install claude_analysis to %s: %w", destPath, err)
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
	// Always use connectivity-based selection for MLOP URL
	chosen := selectGaisfURL()

	// Decide target paths: prefer managed system path if writable; else fallback to user-level
	managedSettingsPath, managedBinDir := managedPaths()
	useManaged := false
	if managedSettingsPath != "" {
		if err := os.MkdirAll(filepath.Dir(managedSettingsPath), 0o755); err == nil {
			// try a small write test by writing settings later; keep a flag
			useManaged = true
		}
	}

	// Compute desired hook path and optionally copy binary into managed directory
	hookPath := fmt.Sprintf("~/.claude/claude_analysis-%s", platformSuffix())
	if runtime.GOOS == "windows" {
		hookPath += ".exe"
	}
	if useManaged && managedBinDir != "" {
		// Destination binary name mirrors platform suffix
		destName := fmt.Sprintf("claude_analysis-%s", platformSuffix())
		if runtime.GOOS == "windows" {
			destName += ".exe"
		}
		systemBin := filepath.Join(managedBinDir, destName)
		if err := os.MkdirAll(managedBinDir, 0o755); err == nil {
			if err := copyFile(installedBinaryPath, systemBin, 0o755); err == nil {
				// Use system path for hook; quote on macOS to survive space in "Application Support"
				if runtime.GOOS == "darwin" {
					hookPath = fmt.Sprintf("'%s'", systemBin)
				} else {
					hookPath = systemBin
				}
			}
		}
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

	// Try writing managed-settings.json first if allowed
	if useManaged && managedSettingsPath != "" {
		if err := os.WriteFile(managedSettingsPath, data, 0o644); err == nil {
			fmt.Println("Wrote managed settings:", managedSettingsPath)
			return nil
		}
		// If writing failed, fall through to user-level
	}

	// Fallback to user-level settings.json
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

// managedPaths returns (settingsFilePath, binDir) for system-managed configuration by OS.
// Returns empty strings when OS is unsupported.
func managedPaths() (string, string) {
	switch runtime.GOOS {
	case "darwin":
		dir := filepath.Join("/Library", "Application Support", "ClaudeCode")
		return filepath.Join(dir, "managed-settings.json"), dir
	case "linux":
		dir := filepath.Join("/etc", "claude-code")
		return filepath.Join(dir, "managed-settings.json"), dir
	case "windows":
		// Use ProgramData for system-wide state
		// Note: filepath.Join on Windows will use backslashes when built on Windows; we construct literal path here
		dir := `C:\\ProgramData\\ClaudeCode`
		return dir + `\\managed-settings.json`, dir
	default:
		return "", ""
	}
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

// (no-op: removed unused zipBytes helper)
