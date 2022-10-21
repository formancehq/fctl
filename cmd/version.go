package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version   = "develop"
	BuildDate = "-"
	Commit    = "-"
)

func newVersionCommand() *cobra.Command {
	return newCommand("version",
		withShortDescription("Get version"),
		withRun(func(cmd *cobra.Command, args []string) {
			fmt.Printf("Version: %s \n", Version)
			fmt.Printf("Date: %s \n", BuildDate)
			fmt.Printf("Commit: %s \n", Commit)
		}),
	)
}
