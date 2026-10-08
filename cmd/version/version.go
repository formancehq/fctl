package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Release builds inject these values through GoReleaser ldflags.
var (
	Version   = "v4.0.0-dev"
	Commit    = "-"
	BuildDate = "-"
)

func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print fctl version and build information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "fctl %s (commit: %s, built: %s)\n", Version, Commit, BuildDate)
			return err
		},
	}
}
