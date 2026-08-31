package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	var envFlags []string
	var platformOverride string
	var noDownload bool

	cmd := &cobra.Command{
		Use:     "run <agent-id> [-- extra-args...]",
		Aliases: []string{"exec", "serve", "start"},
		Short:   "Resolve and execute an ACP agent process",
		Long: `Resolve and execute an ACP agent as a child process.

Aliases:
  acprun run <agent-id> [args...]
  acprun serve <agent-id> [args...]
  acprun exec <agent-id> [args...]
  acprun start <agent-id> [args...]

Use explicit runner subcommands when you want to avoid any potential ambiguity
with management subcommands (e.g. running an agent whose ID is 'list' or 'cache').`,
		Args: cobra.MinimumNArgs(1),
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true,
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID := args[0]
			extraArgs := args[1:]

			extraEnv := make(map[string]string)
			for _, envStr := range envFlags {
				parts := strings.SplitN(envStr, "=", 2)
				if len(parts) == 2 {
					extraEnv[parts[0]] = parts[1]
				} else {
					return fmt.Errorf("invalid environment variable format %q, expected KEY=VALUE", envStr)
				}
			}

			return runAgent(cmd.Context(), agentID, extraArgs, extraEnv, platformOverride, noDownload)
		},
	}

	cmd.Flags().StringArrayVarP(&envFlags, "env", "e", nil, "Set environment variable (KEY=VALUE)")
	cmd.Flags().StringVarP(&platformOverride, "platform", "p", "", "Target platform override")
	cmd.Flags().BoolVar(&noDownload, "no-download", false, "Do not download binary archive if missing from cache")

	return cmd
}
