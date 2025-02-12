package ui

import (
	"fmt"
	"net/url"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type (
	errMsg error
)

type WebhookModel struct {
	localUrl textinput.Model
	quit     bool
	done     bool
	err      error
}

// Implement TerminalModel methods for WelcomeModel
func (m WebhookModel) Init() tea.Cmd {
	tea.SetWindowTitle("Localhook - Webhook Tunnel")
	return textinput.Blink
}

func (m WebhookModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.quit = true
			return m, tea.Quit

		case tea.KeyEnter:
			_, err := url.ParseRequestURI(m.localUrl.Value())
			if err != nil {
				m.localUrl.SetValue("")
				m.localUrl.Placeholder = "enter a valid URL"
			}
			m.done = true
			return m, tea.Quit

		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	m.localUrl, cmd = m.localUrl.Update(msg)
	return m, cmd
}

func (m WebhookModel) View() string {
	if m.done {
		return fmt.Sprintf("%s\n\n", logo)
	}

	return fmt.Sprintf(
		"%s\n\nEnter your Local URL:\n\n%s\n\n%s",
		logo,
		m.localUrl.View(),
		"(esc to quit)",
	) + "\n"
}

func (m WebhookModel) GetLocalURL() (string, string, string) {
	// Assuming m.localUrl exists and contains the full URL
	u, err := url.Parse(m.localUrl.Value())
	if err != nil {
		return "", "", ""
	}

	host := u.Hostname()
	port := u.Port()
	path := u.Path

	return host, port, path
}

// Implement a struct that satisfies Terminal
type WebhookTerminal struct {
	model TerminalModel
}

// Ensure WlcmTerminal satisfies Terminal interface
func (t *WebhookTerminal) Init() tea.Cmd {
	return t.model.Init()
}

func (t *WebhookTerminal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return t.model.Update(msg)
}

func (t *WebhookTerminal) View() string {
	return t.model.View()
}

func (m WebhookModel) CloseConn() bool {
	return m.quit
}

func InitWebhookTerminal() *WebhookModel {
	ti := textinput.New()
	ti.Placeholder = "http://localhost:3000/payment/webhook (press any key to start)"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 150

	return &WebhookModel{
		localUrl: ti,
		err:      nil,
	}
}
