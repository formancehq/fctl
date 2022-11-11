package profiles

import (
	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/formancehq/fctl/cmd/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newProfilesRenameCommand() *cobra.Command {
	return cmdbuilder.NewCommand("rename",
		cmdbuilder.WithArgs(cobra.ExactArgs(2)),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			oldName := args[0]
			newName := args[1]

			config, err := config.GetConfig()
			if err != nil {
				return err
			}

			p := config.GetProfile(oldName)
			if p == nil {
				return errors.New("profile not found")
			}

			if err := config.DeleteProfile(oldName); err != nil {
				return err
			}
			if config.GetCurrentProfileName() == oldName {
				config.SetCurrentProfile(newName, p)
			} else {
				config.SetProfile(newName, p)
			}

			return errors.Wrap(config.Persist(), "Updating config")
		}),
	)
}
