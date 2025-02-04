package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// TerminalModel holds the functions and the Model Itself
type TerminalModel interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (tea.Model, tea.Cmd)
	View() string
}

// Terminal Interface is the wrapper to Model with all state function
type Terminal interface {
	TerminalModel
	InitialModel() TerminalModel
}
