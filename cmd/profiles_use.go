package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newProfilesUseCommand() *cobra.Command {
	return newCommand("use",
		withArgs(cobra.ExactArgs(1)),
		withRunE(func(cmd *cobra.Command, args []string) error {
			config, err := getConfig()
			if err != nil {
				return err
			}
			config.CurrentProfile = args[0]
			return errors.Wrap(getConfigManager().UpdateConfig(config), "Updating config")
		}),
	)
}
