package profiles

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewDeleteCommand() *cobra.Command {
	return internal.NewCommand("delete",
		internal.WithArgs(cobra.ExactArgs(1)),
		internal.WithShortDescription("Delete a profile"),
		internal.WithValidArgsFunction(ProfileNamesAutoCompletion),
		internal.WithRunE(func(cmd *cobra.Command, args []string) error {

			config, err := internal.Get(cmd)
			if err != nil {
				return err
			}
			if err := config.DeleteProfile(args[0]); err != nil {
				return err
			}

			if err := config.Persist(); err != nil {
				return errors.Wrap(err, "updating config")
			}
			internal.Success(cmd.OutOrStdout(), "Profile deleted!")
			return nil
		}),
	)
}
