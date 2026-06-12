package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"edge-cli/client"
	"edge-cli/config"

	"github.com/spf13/cobra"
)

var servicesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all services deployed on this edge",
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

		c := client.New(config.URL(), token, systemKey)

		services, err := c.ListServices()
		if err != nil {
			return fmt.Errorf("failed to list services: %w", err)
		}

		running, err := c.ListRunning()
		if err != nil {
			return fmt.Errorf("failed to list running instances: %w", err)
		}

		// Count running instances per service name.
		instanceCount := make(map[string]int)
		for _, info := range running {
			instanceCount[info.CodeName]++
		}

		onlyRunning, _ := cmd.Flags().GetBool("running")
		asJSON, _ := cmd.Flags().GetBool("json")

		if onlyRunning {
			filtered := services[:0]
			for _, svc := range services {
				if instanceCount[svc.Name] > 0 {
					filtered = append(filtered, svc)
				}
			}
			services = filtered
		}

		if asJSON {
			type row struct {
				Name      string `json:"name"`
				Instances int    `json:"instances"`
				Engine    string `json:"engine_type"`
				Timeout   string `json:"timeout"`
				Logging   string `json:"logging"`
			}
			rows := make([]row, 0, len(services))
			for _, svc := range services {
				rows = append(rows, row{
					Name:      svc.Name,
					Instances: instanceCount[svc.Name],
					Engine:    formatEngine(svc.EngineType),
					Timeout:   formatTimeout(svc.ExecutionTimeout),
					Logging:   formatBool(svc.LoggingEnabled),
				})
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(rows)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tINSTANCES\tENGINE\tTIMEOUT\tLOGGING")
		for _, svc := range services {
			fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\n",
				svc.Name,
				instanceCount[svc.Name],
				formatEngine(svc.EngineType),
				formatTimeout(svc.ExecutionTimeout),
				formatBool(svc.LoggingEnabled),
			)
		}
		return w.Flush()
	},
}

func init() {
	servicesCmd.AddCommand(servicesListCmd)
	servicesListCmd.Flags().Bool("running", false, "Show only services with at least one running instance")
	servicesListCmd.Flags().Bool("json", false, "Output as JSON")
}

func formatTimeout(t int) string {
	if t < 0 {
		return "never"
	}
	return fmt.Sprintf("%ds", t)
}

func formatEngine(e int) string {
	switch e {
	case 0:
		return "duk"
	case 1:
		return "v8"
	default:
		return fmt.Sprintf("unknown(%d)", e)
	}
}

func formatBool(b bool) string {
	if b {
		return "on"
	}
	return "off"
}
