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

var collectionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all collections in the system",
	Args:  cobra.NoArgs,
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

		outputFmt, _ := cmd.Flags().GetString("output")

		c := client.New(config.URL(), token, systemKey)
		cols, err := c.ListCollections()
		if err != nil {
			return fmt.Errorf("failed to list collections: %w", err)
		}

		if outputFmt == "json" {
			return json.NewEncoder(os.Stdout).Encode(cols)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tID")
		for _, col := range cols {
			fmt.Fprintf(w, "%s\t%s\n", col.Name, col.ID)
		}
		return w.Flush()
	},
}

func init() {
	collectionsCmd.AddCommand(collectionsListCmd)
	collectionsListCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
}
