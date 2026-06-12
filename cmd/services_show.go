package cmd

import (
	"fmt"
	"text/tabwriter"
	"time"
	"os"

	"edge-cli/client"
	"edge-cli/config"

	"github.com/spf13/cobra"
)

var servicesShowCmd = &cobra.Command{
	Use:   "show <service-name>",
	Short: "Show details and running instances for a service",
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

		c := client.New(config.URL(), token, systemKey)

		svc, err := c.GetService(name)
		if err != nil {
			return fmt.Errorf("failed to get service %q: %w", name, err)
		}

		running, err := c.ListRunning()
		if err != nil {
			return fmt.Errorf("failed to list running instances: %w", err)
		}

		var instances []instanceRow
		for id, info := range running {
			if info.CodeName != name {
				continue
			}
			instances = append(instances, instanceRow{
				id:            id,
				started:       info.Started,
				isTerminating: info.IsTerminating,
				heapBytes:     info.HeapStatistics.TotalBytesAllocated(),
				heapError:     info.HeapStatistics.Error,
			})
		}

		// Metadata section
		fmt.Printf("Service:      %s\n", svc.Name)
		fmt.Printf("  System Key:   %s\n", svc.SystemKey)
		fmt.Printf("  Engine:       %s\n", svc.EngineType)
		fmt.Printf("  Timeout:      %s\n", formatTimeout(svc.ExecutionTimeout))
		fmt.Printf("  Concurrency:  %d\n", svc.Concurrency)
		loggingStr := formatBool(svc.LoggingEnabled)
		if svc.LoggingEnabled {
			loggingStr = fmt.Sprintf("on (level: %s, TTL: %d min)", svc.LogLevel, svc.LogTTLMinutes)
		}
		fmt.Printf("  Logging:      %s\n", loggingStr)
		fmt.Printf("  Auto-scale:   %s\n", formatBool(svc.AutoScale))
		fmt.Printf("  Run on edge:  %s\n", formatBool(svc.RunOnEdge))
		fmt.Println()

		fmt.Printf("Running Instances (%d):\n", len(instances))
		if len(instances) == 0 {
			fmt.Println("  (none)")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "  ID\tSTARTED\tTERMINATING\tHEAP USED")
		for _, inst := range instances {
			startedStr := time.Unix(0, inst.started*int64(time.Millisecond)).Format("2006-01-02 15:04:05")
			heapStr := formatBytes(inst.heapBytes)
			if inst.heapError != "" {
				heapStr = "-"
			}
			fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n",
				inst.id,
				startedStr,
				formatBool(inst.isTerminating),
				heapStr,
			)
		}
		return w.Flush()
	},
}

type instanceRow struct {
	id            string
	started       int64
	isTerminating bool
	heapBytes     uint64
	heapError     string
}

func init() {
	servicesCmd.AddCommand(servicesShowCmd)
}

func formatBytes(b uint64) string {
	switch {
	case b >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(b)/1024/1024)
	case b >= 1024:
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	default:
		return fmt.Sprintf("%d B", b)
	}
}
