package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"edge-cli/config"
	"edge-cli/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive TUI (direct mode)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		edgeConfigPath, _ := cmd.Flags().GetString("edge-config")
		systemKey, err := resolveSystemKey(edgeConfigPath)
		if err != nil {
			return err
		}
		token := config.Token()
		if token == "" {
			return fmt.Errorf("not authenticated — run: edge-cli auth login")
		}
		c := newClient(systemKey)
		return tui.Run(c)
	},
}

var tuiProxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Connect via platform proxy — prompts for platform URL, lists edges to pick from",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		platformURL, _ := cmd.Flags().GetString("platform")
		systemKey, _ := cmd.Flags().GetString("system-key")
		token := config.Token()
		if token == "" {
			return fmt.Errorf("not authenticated — run: edge-cli auth login")
		}
		if platformURL == "" {
			platformURL = config.ProxyURL()
		}
		if systemKey == "" {
			systemKey = config.SystemKey()
		}
		// tui.RunProxy handles the interactive edge picker then launches the TUI
		return tui.RunProxy(platformURL, token, systemKey)
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
	tuiCmd.AddCommand(tuiProxyCmd)
	tuiProxyCmd.Flags().String("platform", "", "Platform URL (e.g. https://platform.clearblade.com)")
	tuiProxyCmd.Flags().String("system-key", "", "System key on the platform")
}
