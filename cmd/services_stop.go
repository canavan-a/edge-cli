package cmd

import (
	"fmt"

	"edge-cli/client"
	"edge-cli/config"

	"github.com/spf13/cobra"
)

var servicesStopCmd = &cobra.Command{
	Use:   "stop <service-name>",
	Short: "Stop all running instances of a service",
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

		timeout, _ := cmd.Flags().GetInt("timeout")
		instanceID, _ := cmd.Flags().GetString("instance")

		c := client.New(config.URL(), token, systemKey)

		// If a specific instance was given, stop only that one.
		if instanceID != "" {
			if err := c.StopInstance(instanceID, timeout); err != nil {
				return fmt.Errorf("failed to stop instance %s: %w", instanceID, err)
			}
			fmt.Printf("Stopped instance %s.\n", instanceID)
			return nil
		}

		// Otherwise stop every running instance of the named service.
		running, err := c.ListRunning()
		if err != nil {
			return fmt.Errorf("failed to list running instances: %w", err)
		}

		var stopped int
		for id, info := range running {
			if info.CodeName != name {
				continue
			}
			if err := c.StopInstance(id, timeout); err != nil {
				fmt.Printf("Warning: failed to stop instance %s: %s\n", id, err)
				continue
			}
			fmt.Printf("Stopped instance %s.\n", id)
			stopped++
		}

		if stopped == 0 {
			fmt.Printf("No running instances found for %q.\n", name)
		} else {
			fmt.Printf("Stopped %d instance(s) of %q.\n", stopped, name)
		}
		return nil
	},
}

func init() {
	servicesCmd.AddCommand(servicesStopCmd)
	servicesStopCmd.Flags().String("instance", "", "Stop a specific instance by ID (default: stop all instances)")
	servicesStopCmd.Flags().Int("timeout", 30, "Graceful shutdown timeout in seconds")
}
