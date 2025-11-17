package versions

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/pkg"
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
