package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"claude_analysis/cmd/installer/internal/auth"
	"claude_analysis/cmd/installer/internal/config"
)

// Update function for the TUI
func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.CurrentView {
	case MainMenuView:
		return m.updateMainMenu(msg)
	case GAISFConfigView:
		return m.updateGAISFConfig(msg)
	case InputView:
		return m.updateInput(msg)
	case OperationView:
		return m.updateOperation(msg)
	default:
		return m, nil
	}
}

func (m Model) updateMainMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.List.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "q":
			m.Quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.List.SelectedItem().(Item)
			if ok {
				if i.Action == nil { // Exit option
					m.Quitting = true
					return m, tea.Quit
				}

				// Special handling for Update GAISF API Key
				if i.TitleText == "🔑 Update GAISF API Key" {
					m.Choice = i.TitleText
					m.CurrentView = GAISFConfigView
					m.GAISFConfig = NewGAISFConfig()
					return m, nil
				}

				m.Choice = i.TitleText
				m.Operation = "Executing: " + i.TitleText
				m.Result = ""
				m.IsError = false
				m.CurrentView = OperationView
				return m, m.executeOperation(i.Action, i.IsFullInstall)
			}
		}
	}

	var cmd tea.Cmd
	m.List, cmd = m.List.Update(msg)
	return m, cmd
}

func (m Model) updateGAISFConfig(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.GAISFList.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "q":
			m.Quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.GAISFList.SelectedItem().(Item)
			if ok {
				switch i.TitleText {
				case "🔑 Auto-configure GAISF token":
					m.GAISFConfig.AutoLogin = true
					m.GAISFConfig.Stage = "username"
					m.InputPrompt = "Enter username:"
					m.InputType = "username"
					m.TextInput.Placeholder = "username"
					m.TextInput.SetValue("")
					m.TextInput.EchoMode = textinput.EchoNormal
					m.TextInput.Focus()
					m.CurrentView = InputView
					return m, nil

				case "📝 Manual token input":
					m.GAISFConfig.AutoLogin = false
					m.GAISFConfig.Stage = "token"
					m.InputPrompt = "Enter your GAISF token:"
					m.InputType = "token"
					m.TextInput.Placeholder = "GAISF token"
					m.TextInput.SetValue("")
					m.TextInput.EchoMode = textinput.EchoPassword
					m.TextInput.EchoCharacter = '•'
					m.TextInput.Focus()
					m.CurrentView = InputView
					return m, nil

				case "⏭️  Skip GAISF configuration":
					m.GAISFConfig.Stage = "complete"
					// Execute the actual update after skipping GAISF config
					if m.Choice == "🔑 Update GAISF API Key" {
						m.CurrentView = OperationView
						m.Operation = "Executing: " + m.Choice
						return m, m.executeGAISFUpdate("")
					}
					// For other cases, just go back to main menu
					m.CurrentView = MainMenuView
					return m, nil
				}
			}

		case "esc":
			m.CurrentView = MainMenuView
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.GAISFList, cmd = m.GAISFList.Update(msg)
	return m, cmd
}

func (m Model) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c":
			m.Quitting = true
			return m, tea.Quit

		case "enter":
			value := m.TextInput.Value()
			m.TextInput.SetValue("")

			switch m.GAISFConfig.Stage {
			case "username":
				m.GAISFConfig.Username = value
				m.GAISFConfig.Stage = "password"
				m.InputPrompt = "Enter password:"
				m.InputType = "password"
				m.TextInput.Placeholder = "password"
				m.TextInput.EchoMode = textinput.EchoPassword
				m.TextInput.EchoCharacter = '•'
				m.TextInput.Focus()
				return m, nil

			case "password":
				m.GAISFConfig.Password = value
				m.GAISFConfig.Stage = "processing"
				m.TextInput.EchoMode = textinput.EchoNormal
				m.CurrentView = OperationView
				m.Operation = "🔐 Authenticating with GAISF..."
				return m, m.processGaisfAuth()

			case "token":
				m.GAISFConfig.Token = value
				m.GAISFConfig.Stage = "complete"
				m.TextInput.EchoMode = textinput.EchoNormal
				// Now execute the GAISF update with the token
				m.CurrentView = OperationView
				m.Operation = "Updating GAISF configuration..."
				return m, m.executeGAISFUpdate(value)
			}

		case "esc":
			m.CurrentView = GAISFConfigView
			m.GAISFConfig.Stage = "choice"
			m.TextInput.EchoMode = textinput.EchoNormal
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.TextInput, cmd = m.TextInput.Update(msg)
	return m, cmd
}

func (m Model) updateOperation(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "q":
			m.Quitting = true
			return m, tea.Quit

		case "enter", "esc":
			m.CurrentView = MainMenuView
			return m, nil
		}

	case OperationResult:
		m.Result = msg.Message
		m.IsError = msg.IsError

		// Auto-switch to GAISF configuration if this was a full install
		if msg.AutoSwitchToGAISF && !msg.IsError {
			m.CurrentView = GAISFConfigView
			m.GAISFConfig = NewGAISFConfig()
			return m, nil
		}

		return m, nil
	}
	return m, nil
}

// View function for the TUI
func (m Model) View() string {
	if m.Quitting {
		return QuitTextStyle.Render("👋 Thank you for using Claude Code Installer!\n")
	}

	switch m.CurrentView {
	case MainMenuView:
		return "\n" + m.List.View()

	case GAISFConfigView:
		if m.GAISFConfig.Stage == "choice" {
			return "\n" + m.GAISFList.View()
		} else if m.GAISFConfig.Stage == "processing" {
			return fmt.Sprintf(
				"\n%s\n\n%s\n",
				HeaderStyle.Render("🔑 GAISF API Authentication Setup"),
				"🔐 Authenticating with GAISF...\n\n⏳ Please wait...",
			)
		} else {
			return fmt.Sprintf(
				"\n%s\n\n%s\n",
				HeaderStyle.Render("🔑 GAISF API Authentication Setup"),
				"Processing GAISF configuration...",
			)
		}

	case InputView:
		var promptText string
		switch m.GAISFConfig.Stage {
		case "username":
			promptText = "Enter your username:"
		case "password":
			promptText = "Enter your password (hidden):"
		case "token":
			promptText = "Enter your GAISF token (hidden):"
		default:
			promptText = m.InputPrompt
		}

		return fmt.Sprintf(
			"\n%s\n\n%s\n%s\n\n%s\n",
			HeaderStyle.Render("📝 Input Required"),
			PromptStyle.Render(promptText),
			InputStyle.Render(m.TextInput.View()),
			"Press Enter to confirm, Esc to go back",
		)

	case OperationView:
		statusMsg := m.Operation
		if m.Result != "" {
			if m.IsError {
				statusMsg += "\n\n" + ErrorStyle.Render(m.Result)
			} else {
				statusMsg += "\n\n" + SuccessStyle.Render(m.Result)
			}
			statusMsg += "\n\nPress Enter to return to main menu..."
		} else {
			statusMsg += "\n\n⏳ Processing..."
		}
		return fmt.Sprintf(
			"\n%s\n\n%s\n",
			HeaderStyle.Render("🔄 Operation in Progress"),
			statusMsg,
		)

	default:
		return ""
	}
}

// Command to execute operations
func (m Model) executeOperation(action func() error, isFullInstall bool) tea.Cmd {
	return func() tea.Msg {
		err := action()
		if err != nil {
			return OperationResult{
				Message:           fmt.Sprintf("❌ Error: %v", err),
				IsError:           true,
				AutoSwitchToGAISF: false,
			}
		}

		return OperationResult{
			Message:           "✅ Operation completed successfully!",
			IsError:           false,
			AutoSwitchToGAISF: isFullInstall,
		}
	}
}

// Execute GAISF configuration update
func (m Model) executeGAISFUpdate(token string) tea.Cmd {
	return func() tea.Msg {
		// Update settings with the new token
		if err := config.UpdateClaudeCodeSettings(token); err != nil {
			return OperationResult{
				Message: fmt.Sprintf("❌ Failed to update settings: %v", err),
				IsError: true,
			}
		}

		return OperationResult{
			Message: "✅ GAISF API Key updated successfully!",
			IsError: false,
		}
	}
}

// Process GAISF authentication
func (m Model) processGaisfAuth() tea.Cmd {
	return func() tea.Msg {
		if m.GAISFConfig.AutoLogin {
			token, err := auth.GetGAISFToken(m.GAISFConfig.Username, m.GAISFConfig.Password)
			if err != nil {
				return OperationResult{
					Message: fmt.Sprintf("❌ Failed to get GAISF token: %v", err),
					IsError: true,
				}
			}
			if updateErr := config.UpdateClaudeCodeSettings(token); updateErr != nil {
				return OperationResult{
					Message: fmt.Sprintf("❌ Failed to update settings: %v", updateErr),
					IsError: true,
				}
			}
		}

		return OperationResult{
			Message: "✅ GAISF authentication and configuration updated successfully!",
			IsError: false,
		}
	}
}

// GAISF Config Model Update functions
func (m *GAISFConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case GAISFAuthResult:
		if msg.Error != nil {
			m.Error = fmt.Sprintf("Authentication failed: %v", msg.Error)
			m.Config.Stage = "choice"
			m.TextInput.EchoMode = textinput.EchoNormal
			m.TextInput.SetValue("")
		} else {
			m.Result.Token = msg.Token
			m.Quitting = true
			return m, tea.Quit
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.Quitting = true
			m.Result.Skipped = true
			return m, tea.Quit

		case "1":
			if m.Config.Stage == "choice" {
				m.Config.AutoLogin = true
				m.Config.Stage = "username"
				m.TextInput.Placeholder = "Enter username"
				m.TextInput.SetValue("")
			}
			return m, nil

		case "2":
			if m.Config.Stage == "choice" {
				m.Config.AutoLogin = false
				m.Config.Stage = "token"
				m.TextInput.Placeholder = "Enter GAISF token"
				m.TextInput.EchoMode = textinput.EchoPassword
				m.TextInput.EchoCharacter = '•'
				m.TextInput.SetValue("")
			}
			return m, nil

		case "3":
			if m.Config.Stage == "choice" {
				m.Result.Skipped = true
				m.Quitting = true
				return m, tea.Quit
			}

		case "enter":
			if m.Config.Stage == "choice" {
				m.Result.Skipped = true
				m.Quitting = true
				return m, tea.Quit
			}
			return m.handleEnter()

		case "esc":
			if m.Config.Stage != "choice" {
				m.Config.Stage = "choice"
				m.TextInput.EchoMode = textinput.EchoNormal
				m.TextInput.SetValue("")
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.TextInput, cmd = m.TextInput.Update(msg)
	return m, cmd
}

func (m *GAISFConfigModel) handleEnter() (tea.Model, tea.Cmd) {
	value := strings.TrimSpace(m.TextInput.Value())

	switch m.Config.Stage {
	case "username":
		if value == "" {
			m.Error = "Username cannot be empty"
			return m, nil
		}
		m.Config.Username = value
		m.Config.Stage = "password"
		m.TextInput.Placeholder = "Enter password"
		m.TextInput.EchoMode = textinput.EchoPassword
		m.TextInput.EchoCharacter = '•'
		m.TextInput.SetValue("")
		m.Error = ""
		return m, nil

	case "password":
		if value == "" {
			m.Error = "Password cannot be empty"
			return m, nil
		}
		m.Config.Password = value
		return m, m.authenticateGAISF()

	case "token":
		if value == "" {
			m.Error = "Token cannot be empty"
			return m, nil
		}
		m.Result.Token = value
		m.Quitting = true
		return m, tea.Quit
	}

	return m, nil
}

func (m *GAISFConfigModel) authenticateGAISF() tea.Cmd {
	return func() tea.Msg {
		token, err := auth.GetGAISFToken(m.Config.Username, m.Config.Password)
		if err != nil {
			return GAISFAuthResult{Error: err}
		}
		return GAISFAuthResult{Token: token}
	}
}

func (m *GAISFConfigModel) View() string {
	if m.Quitting {
		if m.Result.Skipped {
			return "⏭️  Skipping GAISF configuration...\n"
		}
		return "✅ GAISF configuration completed!\n"
	}

	var content strings.Builder
	content.WriteString(HeaderStyle.Render("🔑 GAISF API Authentication Setup"))
	content.WriteString("\n\n")

	switch m.Config.Stage {
	case "choice":
		content.WriteString("Configure GAISF token for API authentication?\n\n")
		content.WriteString("1. 🔑 Auto-configure GAISF token (Login with username/password)\n")
		content.WriteString("2. 📝 Manual token input (Enter GAISF token manually)\n")
		content.WriteString("3. ⏭️  Skip GAISF configuration (Continue without authentication)\n\n")
		content.WriteString(PromptStyle.Render("Please select an option (1-3):"))

	case "username":
		content.WriteString("Enter your username:\n\n")
		content.WriteString(InputStyle.Render(m.TextInput.View()))
		content.WriteString("\n\nPress Enter to continue, Esc to go back")

	case "password":
		content.WriteString("Enter your password (hidden):\n\n")
		content.WriteString(InputStyle.Render(m.TextInput.View()))
		content.WriteString("\n\nPress Enter to authenticate, Esc to go back")

	case "token":
		content.WriteString("Enter your GAISF token (hidden):\n\n")
		content.WriteString(InputStyle.Render(m.TextInput.View()))
		content.WriteString("\n\nPress Enter to continue, Esc to go back")

	case "processing":
		content.WriteString("🔐 Authenticating with GAISF...\n\n⏳ Please wait...")
	}

	if m.Error != "" {
		content.WriteString("\n\n")
		content.WriteString(ErrorStyle.Render("❌ " + m.Error))
	}

	return content.String()
}
