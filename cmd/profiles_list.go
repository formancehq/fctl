package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newProfilesListCommand() *cobra.Command {
	return newCommand("list",
		withRunE(func(cmd *cobra.Command, args []string) error {

			fmt.Println(viper.GetString(profileFlag))

			config, err := getConfig()
			if err != nil {
				return err
			}

			for p := range config.Profiles {
				fmt.Fprint(cmd.OutOrStdout(), "- ", p)
				if config.CurrentProfile == p {
					fmt.Fprint(cmd.OutOrStdout(), " *")
				}
				fmt.Fprintln(cmd.OutOrStdout())
			}
			return nil
		}))
}
