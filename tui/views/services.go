package views

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"edge-cli/client"
	"edge-cli/config"
	"edge-cli/models"
)

// LogoutMsg is returned when the user logs out — the caller should clear the
// cached edge token and re-enter the proxy flow.
type LogoutMsg struct{ EdgeName string }

type ServicesModel struct {
	client    *client.Client
	connLabel string
	edgeName  string // non-empty in proxy mode, used for logout
	state     servicesState
	list      list.Model
	spinner   spinner.Model
	detail    *ServiceDetailModel
	err       error
	width     int
	height    int
}

type servicesState int

const (
	stateLoading servicesState = iota
	stateList
	stateDetail
)

type serviceItem struct{ svc models.DBCodeMeta }

func (i serviceItem) Title() string       { return i.svc.Name }
func (i serviceItem) Description() string { return "" }
func (i serviceItem) FilterValue() string { return i.svc.Name }

type servicesLoadedMsg struct{ services []models.DBCodeMeta }
type errMsg struct{ err error }

// compactDelegate renders each service as a single line:
//   ▶ serviceName        duk  conc:1  log:debug
type compactDelegate struct{}

func (d compactDelegate) Height() int                               { return 1 }
func (d compactDelegate) Spacing() int                             { return 0 }
func (d compactDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d compactDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(serviceItem)
	if !ok {
		return
	}

	svc := i.svc
	name := svc.Name
	meta := fmt.Sprintf("%s  conc:%-2d  log:%s",
		engineLabel(svc.EngineType),
		svc.Concurrency,
		logLevelLabel(svc.LoggingEnabled, svc.LogLevel),
	)

	selected := index == m.Index()

	nameStyle := lipgloss.NewStyle().Width(36)
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	if selected {
		nameStyle = nameStyle.Foreground(lipgloss.Color("86")).Bold(true)
		metaStyle = metaStyle.Foreground(lipgloss.Color("243"))
		fmt.Fprintf(w, "  %s  %s",
			nameStyle.Render("▶ "+name),
			metaStyle.Render(meta),
		)
	} else {
		nameStyle = nameStyle.Foreground(lipgloss.Color("252"))
		fmt.Fprintf(w, "  %s  %s",
			nameStyle.Render("  "+name),
			metaStyle.Render(meta),
		)
	}
}

func NewServicesModel(c *client.Client) ServicesModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	connLabel := c.ConnectionLabel()
	l := list.New(nil, compactDelegate{}, 0, 0)
	l.Title = "edge-cli  [" + connLabel + "]"
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Padding(0, 1)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(true)

	return ServicesModel{
		client:    c,
		connLabel: connLabel,
		edgeName:  c.EdgeName(),
		state:     stateLoading,
		list:      l,
		spinner:   s,
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
		case "L":
			if m.edgeName != "" {
				_ = config.ClearEdgeToken(m.edgeName)
				return m, tea.Quit
			}
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
	body := m.list.View()
	if m.edgeName != "" {
		body += "\n" + helpStyle.Render("  L logout from edge")
	}
	return body
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
		return "unk"
	}
}

func logLevelLabel(enabled bool, level string) string {
	if !enabled {
		return "off"
	}
	if level == "" {
		return "on"
	}
	return level
}

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

func helpBar(bindings ...string) string {
	return helpStyle.Render(strings.Join(bindings, "  ·  "))
}
