package views

import (
	"fmt"
	"sort"
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

// instanceEntry pairs an instance ID with its info for stable ordered display.
type instanceEntry struct {
	id   string
	info models.RunningServiceInfo
}

type ServiceDetailModel struct {
	client *client.Client
	svc    models.DBCodeMeta
	tab    detailTab
	done   bool
	width  int
	height int

	// info tab
	instances   []instanceEntry // ordered snapshot
	cursor      int             // which instance is selected
	confirm     string          // non-empty = show confirmation prompt ("stop-all" | "kill:<id>")
	statusMsg   string          // transient feedback line
	infoLoading bool

	// code tab
	codeViewport viewport.Model
	codeLoaded   bool
	codeSpinner  spinner.Model

	// logs tab
	logsViewport  viewport.Model
	logLines      []string
	latestLogTime int64
	logSpinner    spinner.Model
	logLoading    bool
}

// ── messages ────────────────────────────────────────────────────────────────

type codeLoadedMsg struct{ code string }
type logsLoadedMsg struct {
	entries []models.LogEntry
	initial bool
}
type runningLoadedMsg struct{ running map[string]models.RunningServiceInfo }
type logTickMsg struct{}
type infoTickMsg struct{}
type actionDoneMsg struct{ status string }

// ── constructor ─────────────────────────────────────────────────────────────

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
		client:      c,
		svc:         svc,
		tab:         tabInfo,
		width:       w,
		height:      h,
		codeViewport: cv,
		logsViewport: lv,
		codeSpinner:  cs,
		logSpinner:   ls,
		infoLoading:  true,
	}
}

func (m ServiceDetailModel) Init() tea.Cmd {
	return tea.Batch(fetchRunning(m.client), scheduleInfoPoll())
}

// ── update ───────────────────────────────────────────────────────────────────

func (m ServiceDetailModel) Update(msg tea.Msg) (ServiceDetailModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.KeyMsg:
		// confirmation prompt swallows all keys
		if m.confirm != "" {
			switch msg.String() {
			case "y", "Y":
				action := m.confirm
				m.confirm = ""
				cmds = append(cmds, m.executeAction(action))
			default:
				m.confirm = ""
				m.statusMsg = "cancelled"
			}
			return m, tea.Batch(cmds...)
		}

		switch msg.String() {
		case "esc", "backspace":
			m.done = true
			return m, nil

		case "tab":
			m.tab = (m.tab + 1) % detailTab(len(tabs))
			cmds = append(cmds, m.onTabSwitch())
			return m, tea.Batch(cmds...)

		case "1":
			m.tab = tabInfo
		case "2":
			m.tab = tabCode
			cmds = append(cmds, m.onTabSwitch())
			return m, tea.Batch(cmds...)
		case "3":
			m.tab = tabLogs
			cmds = append(cmds, m.onTabSwitch())
			return m, tea.Batch(cmds...)

		case "up", "k":
			if m.tab == tabInfo && m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.tab == tabInfo && m.cursor < len(m.instances)-1 {
				m.cursor++
			}

		case "s":
			if m.tab == tabInfo {
				cmds = append(cmds, startService(m.client, m.svc.Name))
				m.statusMsg = "starting…"
			}

		case "S":
			if m.tab == tabInfo && len(m.instances) > 0 {
				m.confirm = "stop-all"
			}

		case "x", "enter":
			if m.tab == tabInfo && len(m.instances) > 0 {
				id := m.instances[m.cursor].id
				m.confirm = "kill:" + id
			}
		}

	case runningLoadedMsg:
		m.infoLoading = false
		entries := make([]instanceEntry, 0, len(msg.running))
		for id, info := range msg.running {
			if info.CodeName == m.svc.Name {
				entries = append(entries, instanceEntry{id, info})
			}
		}
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].info.Started < entries[j].info.Started
		})
		m.instances = entries
		if m.cursor >= len(m.instances) {
			m.cursor = max(0, len(m.instances)-1)
		}

	case infoTickMsg:
		cmds = append(cmds, fetchRunning(m.client), scheduleInfoPoll())

	case actionDoneMsg:
		m.statusMsg = msg.status
		cmds = append(cmds, fetchRunning(m.client))

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

	// Forward scroll to active viewport.
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

func (m *ServiceDetailModel) executeAction(action string) tea.Cmd {
	switch {
	case action == "stop-all":
		ids := make([]string, len(m.instances))
		for i, e := range m.instances {
			ids[i] = e.id
		}
		return stopAll(m.client, ids)
	case strings.HasPrefix(action, "kill:"):
		id := strings.TrimPrefix(action, "kill:")
		return stopInstance(m.client, id)
	}
	return nil
}

// ── view ─────────────────────────────────────────────────────────────────────

func (m ServiceDetailModel) View() string {
	header := m.renderHeader()
	tabBar := m.renderTabBar()
	body := m.renderBody()

	var helpBindings []string
	switch m.tab {
	case tabInfo:
		helpBindings = []string{"esc back", "tab next", "s start", "S stop all", "↑↓ select instance", "x/enter kill"}
	default:
		helpBindings = []string{"esc back", "tab next", "↑↓ scroll"}
	}
	help := helpBar(helpBindings...)

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
	if m.confirm != "" {
		return m.renderConfirm()
	}
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

func (m ServiceDetailModel) renderConfirm() string {
	warn := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	var label string
	if m.confirm == "stop-all" {
		label = fmt.Sprintf("stop all %d instance(s)", len(m.instances))
	} else {
		id := strings.TrimPrefix(m.confirm, "kill:")
		label = fmt.Sprintf("kill instance %s", shortID(id))
	}
	return fmt.Sprintf("\n  %s  press y to confirm, any other key to cancel\n",
		warn.Render("⚠ "+label+"?"))
}

func (m ServiceDetailModel) renderInfo() string {
	b := &strings.Builder{}

	// ── service metadata ──────────────────────────────────────────────────
	section := lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	timeout := "never"
	if m.svc.ExecutionTimeout > 0 {
		timeout = fmt.Sprintf("%ds", m.svc.ExecutionTimeout)
	}

	fmt.Fprintf(b, "  %s\n", section.Render("Runtime Settings"))
	fmt.Fprintf(b, "    Engine:          %s  (version %d)\n", engineLabel(m.svc.EngineType), m.svc.Version)
	fmt.Fprintf(b, "    Exec timeout:    %s\n", timeout)
	fmt.Fprintf(b, "    Run on edge:     %v\n", m.svc.RunOnEdge)
	fmt.Fprintf(b, "    Run on platform: %v\n", m.svc.RunOnPlatform)
	if len(m.svc.Topics) > 0 {
		fmt.Fprintf(b, "    Topics:          %s\n", strings.Join(m.svc.Topics, ", "))
	}
	fmt.Fprintln(b)

	fmt.Fprintf(b, "  %s\n", section.Render("Logging"))
	fmt.Fprintf(b, "    Enabled:  %v\n", m.svc.LoggingEnabled)
	fmt.Fprintf(b, "    Level:    %s\n", m.svc.LogLevel)
	fmt.Fprintf(b, "    TTL:      %d min\n", m.svc.LogTTLMinutes)
	fmt.Fprintln(b)

	fmt.Fprintf(b, "  %s\n", section.Render("Concurrency Settings"))
	fmt.Fprintf(b, "    Instances:     %d\n", m.svc.Concurrency)
	fmt.Fprintf(b, "    Auto-balance:  %v\n", m.svc.AutoBalance)
	fmt.Fprintf(b, "    Auto-scale:    %v\n", m.svc.AutoScale)
	if m.svc.AutoScale {
		fmt.Fprintf(b, "    Min scale:     %d\n", m.svc.MinScaleConcurrency)
		fmt.Fprintf(b, "    Max scale:     %d\n", m.svc.MaxScaleConcurrency)
	}
	fmt.Fprintln(b)

	// ── status line ───────────────────────────────────────────────────────
	if m.statusMsg != "" {
		status := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
		fmt.Fprintf(b, "  %s\n\n", status.Render(m.statusMsg))
	}

	// ── running instances ─────────────────────────────────────────────────
	if m.infoLoading {
		fmt.Fprintf(b, "  Loading instances…\n")
		return b.String()
	}

	fmt.Fprintf(b, "  Running instances: %d\n", len(m.instances))
	if len(m.instances) == 0 {
		return b.String()
	}

	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	termStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	for i, e := range m.instances {
		started := time.Unix(0, e.info.Started).Format("2006-01-02 15:04:05")
		heap := formatBytesLocal(e.info.HeapStatistics.TotalBytesAllocated())
		if e.info.HeapStatistics.Error != "" {
			heap = "-"
		}

		terminating := ""
		if e.info.IsTerminating {
			terminating = termStyle.Render(" [terminating]")
		}

		line := fmt.Sprintf("    %s  started %s  heap %s%s",
			shortID(e.id), started, heap, terminating)

		if i == m.cursor {
			fmt.Fprintf(b, "%s\n", selectedStyle.Render("  ▶ "+strings.TrimLeft(line, " ")))
		} else {
			fmt.Fprintf(b, "%s\n", dimStyle.Render(line))
		}
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

// ── commands ─────────────────────────────────────────────────────────────────

func fetchRunning(c *client.Client) tea.Cmd {
	return func() tea.Msg {
		r, err := c.ListRunning()
		if err != nil {
			return errMsg{err}
		}
		return runningLoadedMsg{r}
	}
}

func scheduleInfoPoll() tea.Cmd {
	return tea.Tick(2*time.Second, func(_ time.Time) tea.Msg {
		return infoTickMsg{}
	})
}

func startService(c *client.Client, name string) tea.Cmd {
	return func() tea.Msg {
		if err := c.StartService(name, nil); err != nil {
			return actionDoneMsg{"start failed: " + err.Error()}
		}
		return actionDoneMsg{"started"}
	}
}

func stopInstance(c *client.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if err := c.StopInstance(id, 30); err != nil {
			return actionDoneMsg{"kill failed: " + err.Error()}
		}
		return actionDoneMsg{"killed " + shortID(id)}
	}
}

func stopAll(c *client.Client, ids []string) tea.Cmd {
	return func() tea.Msg {
		var failed int
		for _, id := range ids {
			if err := c.StopInstance(id, 30); err != nil {
				failed++
			}
		}
		if failed > 0 {
			return actionDoneMsg{fmt.Sprintf("stopped %d/%d instances", len(ids)-failed, len(ids))}
		}
		return actionDoneMsg{fmt.Sprintf("stopped all %d instances", len(ids))}
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

// ── helpers ──────────────────────────────────────────────────────────────────

func viewportHeight(total int) int {
	if total < 8 {
		return 4
	}
	return total - 6
}

func numberLines(code string) string {
	lines := strings.Split(code, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	w := len(fmt.Sprintf("%d", len(lines)))
	numStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	var b strings.Builder
	for i, line := range lines {
		b.WriteString(numStyle.Render(fmt.Sprintf("%*d  ", w, i+1)))
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

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
