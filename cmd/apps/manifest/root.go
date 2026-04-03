package manifest

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewCommand("manifest",
		fctl.WithShortDescription("Manage app manifests"),
		fctl.WithChildCommands(
			NewPush(),
			NewList(),
		),
	)
}
