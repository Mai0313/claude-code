package main

import (
	"archive/zip"
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
		MLOPHosts:     []string{"https://mlop-azure-gateway.mediatek.inc"},
		RegistryHosts: []string{"https://oa-mirror.mediatek.inc/repository/npm"},
	},
	{
		Domain:        "swrd",
		MLOPHosts:     []string{"https://mlop-azure-rddmz.mediatek.inc"},
		RegistryHosts: []string{"https://swrd-mirror.mediatek.inc/repository/npm"},
	},
}

// Environment is the resolved and connectivity-validated selection
type Environment struct {
	Config      EnvironmentConfig
	MLOPBaseURL string // with scheme
	RegistryURL string // with scheme (or empty to use default)
}

var selectedEnv *Environment

func clearScreen() {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	} else {
		fmt.Print("\033[H\033[2J")
	}
}

func main() {
	// Ensure child processes that support NO_COLOR also disable colorized output
	_ = os.Setenv("NO_COLOR", "1")
	// Allow self-signed certs for current process
	_ = os.Setenv("NODE_TLS_REJECT_UNAUTHORIZED", "0")

	// Clear screen and show welcome
	clearScreen()
	showMainMenu()
}

// Menu item structure
type MenuItem struct {
	Label       string
	Description string
	Action      func() error
}

func showMainMenu() {
	menuItems := []MenuItem{
		{
			Label:       "🚀 Full Installation",
			Description: "Node.js + Claude CLI + Configuration",
			Action:      runFullInstall,
		},
		{
			Label:       "🔑 Update GAISF API Key",
			Description: "Update GAISF token in existing configuration",
			Action:      updateGAISFKey,
		},
		{
			Label:       "📦 Install Node.js",
			Description: "Install Node.js version 22+",
			Action:      installNodeJS,
		},
		{
			Label:       "🤖 Install/Update Claude CLI",
			Description: "Install or update Claude CLI package",
			Action:      installOrUpdateClaude,
		},
		{
			Label:       "❌ Exit",
			Description: "Quit the program",
			Action:      nil,
		},
	}

	for {
		selectedIndex := showInteractiveMenu(menuItems)

		if selectedIndex == len(menuItems)-1 { // Exit option
			fmt.Println("👋 Thank you for using Claude Code Installer!")
			pauseIfInteractive()
			return
		}

		// Execute selected action
		if menuItems[selectedIndex].Action != nil {
			executeWithErrorHandling(menuItems[selectedIndex].Label, menuItems[selectedIndex].Action)
		}

		fmt.Println()
		fmt.Println("Press Enter to return to main menu...")
		reader := bufio.NewReader(os.Stdin)
		reader.ReadString('\n')
		clearScreen() // Clear screen
	}
}

func showInteractiveMenu(items []MenuItem) int {
	for {
		// Clear screen and move cursor to top
		clearScreen()
		fmt.Println(`
		=============================================
		🤖 Claude Code Installer & Configuration Tool
		=============================================
		`)
		fmt.Println()

		// Display menu items with numbers
		for i, item := range items {
			fmt.Printf("%d. %s", i+1, item.Label)
			if item.Description != "" {
				fmt.Printf(" (%s)", item.Description)
			}
			fmt.Println()
		}

		fmt.Println()
		fmt.Printf("Please select an option (1-%d) [default: 1]: ", len(items))

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		// Default to first option if empty input
		if input == "" {
			return 0
		}

		// Handle quit options
		if strings.ToLower(input) == "q" || strings.ToLower(input) == "quit" || strings.ToLower(input) == "exit" {
			return len(items) - 1
		}

		// Parse number selection
		var selectedIndex int
		if _, err := fmt.Sscanf(input, "%d", &selectedIndex); err == nil {
			selectedIndex-- // Convert to 0-based index
			if selectedIndex >= 0 && selectedIndex < len(items) {
				return selectedIndex
			}
		}

		fmt.Printf("Invalid selection. Please enter a number between 1 and %d.\n", len(items))
		fmt.Println("Press Enter to continue...")
		reader.ReadString('\n')
	}
}

func showGAISFConfigMenu() int {
	gaifsMenuItems := []MenuItem{
		{
			Label:       "🔑 Auto-configure GAISF token",
			Description: "Login with username/password to get token",
			Action:      nil,
		},
		{
			Label:       "📝 Manual token input",
			Description: "Enter GAISF token manually",
			Action:      nil,
		},
		{
			Label:       "⏭️  Skip GAISF configuration",
			Description: "Continue without API authentication",
			Action:      nil,
		},
	}

	for {
		// Clear screen and move cursor to top
		clearScreen()

		fmt.Println("🔑 GAISF API Authentication Setup")
		fmt.Println()
		fmt.Println("Configure GAISF token for API authentication?")
		fmt.Println()

		// Display menu items with numbers
		for i, item := range gaifsMenuItems {
			fmt.Printf("%d. %s", i+1, item.Label)
			if item.Description != "" {
				fmt.Printf(" (%s)", item.Description)
			}
			fmt.Println()
		}

		fmt.Println()
		fmt.Printf("Please select an option (1-%d) [default: 1]: ", len(gaifsMenuItems))

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		// Default to first option if empty input
		if input == "" {
			return 0
		}

		// Handle quit options - skip configuration
		if strings.ToLower(input) == "q" || strings.ToLower(input) == "quit" || strings.ToLower(input) == "exit" {
			return 2 // Skip configuration (index 2)
		}

		// Parse number selection
		var selectedIndex int
		if _, err := fmt.Sscanf(input, "%d", &selectedIndex); err == nil {
			selectedIndex-- // Convert to 0-based index
			if selectedIndex >= 0 && selectedIndex < len(gaifsMenuItems) {
				return selectedIndex
			}
		}

		fmt.Printf("Invalid selection. Please enter a number between 1 and %d.\n", len(gaifsMenuItems))
		fmt.Println("Press Enter to continue...")
		reader.ReadString('\n')
	}
}

func executeWithErrorHandling(operationName string, operation func() error) {
	clearScreen() // Clear screen
	fmt.Printf("Executing: %s\n", operationName)
	fmt.Println(strings.Repeat("=", 50))

	if err := operation(); err != nil {
		fmt.Printf("❌ Error: %s failed: %v\n", operationName, err)
	} else {
		fmt.Printf("✅ %s completed successfully!\n", operationName)
	}

	fmt.Println(strings.Repeat("=", 50))
}

func runFullInstall() error {
	fmt.Println("🚀 Starting full Claude Code installation...")
	return run()
}

func updateGAISFKey() error {
	fmt.Println("🔑 Updating GAISF API Key...")

	// Check if settings.json exists
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to get home dir: %w", err)
	}
	settingsPath := filepath.Join(homeDir, ".claude", "settings.json")

	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		return fmt.Errorf("settings.json not found at %s. Please run full installation first", settingsPath)
	}

	// Load existing settings
	var settings Settings
	if existingData, err := os.ReadFile(settingsPath); err == nil {
		if err := json.Unmarshal(existingData, &settings); err != nil {
			return fmt.Errorf("failed to parse existing settings: %w", err)
		}
	}

	// Get new GAISF token
	chosen := selectAvailableUrl()
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Print("Enter password: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	if username == "" || password == "" {
		return errors.New("username and password are required")
	}

	fmt.Printf("Attempting to get GAISF token for user: %s\n", username)
	token, err := getGAISFToken(username, password)
	if err != nil {
		fmt.Printf("Failed to get GAISF token automatically: %v\n", err)
		fmt.Println("=== Manual GAISF Token Setup ===")
		fmt.Printf("Login URL: %s\n", chosen.MLOPBaseURL+"/auth/login")
		fmt.Print("Enter your GAISF token: ")
		manualToken, _ := reader.ReadString('\n')
		token = strings.TrimSpace(manualToken)
		if token == "" {
			return errors.New("no token provided")
		}
	}

	// Update settings with new token
	if settings.Env == nil {
		settings.Env = make(map[string]string)
	}
	settings.Env["ANTHROPIC_CUSTOM_HEADERS"] = "api-key: " + token

	// Save updated settings
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(settingsPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write settings: %w", err)
	}

	fmt.Println("✅ GAISF API Key updated successfully!")
	return nil
}

func installNodeJS() error {
	fmt.Println("📦 Installing Node.js...")

	if checkNodeVersion() {
		fmt.Println("✅ Node.js version >= 22 already installed!")
		return nil
	}

	if isCommandAvailable("node") {
		fmt.Println("Node.js found but version is less than 22. Upgrading...")
	} else {
		fmt.Println("Node.js not found. Installing...")
	}

	switch runtime.GOOS {
	case "windows":
		if err := installNodeWindows(); err != nil {
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

	fmt.Println("✅ Node.js installation completed!")
	return nil
}

func installOrUpdateClaude() error {
	fmt.Println("🤖 Installing/Updating Claude Code CLI...")

	if err := installClaudeCLI(); err != nil {
		return fmt.Errorf("failed to install/update Claude CLI: %w", err)
	}

	fmt.Println("✅ Claude Code CLI installation/update completed!")
	return nil
}

func run() error {
	fmt.Println("\n📋 Installation Steps:")
	fmt.Println("1. ✓ Check/Install Node.js")
	fmt.Println("2. ✓ Install Claude CLI")
	fmt.Println("3. ✓ Install claude_analysis binary")
	fmt.Println("4. ✓ Generate settings.json")
	fmt.Println()

	// 1) Node.js check/install guidance
	fmt.Println("📦 Step 1: Checking Node.js...")
	if !checkNodeVersion() {
		if isCommandAvailable("node") {
			fmt.Println("Node.js found but version is less than 22. Upgrading...")
		} else {
			fmt.Println("Node.js not found. Installing...")
		}

		switch runtime.GOOS {
		case "windows":
			if err := installNodeWindows(); err != nil {
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
	} else {
		fmt.Println("✅ Node.js version >= 22 found. Skipping Node.js installation.")
	}

	// 2) Install @anthropic-ai/claude-code with registry fallbacks
	fmt.Println("\n🤖 Step 2: Installing/Updating Claude CLI...")
	if err := installClaudeCLI(); err != nil {
		return err
	}

	// 3) Move claude_analysis to ~/.claude with platform-specific name
	fmt.Println("\n⚙️  Step 3: Installing claude_analysis binary...")
	destPath, err := installClaudeAnalysisBinary()
	if err != nil {
		return err
	}

	// 4) Generate settings.json to ~/.claude/settings.json
	fmt.Println("\n📝 Step 4: Generating configuration...")
	if err := writeSettingsJSON(destPath); err != nil {
		return err
	}

	fmt.Println("\n🎉 Installation completed successfully!")
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
	env := selectAvailableUrl()
	baseURL := env.MLOPBaseURL
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
		if err := runLoggedCmd("brew", "install", "node@22"); err == nil {
			_ = runLoggedCmd("brew", "link", "--overwrite", "--force", "node@22")
			return nil
		}
		if err := runLoggedCmd("brew", "install", "node"); err == nil {
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
		_ = runLoggedCmd("sudo", "apt-get", "update")
		if err := runLoggedCmd("sudo", "apt-get", "install", "-y", "nodejs", "npm"); err == nil {
			return nil
		}
		// Try NodeSource for Node 22
		if isCommandAvailable("curl") {
			if err := runLoggedShell("curl -fsSL https://deb.nodesource.com/setup_22.x | sudo -E bash -"); err == nil {
				if err := runLoggedCmd("sudo", "apt-get", "install", "-y", "nodejs"); err == nil {
					return nil
				}
			}
		}
	}
	if isCommandAvailable("dnf") {
		_ = runLoggedCmd("sudo", "dnf", "-y", "module", "enable", "nodejs:22")
		if err := runLoggedCmd("sudo", "dnf", "-y", "install", "nodejs"); err == nil {
			return nil
		}
	}
	if isCommandAvailable("yum") {
		if err := runLoggedCmd("sudo", "yum", "-y", "install", "nodejs", "npm"); err == nil {
			return nil
		}
	}
	if isCommandAvailable("pacman") {
		if err := runLoggedCmd("sudo", "pacman", "-Sy", "--noconfirm", "nodejs", "npm"); err == nil {
			return nil
		}
	}
	fmt.Println("Unable to install Node.js automatically on Linux. Please install Node.js LTS (v22) from https://nodejs.org/ and re-run this installer.")
	return errors.New("node.js not installed")
}

// installNodeWindows downloads the specified Node.js zip, extracts it to Program Files, and sets user env vars.
func installNodeWindows() error {
	const nodeZipName = "node-v22.18.0-win-x64.zip"
	// Install under user's home to avoid requiring Administrator
	targetDir, derr := getNodeInstallDir()
	if derr != nil {
		return derr
	}

	// Locate zip next to the installer executable
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("os.Executable failed: %w", err)
	}
	exeDir := filepath.Dir(exe)
	zipPath := filepath.Join(exeDir, nodeZipName)
	if _, err := os.Stat(zipPath); err != nil {
		return fmt.Errorf("required %s not found next to installer at %s: %w", nodeZipName, exeDir, err)
	}

	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("create target dir %s: %w (try running as Administrator)", targetDir, err)
	}

	fmt.Printf("Extracting Node.js from %s to %s...\n", zipPath, targetDir)
	if err := unzip(zipPath, targetDir); err != nil {
		return fmt.Errorf("extract node zip: %w", err)
	}

	// Some Node.js zips wrap files in a single version folder. Flatten it.
	if err := flattenIfSingleSubdir(targetDir); err != nil {
		fmt.Printf("Warning: Failed to flatten node directory: %v\n", err)
	}

	// Persist user environment variables (User scope)
	if err := setWindowsUserEnv("NODE_HOME", targetDir); err != nil {
		fmt.Printf("Warning: Failed to set NODE_HOME (user): %v\n", err)
	}
	// Ensure PATH includes targetDir
	if err := ensureWindowsUserPathIncludes(targetDir); err != nil {
		fmt.Printf("Warning: Failed to update PATH (user): %v\n", err)
	}

	// Also update current process environment so subsequent steps in this run can use node/npm immediately
	_ = os.Setenv("NODE_HOME", targetDir)
	_ = os.Setenv("PATH", addToPath(os.Getenv("PATH"), targetDir))

	// Broadcast environment change so future processes can pick up updated user env without reboot
	if err := broadcastWindowsEnvChange(); err != nil {
		fmt.Printf("Warning: Failed to broadcast environment change: %v\n", err)
	}

	fmt.Println("Node.js installed on Windows.")
	return nil
}

// unzip extracts a zip archive to the destination directory. Overwrites existing files.
func unzip(srcZip, destDir string) error {
	r, err := zip.OpenReader(srcZip)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		// Resolve path and prevent ZipSlip
		fpath := filepath.Join(destDir, f.Name)
		if !strings.HasPrefix(filepath.Clean(fpath)+string(os.PathSeparator), filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path in zip: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fpath), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			rc.Close()
			return err
		}
		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return err
		}
		out.Close()
		rc.Close()
	}
	return nil
}

// flattenIfSingleSubdir moves contents up if destDir contains exactly one subdirectory and no files.
func flattenIfSingleSubdir(destDir string) error {
	entries, err := os.ReadDir(destDir)
	if err != nil {
		return err
	}
	var dirs []os.DirEntry
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e)
		} else {
			// file at root -> do nothing
			return nil
		}
	}
	if len(dirs) != 1 {
		return nil
	}
	sub := filepath.Join(destDir, dirs[0].Name())
	// Move all items from sub up to destDir
	items, err := os.ReadDir(sub)
	if err != nil {
		return err
	}
	for _, it := range items {
		from := filepath.Join(sub, it.Name())
		to := filepath.Join(destDir, it.Name())
		if err := os.Rename(from, to); err != nil {
			// Fallback to copy if rename fails across volumes (unlikely)
			if it.IsDir() {
				if err2 := copyDir(from, to); err2 != nil {
					return err
				}
				_ = os.RemoveAll(from)
			} else {
				if err2 := copyFile(from, to, 0o755); err2 != nil {
					return err
				}
				_ = os.Remove(from)
			}
		}
	}
	// Remove now-empty subdir
	_ = os.RemoveAll(sub)
	return nil
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := copyDir(s, d); err != nil {
				return err
			}
		} else {
			if err := copyFile(s, d, 0o755); err != nil {
				return err
			}
		}
	}
	return nil
}

func findWindowsNpmFallback() string {
	var bases []string
	if dir, err := getNodeInstallDir(); err == nil {
		bases = append(bases, dir)
	}
	bases = append(bases, `C:\Program Files\nodejs`)
	for _, base := range bases {
		entries, err := os.ReadDir(base)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				p := filepath.Join(base, e.Name(), "npm.cmd")
				if _, err := os.Stat(p); err == nil {
					return p
				}
			}
		}
	}
	return ""
}

func setWindowsUserEnv(name, value string) error {
	// Use PowerShell to persist user-level environment variable
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command",
		fmt.Sprintf("[Environment]::SetEnvironmentVariable('%s','%s','User')", name, value))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func getWindowsUserEnv(name string) (string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command",
		fmt.Sprintf("[Environment]::GetEnvironmentVariable('%s','User')", name))
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func ensureWindowsUserPathIncludes(dir string) error {
	existing, err := getWindowsUserEnv("Path")
	if err != nil {
		existing = os.Getenv("PATH") // fallback to process PATH
	}
	updated := addToPath(existing, dir)
	if updated == existing {
		return nil // already present
	}
	return setWindowsUserEnv("Path", updated)
}

func addToPath(pathVar, dir string) string {
	if dir == "" {
		return pathVar
	}
	// Windows PATH uses ';' separator.
	sep := ";"
	// Normalize for comparison
	target := strings.ToLower(filepath.Clean(dir))
	var parts []string
	if pathVar != "" {
		parts = strings.Split(pathVar, sep)
	}
	for _, p := range parts {
		if strings.ToLower(filepath.Clean(strings.TrimSpace(p))) == target {
			return pathVar // already included
		}
	}
	if pathVar == "" {
		return dir
	}
	if strings.HasSuffix(pathVar, sep) {
		return pathVar + dir
	}
	return pathVar + sep + dir
}

// getNodeInstallDir returns the managed Node.js install directory under the current user's home.
// Example: %USERPROFILE%\.claude\nodejs
func getNodeInstallDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to resolve user home directory: %w", err)
	}
	home = strings.TrimSpace(home)
	if home == "" {
		return "", errors.New("user home directory is empty")
	}
	return filepath.Join(home, ".claude", "nodejs"), nil
}

// broadcastWindowsEnvChange notifies the system that environment variables changed.
// This helps new processes see updated user env without requiring a full logoff.
func broadcastWindowsEnvChange() error {
	ps := `Add-Type @"
using System;
using System.Runtime.InteropServices;
public static class NativeMethods {
	[DllImport("user32.dll", SetLastError=true, CharSet=CharSet.Auto)]
	public static extern IntPtr SendMessageTimeout(IntPtr hWnd, uint Msg, IntPtr wParam, string lParam, uint fuFlags, uint uTimeout, out IntPtr lpdwResult);
}
"@; [IntPtr]$r=[IntPtr]::Zero; [void][NativeMethods]::SendMessageTimeout([IntPtr]0xffff, 0x1A, [IntPtr]::Zero, 'Environment', 0x0002, 5000, [ref]$r)`
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", ps)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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

// selectAvailableUrl resolves and caches the environment selection by testing connectivity
func selectAvailableUrl() *Environment {
	if selectedEnv != nil {
		return selectedEnv
	}

	// Try each configured environment in order; pick the first with a reachable MLOP host
	for _, cfg := range environmentConfigs {
		var chosenMLOP string
		for _, httpsURL := range cfg.MLOPHosts {
			// Use a shorter timeout to avoid long delays when hosts are not reachable.
			if checkURLReachability(httpsURL, 2*time.Second) == nil {
				chosenMLOP = httpsURL
				break
			}
		}
		if chosenMLOP == "" {
			continue
		}
		// Registry is optional; use first reachable, else empty to fall back to default
		var chosenRegistry string
		for _, httpsURL := range cfg.RegistryHosts {
			if checkURLReachability(httpsURL, 2*time.Second) == nil {
				chosenRegistry = httpsURL
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
		fmt.Printf("Warning: Falling back to default environment without connectivity check (domain=%s, mlop=%s)\n", cfg.Domain, selectedEnv.MLOPBaseURL)
		return selectedEnv
	}

	// Should not happen; create a stub
	selectedEnv = &Environment{Config: EnvironmentConfig{Domain: "oa"}, MLOPBaseURL: "https://mlop-azure-gateway.mediatek.inc"}
	return selectedEnv
}

func npmPath() string {
	// Prefer npm next to node if node is found
	if p, err := exec.LookPath("npm"); err == nil {
		return p
	}
	// Windows-specific fallback to standard installation directory
	if runtime.GOOS == "windows" {
		// Prefer our managed install directory under user's home first
		if dir, err := getNodeInstallDir(); err == nil {
			candidate := filepath.Join(dir, "npm.cmd")
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
		// Also check one-level deeper if extracted into a versioned folder under either base
		if p := findWindowsNpmFallback(); p != "" {
			return p
		}
	}
	return "npm" // rely on PATH
}

func installClaudeCLI() error {
	// Use the best working registry found by selectRegistryURL (mapping-based)
	chosen := selectAvailableUrl()

	args := []string{"install", "-g", "@anthropic-ai/claude-code@latest", "--no-color", "--silent"}
	if chosen.RegistryURL != "" {
		args = append(args, "--registry="+chosen.RegistryURL)
		fmt.Printf("📦 Installing @anthropic-ai/claude-code with registry: %s\n", chosen.RegistryURL)
	} else {
		fmt.Println("📦 Installing @anthropic-ai/claude-code via default registry...")
	}

	if err := runLoggedCmd(npmPath(), args...); err != nil {
		return fmt.Errorf("npm install failed: %w", err)
	}

	// Verify installation
	if err := verifyClaudeInstalled(); err != nil {
		return fmt.Errorf("installation verification failed: %w", err)
	}

	fmt.Println("✅ Claude CLI installed successfully!")
	return nil
}

func verifyClaudeInstalled() error {
	if path, ok := findClaudeBinary(); ok {
		return runLoggedCmd(path, "--version")
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
	fmt.Printf("✅ Installed claude_analysis to: %s\n", destPath)
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
	// Resolve settings path and load existing settings (if any) for merge
	homeDir, _ := os.UserHomeDir()
	targetDir := filepath.Join(homeDir, ".claude")
	target := filepath.Join(targetDir, "settings.json")

	var existingSettings *Settings

	if _, err := os.Stat(target); err == nil {
		if existingData, rerr := os.ReadFile(target); rerr == nil {
			var es Settings
			if jerr := json.Unmarshal(existingData, &es); jerr == nil {
				existingSettings = &es
			} else {
				fmt.Printf("⚠️  Warning: Existing settings.json is not valid JSON; proceeding with defaults: %v\n", jerr)
			}
		}
	}

	// Always use connectivity-based selection for MLOP URL via environment selection
	chosen := selectAvailableUrl()

	// Try to get GAISF token for API authentication (only ask when we're going to write)
	var apiKeyHeader string
	gaisfChoice := showGAISFConfigMenu()

	if gaisfChoice == 0 { // Auto-configure
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter username: ")
		username, _ := reader.ReadString('\n')
		username = strings.TrimSpace(username)
		fmt.Print("Enter password: ")
		password, _ := reader.ReadString('\n')
		password = strings.TrimSpace(password)

		if username != "" && password != "" {
			fmt.Printf("🔐 Attempting to get GAISF token for user: %s\n", username)
			if token, err := getGAISFToken(username, password); err == nil {
				apiKeyHeader = "api-key: " + token
				fmt.Println("✅ GAISF token obtained successfully.")
			} else {
				fmt.Printf("⚠️  Warning: Failed to get GAISF token: %v\n", err)
				fmt.Println("=== Manual GAISF Token Setup ===")
				fmt.Println("Follow steps in your browser to get your GAISF token then paste it below.")
				fmt.Printf("Login URL: %s\n", chosen.MLOPBaseURL+"/auth/login")
				fmt.Print("Enter your GAISF token (or press Enter to skip): ")
				apiKey, _ := reader.ReadString('\n')
				apiKey = strings.TrimSpace(apiKey)
				if apiKey != "" {
					apiKeyHeader = "api-key: " + apiKey
					fmt.Println("✅ GAISF token configured successfully.")
				} else {
					fmt.Println("⏭️  Skipping GAISF token configuration...")
				}
			}
		}
	} else if gaisfChoice == 1 { // Manual token input
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("📝 Manual GAISF Token Input")
		fmt.Printf("Login URL: %s\n", chosen.MLOPBaseURL+"/auth/login")
		fmt.Println("Please get your GAISF token from the above URL and paste it below.")
		fmt.Print("Enter your GAISF token: ")
		apiKey, _ := reader.ReadString('\n')
		apiKey = strings.TrimSpace(apiKey)
		if apiKey != "" {
			apiKeyHeader = "api-key: " + apiKey
			fmt.Println("✅ GAISF token configured successfully.")
		} else {
			fmt.Println("⏭️  No token provided, skipping GAISF configuration...")
		}
	} else { // Skip configuration
		fmt.Println("⏭️  Skipping GAISF token configuration.")
	}

	// Use the actual installed binary path
	hookPath := installedBinaryPath

	// Build settings from existing (if any) and ensure unified defaults
	var settings Settings
	if existingSettings != nil {
		fmt.Println("📋 Found existing settings, merging configurations...")
		settings = *existingSettings
	}
	ensureDefaultSettings(&settings, hookPath, chosen.MLOPBaseURL, "")

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
	fmt.Printf("✅ Wrote settings to: %s\n", target)
	return nil
}

// ensureDefaultSettings applies unified defaults and required structure to settings.
// It is idempotent and can be called whether settings was empty or loaded from disk.
func ensureDefaultSettings(settings *Settings, hookPath, baseURL, customHeader string) {
	if settings.Env == nil {
		settings.Env = make(map[string]string)
	}
	applyDefaultEnv(settings.Env, baseURL, customHeader)
	// Hard-enable flags required by the app
	settings.IncludeCoAuthoredBy = true
	settings.EnableAllProjectMcpServers = true
	// Ensure required Stop hook exists and points to provided hookPath
	if settings.Hooks == nil {
		settings.Hooks = make(map[string][]Hook)
	}
	settings.Hooks["Stop"] = []Hook{{Matcher: "*", Hooks: []Hook{{Type: "command", Command: hookPath}}}}
}

func runLoggedCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	// Ensure color is disabled for child processes that honor NO_COLOR
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error: Command failed: %s %v - %v\n", name, args, err)
	}
	return err
}

func runLoggedShell(script string) error {
	cmd := exec.Command("sh", "-lc", script)
	// Ensure color is disabled for child processes that honor NO_COLOR
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error: Shell script failed: %s - %v\n", script, err)
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
