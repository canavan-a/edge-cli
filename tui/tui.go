package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"edge-cli/client"
	"edge-cli/config"
	"edge-cli/tui/views"
)

func promptEdgeAuth(platformURL, platformToken, systemKey, edgeName string) (string, error) {
	credPrompt := views.NewEdgeCredPromptModel(edgeName)
	credProg := tea.NewProgram(credPrompt, tea.WithAltScreen())
	credResult, err := credProg.Run()
	if err != nil {
		return "", err
	}
	cp := credResult.(views.EdgeCredPromptModel)
	if cp.Cancelled() {
		return "", nil
	}
	edgeToken, err := client.AuthenticateEdgeViaProxy(platformURL, platformToken, systemKey, edgeName, cp.Email(), cp.Password())
	if err != nil {
		return "", fmt.Errorf("edge auth failed: %w", err)
	}
	_ = config.SaveEdgeToken(edgeName, edgeToken)
	return edgeToken, nil
}

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
	// Fill from saved config if not provided via flags.
	if platformURL == "" {
		platformURL = config.ProxyURL()
	}
	if systemKey == "" {
		systemKey = config.SystemKey()
	}

	// If we don't have a platform token yet, prompt for platform credentials.
	if token == "" || (platformURL != "" && systemKey != "") {
		// Check if we have a saved edge token — if so we can skip platform auth entirely.
		savedEdge := config.EdgeName()
		if savedEdge != "" && config.EdgeToken(savedEdge) != "" {
			edgeToken := config.EdgeToken(savedEdge)
			proxyClient := client.NewProxy(platformURL, edgeToken, systemKey, savedEdge)
			return Run(proxyClient)
		}
	}

	if platformURL == "" || systemKey == "" {
		prompt := views.NewPlatformPromptModel(platformURL, systemKey)
		p := tea.NewProgram(prompt, tea.WithAltScreen())
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

	if token == "" {
		token = config.Token()
	}

	// Check again for saved edge after we have a platform URL.
	savedEdge := config.EdgeName()
	if savedEdge != "" && config.EdgeToken(savedEdge) != "" {
		edgeToken := config.EdgeToken(savedEdge)
		proxyClient := client.NewProxy(platformURL, edgeToken, systemKey, savedEdge)
		return Run(proxyClient)
	}

	// No saved edge — show the picker.
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

	// Persist the proxy config so future runs skip the picker.
	_ = config.SaveProxyConfig(platformURL, edgeName)
	if systemKey != "" {
		config.SetSystemKey(systemKey)
	}

	// Use cached edge token if available, otherwise prompt for credentials.
	edgeToken := config.EdgeToken(edgeName)
	if edgeToken == "" {
		edgeToken, err = promptEdgeAuth(platformURL, token, systemKey, edgeName)
		if err != nil {
			return err
		}
		if edgeToken == "" {
			return nil // cancelled
		}
	}

	proxyClient := client.NewProxy(platformURL, edgeToken, systemKey, edgeName)
	return Run(proxyClient)
}
