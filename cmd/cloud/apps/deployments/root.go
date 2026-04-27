package deployments

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewCommand("deployments",
		fctl.WithShortDescription("Manage deployments"),
		fctl.WithAliases("deploy", "dep"),
		fctl.WithChildCommands(
			NewCreate(),
			NewList(),
			NewShow(),
			NewLogs(),
		),
	)
}
