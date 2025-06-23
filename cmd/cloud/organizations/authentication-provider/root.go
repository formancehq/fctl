package authentication_provider

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return fctl.NewMembershipCommand("authentication-provider",
		fctl.WithShortDescription("Authentication provider management"),
		fctl.WithChildCommands(
			NewConfigureCommand(),
			NewDeleteCommand(),
			NewShowCommand(),
		),
	)
}
