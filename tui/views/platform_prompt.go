package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"edge-cli/client"
)

type promptField int

const (
	fieldPlatformURL promptField = iota
	fieldEmail
	fieldPassword
	fieldSystemKey
	fieldCount
)

type PlatformPromptModel struct {
	inputs    [fieldCount]textinput.Model
	focused   promptField
	cancelled bool
	token     string // set after successful auth
	err       string
}

func NewPlatformPromptModel(prefillURL, prefillSystemKey string) PlatformPromptModel {
	placeholders := []string{"https://platform.clearblade.com", "you@example.com", "", "abc123..."}
	prefills := []string{prefillURL, "", "", prefillSystemKey}

	var inputs [fieldCount]textinput.Model
	for i := range inputs {
		t := textinput.New()
		t.Placeholder = placeholders[i]
		t.CharLimit = 256
		t.SetValue(prefills[i])
		if promptField(i) == fieldPassword {
			t.EchoMode = textinput.EchoPassword
			t.EchoCharacter = '•'
		}
		inputs[i] = t
	}
	inputs[fieldPlatformURL].Focus()

	return PlatformPromptModel{inputs: inputs}
}

// NewPlatformPromptModelEmpty shows a fully empty prompt with no pre-fills.
func NewPlatformPromptModelEmpty() PlatformPromptModel {
	return NewPlatformPromptModel("", "")
}

func (m PlatformPromptModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m PlatformPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "tab", "down", "enter":
			if msg.String() == "enter" && m.focused == fieldCount-1 {
				return m, m.submit()
			}
			m.inputs[m.focused].Blur()
			m.focused = (m.focused + 1) % fieldCount
			m.inputs[m.focused].Focus()
			return m, textinput.Blink
		case "shift+tab", "up":
			m.inputs[m.focused].Blur()
			if m.focused == 0 {
				m.focused = fieldCount - 1
			} else {
				m.focused--
			}
			m.inputs[m.focused].Focus()
			return m, textinput.Blink
		}
	case authResultMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.token = msg.token
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	return m, cmd
}

func (m PlatformPromptModel) View() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render("Connect via Platform Proxy")
	labels := []string{"Platform URL", "Email", "Password", "System Key"}

	var b strings.Builder
	fmt.Fprintln(&b, title)
	fmt.Fprintln(&b)
	for i, input := range m.inputs {
		label := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(labels[i])
		fmt.Fprintf(&b, "  %s\n  %s\n\n", label, input.View())
	}
	if m.err != "" {
		fmt.Fprintf(&b, "  %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("✗ "+m.err))
	}
	fmt.Fprintln(&b, helpBar("tab next field", "enter confirm", "esc cancel"))
	return b.String()
}

func (m *PlatformPromptModel) submit() tea.Cmd {
	platformURL := strings.TrimRight(m.inputs[fieldPlatformURL].Value(), "/")
	email := m.inputs[fieldEmail].Value()
	password := m.inputs[fieldPassword].Value()
	return func() tea.Msg {
		token, err := client.Authenticate(platformURL, email, password)
		return authResultMsg{token: token, err: err}
	}
}

type authResultMsg struct {
	token string
	err   error
}

func (m PlatformPromptModel) Cancelled() bool    { return m.cancelled }
func (m PlatformPromptModel) PlatformURL() string { return strings.TrimRight(m.inputs[fieldPlatformURL].Value(), "/") }
func (m PlatformPromptModel) SystemKey() string   { return m.inputs[fieldSystemKey].Value() }
func (m PlatformPromptModel) Token() string       { return m.token }
