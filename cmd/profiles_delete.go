package cmd

import (
	"fmt"

	"github.com/formancehq/fctl/pkg"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newProfilesDeleteCommand() *cobra.Command {
	return newCommand("delete",
		withArgs(cobra.ExactArgs(1)),
		withRunE(func(cmd *cobra.Command, args []string) error {
			if err := fctl.ConfigFromContext(cmd.Context()).DeleteProfile(args[0]); err != nil {
				return err
			}
			if err := fctl.ConfigManagerFromContext(cmd.Context()).
				UpdateConfig(fctl.ConfigFromContext(cmd.Context())); err != nil {
				return errors.Wrap(err, "updating config")
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Profile deleted.")
			return nil
		}),
	)
}
