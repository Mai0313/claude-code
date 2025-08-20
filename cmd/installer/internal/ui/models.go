package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// View types
type ViewType int

const (
	MainMenuView ViewType = iota
	GAISFConfigView
	InputView
	OperationView
)

// Menu item for list
type Item struct {
	TitleText, DescText string
	Action              func() error
	IsFullInstall       bool
}

func (i Item) FilterValue() string { return i.TitleText }
func (i Item) Title() string       { return i.TitleText }
func (i Item) Description() string { return i.DescText }

// Main model
type Model struct {
	List        list.Model
	GAISFList   list.Model
	TextInput   textinput.Model
	CurrentView ViewType
	Choice      string
	Quitting    bool
	Operation   string
	Result      string
	IsError     bool
	InputPrompt string
	InputType   string // "username", "password", "token"
	GAISFConfig *GAISFConfig
}

// GAISF configuration state
type GAISFConfig struct {
	Stage     string // "choice", "username", "password", "token", "processing", "complete"
	Username  string
	Password  string
	Token     string
	AutoLogin bool
}

func NewGAISFConfig() *GAISFConfig {
	return &GAISFConfig{
		Stage: "choice",
	}
}

// Custom item delegate for styling
type ItemDelegate struct{}

func (d ItemDelegate) Height() int                             { return 1 }
func (d ItemDelegate) Spacing() int                            { return 0 }
func (d ItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d ItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(Item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i.TitleText)

	fn := ItemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return SelectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

// Message types
type OperationResult struct {
	Message           string
	IsError           bool
	AutoSwitchToGAISF bool // New field to indicate auto-switch to GAISF
}

// GAISF authentication result
type GAISFAuthResult struct {
	Token string
	Error error
}

// GAISF configuration result
type GAISFResult struct {
	Token   string
	Skipped bool
}

// Dedicated GAISF configuration model
type GAISFConfigModel struct {
	TextInput textinput.Model
	Config    *GAISFConfig
	Result    *GAISFResult
	Quitting  bool
	Error     string
}

func NewGAISFConfigModel() *GAISFConfigModel {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 60

	return &GAISFConfigModel{
		TextInput: ti,
		Config:    NewGAISFConfig(),
		Result:    &GAISFResult{},
	}
}

func (m *GAISFConfigModel) Init() tea.Cmd {
	return textinput.Blink
}
