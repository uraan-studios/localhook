package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	activeItemStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("blue")).Bold(true)
	selectedItemStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("green")).Bold(true)
	inactiveItemStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("white"))
)

type Model struct {
	choices  []string         // items on the to-do list
	cursor   int              // which to-do list item our cursor is pointing at
	selected map[int]struct{} // which to-do items are selected
	choice   int              // selected choice to return
	done     bool             // track if user has made a selection
}

func InitialModel() Model {
	return Model{
		choices:  []string{"New Web Hook", "Delete Hook", "Donate"},
		selected: make(map[int]struct{}),
		choice:   0,
		done:     false,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.SetWindowTitle("Localhook - Make localhost webhook Public")
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter":
			m.choice = m.cursor + 1 // Set the selected choice (1-based index)
			m.done = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m Model) View() string {
	if m.done {
		return fmt.Sprintf("You selected: %d\n", m.choice)
	}

	s := "Choose an option:\n\n"
	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		checked := " "
		if _, ok := m.selected[i]; ok {
			checked = "x"
			s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, selectedItemStyle.Render(choice))
		} else {
			s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, inactiveItemStyle.Render(choice))
		}
	}

	s += "\nPress Enter to confirm, q to quit.\n"
	return s
}

func (m Model) Choice() int {
	return m.choice
}
