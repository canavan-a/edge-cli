package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the current version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("edge-cli %s\n", version)
		if isDevBuild() {
			fmt.Println("repo:     (dev build — repo not set)")
		} else {
			fmt.Printf("repo:     https://github.com/%s\n", repo)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
