// Package cli provides Cobra command implementations for acprun.
package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/baldaworks/acprun/internal/registry"
	"github.com/baldaworks/acprun/internal/resolver"
	"github.com/baldaworks/acprun/internal/runner"
)

type globalFlags struct {
	registryURL string
	cacheDir    string
	offline     bool
	verbose     bool
}

var globals globalFlags

// NewRootCmd creates and configures the root Cobra command for acprun.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "acprun [<agent-id> [args...] | command]",
		Short: "acprun - Discover, resolve, and run ACP (Agent Client Protocol) agents",
		Long: `acprun is a CLI tool and runner for discovering, resolving, and running
agents published in the Agent Client Protocol (ACP) Registry.

Supports one-shot execution:
  acprun <agent-id> [agent-args...]

As well as explicit runner and management subcommands:
  acprun run / serve / exec / start <agent-id> [args...]
  acprun list
  acprun info <agent-id>
  acprun resolve <agent-id>
  acprun cache [path|clean]
  acprun version`,
		SilenceUsage:  true,
		SilenceErrors: true,
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true,
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			InitLogger(globals.verbose)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			// One-shot execution: args[0] is agentID, args[1:] are extra arguments
			agentID := args[0]
			extraArgs := args[1:]

			return runAgent(cmd.Context(), agentID, extraArgs, nil, "", false)
		},
	}

	rootCmd.PersistentFlags().StringVarP(&globals.registryURL, "registry", "r", "", "ACP Registry URL (default: official CDN, env: ACP_REGISTRY_URL)")
	rootCmd.PersistentFlags().StringVar(&globals.cacheDir, "cache-dir", "", "Cache directory path (default: $USER_CACHE_DIR/acprun, env: ACP_CACHE_DIR)")
	rootCmd.PersistentFlags().BoolVar(&globals.offline, "offline", false, "Offline mode: use cached manifests and binaries only")
	rootCmd.PersistentFlags().BoolVarP(&globals.verbose, "verbose", "v", false, "Enable verbose output")

	// Register subcommands
	rootCmd.AddCommand(newListCmd())
	rootCmd.AddCommand(newInfoCmd())
	rootCmd.AddCommand(newResolveCmd())
	rootCmd.AddCommand(newRunCmd())
	rootCmd.AddCommand(newCacheCmd())
	rootCmd.AddCommand(newVersionCmd())

	return rootCmd
}

// Execute runs the CLI and returns the process exit code.
func Execute() int {
	return ExecuteArgs(os.Args[1:])
}

// ExecuteArgs runs the CLI with the provided arguments and returns the process exit code.
func ExecuteArgs(args []string) int {
	rootCmd := NewRootCmd()
	ctx := context.Background()

	rootCmd.SetArgs(NormalizeArgs(rootCmd, args))

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}

// NormalizeArgs transforms CLI arguments so that one-shot agent execution (e.g. `acprun <agent-id>`)
// is routed to the `run` subcommand when the first positional argument is not a known subcommand.
func NormalizeArgs(cmd *cobra.Command, args []string) []string {
	i := 0
	for i < len(args) {
		arg := args[i]

		// Handle POSIX end of options delimiter
		if arg == "--" {
			if i+1 < len(args) && !isSubcommand(cmd, args[i+1]) {
				newArgs := make([]string, 0, len(args)+1)
				newArgs = append(newArgs, "run")
				newArgs = append(newArgs, args...)
				return newArgs
			}
			return args
		}

		if strings.HasPrefix(arg, "-") {
			// Flag with '=' format, e.g. --registry=https://... or -r=https://...
			if strings.Contains(arg, "=") {
				i++
				continue
			}

			flagName := strings.TrimLeft(arg, "-")
			if isFlagWithValue(cmd, flagName) {
				// Flag consumes next argument as its value
				i += 2
				continue
			}

			// Boolean or standalone flag
			i++
			continue
		}

		// First non-flag argument encountered
		if isSubcommand(cmd, arg) {
			// Recognized subcommand; leave args as is
			return args
		}

		// Found agent ID: route to `run` subcommand by prepending "run"
		newArgs := make([]string, 0, len(args)+1)
		newArgs = append(newArgs, "run")
		newArgs = append(newArgs, args...)
		return newArgs
	}

	return args
}

func isSubcommand(cmd *cobra.Command, name string) bool {
	if name == "help" || name == "completion" || strings.HasPrefix(name, "__complete") {
		return true
	}
	for _, sub := range cmd.Commands() {
		if sub.Name() == name {
			return true
		}
		for _, alias := range sub.Aliases {
			if alias == name {
				return true
			}
		}
	}
	return false
}

func isFlagWithValue(cmd *cobra.Command, flagName string) bool {
	if len(flagName) == 1 {
		if f := cmd.PersistentFlags().ShorthandLookup(flagName); f != nil {
			return f.Value.Type() != "bool"
		}
		if f := cmd.Flags().ShorthandLookup(flagName); f != nil {
			return f.Value.Type() != "bool"
		}
		for _, sub := range cmd.Commands() {
			if f := sub.Flags().ShorthandLookup(flagName); f != nil {
				return f.Value.Type() != "bool"
			}
		}
		return false
	}

	if f := cmd.PersistentFlags().Lookup(flagName); f != nil {
		return f.Value.Type() != "bool"
	}
	if f := cmd.Flags().Lookup(flagName); f != nil {
		return f.Value.Type() != "bool"
	}
	for _, sub := range cmd.Commands() {
		if f := sub.Flags().Lookup(flagName); f != nil {
			return f.Value.Type() != "bool"
		}
	}

	return false
}

func getRegistryClient() *registry.Client {
	return registry.NewClient(registry.ClientOptions{
		RegistryURL: globals.registryURL,
		CacheDir:    globals.cacheDir,
		Offline:     globals.offline,
	})
}

func getResolver(client *registry.Client) *resolver.Resolver {
	return resolver.NewResolver(client.CacheManager(), nil)
}

func runAgent(ctx context.Context, agentID string, extraArgs []string, extraEnv map[string]string, platformOverride string, noDownload bool) error {
	client := getRegistryClient()
	res := getResolver(client)

	agent, err := client.GetAgent(ctx, agentID)
	if err != nil {
		return err
	}

	resolved, err := res.Resolve(ctx, agent, resolver.ResolveOptions{
		Platform:   platformOverride,
		ExtraArgs:  extraArgs,
		ExtraEnv:   extraEnv,
		NoDownload: noDownload,
	})
	if err != nil {
		return fmt.Errorf("failed to resolve agent %q: %w", agentID, err)
	}

	r := runner.NewRunner()
	exitCode, err := r.Run(ctx, resolved)
	if err != nil {
		return fmt.Errorf("agent execution failed: %w", err)
	}

	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}
