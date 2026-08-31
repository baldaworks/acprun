package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/baldaworks/acprun/internal/registry"
)

func newCacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage local cache directory and stored manifests/binaries",
	}

	cmd.AddCommand(newCachePathCmd())
	cmd.AddCommand(newCacheCleanCmd())

	return cmd
}

func newCachePathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the absolute path to the local ACP cache directory",
		Run: func(cmd *cobra.Command, args []string) {
			client := getRegistryClient()
			fmt.Println(client.CacheManager().RootDir())
		},
	}
}

func newCacheCleanCmd() *cobra.Command {
	var all bool
	var manifestsOnly bool
	var downloadsOnly bool

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Purge cached manifests and downloaded files",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getRegistryClient()
			cm := client.CacheManager()

			if !all && !manifestsOnly && !downloadsOnly {
				// Default to cleaning manifests and downloads, keeping extracted binaries unless --all is specified
				downloadsOnly = true
				manifestsOnly = true
			}

			err := cm.Clean(registry.CleanOptions{
				All:           all,
				ManifestsOnly: manifestsOnly,
				DownloadsOnly: downloadsOnly,
			})
			if err != nil {
				return fmt.Errorf("failed to clean cache: %w", err)
			}

			fmt.Println("Cache cleaned successfully.")
			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Purge all cache contents including extracted agent binaries")
	cmd.Flags().BoolVar(&manifestsOnly, "manifests-only", false, "Purge only cached registry manifests")
	cmd.Flags().BoolVar(&downloadsOnly, "downloads-only", false, "Purge only downloaded raw archives")

	return cmd
}
