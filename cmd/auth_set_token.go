package cmd

import (
	"fmt"
	"os"

	"edge-cli/config"

	"github.com/spf13/cobra"
)

var setTokenCmd = &cobra.Command{
	Use:   "set-token <token>",
	Short: "Save a dev token directly without logging in",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		token := args[0]
		email, _ := cmd.Flags().GetString("email")

		if err := config.SaveCredentials(token, email); err != nil {
			return fmt.Errorf("failed to save token: %w", err)
		}

		fmt.Fprintf(os.Stdout, "Token saved to %s\n", config.ConfigFile())
		return nil
	},
}

func init() {
	authCmd.AddCommand(setTokenCmd)
	setTokenCmd.Flags().String("email", "", "Optional email label for auth status display")
}
