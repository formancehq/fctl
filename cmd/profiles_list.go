package cmd

import (
	"fmt"

	"github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
)

func newProfilesListCommand() *cobra.Command {
	return newCommand("list",
		withRunE(func(cmd *cobra.Command, args []string) error {
			for p := range fctl.ConfigFromContext(cmd.Context()).Profiles {
				fmt.Fprint(cmd.OutOrStdout(), "- ", p)
				if fctl.ConfigFromContext(cmd.Context()).CurrentProfile == p {
					fmt.Fprint(cmd.OutOrStdout(), " *")
				}
				fmt.Fprintln(cmd.OutOrStdout())
			}
			return nil
		}))
}
