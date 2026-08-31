package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newInfoCmd() *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:     "info <agent-id>",
		Aliases: []string{"show"},
		Short:   "Display detailed metadata and distributions for an agent",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID := args[0]
			client := getRegistryClient()

			agent, err := client.GetAgent(cmd.Context(), agentID)
			if err != nil {
				return err
			}

			if asJSON {
				data, err := json.MarshalIndent(agent, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
				return nil
			}

			fmt.Printf("ID:           %s\n", agent.ID)
			fmt.Printf("Name:         %s\n", agent.Name)
			fmt.Printf("Version:      %s\n", agent.Version)
			if agent.Description != "" {
				fmt.Printf("Description:  %s\n", agent.Description)
			}
			if len(agent.Authors) > 0 {
				fmt.Printf("Authors:      %s\n", strings.Join(agent.Authors, ", "))
			}
			if agent.License != "" {
				fmt.Printf("License:      %s\n", agent.License)
			}
			if agent.Website != "" {
				fmt.Printf("Website:      %s\n", agent.Website)
			}
			if agent.Repository != "" {
				fmt.Printf("Repository:   %s\n", agent.Repository)
			}

			fmt.Println("\nDistributions:")
			if len(agent.Distribution.Binary) > 0 {
				fmt.Println("  Binary:")
				for platform, target := range agent.Distribution.Binary {
					fmt.Printf("    - Platform: %s\n", platform)
					fmt.Printf("      Archive:  %s\n", target.Archive)
					fmt.Printf("      Command:  %s\n", target.Cmd)
					if len(target.Args) > 0 {
						fmt.Printf("      Args:     %v\n", target.Args)
					}
					if target.SHA256 != "" {
						fmt.Printf("      SHA256:   %s\n", target.SHA256)
					}
				}
			}
			if agent.Distribution.NPX != nil {
				fmt.Println("  NPX:")
				fmt.Printf("    Package:  %s\n", agent.Distribution.NPX.Package)
				if len(agent.Distribution.NPX.Args) > 0 {
					fmt.Printf("    Args:     %v\n", agent.Distribution.NPX.Args)
				}
				if len(agent.Distribution.NPX.Env) > 0 {
					fmt.Printf("    Env:      %v\n", agent.Distribution.NPX.Env)
				}
			}
			if agent.Distribution.UVX != nil {
				fmt.Println("  UVX:")
				fmt.Printf("    Package:  %s\n", agent.Distribution.UVX.Package)
				if len(agent.Distribution.UVX.Args) > 0 {
					fmt.Printf("    Args:     %v\n", agent.Distribution.UVX.Args)
				}
				if len(agent.Distribution.UVX.Env) > 0 {
					fmt.Printf("    Env:      %v\n", agent.Distribution.UVX.Env)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "Output agent metadata as JSON")

	return cmd
}
