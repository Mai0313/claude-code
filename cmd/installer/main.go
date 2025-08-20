package main

import (
	"archive/zip"
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

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	// Temporary solution for OA, since some computer can access the default registry url.
	{
		Domain:        "oa",
		MLOPHosts:     []string{"https://mlop-azure-gateway.mediatek.inc"},
		RegistryHosts: []string{"https://registry.npmjs.org"},
	},
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

// Styles for the TUI
var (
	titleStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#7D56F4")).Padding(0, 1).Bold(true)
	headerStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true).MarginBottom(1)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("#EE6FF8")).Bold(true)

	paginationStyle = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle       = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle   = lipgloss.NewStyle().Margin(1, 0, 2, 4)

	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F87")).Bold(true)
	inputStyle   = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#7D56F4")).Padding(0, 1)
	promptStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
)

// View types
type viewType int

const (
	mainMenuView viewType = iota
	gaisfConfigView
	inputView
	operationView
)

// Menu item for list
type item struct {
	title, desc   string
	action        func() error
	isFullInstall bool
}

func (i item) FilterValue() string { return i.title }
func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }

// Main model
type model struct {
	list        list.Model
	gaisfList   list.Model
	textInput   textinput.Model
	currentView viewType
	choice      string
	quitting    bool
	operation   string
	result      string
	isError     bool
	inputPrompt string
	inputType   string // "username", "password", "token"
	gaisfConfig *GAISFConfig
}

// GAISF configuration state
type GAISFConfig struct {
	stage     string // "choice", "username", "password", "token", "processing", "complete"
	username  string
	password  string
	token     string
	autoLogin bool
}

func newGAISFConfig() *GAISFConfig {
	return &GAISFConfig{
		stage: "choice",
	}
}

func main() {
	// Ensure child processes that support NO_COLOR also disable colorized output
	os.Setenv("NO_COLOR", "1")
	// Allow self-signed certs for current process
	os.Setenv("NODE_TLS_REJECT_UNAUTHORIZED", "0")

	// Create main menu items
	items := []list.Item{
		item{
			title:         "🚀 Full Installation",
			desc:          "Node.js + Claude CLI + Configuration",
			action:        runFullInstall,
			isFullInstall: true,
		},
		item{
			title:         "🔑 Update GAISF API Key",
			desc:          "Update GAISF token in existing configuration",
			action:        func() error { return updateClaudeCodeSettings() },
			isFullInstall: false,
		},
		item{
			title:         "📦 Install Node.js",
			desc:          "Install Node.js version 22+",
			action:        installNodeJS,
			isFullInstall: false,
		},
		item{
			title:         "🤖 Install/Update Claude CLI",
			desc:          "Install or update Claude CLI package",
			action:        installOrUpdateClaude,
			isFullInstall: false,
		},
		item{
			title:         "❌ Exit",
			desc:          "Quit the program",
			action:        nil,
			isFullInstall: false,
		},
	}

	const defaultWidth = 80
	const listHeight = 14

	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "Claude Code Installer & Configuration Tool"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	// Create text input for forms
	ti := textinput.New()
	ti.Placeholder = "Enter value..."
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 60

	// Create GAISF configuration menu items
	gaisfItems := []list.Item{
		item{
			title:  "🔑 Auto-configure GAISF token",
			desc:   "Login with username/password to get token",
			action: nil,
		},
		item{
			title:  "📝 Manual token input",
			desc:   "Enter GAISF token manually",
			action: nil,
		},
		item{
			title:  "⏭️  Skip GAISF configuration",
			desc:   "Continue without API authentication",
			action: nil,
		},
	}

	gl := list.New(gaisfItems, itemDelegate{}, defaultWidth, listHeight)
	gl.Title = "GAISF API Authentication Setup"
	gl.SetShowStatusBar(false)
	gl.SetFilteringEnabled(false)
	gl.Styles.Title = titleStyle
	gl.Styles.PaginationStyle = paginationStyle
	gl.Styles.HelpStyle = helpStyle

	m := model{
		list:        l,
		gaisfList:   gl,
		textInput:   ti,
		currentView: mainMenuView,
		gaisfConfig: newGAISFConfig(),
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("❌ Error running program: %v", err)
		os.Exit(1)
	}
}

// Custom item delegate for styling
type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i.title)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

// Update function for the TUI
func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.currentView {
	case mainMenuView:
		return m.updateMainMenu(msg)
	case gaisfConfigView:
		return m.updateGaisfConfig(msg)
	case inputView:
		return m.updateInput(msg)
	case operationView:
		return m.updateOperation(msg)
	default:
		return m, nil
	}
}

func (m model) updateMainMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				if i.action == nil { // Exit option
					m.quitting = true
					return m, tea.Quit
				}

				// Special handling for Update GAISF API Key
				if i.title == "🔑 Update GAISF API Key" {
					m.choice = i.title
					m.currentView = gaisfConfigView
					m.gaisfConfig = newGAISFConfig()
					return m, nil
				}

				m.choice = i.title
				m.operation = "Executing: " + i.title
				m.result = ""
				m.isError = false
				m.currentView = operationView
				return m, m.executeOperation(i.action, i.isFullInstall)
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) updateGaisfConfig(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.gaisfList.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.gaisfList.SelectedItem().(item)
			if ok {
				switch i.title {
				case "🔑 Auto-configure GAISF token":
					m.gaisfConfig.autoLogin = true
					m.gaisfConfig.stage = "username"
					m.inputPrompt = "Enter username:"
					m.inputType = "username"
					m.textInput.Placeholder = "username"
					m.textInput.SetValue("")
					m.textInput.EchoMode = textinput.EchoNormal
					m.textInput.Focus()
					m.currentView = inputView
					return m, nil

				case "📝 Manual token input":
					m.gaisfConfig.autoLogin = false
					m.gaisfConfig.stage = "token"
					m.inputPrompt = "Enter your GAISF token:"
					m.inputType = "token"
					m.textInput.Placeholder = "GAISF token"
					m.textInput.SetValue("")
					m.textInput.EchoMode = textinput.EchoPassword
					m.textInput.EchoCharacter = '•'
					m.textInput.Focus()
					m.currentView = inputView
					return m, nil

				case "⏭️  Skip GAISF configuration":
					m.gaisfConfig.stage = "complete"
					// Execute the actual update after skipping GAISF config
					if m.choice == "🔑 Update GAISF API Key" {
						m.currentView = operationView
						m.operation = "Executing: " + m.choice
						return m, m.executeGAISFUpdate("")
					}
					return m, nil
				}
			}

		case "esc":
			m.currentView = mainMenuView
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.gaisfList, cmd = m.gaisfList.Update(msg)
	return m, cmd
}

func (m model) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			value := m.textInput.Value()
			m.textInput.SetValue("")

			switch m.gaisfConfig.stage {
			case "username":
				m.gaisfConfig.username = value
				m.gaisfConfig.stage = "password"
				m.inputPrompt = "Enter password:"
				m.inputType = "password"
				m.textInput.Placeholder = "password"
				m.textInput.EchoMode = textinput.EchoPassword
				m.textInput.EchoCharacter = '•'
				m.textInput.Focus()
				return m, nil

			case "password":
				m.gaisfConfig.password = value
				m.gaisfConfig.stage = "processing"
				m.textInput.EchoMode = textinput.EchoNormal
				m.currentView = operationView
				m.operation = "🔐 Authenticating with GAISF..."
				return m, m.processGaisfAuth()

			case "token":
				m.gaisfConfig.token = value
				m.gaisfConfig.stage = "complete"
				m.textInput.EchoMode = textinput.EchoNormal
				// Now execute the GAISF update with the token
				m.currentView = operationView
				m.operation = "Updating GAISF configuration..."
				return m, m.executeGAISFUpdate(value)
			}

		case "esc":
			m.currentView = gaisfConfigView
			m.gaisfConfig.stage = "choice"
			m.textInput.EchoMode = textinput.EchoNormal
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// Execute GAISF configuration update
func (m model) executeGAISFUpdate(token string) tea.Cmd {
	return func() tea.Msg {
		// Update settings with the new token
		if err := updateClaudeCodeSettings(token); err != nil {
			return operationResult{
				message: fmt.Sprintf("❌ Failed to update settings: %v", err),
				isError: true,
			}
		}

		return operationResult{
			message: "✅ GAISF API Key updated successfully!",
			isError: false,
		}
	}
}

// Process GAISF authentication
func (m model) processGaisfAuth() tea.Cmd {
	return func() tea.Msg {
		if m.gaisfConfig.autoLogin {
			token, err := getGAISFToken(m.gaisfConfig.username, m.gaisfConfig.password)
			if err != nil {
				return operationResult{
					message: fmt.Sprintf("❌ Failed to get GAISF token: %v", err),
					isError: true,
				}
			}
			if updateErr := updateClaudeCodeSettings(token); updateErr != nil {
				return operationResult{
					message: fmt.Sprintf("❌ Failed to update settings: %v", updateErr),
					isError: true,
				}
			}
		}

		return operationResult{
			message: "✅ GAISF authentication and configuration updated successfully!",
			isError: false,
		}
	}
}

func (m model) updateOperation(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "enter", "esc":
			m.currentView = mainMenuView
			return m, nil
		}

	case operationResult:
		m.result = msg.message
		m.isError = msg.isError

		// Auto-switch to GAISF configuration if this was a full install
		if msg.autoSwitchToGAISF && !msg.isError {
			m.currentView = gaisfConfigView
			m.gaisfConfig = newGAISFConfig()
			return m, nil
		}

		return m, nil
	}
	return m, nil
}

// Message types
type operationResult struct {
	message           string
	isError           bool
	autoSwitchToGAISF bool // New field to indicate auto-switch to GAISF
}

// Command to execute operations
func (m model) executeOperation(action func() error, isFullInstall bool) tea.Cmd {
	return func() tea.Msg {
		err := action()
		if err != nil {
			return operationResult{
				message:           fmt.Sprintf("❌ Error: %v", err),
				isError:           true,
				autoSwitchToGAISF: false,
			}
		}

		return operationResult{
			message:           "✅ Operation completed successfully!",
			isError:           false,
			autoSwitchToGAISF: isFullInstall,
		}
	}
}

// View function for the TUI
func (m model) View() string {
	if m.quitting {
		return quitTextStyle.Render("👋 Thank you for using Claude Code Installer!\n")
	}

	switch m.currentView {
	case mainMenuView:
		return "\n" + m.list.View()

	case gaisfConfigView:
		if m.gaisfConfig.stage == "choice" {
			return "\n" + m.gaisfList.View()
		} else {
			return fmt.Sprintf(
				"\n%s\n\n%s\n",
				headerStyle.Render("🔑 GAISF API Authentication Setup"),
				"Processing GAISF configuration...",
			)
		}

	case inputView:
		var promptText string
		switch m.gaisfConfig.stage {
		case "username":
			promptText = "Enter your username:"
		case "password":
			promptText = "Enter your password (hidden):"
		case "token":
			promptText = "Enter your GAISF token (hidden):"
		default:
			promptText = m.inputPrompt
		}

		return fmt.Sprintf(
			"\n%s\n\n%s\n%s\n\n%s\n",
			headerStyle.Render("📝 Input Required"),
			promptStyle.Render(promptText),
			inputStyle.Render(m.textInput.View()),
			"Press Enter to confirm, Esc to go back",
		)

	case operationView:
		statusMsg := m.operation
		if m.result != "" {
			if m.isError {
				statusMsg += "\n\n" + errorStyle.Render(m.result)
			} else {
				statusMsg += "\n\n" + successStyle.Render(m.result)
			}
			statusMsg += "\n\nPress Enter to return to main menu..."
		} else {
			statusMsg += "\n\n⏳ Processing..."
		}
		return fmt.Sprintf(
			"\n%s\n\n%s\n",
			headerStyle.Render("🔄 Operation in Progress"),
			statusMsg,
		)

	default:
		return ""
	}
}

// Menu item structure - kept for compatibility
type MenuItem struct {
	Label         string
	Description   string
	Action        func() error
	IsFullInstall bool // New field to identify full install
}

func runFullInstall() error {
	fmt.Println("🚀 Starting full Claude Code installation...")

	// 1) Node.js check/install guidance
	installNodeJS()

	// 2) Install @anthropic-ai/claude-code with registry fallbacks
	// and move claude_analysis to ~/.claude with platform-specific name
	installOrUpdateClaude()

	fmt.Println("🎉 Installation completed successfully!")
	fmt.Println("� Automatically switching to GAISF API Key configuration...")
	return nil
}

func installNodeJS() error {
	fmt.Println("📦 Step 1: Checking Node.js...")
	if !checkNodeVersion() {
		if isCommandAvailable("node") {
			fmt.Println("⚡ Node.js found but version is less than 22. Upgrading...")
		} else {
			fmt.Println("📦 Node.js not found. Installing...")
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
	}

	fmt.Println("✅ Node.js version >= 22 found. Skipping Node.js installation.")
	return nil
}

func installOrUpdateClaude() error {
	fmt.Println("🤖 Installing/Updating Claude Code CLI...")

	if err := installClaudeCLI(); err != nil {
		return fmt.Errorf("failed to install/update Claude CLI: %w", err)
	}

	fmt.Println("✅ Claude Code CLI installation/update completed!")
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to get home dir: %w", err)
	}
	targetDir := filepath.Join(homeDir, ".claude")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("failed to create %s: %w", targetDir, err)
	}

	// Determine source binary path: same directory as this installer
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("os.Executable failed: %w", err)
	}
	srcDir := filepath.Dir(exe)
	srcName := exeName("claude_analysis")
	srcPath := filepath.Join(srcDir, srcName)
	if _, err := os.Stat(srcPath); err != nil {
		return fmt.Errorf("expected %s next to installer: %w", srcName, err)
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
		return fmt.Errorf("failed to install claude_analysis to %s: %w", destPath, err)
	}
	fmt.Printf("✅ Installed claude_analysis to: %s\n", destPath)
	return nil
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
	fmt.Println("❌ Unable to install Node.js automatically on macOS. Please install Node.js LTS from https://nodejs.org/ and re-run this installer.")
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
	fmt.Println("❌ Unable to install Node.js automatically on Linux. Please install Node.js LTS (v22) from https://nodejs.org/ and re-run this installer.")
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

	fmt.Printf("📦 Extracting Node.js from %s to %s...\n", zipPath, targetDir)
	if err := unzip(zipPath, targetDir); err != nil {
		return fmt.Errorf("extract node zip: %w", err)
	}

	// Some Node.js zips wrap files in a single version folder. Flatten it.
	if err := flattenIfSingleSubdir(targetDir); err != nil {
		fmt.Printf("⚠️ Warning: Failed to flatten node directory: %v\n", err)
	}

	// Persist user environment variables (User scope)
	if err := setWindowsUserEnv("NODE_HOME", targetDir); err != nil {
		fmt.Printf("⚠️ Warning: Failed to set NODE_HOME (user): %v\n", err)
	}
	// Ensure PATH includes targetDir
	if err := ensureWindowsUserPathIncludes(targetDir); err != nil {
		fmt.Printf("⚠️ Warning: Failed to update PATH (user): %v\n", err)
	}

	// Also update current process environment so subsequent steps in this run can use node/npm immediately
	_ = os.Setenv("NODE_HOME", targetDir)
	_ = os.Setenv("PATH", addToPath(os.Getenv("PATH"), targetDir))

	// Broadcast environment change so future processes can pick up updated user env without reboot
	if err := broadcastWindowsEnvChange(); err != nil {
		fmt.Printf("⚠️ Warning: Failed to broadcast environment change: %v\n", err)
	}

	fmt.Println("✅ Node.js installed on Windows.")
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
		fmt.Printf("⚠️ Warning: Falling back to default environment without connectivity check (domain=%s, mlop=%s)\n", cfg.Domain, selectedEnv.MLOPBaseURL)
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

// installClaudeCLI installs the @anthropic-ai/claude-code package using npm.
// It first tries the default npm registry, and if that fails, it looks for a fallback registry from the available environments.
// It verifies the installation by checking if the `claude --version` command works.
func installClaudeCLI() error {
	baseArgs := []string{"install", "-g", "@anthropic-ai/claude-code@latest", "--no-color"}

	// --- 步驟 1: 嘗試使用預設 registry 安裝 ---
	fmt.Println("📦 Attempting to install @anthropic-ai/claude-code with default registry...")
	err := runLoggedCmd(npmPath(), baseArgs...)

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

	chosen := selectAvailableUrl()
	if chosen.RegistryURL == "" {
		// 如果沒有找到備用 registry，返回第一次嘗試的錯誤
		return fmt.Errorf("npm install failed with default registry and no fallback registry is available: %w", err)
	}

	// 建立帶有 registry 的新參數
	retryArgs := append(baseArgs, "--registry="+chosen.RegistryURL)
	fmt.Printf("📦 Retrying installation with registry: %s\n", chosen.RegistryURL)

	// 執行重試
	if retryErr := runLoggedCmd(npmPath(), retryArgs...); retryErr != nil {
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

// New TUI-based settings configuration with optional token parameter
func updateClaudeCodeSettings(token ...string) error {
	fmt.Println("🔑 Updating GAISF API Key...")
	// Resolve settings path and load existing settings (if any) for merge
	homeDir, _ := os.UserHomeDir()
	targetDir := filepath.Join(homeDir, ".claude")
	hookPath := filepath.Join(homeDir, ".claude", exeName("claude_analysis"))
	target := filepath.Join(targetDir, "settings.json")

	var existingSettings *Settings

	if _, err := os.Stat(target); err == nil {
		if existingData, rerr := os.ReadFile(target); rerr == nil {
			var es Settings
			if jerr := json.Unmarshal(existingData, &es); jerr == nil {
				existingSettings = &es
			} else {
				fmt.Printf("⚠️ Warning: Existing settings.json is not valid JSON; proceeding with defaults: %v\n", jerr)
			}
		}
	}

	// Always use connectivity-based selection for MLOP URL via environment selection
	chosen := selectAvailableUrl()

	var gaisfToken string

	// Check if token is provided as parameter
	if len(token) > 0 && token[0] != "" {
		gaisfToken = token[0]
		fmt.Println("🔑 Using provided token...")
	} else {
		// Create GAISF configuration TUI
		gaisfResult, err := runGAISFConfigTUI()
		if err != nil {
			return fmt.Errorf("GAISF configuration failed: %w", err)
		}
		gaisfToken = gaisfResult.token
	}

	// Build settings from existing (if any) and ensure unified defaults
	var settings Settings
	if existingSettings != nil {
		fmt.Println("📋 Found existing settings, merging configurations...")
		settings = *existingSettings
	}
	ensureDefaultSettings(&settings, hookPath, chosen.MLOPBaseURL, "")

	// Add custom headers if GAISF token was obtained
	if gaisfToken != "" {
		settings.Env["ANTHROPIC_CUSTOM_HEADERS"] = "api-key: " + gaisfToken
	} else {
		// Remove the header if no token provided (skip case)
		delete(settings.Env, "ANTHROPIC_CUSTOM_HEADERS")
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

type GAISFResult struct {
	token   string
	skipped bool
}

// Run GAISF configuration TUI
func runGAISFConfigTUI() (*GAISFResult, error) {
	// Create a separate TUI program for GAISF configuration
	gaisfModel := &gaisfConfigModel{
		textInput: textinput.New(),
		config:    newGAISFConfig(),
		result:    &GAISFResult{},
	}

	gaisfModel.textInput.Focus()
	gaisfModel.textInput.CharLimit = 500
	gaisfModel.textInput.Width = 60

	p := tea.NewProgram(gaisfModel)
	_, err := p.Run()
	if err != nil {
		return nil, err
	}

	return gaisfModel.result, nil
}

// Dedicated GAISF configuration model
type gaisfConfigModel struct {
	textInput textinput.Model
	config    *GAISFConfig
	result    *GAISFResult
	quitting  bool
	error     string
}

func (m *gaisfConfigModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *gaisfConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case gaisfAuthResult:
		if msg.error != nil {
			m.error = fmt.Sprintf("Authentication failed: %v", msg.error)
			m.config.stage = "choice"
			m.textInput.EchoMode = textinput.EchoNormal
			m.textInput.SetValue("")
		} else {
			m.result.token = msg.token
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			m.result.skipped = true
			return m, tea.Quit

		case "1":
			if m.config.stage == "choice" {
				m.config.autoLogin = true
				m.config.stage = "username"
				m.textInput.Placeholder = "Enter username"
				m.textInput.SetValue("")
			}
			return m, nil

		case "2":
			if m.config.stage == "choice" {
				m.config.autoLogin = false
				m.config.stage = "token"
				m.textInput.Placeholder = "Enter GAISF token"
				m.textInput.EchoMode = textinput.EchoPassword
				m.textInput.EchoCharacter = '•'
				m.textInput.SetValue("")
			}
			return m, nil

		case "3":
			if m.config.stage == "choice" {
				m.result.skipped = true
				m.quitting = true
				return m, tea.Quit
			}

		case "enter":
			if m.config.stage == "choice" {
				m.result.skipped = true
				m.quitting = true
				return m, tea.Quit
			}
			return m.handleEnter()

		case "esc":
			if m.config.stage != "choice" {
				m.config.stage = "choice"
				m.textInput.EchoMode = textinput.EchoNormal
				m.textInput.SetValue("")
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m *gaisfConfigModel) handleEnter() (tea.Model, tea.Cmd) {
	value := strings.TrimSpace(m.textInput.Value())

	switch m.config.stage {
	case "username":
		if value == "" {
			m.error = "Username cannot be empty"
			return m, nil
		}
		m.config.username = value
		m.config.stage = "password"
		m.textInput.Placeholder = "Enter password"
		m.textInput.EchoMode = textinput.EchoPassword
		m.textInput.EchoCharacter = '•'
		m.textInput.SetValue("")
		m.error = ""
		return m, nil

	case "password":
		if value == "" {
			m.error = "Password cannot be empty"
			return m, nil
		}
		m.config.password = value
		return m, m.authenticateGAISF()

	case "token":
		if value == "" {
			m.error = "Token cannot be empty"
			return m, nil
		}
		m.result.token = value
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}

func (m *gaisfConfigModel) authenticateGAISF() tea.Cmd {
	return func() tea.Msg {
		token, err := getGAISFToken(m.config.username, m.config.password)
		if err != nil {
			return gaisfAuthResult{error: err}
		}
		return gaisfAuthResult{token: token}
	}
}

type gaisfAuthResult struct {
	token string
	error error
}

func (m *gaisfConfigModel) View() string {
	if m.quitting {
		if m.result.skipped {
			return "⏭️  Skipping GAISF configuration...\n"
		}
		return "✅ GAISF configuration completed!\n"
	}

	var content strings.Builder
	content.WriteString(headerStyle.Render("🔑 GAISF API Authentication Setup"))
	content.WriteString("\n\n")

	switch m.config.stage {
	case "choice":
		content.WriteString("Configure GAISF token for API authentication?\n\n")
		content.WriteString("1. 🔑 Auto-configure GAISF token (Login with username/password)\n")
		content.WriteString("2. 📝 Manual token input (Enter GAISF token manually)\n")
		content.WriteString("3. ⏭️  Skip GAISF configuration (Continue without authentication)\n\n")
		content.WriteString(promptStyle.Render("Please select an option (1-3):"))

	case "username":
		content.WriteString("Enter your username:\n\n")
		content.WriteString(inputStyle.Render(m.textInput.View()))
		content.WriteString("\n\nPress Enter to continue, Esc to go back")

	case "password":
		content.WriteString("Enter your password (hidden):\n\n")
		content.WriteString(inputStyle.Render(m.textInput.View()))
		content.WriteString("\n\nPress Enter to authenticate, Esc to go back")

	case "token":
		content.WriteString("Enter your GAISF token (hidden):\n\n")
		content.WriteString(inputStyle.Render(m.textInput.View()))
		content.WriteString("\n\nPress Enter to continue, Esc to go back")

	case "processing":
		content.WriteString("🔐 Authenticating with GAISF...\n\n⏳ Please wait...")
	}

	if m.error != "" {
		content.WriteString("\n\n")
		content.WriteString(errorStyle.Render("❌ " + m.error))
	}

	return content.String()
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
		fmt.Printf("❌ Error: Command failed: %s %v - %v\n", name, args, err)
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
		fmt.Printf("❌ Error: Shell script failed: %s - %v\n", script, err)
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
