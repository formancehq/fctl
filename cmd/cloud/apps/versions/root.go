package versions

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return fctl.NewCommand("versions",
		fctl.WithShortDescription("Manage app versions"),
		fctl.WithChildCommands(
			NewList(),
			NewShow(),
			NewArchive(),
			NewManifest(),
		),
	)
}
