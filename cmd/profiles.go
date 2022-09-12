package cmd

import (
	"github.com/spf13/cobra"
)

var profilesCommand = &cobra.Command{
	Use: "profiles",
}

func init() {
	rootCommand.AddCommand(profilesCommand)
}
