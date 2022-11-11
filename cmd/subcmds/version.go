package subcmds

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/spf13/cobra"
)

var (
	Version   = "develop"
	BuildDate = "-"
	Commit    = "-"
)

func NewVersionCommand() *cobra.Command {
	return cmdbuilder.NewCommand("version",
		cmdbuilder.WithShortDescription("Get version"),
		cmdbuilder.WithRun(func(cmd *cobra.Command, args []string) {
			fmt.Printf("Version: %s \n", Version)
			fmt.Printf("Date: %s \n", BuildDate)
			fmt.Printf("Commit: %s \n", Commit)
		}),
	)
}
