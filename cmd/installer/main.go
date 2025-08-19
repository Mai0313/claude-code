package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/net/html"
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

var logger *zap.Logger

// EnvironmentConfig represents a domain environment and its associated endpoints
type EnvironmentConfig struct {
	// Domain is the value to send to the login endpoint (e.g., "oa", "swrd")
	Domain string
	// MLOPHosts are the candidate base hosts for GAISF/MLOP gateway
	MLOPHosts []string
	// RegistryHosts are the candidate npm registry mirrors for this domain
	RegistryHosts []string
}

// environmentConfigs defines the available domain environments and their mappings.
// Add new mappings here to support additional domains.
var environmentConfigs = []EnvironmentConfig{
	{
		Domain:        "oa",
		MLOPHosts:     []string{"mlop-azure-gateway.mediatek.inc"},
		RegistryHosts: []string{"oa-mirror.mediatek.inc/repository/npm"},
	},
	{
		Domain:        "swrd",
		MLOPHosts:     []string{"mlop-azure-rddmz.mediatek.inc"},
		RegistryHosts: []string{"swrd-mirror.mediatek.inc/repository/npm"},
	},
}

// Environment is the resolved and connectivity-validated selection
type Environment struct {
	Config      EnvironmentConfig
	MLOPBaseURL string // with scheme
	RegistryURL string // with scheme (or empty to use default)
}

var selectedEnv *Environment

// initLogger initializes the zap logger with console output
func initLogger() {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.CallerKey = zapcore.OmitKey     // Remove caller info for cleaner output
	config.EncoderConfig.StacktraceKey = zapcore.OmitKey // Remove stacktrace for cleaner output
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05")

	var err error
	logger, err = config.Build()
	if err != nil {
		// Fallback to a basic logger if config fails
		logger = zap.NewExample()
	}
}

func main() {
	initLogger()
	defer logger.Sync()

	err := run()
	if err != nil {
		logger.Error("Installation failed", zap.Error(err))
	} else {
		logger.Info("Installation completed successfully.")
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
			logger.Info("Node.js found but version is less than 22. Upgrading...")
		} else {
			logger.Info("Node.js not found. Installing...")
		}

		switch runtime.GOOS {
		case "windows":
			// Per requirement: prompt user to download MSI and exit.
			logger.Info("Node.js not found or version < 22. Please download and install Node.js LTS from:",
				zap.String("url", "https://nodejs.org/dist/v22.18.0/node-v22.18.0-arm64.msi"))
			logger.Info("After installation, re-run this installer.")
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
		logger.Info("Node.js version >= 22 found. Skipping Node.js installation.")
	}

	// 2) Install @anthropic-ai/claude-code with registry fallbacks
	logger.Info("Installing/Updating Claude CLI (@anthropic-ai/claude-code@latest)...")
	if err := installClaudeCLI(); err != nil {
		return err
	}

	// 3) Move claude_analysis to ~/.claude with platform-specific name
	destPath, err := installClaudeAnalysisBinary()
	if err != nil {
		return err
	}

	// 4) Generate settings.json to ~/.claude/settings.json
	if err := writeSettingsJSON(destPath); err != nil {
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

// getGAISFToken performs login to get GAISF token from the MLOP gateway
// Returns the GAISF token string or error if login fails
func getGAISFToken(username, password string) (string, error) {
	// Use connectivity-selected base URL and domain via unified environment selection
	baseURL := selectGaisfURL()
	env := ensureEnvironmentSelected()
	loginURL := strings.TrimRight(baseURL, "/") + "/auth/login"

	// Cookie-aware HTTP client with redirect support (default)
	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", fmt.Errorf("failed to create cookie jar: %w", err)
	}
	client := &http.Client{Jar: jar, Timeout: 30 * time.Second}

	// Step 1: GET login page and parse CSRF from input[name="_csrf"]
	resp, err := client.Get(loginURL)
	if err != nil {
		return "", fmt.Errorf("failed to get login page: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return "", fmt.Errorf("login page request failed, status: %d", resp.StatusCode)
	}

	csrfToken, err := extractCSRFToken(resp.Body)
	if err != nil || csrfToken == "" {
		return "", fmt.Errorf("unable to find CSRF token on login page: %w", err)
	}

	// Step 2: POST credentials to /auth/login
	form := url.Values{
		"_csrf":            {csrfToken},
		"username":         {username},
		"password":         {password},
		"expiration_hours": {"720"}, // 30 * 24
		"domain":           {env.Config.Domain},
	}

	req, err := http.NewRequest(http.MethodPost, loginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", loginURL)

	resp2, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("login request failed: %w", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode < 200 || resp2.StatusCode >= 400 {
		// Read a small portion for context
		body, _ := io.ReadAll(io.LimitReader(resp2.Body, 1024))
		return "", fmt.Errorf("login failed, status %d: %s", resp2.StatusCode, string(body))
	}

	// Step 3: Parse token from first <textarea>
	token, err := extractFirstTextarea(resp2.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse token from response: %w", err)
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return "", errors.New("could not find token in login response")
	}
	return token, nil
}

// extractCSRFToken parses the HTML document and returns the value of input[name="_csrf"].
func extractCSRFToken(r io.Reader) (string, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return "", err
	}
	var csrf string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && strings.EqualFold(n.Data, "input") {
			var nameVal, val string
			for _, a := range n.Attr {
				if strings.EqualFold(a.Key, "name") {
					nameVal = a.Val
				}
				if strings.EqualFold(a.Key, "value") {
					val = a.Val
				}
			}
			if nameVal == "_csrf" {
				csrf = val
				return
			}
		}
		for c := n.FirstChild; c != nil && csrf == ""; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	if csrf == "" {
		return "", errors.New("_csrf not found")
	}
	return csrf, nil
}

// extractFirstTextarea returns the text content of the first <textarea> element.
func extractFirstTextarea(r io.Reader) (string, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return "", err
	}
	var result string
	var found bool
	textContent := func(n *html.Node) string {
		var b strings.Builder
		var walk func(*html.Node)
		walk = func(nn *html.Node) {
			if nn.Type == html.TextNode {
				b.WriteString(nn.Data)
			}
			for c := nn.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
		}
		walk(n)
		return b.String()
	}

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && strings.EqualFold(n.Data, "textarea") && !found {
			result = textContent(n)
			found = true
			return
		}
		for c := n.FirstChild; c != nil && !found; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	if !found {
		return "", errors.New("no textarea found")
	}
	return result, nil
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
	logger.Info("Unable to install Node.js automatically on macOS. Please install Node.js LTS from https://nodejs.org/ and re-run this installer.")
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
	logger.Info("Unable to install Node.js automatically on Linux. Please install Node.js LTS (v22) from https://nodejs.org/ and re-run this installer.")
	return errors.New("node.js not installed")
}

func npmPath() string {
	// Prefer npm next to node if node is found
	if p, err := exec.LookPath("npm"); err == nil {
		return p
	}
	return "npm" // rely on PATH
}

// checkConnectivity tests connectivity to a base URL using HTTPS only with a lightweight GET.
// Returns the working URL (with scheme) or empty string if not reachable within timeout.
func checkConnectivity(baseURL string, timeout time.Duration) string {
	// Extract hostname from baseURL (remove any existing scheme)
	hostname := strings.TrimPrefix(baseURL, "https://")
	hostname = strings.TrimPrefix(hostname, "http://")
	hostname = strings.TrimSuffix(hostname, "/")

	httpsURL := "https://" + hostname
	if checkURLReachability(httpsURL, timeout) == nil {
		return httpsURL
	}
	return ""
}

// checkURLReachability performs an HTTP HEAD request to test if URL is reachable
func checkURLReachability(url string, timeout time.Duration) error {
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: false},
		},
	}

	// Single lightweight GET request. Many services don't support HEAD reliably.
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("User-Agent", "claude-installer/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	io.Copy(io.Discard, io.LimitReader(resp.Body, 512))
	resp.Body.Close()
	// Treat any HTTP response as reachable (even 4xx/5xx indicate server reachable and TLS/DNS ok).
	return nil
}

// selectRegistryURL checks connectivity and returns the best npm registry to use
func selectRegistryURL() string {
	env := ensureEnvironmentSelected()
	if env.RegistryURL != "" {
		logger.Info("Using registry", zap.String("url", env.RegistryURL))
	}
	return env.RegistryURL // empty means use default registry
}

// selectGaisfURL checks connectivity and returns the best MLOP URL to use
func selectGaisfURL() string {
	env := ensureEnvironmentSelected()
	logger.Info("Using MLOP gateway", zap.String("url", env.MLOPBaseURL), zap.String("domain", env.Config.Domain))
	return env.MLOPBaseURL
}

// ensureEnvironmentSelected resolves and caches the environment selection by testing connectivity
func ensureEnvironmentSelected() *Environment {
	if selectedEnv != nil {
		return selectedEnv
	}

	// Try each configured environment in order; pick the first with a reachable MLOP host
	for _, cfg := range environmentConfigs {
		var chosenMLOP string
		for _, host := range cfg.MLOPHosts {
			// Use a shorter timeout to avoid long delays when hosts are not reachable.
			if url := checkConnectivity(host, 2*time.Second); url != "" {
				chosenMLOP = url
				break
			}
		}
		if chosenMLOP == "" {
			continue
		}
		// Registry is optional; use first reachable, else empty to fall back to default
		var chosenRegistry string
		for _, host := range cfg.RegistryHosts {
			if url := checkConnectivity(host, 2*time.Second); url != "" {
				chosenRegistry = url
				break
			}
		}
		selectedEnv = &Environment{
			Config:      cfg,
			MLOPBaseURL: chosenMLOP,
			RegistryURL: chosenRegistry,
		}
		return selectedEnv
	}

	// As a last resort, fall back to the first environment with HTTPS for MLOP host, without connectivity check
	if len(environmentConfigs) > 0 {
		cfg := environmentConfigs[0]
		mlopHost := cfg.MLOPHosts[0]
		selectedEnv = &Environment{
			Config:      cfg,
			MLOPBaseURL: "https://" + strings.TrimSuffix(strings.TrimPrefix(mlopHost, "https://"), "/"),
			RegistryURL: "", // default registry
		}
		logger.Warn("Falling back to default environment without connectivity check", zap.String("domain", cfg.Domain), zap.String("mlop", selectedEnv.MLOPBaseURL))
		return selectedEnv
	}

	// Should not happen; create a stub
	selectedEnv = &Environment{Config: EnvironmentConfig{Domain: "oa"}, MLOPBaseURL: "https://mlop-azure-gateway.mediatek.inc"}
	return selectedEnv
}

func installClaudeCLI() error {
	// Use the best working registry found by selectRegistryURL (mapping-based)
	registry := selectRegistryURL()

	args := []string{"install", "-g", "@anthropic-ai/claude-code@latest"}
	if registry != "" {
		args = append(args, "--registry="+registry)
		logger.Info("Installing @anthropic-ai/claude-code", zap.String("registry", registry))
	} else {
		logger.Info("Installing @anthropic-ai/claude-code via default registry...")
	}

	if err := runCmdLogged(npmPath(), args...); err != nil {
		return fmt.Errorf("npm install failed: %w", err)
	}

	// Verify installation
	if err := verifyClaudeInstalled(); err != nil {
		return fmt.Errorf("installation verification failed: %w", err)
	}

	return nil
}

func verifyClaudeInstalled() error {
	if path, ok := findClaudeBinary(); ok {
		return runCmdLogged(path, "--version")
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
	logger.Info("Installed claude_analysis", zap.String("path", destPath))
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

func writeSettingsJSON(installedBinaryPath string) error {
	// Determine settings path first and handle overwrite/backup prompt early
	homeDir, _ := os.UserHomeDir()
	targetDir := filepath.Join(homeDir, ".claude")
	target := filepath.Join(targetDir, "settings.json")

	var (
		existingSettings    *Settings
		shouldWriteSettings = true
	)

	if _, err := os.Stat(target); err == nil {
		// Read existing for potential merge before renaming
		if existingData, rerr := os.ReadFile(target); rerr == nil {
			var es Settings
			if jerr := json.Unmarshal(existingData, &es); jerr == nil {
				existingSettings = &es
			} else {
				logger.Warn("Existing settings.json is not valid JSON; will overwrite with defaults", zap.Error(jerr))
			}
		}

		if !askYesNo("settings.json already exists. Overwrite it? (y/N): ", false) {
			logger.Info("User chose not to overwrite existing settings.json; skipping settings update.")
			shouldWriteSettings = false
		} else {
			// Backup existing before overwrite
			if err := os.MkdirAll(targetDir, 0o755); err != nil {
				return fmt.Errorf("failed to ensure %s: %w", targetDir, err)
			}
			backupName := fmt.Sprintf("settings.backup_%s.json", time.Now().Format("20060102_150405"))
			backupPath := filepath.Join(targetDir, backupName)
			if err := os.Rename(target, backupPath); err != nil {
				return fmt.Errorf("failed to backup existing settings.json: %w", err)
			}
			logger.Info("Backed up existing settings.json", zap.String("backup", backupPath))
		}
	}

	if !shouldWriteSettings {
		return nil
	}

	// Always use connectivity-based selection for MLOP URL via environment selection
	chosen := selectGaisfURL()

	// Try to get GAISF token for API authentication (only ask when we're going to write)
	var apiKeyHeader string
	if askYesNo("Do you want to configure GAISF token for API authentication? (y/N): ", false) {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter username: ")
		username, _ := reader.ReadString('\n')
		username = strings.TrimSpace(username)
		fmt.Print("Enter password: ")
		password, _ := reader.ReadString('\n')
		password = strings.TrimSpace(password)

		if username != "" && password != "" {
			logger.Info("Attempting to get GAISF token", zap.String("user", username))
			if token, err := getGAISFToken(username, password); err == nil {
				apiKeyHeader = "api-key: " + token
				logger.Info("GAISF token obtained successfully.")
			} else {
				logger.Warn("Failed to get GAISF token", zap.Error(err))
				logger.Info("=== Manual GAISF Token Setup ===")
				logger.Info("Follow steps in your browser to get your GAISF token then paste it below.")
				logger.Info("Login URL:", zap.String("url", chosen+"/auth/login"))
				fmt.Print("Enter your GAISF token (or press Enter to skip): ")
				apiKey, _ := reader.ReadString('\n')
				apiKey = strings.TrimSpace(apiKey)
				if apiKey != "" {
					apiKeyHeader = "api-key: " + apiKey
					logger.Info("GAISF token configured successfully.")
				} else {
					logger.Info("Skipping GAISF token configuration...")
				}
			}
		}
	} else {
		logger.Info("Skipping GAISF token configuration.")
	}

	// Use the actual installed binary path
	hookPath := installedBinaryPath

	// Initialize with default settings
	settings := Settings{
		Env:                        map[string]string{},
		IncludeCoAuthoredBy:        true,
		EnableAllProjectMcpServers: true,
		Hooks: map[string][]Hook{
			"Stop": {
				{Matcher: "*", Hooks: []Hook{{Type: "command", Command: hookPath}}},
			},
		},
	}
	applyDefaultEnv(settings.Env, chosen, "")

	// If we had valid existing settings, start from them and merge updates
	if existingSettings != nil {
		logger.Info("Found existing settings, merging configurations before overwrite...")
		settings = *existingSettings
		if settings.Env == nil {
			settings.Env = make(map[string]string)
		}
		applyDefaultEnv(settings.Env, chosen, "")
		settings.IncludeCoAuthoredBy = true
		settings.EnableAllProjectMcpServers = true
		if settings.Hooks == nil {
			settings.Hooks = make(map[string][]Hook)
		}
		settings.Hooks["Stop"] = []Hook{{Matcher: "*", Hooks: []Hook{{Type: "command", Command: hookPath}}}}
	}

	// Add custom headers if GAISF token was obtained
	if apiKeyHeader != "" {
		settings.Env["ANTHROPIC_CUSTOM_HEADERS"] = apiKeyHeader
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Create target directory and write settings.json
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("failed to create %s: %w", targetDir, err)
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", target, err)
	}
	logger.Info("Wrote settings", zap.String("path", target))
	return nil
}

func runCmdLogged(name string, args ...string) error {
	logger.Debug("Executing command", zap.String("command", name), zap.Strings("args", args))
	fmt.Printf("$ %s %s\n", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		logger.Error("Command failed", zap.String("command", name), zap.Strings("args", args), zap.Error(err))
	}
	return err
}

func runShellLogged(script string) error {
	logger.Debug("Executing shell script", zap.String("script", script))
	fmt.Printf("$ sh -lc %q\n", script)
	cmd := exec.Command("sh", "-lc", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		logger.Error("Shell script failed", zap.String("script", script), zap.Error(err))
	}
	return err
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

// findClaudeBinary attempts to locate the claude CLI either on PATH or in npm's global bin directory.
func findClaudeBinary() (string, bool) {
	if p, err := exec.LookPath("claude"); err == nil {
		return p, true
	}
	out, err := exec.Command(npmPath(), "bin", "-g").Output()
	if err != nil {
		return "", false
	}
	binDir := strings.TrimSpace(string(out))
	p := filepath.Join(binDir, exeName("claude"))
	if _, err := os.Stat(p); err == nil {
		return p, true
	}
	return "", false
}

// askYesNo prompts the user and returns true for yes/false for no. Defaults apply when input is empty.
func askYesNo(prompt string, defaultYes bool) bool {
	fmt.Print(prompt)
	r := bufio.NewReader(os.Stdin)
	resp, _ := r.ReadString('\n')
	resp = strings.TrimSpace(strings.ToLower(resp))
	if resp == "" {
		return defaultYes
	}
	return resp == "y" || resp == "yes"
}

// applyDefaultEnv sets/overwrites the expected env defaults used by settings.json
func applyDefaultEnv(env map[string]string, baseURL string, customHeader string) {
	env["DISABLE_TELEMETRY"] = "1"
	env["CLAUDE_CODE_USE_BEDROCK"] = "1"
	env["ANTHROPIC_BEDROCK_BASE_URL"] = baseURL
	env["CLAUDE_CODE_ENABLE_TELEMETRY"] = "1"
	env["CLAUDE_CODE_SKIP_BEDROCK_AUTH"] = "1"
	env["CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC"] = "1"
	env["NODE_TLS_REJECT_UNAUTHORIZED"] = "0" // Allow self-signed certs for MLOP
	if customHeader != "" {
		env["ANTHROPIC_CUSTOM_HEADERS"] = customHeader
	}
}
