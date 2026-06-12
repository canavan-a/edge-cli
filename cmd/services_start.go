package cmd

import (
	"encoding/json"
	"fmt"

	"edge-cli/client"
	"edge-cli/config"

	"github.com/spf13/cobra"
)

var servicesStartCmd = &cobra.Command{
	Use:   "start <service-name>",
	Short: "Trigger execution of a service",
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

		var params map[string]any
		if paramsStr, _ := cmd.Flags().GetString("params"); paramsStr != "" {
			if err := json.Unmarshal([]byte(paramsStr), &params); err != nil {
				return fmt.Errorf("--params must be valid JSON: %w", err)
			}
		}

		c := client.New(config.URL(), token, systemKey)

		if err := c.StartService(name, params); err != nil {
			return fmt.Errorf("failed to start service %q: %w", name, err)
		}

		fmt.Printf("Started %q.\n", name)
		return nil
	},
}

func init() {
	servicesCmd.AddCommand(servicesStartCmd)
	servicesStartCmd.Flags().String("params", "", `Optional JSON params to pass to the service, e.g. '{"key":"value"}'`)
}
