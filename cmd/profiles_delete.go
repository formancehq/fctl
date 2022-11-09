package cmd

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newProfilesDeleteCommand() *cobra.Command {
	return newCommand("delete",
		withArgs(cobra.ExactArgs(1)),
		withRunE(func(cmd *cobra.Command, args []string) error {

			config, err := getConfig()
			if err != nil {
				return err
			}
			if err := config.DeleteProfile(args[0]); err != nil {
				return err
			}

			if err := getConfigManager().UpdateConfig(config); err != nil {
				return errors.Wrap(err, "updating config")
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Profile deleted.")
			return nil
		}),
	)
}
