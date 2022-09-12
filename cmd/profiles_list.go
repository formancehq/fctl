package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listProfilesCommand = &cobra.Command{
	Use: "list",
	RunE: func(cmd *cobra.Command, args []string) error {
		for p := range config.Profiles {
			fmt.Println("-", p)
		}
		return nil
	},
}

func init() {
	profilesCommand.AddCommand(listProfilesCommand)
}
