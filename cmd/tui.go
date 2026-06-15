package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"edge-cli/client"
	"edge-cli/config"
	"edge-cli/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive TUI",
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
		c := client.New(config.URL(), token, systemKey)
		return tui.Run(c)
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
