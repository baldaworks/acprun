package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/baldaworks/acprun/internal/registry"
)

func newListCmd() *cobra.Command {
	var format string
	var distFilter string

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List available agents in the ACP registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getRegistryClient()
			agents, err := client.ListAgents(cmd.Context())
			if err != nil {
				return err
			}

			// Filter by distribution type if requested
			var filtered []registry.Agent
			distFilter = strings.ToLower(strings.TrimSpace(distFilter))
			for _, a := range agents {
				if distFilter == "" || distFilter == "all" {
					filtered = append(filtered, a)
					continue
				}
				types := a.DistributionTypes()
				matched := false
				for _, t := range types {
					if t == distFilter {
						matched = true
						break
					}
				}
				if matched {
					filtered = append(filtered, a)
				}
			}

			switch strings.ToLower(format) {
			case "json":
				data, err := json.MarshalIndent(filtered, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
			case "ids":
				for _, a := range filtered {
					fmt.Println(a.ID)
				}
			default: // "table"
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
				fmt.Fprintln(w, "ID\tNAME\tVERSION\tDISTRIBUTIONS\tDESCRIPTION")
				for _, a := range filtered {
					desc := a.Description
					if len(desc) > 60 {
						desc = desc[:57] + "..."
					}
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
						a.ID,
						a.Name,
						a.Version,
						a.DistributionTypesString(),
						desc,
					)
				}
				w.Flush()
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json, ids")
	cmd.Flags().StringVarP(&distFilter, "distribution", "d", "all", "Filter by distribution type: all, binary, npx, uvx")

	return cmd
}
