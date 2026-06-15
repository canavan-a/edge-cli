package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"edge-cli/client"
	"edge-cli/models"
)

// ServicesModel is the root screen — shows a list of services and lets you
// drill into logs or code for the selected one.
type ServicesModel struct {
	client  *client.Client
	state   servicesState
	list    list.Model
	spinner spinner.Model
	detail  *ServiceDetailModel
	err     error
	width   int
	height  int
}

type servicesState int

const (
	stateLoading servicesState = iota
	stateList
	stateDetail
)

// serviceItem wraps DBCodeMeta to satisfy list.Item.
type serviceItem struct{ svc models.DBCodeMeta }

func (i serviceItem) Title() string       { return i.svc.Name }
func (i serviceItem) Description() string { return engineLabel(i.svc.EngineType) }
func (i serviceItem) FilterValue() string { return i.svc.Name }

type servicesLoadedMsg struct{ services []models.DBCodeMeta }
type errMsg struct{ err error }

func NewServicesModel(c *client.Client) ServicesModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("86")).
		BorderLeftForeground(lipgloss.Color("86"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("243")).
		BorderLeftForeground(lipgloss.Color("86"))

	l := list.New(nil, delegate, 0, 0)
	l.Title = "edge-cli — services"
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Padding(0, 1)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)

	return ServicesModel{
		client:  c,
		state:   stateLoading,
		list:    l,
		spinner: s,
	}
}

func (m ServicesModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchServices(m.client))
}

func (m ServicesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.state == stateDetail {
			detail, cmd := m.detail.Update(msg)
			m.detail = &detail
			if m.detail.done {
				m.state = stateList
				m.detail = nil
			}
			return m, cmd
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			if m.state == stateList {
				if i, ok := m.list.SelectedItem().(serviceItem); ok {
					detail := NewServiceDetailModel(m.client, i.svc, m.width, m.height)
					m.detail = &detail
					m.state = stateDetail
					return m, m.detail.Init()
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height)
		if m.detail != nil {
			m.detail.resize(msg.Width, msg.Height)
		}

	case servicesLoadedMsg:
		items := make([]list.Item, len(msg.services))
		for i, svc := range msg.services {
			items[i] = serviceItem{svc}
		}
		m.list.SetItems(items)
		m.state = stateList
		return m, nil

	case errMsg:
		m.err = msg.err
		m.state = stateList
		return m, nil

	case spinner.TickMsg:
		if m.state == stateLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	if m.state == stateDetail && m.detail != nil {
		detail, cmd := m.detail.Update(msg)
		m.detail = &detail
		return m, cmd
	}

	if m.state == stateList {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m ServicesModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n  error: %s\n\n  press q to quit\n", m.err)
	}
	if m.state == stateLoading {
		return fmt.Sprintf("\n  %s loading services…\n", m.spinner.View())
	}
	if m.state == stateDetail && m.detail != nil {
		return m.detail.View()
	}
	return m.list.View()
}

func fetchServices(c *client.Client) tea.Cmd {
	return func() tea.Msg {
		svcs, err := c.ListServices()
		if err != nil {
			return errMsg{err}
		}
		return servicesLoadedMsg{svcs}
	}
}

func engineLabel(t int) string {
	switch t {
	case 0:
		return "duk"
	case 1:
		return "v8"
	default:
		return "unknown"
	}
}

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

func helpBar(bindings ...string) string {
	return helpStyle.Render(strings.Join(bindings, "  ·  "))
}
