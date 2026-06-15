package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"edge-cli/client"
	"edge-cli/config"
	"edge-cli/models"

	"github.com/spf13/cobra"
)

var servicesLogsCmd = &cobra.Command{
	Use:   "logs [service-name]",
	Short: "Fetch logs for a service (use --follow to tail, --all for all services)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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
		allServices, _ := cmd.Flags().GetBool("all")
		level, _ := cmd.Flags().GetString("level")
		sinceStr, _ := cmd.Flags().GetString("since")
		limit, _ := cmd.Flags().GetInt("limit")
		intervalStr, _ := cmd.Flags().GetString("interval")

		if len(args) == 0 && !allServices {
			return fmt.Errorf("specify a service name or use --all")
		}

		var serviceName string
		if len(args) > 0 {
			serviceName = args[0]
		}

		interval, err := time.ParseDuration(intervalStr)
		if err != nil {
			return fmt.Errorf("invalid --interval %q: %w", intervalStr, err)
		}

		var since time.Time
		if sinceStr != "" {
			dur, err := time.ParseDuration(sinceStr)
			if err != nil {
				// try absolute timestamp
				since, err = time.Parse("2006-01-02 15:04:05", sinceStr)
				if err != nil {
					return fmt.Errorf("invalid --since %q: use a duration (e.g. 30m, 1h) or timestamp (2006-01-02 15:04:05)", sinceStr)
				}
			} else {
				since = time.Now().Add(-dur)
			}
		}

		c := client.New(config.URL(), token, systemKey)

		opts := client.LogQueryOpts{
			ServiceName: serviceName,
			Level:       level,
			Since:       since,
			Limit:       limit,
		}

		entries, err := c.GetLogsV4(opts)
		if err != nil {
			return fmt.Errorf("failed to get logs: %w", err)
		}

		// Track the latest timestamp seen so follow-mode only fetches newer entries.
		var latestTime int64
		for _, e := range entries {
			printLogEntry(e)
			if e.Time > latestTime {
				latestTime = e.Time
			}
		}

		if !follow {
			return nil
		}

		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-sig:
				return nil
			case <-ticker.C:
				pollOpts := client.LogQueryOpts{
					ServiceName: serviceName,
					Level:       level,
					Limit:       limit,
				}
				if latestTime > 0 {
					// Fetch only entries newer than the last one we printed.
					pollOpts.AfterTimeMicros = latestTime
				}

				entries, err := c.GetLogsV4(pollOpts)
				if err != nil {
					fmt.Fprintf(os.Stderr, "poll error: %s\n", err)
					continue
				}
				for _, e := range entries {
					printLogEntry(e)
					if e.Time > latestTime {
						latestTime = e.Time
					}
				}
			}
		}
	},
}

func printLogEntry(e models.LogEntry) {
	ts := time.UnixMicro(e.Time).Format("2006-01-02 15:04:05")
	prefix := fmt.Sprintf("[%s] %s", ts, e.Name)
	if e.Level != "" {
		prefix += " [" + e.Level + "]"
	}
	fmt.Printf("%s: %s\n", prefix, e.Log)
}

func init() {
	servicesCmd.AddCommand(servicesLogsCmd)
	servicesLogsCmd.Flags().BoolP("follow", "f", false, "Poll for new log entries continuously")
	servicesLogsCmd.Flags().BoolP("all", "a", false, "Show logs for all services")
	servicesLogsCmd.Flags().String("level", "", "Filter by log level (e.g. info, warn, error)")
	servicesLogsCmd.Flags().String("since", "", "Show logs since duration (e.g. 30m, 1h) or timestamp (2006-01-02 15:04:05)")
	servicesLogsCmd.Flags().Int("limit", 50, "Max number of log entries to fetch per request")
	servicesLogsCmd.Flags().String("interval", "2s", "Polling interval when using --follow")
}
