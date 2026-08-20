package psu

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewPSUCommand() *cobra.Command {
	return fctl.NewCommand("psu",
		fctl.WithAliases("payment-service-users", "users", "user"),
		fctl.WithShortDescription("Manage payment service users (open banking)"),
		fctl.WithChildCommands(
			NewCreateCommand(),
			NewListCommand(),
			NewShowCommand(),
			NewDeleteCommand(),
			NewForwardCommand(),
			NewCreateLinkCommand(),
		),
	)
}
