package cmd

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var deleteProfileCommand = &cobra.Command{
	Use:  "delete",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.DeleteProfile(args[0]); err != nil {
			return err
		}
		if err := configManager.UpdateConfig(config); err != nil {
			return errors.Wrap(err, "updating config")
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Profile deleted.")
		return nil
	},
}

func init() {
	profilesCommand.AddCommand(deleteProfileCommand)
}
