package cmd

import "github.com/spf13/cobra"

var servicesCmd = &cobra.Command{
	Use:   "services",
	Short: "Manage and inspect services running on this edge",
}

func init() {
	rootCmd.AddCommand(servicesCmd)
}
