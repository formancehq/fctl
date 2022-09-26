package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var useProfileCommand = &cobra.Command{
	Use:  "use",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		config.CurrentProfile = args[0]
		return errors.Wrap(configManager.UpdateConfig(config), "Updating config")
	},
}

func init() {
	profilesCommand.AddCommand(useProfileCommand)
}
