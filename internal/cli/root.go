// Package cli provides Cobra command implementations for acprun.
package cli

import (
	"context"
	"fmt"
	"os"

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
	rootCmd := NewRootCmd()
	ctx := context.Background()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
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
