package views

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type AuthSpinnerModel struct {
	spinner spinner.Model
	label   string
	token   string
	err     error
	authCmd tea.Cmd
}

type AuthDoneMsg struct {
	Token string
	Err   error
}

func NewAuthSpinnerModel(label string, authCmd tea.Cmd) AuthSpinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return AuthSpinnerModel{spinner: s, label: label, authCmd: authCmd}
}

func (m AuthSpinnerModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.authCmd)
}

func (m AuthSpinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case AuthDoneMsg:
		m.token = msg.Token
		m.err = msg.Err
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m AuthSpinnerModel) View() string {
	return fmt.Sprintf("\n  %s %s\n", m.spinner.View(), m.label)
}

func (m AuthSpinnerModel) Token() string { return m.token }
func (m AuthSpinnerModel) Err() error    { return m.err }
