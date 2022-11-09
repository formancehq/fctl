package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newProfilesListCommand() *cobra.Command {
	return newCommand("list",
		withRunE(func(cmd *cobra.Command, args []string) error {

			config, err := getConfig()
			if err != nil {
				return err
			}

			currentProfileName, err := getCurrentProfileName()
			if err != nil {
				return err
			}

			for p := range config.GetProfiles() {
				fmt.Fprint(cmd.OutOrStdout(), "- ", p)
				if currentProfileName == p {
					fmt.Fprint(cmd.OutOrStdout(), " *")
				}
				fmt.Fprintln(cmd.OutOrStdout())
			}
			return nil
		}))
}
