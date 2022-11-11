package profiles

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewUseCommand() *cobra.Command {
	return cmdbuilder.NewCommand("use",
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithShortDescription("Use profile"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			config, err := config.Get()
			if err != nil {
				return err
			}

			config.SetCurrentProfileName(args[0])
			if err := config.Persist(); err != nil {
				return errors.Wrap(err, "Updating config")
			}
			cmdbuilder.Success(cmd.OutOrStdout(), "Selected profile updated!")
			return nil
		}),
	)
}
