package authentication_provider

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
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
