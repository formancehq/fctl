package transferinitiation

import (
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewTransferInitiationCommand() *cobra.Command {
	return fctl.NewCommand("transfer_initiation",
		fctl.WithAliases("ti"),
		fctl.WithShortDescription("Transfer Initiation management"),
		fctl.WithChildCommands(
			NewApproveCommand(),
			NewCreateCommand(),
			NewDeleteCommand(),
			NewListCommand(),
			NewRejectCommand(),
			NewRetryCommand(),
			NewShowCommand(),
			NewUpdateStatusCommand(),
			NewReverseCommand(),
		),
	)
}
