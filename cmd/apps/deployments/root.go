package deployments

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewCommand("deployments",
		fctl.WithShortDescription("Manage app deployments"),
		fctl.WithChildCommands(
			NewCreate(),
			NewList(),
			NewShow(),
			NewDelete(),
			NewDeploy(),
		),
	)
}
