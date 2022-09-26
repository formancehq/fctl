package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var renameProfileCommand = &cobra.Command{
	Use:  "rename",
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		oldName := args[0]
		newName := args[1]

		p, ok := config.Profiles[oldName]
		if !ok {
			return errors.New("profile not found")
		}

		config.Profiles[newName] = p
		delete(config.Profiles, oldName)
		if config.CurrentProfile == oldName {
			config.CurrentProfile = newName
		}

		return errors.Wrap(configManager.UpdateConfig(config), "Updating config")
	},
}

func init() {
	profilesCommand.AddCommand(renameProfileCommand)
}
