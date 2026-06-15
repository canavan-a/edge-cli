package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"edge-cli/client"
	"edge-cli/config"

	"github.com/spf13/cobra"
)

var collectionsQueryCmd = &cobra.Command{
	Use:   "query <collection-name>",
	Short: "Fetch rows from a collection",
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

		limit, _ := cmd.Flags().GetInt("limit")
		sortBy, _ := cmd.Flags().GetString("sort-by")
		sortDesc, _ := cmd.Flags().GetBool("desc")
		outputFmt, _ := cmd.Flags().GetString("output")

		c := client.New(config.URL(), token, systemKey)
		result, err := c.QueryCollection(name, client.CollectionQueryOpts{
			SortBy:   sortBy,
			SortDesc: sortDesc,
			Limit:    limit,
		})
		if err != nil {
			return fmt.Errorf("failed to query collection: %w", err)
		}

		if outputFmt == "json" {
			return json.NewEncoder(os.Stdout).Encode(result.Data)
		}

		if len(result.Data) == 0 {
			fmt.Println("(no rows)")
			return nil
		}

		// Collect and sort column names for stable output; put item_id first.
		colSet := map[string]bool{}
		for _, row := range result.Data {
			for k := range row {
				colSet[k] = true
			}
		}
		cols := make([]string, 0, len(colSet))
		for k := range colSet {
			if k != "item_id" {
				cols = append(cols, k)
			}
		}
		sort.Strings(cols)
		if colSet["item_id"] {
			cols = append([]string{"item_id"}, cols...)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, strings.Join(cols, "\t"))
		for _, row := range result.Data {
			vals := make([]string, len(cols))
			for i, col := range cols {
				if v := row[col]; v != nil {
					vals[i] = fmt.Sprintf("%v", v)
				}
			}
			fmt.Fprintln(w, strings.Join(vals, "\t"))
		}
		return w.Flush()
	},
}

func init() {
	collectionsCmd.AddCommand(collectionsQueryCmd)
	collectionsQueryCmd.Flags().IntP("limit", "n", 25, "Max number of rows to return")
	collectionsQueryCmd.Flags().String("sort-by", "", "Column to sort by")
	collectionsQueryCmd.Flags().Bool("desc", false, "Sort in descending order")
	collectionsQueryCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
}
