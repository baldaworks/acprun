package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/baldaworks/acprun/internal/resolver"
)

func newResolveCmd() *cobra.Command {
	var platformOverride string
	var formatOverride string
	var asJSON bool
	var noDownload bool

	cmd := &cobra.Command{
		Use:   "resolve <agent-id> [-- extra-args...]",
		Short: "Resolve an agent's execution vector without running the process",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID := args[0]
			extraArgs := args[1:]

			client := getRegistryClient()
			res := getResolver(client)

			agent, err := client.GetAgent(cmd.Context(), agentID)
			if err != nil {
				return err
			}

			resolved, err := res.Resolve(cmd.Context(), agent, resolver.ResolveOptions{
				Platform:   platformOverride,
				Format:     formatOverride,
				NoDownload: noDownload,
				ExtraArgs:  extraArgs,
			})
			if err != nil {
				return fmt.Errorf("failed to resolve agent %q: %w", agentID, err)
			}

			if asJSON {
				data, err := json.MarshalIndent(resolved, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
				return nil
			}

			fmt.Printf("Agent:       %s (version %s)\n", resolved.AgentID, resolved.Version)
			fmt.Printf("Format:      %s\n", resolved.Format)
			fmt.Printf("Executable:  %s\n", resolved.Executable)
			if len(resolved.Args) > 0 {
				fmt.Printf("Args:        %s\n", strings.Join(resolved.Args, " "))
			}
			if resolved.WorkingDir != "" {
				fmt.Printf("WorkingDir:  %s\n", resolved.WorkingDir)
			}
			if len(resolved.Env) > 0 {
				fmt.Println("Environment:")
				for k, v := range resolved.Env {
					fmt.Printf("  %s=%s\n", k, v)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&platformOverride, "platform", "p", "", "Target platform (e.g. linux-x86_64, darwin-aarch64)")
	cmd.Flags().StringVarP(&formatOverride, "format", "f", "", "Distribution format override: binary, npx, uvx")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output resolved command vector as JSON")
	cmd.Flags().BoolVar(&noDownload, "no-download", false, "Do not download binary archive if missing from cache")

	return cmd
}
