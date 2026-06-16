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

	// The platform dev token is not in the edge's local auth DB.
	// Prompt for credentials and authenticate against the edge through the proxy.
	credPrompt := views.NewEdgeCredPromptModel(edgeName)
	credProg := tea.NewProgram(credPrompt, tea.WithAltScreen())
	credResult, err := credProg.Run()
	if err != nil {
		return err
	}
	cp := credResult.(views.EdgeCredPromptModel)
	if cp.Cancelled() {
		return nil
	}

	edgeToken, err := client.AuthenticateEdgeViaProxy(platformURL, token, systemKey, edgeName, cp.Email(), cp.Password())
	if err != nil {
		return fmt.Errorf("edge auth failed: %w", err)
	}

	proxyClient := client.NewProxy(platformURL, edgeToken, systemKey, edgeName)
	return Run(proxyClient)
}
