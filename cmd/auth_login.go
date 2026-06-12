package cmd

import (
	"fmt"
	"os"
	"syscall"

	"edge-cli/client"
	"edge-cli/config"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in and save credentials to ~/.edge-cli/config.yaml",
	RunE: func(cmd *cobra.Command, args []string) error {
		email, _ := cmd.Flags().GetString("email")
		if email == "" {
			fmt.Print("Email: ")
			fmt.Scanln(&email)
		}

		fmt.Print("Password: ")
		passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}

		edgeURL := config.URL()

		token, err := client.Authenticate(edgeURL, email, string(passwordBytes))
		if err != nil {
			return err
		}

		if err := config.SaveCredentials(token, email); err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}

		fmt.Fprintf(os.Stdout, "Logged in as %s. Token saved to %s\n", email, config.ConfigFile())
		return nil
	},
}

func init() {
	authCmd.AddCommand(loginCmd)
	loginCmd.Flags().String("email", "", "Developer email address")
}
