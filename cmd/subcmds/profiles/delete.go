package profiles

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/formancehq/fctl/cmd/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newProfilesDeleteCommand() *cobra.Command {
	return cmdbuilder.NewCommand("delete",
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {

			config, err := config.Get()
			if err != nil {
				return err
			}
			if err := config.DeleteProfile(args[0]); err != nil {
				return err
			}

			if err := config.Persist(); err != nil {
				return errors.Wrap(err, "updating config")
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Profile deleted.")
			return nil
		}),
	)
}
