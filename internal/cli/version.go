package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Version info set at build time or defaults.
var (
	Version = "1.0.0"
	Commit  = "none"
	Date    = "unknown"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version and build information for acprun",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("acprun %s (commit: %s, built: %s, os/arch: %s/%s)\n",
				Version, Commit, Date, runtime.GOOS, runtime.GOARCH)
		},
	}
}
