package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type WelcomeModel struct {
	choices  []string
	cursor   int
	selected map[int]struct{}
	choice   int
	done     bool
}

// Implement TerminalModel methods for WelcomeModel
func (m WelcomeModel) Init() tea.Cmd {
	return tea.SetWindowTitle("Localhook - Make localhost webhook Public")
}

func (m WelcomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			m.choice = m.cursor + 1
			m.done = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m WelcomeModel) View() string {
	if m.done {
		return fmt.Sprintf("%s\n\nYou selected: %d\n", logo, m.choice)
	}

	s := fmt.Sprintf("%s\n\nChoose an option:\n\n", logo)
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

func (m WelcomeModel) GetChoice() int {
	return m.choice
}

// Implement a struct that satisfies Terminal
type WlcmTerminal struct {
	model TerminalModel
}

// Ensure WlcmTerminal satisfies Terminal interface
func (t *WlcmTerminal) Init() tea.Cmd {
	return t.model.Init()
}

func (t *WlcmTerminal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return t.model.Update(msg)
}

func (t *WlcmTerminal) View() string {
	return t.model.View()
}

func InitWlcmTerminal() *WlcmTerminal {
	return &WlcmTerminal{
		model: WelcomeModel{
			choices:  []string{"Tunell Web Hook", "Tunnel Local Site", "Donate"},
			selected: make(map[int]struct{}),
		},
	}
}

// func (t *WlcmTerminal) GetChoice() int {
// 	return t.model.GetChoice()
// }

var logo = `

_                 _ _                 _    
| |               | | |               | |   
| | ___   ___ __ _| | |__   ___   ___ | | __
| |/ _ \ / __/ _' | | '_ \ / _ \ / _ \| |/ /
| | (_) | (_| (_| | | | | | (_) | (_) |   < 
|_|\___/ \___\__,_|_|_| |_|\___/ \___/|_|\_\                                            
                            by Uraan Studios        
                                              											
`
