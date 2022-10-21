package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newProfilesListCommand() *cobra.Command {
	return newCommand("list",
		withRunE(func(cmd *cobra.Command, args []string) error {
			for p := range config.Profiles {
				fmt.Fprint(cmd.OutOrStdout(), "- ", p)
				if config.CurrentProfile == p {
					fmt.Fprint(cmd.OutOrStdout(), " *")
				}
				fmt.Fprintln(cmd.OutOrStdout())
			}
			return nil
		}))
}
