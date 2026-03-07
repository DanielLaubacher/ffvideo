package ui

import "github.com/charmbracelet/lipgloss"

var (
	Title   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	Success = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	Warning = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	Error   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	Subtle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)
