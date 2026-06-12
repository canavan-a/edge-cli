package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"edge-cli/client"
	"edge-cli/config"

	"github.com/spf13/cobra"
)

var servicesLogsCmd = &cobra.Command{
	Use:   "logs <service-name>",
	Short: "Fetch logs for a service (use --follow to tail)",
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

		follow, _ := cmd.Flags().GetBool("follow")
		intervalStr, _ := cmd.Flags().GetString("interval")
		interval, err := time.ParseDuration(intervalStr)
		if err != nil {
			return fmt.Errorf("invalid --interval %q: %w", intervalStr, err)
		}

		c := client.New(config.URL(), token, systemKey)

		// Initial fetch — print all available log runs.
		logs, err := c.GetLogs(name)
		if err != nil {
			return fmt.Errorf("failed to get logs: %w", err)
		}

		seen := make(map[string]bool)
		for _, l := range logs {
			seen[l.ServiceId] = true
			printLogUnit(name, l.ServiceId, l.Time, l.Log)
		}

		if !follow {
			return nil
		}

		// Poll loop — print new runs as they appear.
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-sig:
				return nil
			case <-ticker.C:
				logs, err := c.GetLogs(name)
				if err != nil {
					fmt.Fprintf(os.Stderr, "poll error: %s\n", err)
					continue
				}
				for _, l := range logs {
					if seen[l.ServiceId] {
						continue
					}
					seen[l.ServiceId] = true
					printLogUnit(name, l.ServiceId, l.Time, l.Log)
				}
			}
		}
	},
}

func printLogUnit(serviceName, id, timestamp, log string) {
	fmt.Printf("=== %s [%s] @ %s ===\n", serviceName, id, timestamp)
	if log != "" {
		fmt.Print(log)
		if log[len(log)-1] != '\n' {
			fmt.Println()
		}
	}
}

func init() {
	servicesCmd.AddCommand(servicesLogsCmd)
	servicesLogsCmd.Flags().BoolP("follow", "f", false, "Poll for new log runs continuously")
	servicesLogsCmd.Flags().String("interval", "2s", "Polling interval when using --follow")
}
