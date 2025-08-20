package ui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// Styles for the TUI
var (
	TitleStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#7D56F4")).Padding(0, 1).Bold(true)
	HeaderStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true).MarginBottom(1)
	ItemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	SelectedItemStyle = lipgloss.NewStyle().PaddingLeft(4).Foreground(lipgloss.Color("#EE6FF8")).Bold(true)

	PaginationStyle = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	HelpStyle       = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	QuitTextStyle   = lipgloss.NewStyle().Margin(1, 0, 2, 4)

	SuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	ErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F87")).Bold(true)
	InputStyle   = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#7D56F4")).Padding(0, 1)
	PromptStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
)
