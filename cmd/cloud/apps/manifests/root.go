package manifests

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/cloud/apps/manifests/versions"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewCommand() *cobra.Command {
	return fctl.NewCommand("manifests",
		fctl.WithShortDescription("Manage manifests"),
		fctl.WithAliases("manifest", "mf"),
		fctl.WithChildCommands(
			NewList(),
			NewShow(),
			NewCreate(),
			NewDelete(),
			NewUpdate(),
			versions.NewCommand(),
		),
	)
}
