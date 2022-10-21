package cmd

import (
	cmdctx "github.com/formancehq/fctl/pkg"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newProfilesRenameCommand() *cobra.Command {
	return newCommand("rename",
		withArgs(cobra.ExactArgs(2)),
		withRunE(func(cmd *cobra.Command, args []string) error {
			oldName := args[0]
			newName := args[1]

			p, ok := cmdctx.ConfigFromContext(cmd.Context()).Profiles[oldName]
			if !ok {
				return errors.New("profile not found")
			}

			cmdctx.ConfigFromContext(cmd.Context()).Profiles[newName] = p
			delete(cmdctx.ConfigFromContext(cmd.Context()).Profiles, oldName)
			if cmdctx.ConfigFromContext(cmd.Context()).CurrentProfile == oldName {
				cmdctx.ConfigFromContext(cmd.Context()).CurrentProfile = newName
			}

			return errors.Wrap(cmdctx.ConfigManagerFromContext(cmd.Context()).
				UpdateConfig(cmdctx.ConfigFromContext(cmd.Context())), "Updating config")
		}),
	)
}
