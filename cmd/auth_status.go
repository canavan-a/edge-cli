package cmd

import (
	"fmt"

	"edge-cli/config"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	Run: func(cmd *cobra.Command, args []string) {
		token := config.Token()
		email := config.Email()
		url := config.URL()

		if token == "" {
			fmt.Println("Not logged in.")
			fmt.Printf("Run: edge-cli auth login --email <email>\n")
			return
		}

		if email != "" {
			fmt.Printf("Logged in as %s\n", email)
		} else {
			fmt.Println("Logged in (email unknown)")
		}
		fmt.Printf("Edge URL:   %s\n", url)
		fmt.Printf("Config:     %s\n", config.ConfigFile())
	},
}

func init() {
	authCmd.AddCommand(statusCmd)
}
