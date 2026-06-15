package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"edge-cli/client"
	"edge-cli/tui/views"
)

func Run(c *client.Client) error {
	m := views.NewServicesModel(c)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}

// RunProxy shows the edge picker TUI. On selection it launches the main TUI via proxy.
func RunProxy(platformURL, token, systemKey string) error {
	if platformURL == "" || systemKey == "" {
		picker := views.NewPlatformPromptModel(token)
		p := tea.NewProgram(picker, tea.WithAltScreen())
		result, err := p.Run()
		if err != nil {
			return err
		}
		pm := result.(views.PlatformPromptModel)
		if pm.Cancelled() {
			return nil
		}
		platformURL = pm.PlatformURL()
		systemKey = pm.SystemKey()
		token = pm.Token()
	}

	// Platform client (no edge yet — just to list edges)
	platformClient := client.New(platformURL, token, systemKey)

	picker := views.NewEdgePickerModel(platformClient)
	p := tea.NewProgram(picker, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		return err
	}
	pm := result.(views.EdgePickerModel)
	if pm.Cancelled() {
		return nil
	}

	edgeName := pm.SelectedEdge()
	proxyClient := client.NewProxy(platformURL, token, systemKey, edgeName)
	return Run(proxyClient)
}
