package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type credField int

const (
	credEmail credField = iota
	credPassword
	credFieldCount
)

type EdgeCredPromptModel struct {
	edgeName  string
	inputs    [credFieldCount]textinput.Model
	focused   credField
	cancelled bool
}

func NewEdgeCredPromptModel(edgeName string) EdgeCredPromptModel {
	email := textinput.New()
	email.Placeholder = "you@example.com"
	email.CharLimit = 256
	email.Focus()

	pass := textinput.New()
	pass.Placeholder = "password"
	pass.CharLimit = 256
	pass.EchoMode = textinput.EchoPassword
	pass.EchoCharacter = '•'

	return EdgeCredPromptModel{
		edgeName: edgeName,
		inputs:   [credFieldCount]textinput.Model{email, pass},
	}
}

func (m EdgeCredPromptModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m EdgeCredPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "tab", "down":
			m.inputs[m.focused].Blur()
			m.focused = (m.focused + 1) % credFieldCount
			m.inputs[m.focused].Focus()
			return m, textinput.Blink
		case "shift+tab", "up":
			m.inputs[m.focused].Blur()
			if m.focused == 0 {
				m.focused = credFieldCount - 1
			} else {
				m.focused--
			}
			m.inputs[m.focused].Focus()
			return m, textinput.Blink
		case "enter":
			if m.focused == credPassword {
				return m, tea.Quit
			}
			m.inputs[m.focused].Blur()
			m.focused = credPassword
			m.inputs[m.focused].Focus()
			return m, textinput.Blink
		}
	}

	var cmd tea.Cmd
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	return m, cmd
}

func (m EdgeCredPromptModel) View() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).
		Render(fmt.Sprintf("Sign in to edge: %s", m.edgeName))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	var b strings.Builder
	fmt.Fprintln(&b, title)
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "  %s\n  %s\n\n", dim.Render("Email"), m.inputs[credEmail].View())
	fmt.Fprintf(&b, "  %s\n  %s\n\n", dim.Render("Password"), m.inputs[credPassword].View())
	fmt.Fprintln(&b, helpBar("tab next field", "enter confirm", "esc cancel"))
	return b.String()
}

func (m EdgeCredPromptModel) Cancelled() bool { return m.cancelled }
func (m EdgeCredPromptModel) Email() string    { return m.inputs[credEmail].Value() }
func (m EdgeCredPromptModel) Password() string { return m.inputs[credPassword].Value() }
