package cmd

import (
	cmdctx "github.com/formancehq/fctl/pkg"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newProfilesUseCommand() *cobra.Command {
	return newCommand("use",
		withArgs(cobra.ExactArgs(1)),
		withRunE(func(cmd *cobra.Command, args []string) error {
			cmdctx.ConfigFromContext(cmd.Context()).CurrentProfile = args[0]
			return errors.Wrap(cmdctx.ConfigManagerFromContext(cmd.Context()).
				UpdateConfig(cmdctx.ConfigFromContext(cmd.Context())), "Updating config")
		}),
	)
}
