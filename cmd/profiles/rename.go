package profiles

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewRenameCommand() *cobra.Command {
	return cmdbuilder.NewCommand("rename",
		cmdbuilder.WithArgs(cobra.ExactArgs(2)),
		cmdbuilder.WithShortDescription("Rename a profile"),
		cmdbuilder.WithValidArgsFunction(ProfileNamesAutoCompletion),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			oldName := args[0]
			newName := args[1]

			config, err := config.Get(cmd)
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

			if err := config.Persist(); err != nil {
				return errors.Wrap(config.Persist(), "Updating config")
			}
			return nil
		}),
	)
}
