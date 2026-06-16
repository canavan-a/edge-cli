package views

import (
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"edge-cli/client"
	"edge-cli/config"
	"edge-cli/models"
)

type EdgePickerModel struct {
	client        *client.Client
	list          list.Model
	spinner       spinner.Model
	loading       bool
	cancelled     bool
	selected      string
	selectedToken string
}

type edgeItem struct{ edge models.EdgeInfo }

func (i edgeItem) Title() string {
	if i.edge.IsConnected {
		return i.edge.Name
	}
	return i.edge.Name
}
func (i edgeItem) Description() string { return "" }
func (i edgeItem) FilterValue() string { return i.edge.Name }

type edgesLoadedMsg struct{ edges []models.EdgeInfo }

// edgePickerDelegate renders each edge as a single compact line with connected status.
type edgePickerDelegate struct{}

func (d edgePickerDelegate) Height() int                               { return 1 }
func (d edgePickerDelegate) Spacing() int                             { return 0 }
func (d edgePickerDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d edgePickerDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(edgeItem)
	if !ok {
		return
	}

	selected := index == m.Index()

	var dot string
	var dotStyle lipgloss.Style
	if i.edge.IsConnected {
		dotStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
		dot = "●"
	} else {
		dotStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		dot = "○"
	}

	nameStyle := lipgloss.NewStyle().Width(36)
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	var meta string
	if i.edge.IsConnected {
		meta = i.edge.LastSeenVersion
		if i.edge.PublicAddr != "" {
			meta += "  " + i.edge.PublicAddr
		}
	} else if i.edge.LastDisconnect > 0 {
		t := time.Unix(0, i.edge.LastDisconnect*int64(time.Millisecond))
		meta = "last seen " + t.Format("2006-01-02 15:04")
	}

	prefix := "  "
	if selected {
		nameStyle = nameStyle.Foreground(lipgloss.Color("86")).Bold(true)
		prefix = "▶ "
	} else {
		nameStyle = nameStyle.Foreground(lipgloss.Color("252"))
	}

	fmt.Fprintf(w, "  %s %s  %s",
		dotStyle.Render(dot),
		nameStyle.Render(prefix+i.edge.Name),
		metaStyle.Render(meta),
	)
}

func NewEdgePickerModel(c *client.Client) EdgePickerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	l := list.New(nil, edgePickerDelegate{}, 0, 0)
	l.Title = "Select an edge"
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Padding(0, 1)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(true)

	return EdgePickerModel{
		client:  c,
		list:    l,
		spinner: s,
		loading: true,
	}
}

func (m EdgePickerModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchEdges(m.client))
}

func (m EdgePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "L":
			// Clear all cached proxy state so next launch goes through full flow.
			_ = config.SaveProxyConfig("", "")
			if saved := config.EdgeName(); saved != "" {
				_ = config.ClearEdgeToken(saved)
			}
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			if !m.loading {
				if i, ok := m.list.SelectedItem().(edgeItem); ok {
					m.selected = i.edge.Name
					m.selectedToken = i.edge.Token
					return m, tea.Quit
				}
			}
		}

	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height)

	case edgesLoadedMsg:
		m.loading = false
		// connected first, then disconnected, alphabetical within each group
		connected := []list.Item{}
		disconnected := []list.Item{}
		for _, e := range msg.edges {
			if e.IsConnected {
				connected = append(connected, edgeItem{e})
			} else {
				disconnected = append(disconnected, edgeItem{e})
			}
		}
		m.list.SetItems(append(connected, disconnected...))
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m EdgePickerModel) View() string {
	if m.loading {
		return fmt.Sprintf("\n  %s fetching edges…\n", m.spinner.View())
	}
	return m.list.View() + "\n" + helpBar("enter select", "L logout & clear saved proxy", "esc cancel")
}

func fetchEdges(c *client.Client) tea.Cmd {
	return func() tea.Msg {
		edges, err := c.ListEdges()
		if err != nil {
			return errMsg{err}
		}
		return edgesLoadedMsg{edges}
	}
}

func (m EdgePickerModel) Cancelled() bool           { return m.cancelled }
func (m EdgePickerModel) SelectedEdge() string      { return m.selected }
func (m EdgePickerModel) SelectedEdgeToken() string { return m.selectedToken }
