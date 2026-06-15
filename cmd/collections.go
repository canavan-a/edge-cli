package cmd

import "github.com/spf13/cobra"

var collectionsCmd = &cobra.Command{
	Use:   "collections",
	Short: "Manage and query ClearBlade collections",
}

func init() {
	rootCmd.AddCommand(collectionsCmd)
}
