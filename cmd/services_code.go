package cmd

import (
	"fmt"
	"strings"

	"edge-cli/client"
	"edge-cli/config"

	"github.com/spf13/cobra"
)

var servicesCodeCmd = &cobra.Command{
	Use:   "code <service-name>",
	Short: "Print the source code of a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		edgeConfigPath, _ := cmd.Flags().GetString("edge-config")
		systemKey, err := resolveSystemKey(edgeConfigPath)
		if err != nil {
			return err
		}
		token := config.Token()
		if token == "" {
			return fmt.Errorf("not authenticated — run: edge-cli auth login")
		}

		lineNumbers, _ := cmd.Flags().GetBool("line-numbers")

		c := client.New(config.URL(), token, systemKey)
		code, err := c.GetServiceCode(name)
		if err != nil {
			return fmt.Errorf("failed to get code for %q: %w", name, err)
		}

		if !lineNumbers {
			fmt.Print(code)
			if len(code) > 0 && code[len(code)-1] != '\n' {
				fmt.Println()
			}
			return nil
		}

		lines := strings.Split(code, "\n")
		// trim trailing empty line from split
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		width := len(fmt.Sprintf("%d", len(lines)))
		for i, line := range lines {
			fmt.Printf("%*d  %s\n", width, i+1, line)
		}
		return nil
	},
}

func init() {
	servicesCmd.AddCommand(servicesCodeCmd)
	servicesCodeCmd.Flags().BoolP("line-numbers", "l", false, "Print line numbers")
}
