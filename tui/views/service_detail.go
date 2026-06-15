package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"edge-cli/client"
	"edge-cli/models"
)

type detailTab int

const (
	tabInfo detailTab = iota
	tabCode
	tabLogs
)

var tabs = []string{"Info", "Code", "Logs"}

type ServiceDetailModel struct {
	client  *client.Client
	svc     models.DBCodeMeta
	tab     detailTab
	done    bool
	width   int
	height  int

	// info
	running map[string]models.RunningServiceInfo

	// code
	codeViewport viewport.Model
	codeLoaded   bool
	codeSpinner  spinner.Model

	// logs
	logsViewport  viewport.Model
	logLines      []string
	latestLogTime int64
	logSpinner    spinner.Model
	logLoading    bool
}

type codeLoadedMsg struct{ code string }
type logsLoadedMsg struct {
	entries []models.LogEntry
	initial bool
}
type runningLoadedMsg struct{ running map[string]models.RunningServiceInfo }
type logTickMsg struct{}

func NewServiceDetailModel(c *client.Client, svc models.DBCodeMeta, w, h int) ServiceDetailModel {
	cs := spinner.New()
	cs.Spinner = spinner.Dot
	cs.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ls := spinner.New()
	ls.Spinner = spinner.Dot
	ls.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	vpH := viewportHeight(h)
	cv := viewport.New(w, vpH)
	lv := viewport.New(w, vpH)

	return ServiceDetailModel{
		client:       c,
		svc:          svc,
		tab:          tabInfo,
		width:        w,
		height:       h,
		codeViewport: cv,
		logsViewport: lv,
		codeSpinner:  cs,
		logSpinner:   ls,
	}
}

func (m ServiceDetailModel) Init() tea.Cmd {
	return fetchRunning(m.client)
}

func (m ServiceDetailModel) Update(msg tea.Msg) (ServiceDetailModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "backspace":
			m.done = true
			return m, nil
		case "tab":
			m.tab = (m.tab + 1) % detailTab(len(tabs))
			return m, m.onTabSwitch()
		case "1":
			m.tab = tabInfo
		case "2":
			m.tab = tabCode
			return m, m.onTabSwitch()
		case "3":
			m.tab = tabLogs
			return m, m.onTabSwitch()
		}

	case runningLoadedMsg:
		m.running = msg.running

	case codeLoadedMsg:
		m.codeLoaded = true
		m.codeViewport.SetContent(numberLines(msg.code))
		m.codeViewport.GotoTop()

	case logsLoadedMsg:
		if msg.initial {
			m.logLoading = false
		}
		for _, e := range msg.entries {
			ts := time.UnixMicro(e.Time).Format("15:04:05")
			level := e.Level
			if level == "" {
				level = "info"
			}
			m.logLines = append(m.logLines, fmt.Sprintf("[%s] %s: %s", ts, level, e.Log))
			if e.Time > m.latestLogTime {
				m.latestLogTime = e.Time
			}
		}
		m.logsViewport.SetContent(strings.Join(m.logLines, "\n"))
		m.logsViewport.GotoBottom()
		cmds = append(cmds, scheduleLogPoll())

	case logTickMsg:
		cmds = append(cmds, fetchLogs(m.client, m.svc.Name, m.latestLogTime))

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.codeSpinner, cmd = m.codeSpinner.Update(msg)
		cmds = append(cmds, cmd)
		m.logSpinner, cmd = m.logSpinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Forward scroll events to the active viewport.
	switch m.tab {
	case tabCode:
		var cmd tea.Cmd
		m.codeViewport, cmd = m.codeViewport.Update(msg)
		cmds = append(cmds, cmd)
	case tabLogs:
		var cmd tea.Cmd
		m.logsViewport, cmd = m.logsViewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *ServiceDetailModel) onTabSwitch() tea.Cmd {
	switch m.tab {
	case tabCode:
		if !m.codeLoaded {
			return tea.Batch(m.codeSpinner.Tick, fetchCode(m.client, m.svc.Name))
		}
	case tabLogs:
		if len(m.logLines) == 0 {
			m.logLoading = true
			return tea.Batch(m.logSpinner.Tick, fetchLogs(m.client, m.svc.Name, 0))
		}
	}
	return nil
}

func (m ServiceDetailModel) View() string {
	header := m.renderHeader()
	tabBar := m.renderTabBar()
	body := m.renderBody()
	help := helpBar("esc back", "tab next tab", "↑↓ scroll")
	return lipgloss.JoinVertical(lipgloss.Left, header, tabBar, body, help)
}

func (m ServiceDetailModel) renderHeader() string {
	style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Padding(0, 1)
	return style.Render(m.svc.Name)
}

func (m ServiceDetailModel) renderTabBar() string {
	active := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Underline(true).
		Padding(0, 2)
	inactive := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 2)

	var parts []string
	for i, t := range tabs {
		if detailTab(i) == m.tab {
			parts = append(parts, active.Render(t))
		} else {
			parts = append(parts, inactive.Render(t))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func (m ServiceDetailModel) renderBody() string {
	switch m.tab {
	case tabInfo:
		return m.renderInfo()
	case tabCode:
		if !m.codeLoaded {
			return fmt.Sprintf("\n  %s loading code…\n", m.codeSpinner.View())
		}
		return m.codeViewport.View()
	case tabLogs:
		if m.logLoading {
			return fmt.Sprintf("\n  %s loading logs…\n", m.logSpinner.View())
		}
		if len(m.logLines) == 0 {
			return "\n  (no logs)\n"
		}
		return m.logsViewport.View()
	}
	return ""
}

func (m ServiceDetailModel) renderInfo() string {
	b := &strings.Builder{}
	fmt.Fprintf(b, "  Engine:      %s\n", engineLabel(m.svc.EngineType))
	fmt.Fprintf(b, "  Logging:     %v (level: %s)\n", m.svc.LoggingEnabled, m.svc.LogLevel)
	fmt.Fprintf(b, "  Auto-scale:  %v\n", m.svc.AutoScale)
	fmt.Fprintf(b, "  Run on edge: %v\n", m.svc.RunOnEdge)
	fmt.Fprintln(b)

	var instances []models.RunningServiceInfo
	for _, info := range m.running {
		if info.CodeName == m.svc.Name {
			instances = append(instances, info)
		}
	}
	fmt.Fprintf(b, "  Running instances: %d\n", len(instances))
	for _, inst := range instances {
		started := time.Unix(0, inst.Started).Format("2006-01-02 15:04:05")
		fmt.Fprintf(b, "    started %s  heap %s\n", started, formatBytesLocal(inst.HeapStatistics.TotalBytesAllocated()))
	}
	return b.String()
}

func (m *ServiceDetailModel) resize(w, h int) {
	m.width = w
	m.height = h
	vpH := viewportHeight(h)
	m.codeViewport.Width = w
	m.codeViewport.Height = vpH
	m.logsViewport.Width = w
	m.logsViewport.Height = vpH
}

func viewportHeight(total int) int {
	if total < 8 {
		return 4
	}
	return total - 6 // header + tabbar + helpbar
}

func fetchRunning(c *client.Client) tea.Cmd {
	return func() tea.Msg {
		r, err := c.ListRunning()
		if err != nil {
			return errMsg{err}
		}
		return runningLoadedMsg{r}
	}
}

func fetchCode(c *client.Client, name string) tea.Cmd {
	return func() tea.Msg {
		code, err := c.GetServiceCode(name)
		if err != nil {
			return errMsg{err}
		}
		return codeLoadedMsg{code}
	}
}

func fetchLogs(c *client.Client, name string, afterMicros int64) tea.Cmd {
	return func() tea.Msg {
		entries, err := c.GetLogsV4(client.LogQueryOpts{
			ServiceName:     name,
			AfterTimeMicros: afterMicros,
			Limit:           100,
		})
		if err != nil {
			return errMsg{err}
		}
		return logsLoadedMsg{entries: entries, initial: afterMicros == 0}
	}
}

func scheduleLogPoll() tea.Cmd {
	return tea.Tick(3*time.Second, func(_ time.Time) tea.Msg {
		return logTickMsg{}
	})
}

func numberLines(code string) string {
	lines := strings.Split(code, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	width := len(fmt.Sprintf("%d", len(lines)))
	numStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	var b strings.Builder
	for i, line := range lines {
		b.WriteString(numStyle.Render(fmt.Sprintf("%*d  ", width, i+1)))
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func formatBytesLocal(b uint64) string {
	switch {
	case b >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(b)/1024/1024)
	case b >= 1024:
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	default:
		return fmt.Sprintf("%d B", b)
	}
}
