package profiles

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newProfilesUseCommand() *cobra.Command {
	return cmdbuilder.NewCommand("use",
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			config, err := config.Get()
			if err != nil {
				return err
			}
			config.SetCurrentProfileName(args[0])
			return errors.Wrap(config.Persist(), "Updating config")
		}),
	)
}
