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
	"regexp"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
	if checkClaudeInstalled() {
		logger.Info("Claude CLI already installed. Skipping installation.")
	} else {
		err := installClaudeCLI()
		if err != nil {
			return err
		}
	}

	// 2.1) Run claude update to ensure latest version
	logger.Info("Updating Claude CLI to latest version...")
	err := runCmdLogged("claude", "update")
	if err != nil {
		logger.Warn("Failed to update Claude CLI", zap.Error(err))
		// Don't return error as this is not critical
	} else {
		logger.Info("Claude CLI updated successfully.")
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

// getGAISFToken performs login to get JWT token from the MLOP gateway
// Returns the JWT token string or error if login fails
func getGAISFToken(username, password string) (string, error) {
	// Get gateway URL using selectGaisfURL
	gatewayURL := selectGaisfURL()
	loginURL := gatewayURL + "/auth/login"

	// Create HTTP client with fresh cookie jar for each attempt
	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", fmt.Errorf("failed to create cookie jar: %w", err)
	}

	client := &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // Disable SSL verification like Python version
			},
		},
	}

	// Step 1: Get CSRF token from login page
	resp, err := client.Get(loginURL)
	if err != nil {
		return "", fmt.Errorf("failed to get login page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login page request failed with status: %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read login page: %w", err)
	}
	responseText := string(body)

	// Extract CSRF token from HTML form with multiple patterns
	csrfPatterns := []string{
		`<input type="hidden" name="_csrf" value="([^"]+)"`,
		`name="_csrf" value="([^"]+)"`,
		`_csrf.*?value="([^"]+)"`,
		`csrf.*?value="([^"]+)"`,
	}

	var csrfToken string
	for _, pattern := range csrfPatterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		if matches := re.FindStringSubmatch(responseText); len(matches) > 1 {
			csrfToken = matches[1]
			break
		}
	}

	// If not found in form inputs, try meta tag
	if csrfToken == "" {
		metaPattern := `<meta name="csrf-token" content="([^"]+)"`
		re := regexp.MustCompile(`(?i)` + metaPattern)
		if matches := re.FindStringSubmatch(responseText); len(matches) > 1 {
			csrfToken = matches[1]
		}
	}

	// Also try cookie-based CSRF token as fallback
	if csrfToken == "" {
		parsedURL, _ := url.Parse(loginURL)
		for _, cookie := range jar.Cookies(parsedURL) {
			if cookie.Name == "csrf_token" || cookie.Name == "_csrf" {
				csrfToken = cookie.Value
				break
			}
		}
	}

	if csrfToken == "" {
		return "", errors.New("could not extract CSRF token from login page")
	}

	// Step 2: Login to get JWT token
	formData := url.Values{
		"_csrf":            {csrfToken},
		"username":         {username},
		"password":         {password},
		"expiration_hours": {"720"},
		"domain":           {"oa"},
	}

	// Create login request with proper headers and URL-encoded form data
	reqBody := formData.Encode()

	req, err := http.NewRequest("POST", loginURL, strings.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", loginURL)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err = client.Do(req)
	if err != nil {
		return "", fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read login response: %w", err)
	}

	responseText = string(body)

	// Check for specific error messages
	if strings.Contains(responseText, "Invalid CSRF token") {
		return "", errors.New("CSRF token validation failed")
	}
	if strings.Contains(responseText, "Login Failed") || strings.Contains(responseText, "Invalid credentials") {
		return "", errors.New("invalid username or password")
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("login failed with status %d: %s", resp.StatusCode, responseText[:min(500, len(responseText))])
	}

	// Extract JWT token using multiple patterns
	jwtPatterns := []string{
		`eyJ[A-Za-z0-9_.-]*\.[A-Za-z0-9_.-]*\.[A-Za-z0-9_.-]*`,
		`"token":\s*"(eyJ[A-Za-z0-9_.-]*\.[A-Za-z0-9_.-]*\.[A-Za-z0-9_.-]*)"`,
		`id="token-value"[^>]*>([^<]+)`,
		`class="token"[^>]*>([^<]+)`,
	}

	for _, pattern := range jwtPatterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(responseText); len(matches) > 0 {
			var token string
			if len(matches) > 1 && matches[1] != "" {
				token = matches[1]
			} else {
				token = matches[0]
			}

			// Validate JWT format
			if strings.Count(token, ".") == 2 && strings.HasPrefix(token, "eyJ") {
				return token, nil
			}
		}
	}

	// If no token found but login seems successful, check for redirect
	if (resp.StatusCode == 200 || resp.StatusCode == 302) && resp.Header.Get("Location") != "" {
		redirectURL := resp.Header.Get("Location")
		if !strings.HasPrefix(redirectURL, "http") {
			// Relative URL, make it absolute
			redirectURL = gatewayURL + redirectURL
		}

		redirectResp, err := client.Get(redirectURL)
		if err == nil {
			defer redirectResp.Body.Close()
			redirectBody, err := io.ReadAll(redirectResp.Body)
			if err == nil {
				redirectText := string(redirectBody)
				for _, pattern := range jwtPatterns {
					re := regexp.MustCompile(pattern)
					if matches := re.FindStringSubmatch(redirectText); len(matches) > 0 {
						var token string
						if len(matches) > 1 && matches[1] != "" {
							token = matches[1]
						} else {
							token = matches[0]
						}

						// Validate JWT format
						if strings.Count(token, ".") == 2 && strings.HasPrefix(token, "eyJ") {
							return token, nil
						}
					}
				}
			}
		}
	}

	return "", fmt.Errorf("could not extract JWT token from login response. Response: %s", responseText[:min(500, len(responseText))])
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false, // Keep certificate verification enabled
			},
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
			logger.Info("Found working registry", zap.String("url", workingURL))
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
			logger.Info("Using MLOP gateway", zap.String("url", workingURL))
			return workingURL
		}
	}

	// Fallback: just use first option with HTTPS if no connectivity check worked
	return "https://mlop-azure-gateway.mediatek.inc"
}

func installClaudeCLI() error {
	// Use the best working registry found by selectRegistryURL
	registry := selectRegistryURL()

	args := []string{"install", "-g", "@anthropic-ai/claude-code"}
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

		// Prompt user for overwrite
		fmt.Print("settings.json already exists. Overwrite it? (y/N): ")
		reader := bufio.NewReader(os.Stdin)
		resp, _ := reader.ReadString('\n')
		resp = strings.TrimSpace(strings.ToLower(resp))
		if resp != "y" && resp != "yes" {
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

	// Always use connectivity-based selection for MLOP URL
	chosen := selectGaisfURL()

	// Try to get JWT token for API authentication (only ask when we're going to write)
	var apiKeyHeader string
	fmt.Print("Do you want to configure JWT token for API authentication? (y/N): ")
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "y" || response == "yes" {
		fmt.Print("Enter username: ")
		username, _ := reader.ReadString('\n')
		username = strings.TrimSpace(username)

		fmt.Print("Enter password: ")
		password, _ := reader.ReadString('\n')
		password = strings.TrimSpace(password)

		if username != "" && password != "" {
			logger.Info("Attempting to get JWT token", zap.String("user", username))
			if token, err := getGAISFToken(username, password); err == nil {
				apiKeyHeader = "api-key: " + token
				logger.Info("JWT token obtained successfully.")
			} else {
				logger.Warn("Failed to get JWT token", zap.Error(err))
				logger.Info("\n=== Manual API Key Setup ===")
				logger.Info("Please follow these steps to manually obtain your API key:")
				logger.Info("1. Open this URL in your browser", zap.String("url", chosen+"/auth/login"))
				logger.Info("2. Log in with your credentials")
				logger.Info("3. Navigate to the API key management section")
				logger.Info("4. Generate or copy your API key")
				logger.Info("5. Paste it below")

				fmt.Print("Enter your API key (or press Enter to skip): ")
				apiKey, _ := reader.ReadString('\n')
				apiKey = strings.TrimSpace(apiKey)

				if apiKey != "" {
					apiKeyHeader = "api-key: " + apiKey
					logger.Info("API key configured successfully.")
				} else {
					logger.Info("Skipping API key configuration...")
				}
			}
		}
	} else {
		logger.Info("Skipping JWT token configuration.")
	}

	// Use the actual installed binary path
	hookPath := installedBinaryPath

	// Initialize with default settings
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
					Hooks:   []Hook{{Type: "command", Command: hookPath}},
				},
			},
		},
	}

	// If we had valid existing settings, start from them and merge updates
	if existingSettings != nil {
		logger.Info("Found existing settings, merging configurations before overwrite...")
		settings = *existingSettings
		if settings.Env == nil {
			settings.Env = make(map[string]string)
		}
		settings.Env["DISABLE_TELEMETRY"] = "1"
		settings.Env["CLAUDE_CODE_USE_BEDROCK"] = "1"
		settings.Env["ANTHROPIC_BEDROCK_BASE_URL"] = chosen
		settings.Env["CLAUDE_CODE_ENABLE_TELEMETRY"] = "1"
		settings.Env["CLAUDE_CODE_SKIP_BEDROCK_AUTH"] = "1"
		settings.Env["CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC"] = "1"

		settings.IncludeCoAuthoredBy = true
		settings.EnableAllProjectMcpServers = true

		if settings.Hooks == nil {
			settings.Hooks = make(map[string][]Hook)
		}
		settings.Hooks["Stop"] = []Hook{{Matcher: "*", Hooks: []Hook{{Type: "command", Command: hookPath}}}}
	}

	// Add custom headers if JWT token was obtained
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
